package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/stormhead-org/backend/internal/jwt"
	"github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/services"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/stormhead-org/backend/internal/proto"
)

type AuthorizationServer struct {
	pb.UnimplementedAuthorizationServiceServer
	logger      *zap.Logger
	jwtManager  *jwt.JWT
	userService services.UserService
}

func NewAuthorizationServer(logger *zap.Logger, jwtManager *jwt.JWT, userService services.UserService) *AuthorizationServer {
	return &AuthorizationServer{
		logger:      logger,
		jwtManager:  jwtManager,
		userService: userService,
	}
}

func (s *AuthorizationServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	user, err := s.userService.Register(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterResponse{
		UserId: user.ID.String(),
	},
	nil
}

func (s *AuthorizationServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	user, session, err := s.userService.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(session.ID.String())
	if err != nil {
		s.logger.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(session.ID.String())
	if err != nil {
		s.logger.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &pb.LoginResponse{
		User: &pb.User{
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

func (s *AuthorizationServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	// Client handles token deletion. Server-side invalidation can be added if needed.
	return &pb.LogoutResponse{}, nil
}

func (s *AuthorizationServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Errorf(codes.InvalidArgument, "refresh token is required")
	}

	sessionID, err := s.jwtManager.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token")
	}

	newAccessToken, err := s.jwtManager.GenerateAccessToken(sessionID)
	if err != nil {
		s.logger.Error("failed to generate new access token", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(sessionID)
	if err != nil {
		s.logger.Error("failed to generate new refresh token", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	},
	nil
}

func (s *AuthorizationServer) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
	if req.Token == "" {
		return nil, status.Errorf(codes.InvalidArgument, "verification token is required")
	}

	if err := s.userService.VerifyEmail(ctx, req.Token); err != nil {
		return nil, err
	}

	return &pb.VerifyEmailResponse{}, nil
}

func (s *AuthorizationServer) RequestPasswordReset(ctx context.Context, req *pb.RequestPasswordResetRequest) (*pb.RequestPasswordResetResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}

	if err := s.userService.RequestPasswordReset(ctx, req.Email); err != nil {
		return nil, err
	}

	return &pb.RequestPasswordResetResponse{}, nil
}

func (s *AuthorizationServer) ConfirmPasswordReset(ctx context.Context, req *pb.ConfirmResetPasswordRequest) (*pb.ConfirmResetPasswordResponse, error) {
	if req.Token == "" {
		return nil, status.Errorf(codes.InvalidArgument, "reset token is required")
	}
	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "new password is required")
	}

	if err := s.userService.ConfirmPasswordReset(ctx, req.Token, req.Password); err != nil {
		return nil, err
	}

	return &pb.ConfirmResetPasswordResponse{}, nil
}

func (s *AuthorizationServer) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	userIDStr, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid user ID in token")
	}

	if req.OldPassword == "" {
		return nil, status.Errorf(codes.InvalidArgument, "old password is required")
	}
	if req.NewPassword == "" {
		return nil, status.Errorf(codes.InvalidArgument, "new password is required")
	}

	if err := s.userService.ChangePassword(ctx, userID, req.OldPassword, req.NewPassword); err != nil {
		return nil, err
	}

	return &pb.ChangePasswordResponse{}, nil
}

func (s *AuthorizationServer) GetCurrentSession(ctx context.Context, req *pb.GetCurrentSessionRequest) (*pb.GetCurrentSessionResponse, error) {
	sessionIDStr, err := middleware.GetSessionID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "not authenticated")
	}
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid session ID in token")
	}

	session, err := s.userService.GetCurrentSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return &pb.GetCurrentSessionResponse{
		Session: &pb.Session{
			SessionId: session.ID.String(),
			UserAgent: session.UserAgent,
			IpAddress: session.IpAddress,
			CreatedAt: timestamppb.New(session.CreatedAt),
			UpdatedAt: timestamppb.New(session.UpdatedAt),
		},
	},
	nil
}

func (s *AuthorizationServer) ListActiveSessions(ctx context.Context, req *pb.ListActiveSessionsRequest) (*pb.ListActiveSessionsResponse, error) {
	userIDStr, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid user ID in token")
	}

	sessions, nextCursor, err := s.userService.ListActiveSessions(ctx, userID, req.Cursor, int(req.Limit))
	if err != nil {
		return nil, err
	}

	pbSessions := make([]*pb.Session, len(sessions))
	for i, session := range sessions {
		pbSessions[i] = &pb.Session{
			SessionId: session.ID.String(),
			UserAgent: session.UserAgent,
			IpAddress: session.IpAddress,
			CreatedAt: timestamppb.New(session.CreatedAt),
			UpdatedAt: timestamppb.New(session.UpdatedAt),
		}
	}

	return &pb.ListActiveSessionsResponse{
		Sessions:   pbSessions,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	},
	nil
}

func (s *AuthorizationServer) RevokeSession(ctx context.Context, req *pb.RevokeSessionRequest) (*pb.RevokeSessionResponse, error) {
	currentSessionIDStr, err := middleware.GetSessionID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "not authenticated")
	}
	currentSessionID, err := uuid.Parse(currentSessionIDStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid session ID in token")
	}

	sessionIDToRevoke, err := uuid.Parse(req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid session ID format")
	}

	if err := s.userService.RevokeSession(ctx, currentSessionID, sessionIDToRevoke); err != nil {
		return nil, err
	}

	return &pb.RevokeSessionResponse{}, nil
}

