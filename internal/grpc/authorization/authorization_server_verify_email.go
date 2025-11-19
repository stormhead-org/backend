package grpcauthorization

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *AuthorizationServer) VerifyEmail(ctx context.Context, req *protopkg.VerifyEmailRequest) (*protopkg.VerifyEmailResponse, error) {
	if req.Token == "" {
		return nil, status.Errorf(codes.InvalidArgument, "verification token is required")
	}

	user, err := s.database.SelectUserByVerificationToken(req.Token)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "user not exist")
	}

	user.IsVerified = true
	if err := s.database.UpdateUser(user); err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &protopkg.VerifyEmailResponse{}, nil
}
