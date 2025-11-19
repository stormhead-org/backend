package grpcauthorization

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *AuthorizationServer) RefreshToken(ctx context.Context, req *protopkg.RefreshTokenRequest) (*protopkg.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Errorf(codes.InvalidArgument, "refresh token is required")
	}

	sessionID, err := s.jwt.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token")
	}

	newAccessToken, err := s.jwt.GenerateAccessToken(sessionID)
	if err != nil {
		s.log.Error("failed to generate new access token", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	newRefreshToken, err := s.jwt.GenerateRefreshToken(sessionID)
	if err != nil {
		s.log.Error("failed to generate new refresh token", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &protopkg.RefreshTokenResponse{
			AccessToken:  newAccessToken,
			RefreshToken: newRefreshToken,
		},
		nil
}
