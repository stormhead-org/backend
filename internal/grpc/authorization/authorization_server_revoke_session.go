package grpcauthorization

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *AuthorizationServer) RevokeSession(ctx context.Context, req *protopkg.RevokeSessionRequest) (*protopkg.RevokeSessionResponse, error) {
	currentSessionIDStr, err := middlewarepkg.GetSessionID(ctx)
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

	// Get current user's session
	userSession, err := s.database.SelectSessionByID(currentSessionID.String())
	if err != nil {
		s.log.Error("failed to retrieve current user session", zap.Error(err), zap.String("sessionID", currentSessionID.String()))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	// Get the session to be revoked
	requestedSession, err := s.database.SelectSessionByID(sessionIDToRevoke.String())
	if err != nil {
		s.log.Error("failed to retrieve requested session to revoke", zap.Error(err), zap.String("sessionID", sessionIDToRevoke.String()))
		return nil, status.Errorf(codes.InvalidArgument, "session not found")
	}

	// Check if the session to be revoked belongs to the current user
	if userSession.UserID != requestedSession.UserID {
		s.log.Warn("attempt to revoke session belonging to another user",
			zap.String("currentUserID", userSession.UserID.String()),
			zap.String("requestedSessionUserID", requestedSession.UserID.String()),
			zap.String("sessionIDToRevoke", sessionIDToRevoke.String()))
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	// Prevent revoking the current active session
	if currentSessionID == sessionIDToRevoke {
		s.log.Warn("attempt to revoke current active session", zap.String("sessionID", currentSessionID.String()))
		return nil, status.Errorf(codes.InvalidArgument, "cannot revoke current active session")
	}

	// Delete session from database
	err = s.database.DeleteSession(requestedSession)
	if err != nil {
		s.log.Error("failed to delete session", zap.Error(err), zap.String("sessionID", sessionIDToRevoke.String()))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &protopkg.RevokeSessionResponse{}, nil
}
