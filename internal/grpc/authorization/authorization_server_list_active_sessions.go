package grpcauthorization

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *AuthorizationServer) ListActiveSessions(ctx context.Context, req *protopkg.ListActiveSessionsRequest) (*protopkg.ListActiveSessionsResponse, error) {
	userIDStr, err := middlewarepkg.GetUserID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid user ID in token")
	}

	const SESSIONS_PER_PAGE = 10
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = SESSIONS_PER_PAGE
	}
	sessions, err := s.database.SelectSessionsByUserID(userID.String(), req.Cursor, int(req.Limit)+1)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	var nextCursor string
	if len(sessions) > int(req.Limit) {
		nextCursor = sessions[req.Limit].ID.String()
		sessions = sessions[:req.Limit]
	}

	pbSessions := make([]*protopkg.Session, len(sessions))
	for i, session := range sessions {
		pbSessions[i] = &protopkg.Session{
			SessionId: session.ID.String(),
			UserAgent: session.UserAgent,
			IpAddress: session.IpAddress,
			CreatedAt: timestamppb.New(session.CreatedAt),
			UpdatedAt: timestamppb.New(session.UpdatedAt),
		}
	}

	return &protopkg.ListActiveSessionsResponse{
			Sessions:   pbSessions,
			NextCursor: nextCursor,
			HasMore:    nextCursor != "",
		},
		nil
}
