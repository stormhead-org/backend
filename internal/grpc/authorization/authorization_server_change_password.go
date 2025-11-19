package grpcauthorization

import (
	"context"

	"github.com/google/uuid"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	securitypkg "github.com/stormhead-org/backend/internal/security"
)

func (s *AuthorizationServer) ChangePassword(ctx context.Context, req *protopkg.ChangePasswordRequest) (*protopkg.ChangePasswordResponse, error) {
	userIDStr, err := middlewarepkg.GetUserID(ctx)
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

	user, err := s.database.SelectUserByID(userID.String())
	if err != nil {
		s.log.Error("failed to retrieve user for password change", zap.Error(err), zap.String("userID", userID.String()))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	// Check old password
	err = securitypkg.ComparePasswords(
		user.Password,
		req.OldPassword,
		user.Salt,
	)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "old password invalid")
	}

	// Validate new password complexity
	if len(req.NewPassword) < 12 {
		return nil, status.Errorf(codes.InvalidArgument, "new password must be at least 12 characters long")
	}

	// Check if new password has been pwned
	isPwned, err := s.hibp.IsPasswordPwned(req.NewPassword)
	if err != nil {
		s.log.Error("failed to check new password against HIBP", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to validate new password")
	}
	if isPwned {
		return nil, status.Errorf(codes.InvalidArgument, "new password has been pwned, please choose a different one")
	}

	// Salt new password
	salt := securitypkg.GenerateSalt()

	hash, err := securitypkg.HashPassword(
		req.NewPassword,
		salt,
	)
	if err != nil {
		s.log.Error("failed to hash new password", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	// Update user in database
	user.Password = hash
	user.Salt = salt

	err = s.database.UpdateUser(user)
	if err != nil {
		s.log.Error("failed to update user with new password", zap.Error(err), zap.String("userID", userID.String()))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	// Revoke all active sessions for the user
	err = s.database.DeleteSessionsByUserID(userID.String())
	if err != nil {
		s.log.Error("failed to delete user sessions after password change", zap.Error(err), zap.String("userID", userID.String()))
		// Do not return an error here, as the password change was successful.
		// Logging the error is sufficient.
	}

	return &protopkg.ChangePasswordResponse{}, nil
}
