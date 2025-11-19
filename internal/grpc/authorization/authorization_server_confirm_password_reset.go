package grpcauthorization

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	protopkg "github.com/stormhead-org/backend/internal/proto"
	securitypkg "github.com/stormhead-org/backend/internal/security"
)

func (s *AuthorizationServer) ConfirmPasswordReset(ctx context.Context, req *protopkg.ConfirmResetPasswordRequest) (*protopkg.ConfirmResetPasswordResponse, error) {
	if req.Token == "" {
		return nil, status.Errorf(codes.InvalidArgument, "reset token is required")
	}

	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "new password is required")
	}

	user, err := s.database.SelectUserByResetToken(req.Token)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "user not exist or token is invalid")
	}

	if user.ResetTokenExpiresAt == nil || time.Now().After(*user.ResetTokenExpiresAt) {
		return nil, status.Errorf(codes.InvalidArgument, "reset token expired or invalid")
	}

	if len(req.Password) < 12 {
		return nil, status.Errorf(codes.InvalidArgument, "password must be at least 12 characters long")
	}

	isPwned, err := s.hibp.IsPasswordPwned(req.Password)
	if err != nil {
		s.log.Error("failed to check password against HIBP", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to validate password")
	}
	if isPwned {
		return nil, status.Errorf(codes.InvalidArgument, "password has been pwned, please choose a different one")
	}

	salt := securitypkg.GenerateSalt()
	hash, err := securitypkg.HashPassword(req.Password, salt)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	user.Password = hash
	user.Salt = salt
	user.ResetToken = ""
	user.ResetTokenExpiresAt = nil

	if err := s.database.UpdateUser(user); err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	if err := s.database.DeleteSessionsByUserID(user.ID.String()); err != nil {
		s.log.Error("failed to delete user sessions after password reset", zap.Error(err))
	}

	return &protopkg.ConfirmResetPasswordResponse{}, nil
}
