package grpcauthorization

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	eventpkg "github.com/stormhead-org/backend/internal/event"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	securitypkg "github.com/stormhead-org/backend/internal/security"
)

func (s *AuthorizationServer) Login(ctx context.Context, req *protopkg.LoginRequest) (*protopkg.LoginResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	// Get user from database
	user, err := s.database.SelectUserByEmail(
		req.Email,
	)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "user not found")
	}
	if !user.IsVerified {
		return nil, status.Errorf(codes.InvalidArgument, "user not verified")
	}

	err = securitypkg.ComparePasswords(
		user.Password,
		req.Password,
		user.Salt,
	)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "password invalid")
	}

	// Obtain user agent and ip address
	userAgent := "unknown"
	m, ok := metadata.FromIncomingContext(ctx)
	if ok {
		userAgent = strings.Join(m["user-agent"], "")
	}

	ipAddress := "unknown"
	p, ok := peer.FromContext(ctx)
	if ok {
		parts := strings.Split(p.Addr.String(), ":")
		if len(parts) == 2 {
			ipAddress = parts[0]
		}
	}

	if userAgent == "unknown" || ipAddress == "unknown" {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	// Check existing sessions
	sessions, err := s.database.SelectSessionsByUserID(user.ID.String(), "", 0)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	for _, session := range sessions {
		if session.IpAddress != ipAddress {
			continue
		}

		if session.UserAgent != userAgent {
			continue
		}

		s.log.Error("multiple login attempt from same client")
		// return nil, status.Errorf(codes.Internal, "multiple login attempt from same client")
	}

	// Create session
	session := ormpkg.Session{
		UserID:    user.ID,
		UserAgent: userAgent,
		IpAddress: ipAddress,
	}
	err = s.database.InsertSession(&session)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	// Write message to broker
	err = s.broker.WriteMessage(
		ctx,
		eventpkg.AUTHORIZATION_LOGIN,
		eventpkg.AuthorizationLoginMessage{
			ID: user.ID.String(),
		},
	)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	accessToken, err := s.jwt.GenerateAccessToken(session.ID.String())
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(session.ID.String())
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &protopkg.LoginResponse{
			User: &protopkg.User{
				Id:          user.ID.String(),
				Slug:        user.Slug,
				Name:        user.Name,
				Description: user.Description,
				Email:       user.Email,
				IsVerified:  user.IsVerified,
			},
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
		nil
}
