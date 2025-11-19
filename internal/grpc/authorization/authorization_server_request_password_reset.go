package grpcauthorization

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	eventpkg "github.com/stormhead-org/backend/internal/event"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	securitypkg "github.com/stormhead-org/backend/internal/security"
)

func (s *AuthorizationServer) RequestPasswordReset(ctx context.Context, req *protopkg.RequestPasswordResetRequest) (*protopkg.RequestPasswordResetResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}

	// Get user from database
	user, err := s.database.SelectUserByEmail(
		req.Email,
	)
	if err != nil {
		// Always return success to prevent enumeration attacks, as specified in T043.
		// If the user is not found, we still return success but do not perform any reset actions.
		s.log.Warn("password reset requested for non-existent user", zap.String("email", req.Email))
		return &protopkg.RequestPasswordResetResponse{}, nil
	}
	if !user.IsVerified {
		// Always return success to prevent enumeration attacks.
		// If the user is not verified, we still return success but do not perform any reset actions.
		s.log.Warn("password reset requested for unverified user", zap.String("email", req.Email), zap.String("userID", user.ID.String()))
		return &protopkg.RequestPasswordResetResponse{}, nil
	}

	// Update user with reset token
	user.ResetToken = securitypkg.GenerateToken()
	expiresAt := time.Now().Add(time.Hour) // Token valid for 1 hour
	user.ResetTokenExpiresAt = &expiresAt

	err = s.database.UpdateUser(user)
	if err != nil {
		s.log.Error("failed to update user with reset token", zap.Error(err), zap.String("userID", user.ID.String()))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	// Write message to broker for email sending
	err = s.broker.WriteMessage(
		ctx,
		eventpkg.AUTHORIZATION_REQUEST_PASSWORD_RESET,
		eventpkg.AuthorizationRequestPasswordReset{
			ID: user.ID.String(),
		},
	)
	if err != nil {
		s.log.Error("failed to write password reset event to broker", zap.Error(err), zap.String("userID", user.ID.String()))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &protopkg.RequestPasswordResetResponse{}, nil
}
