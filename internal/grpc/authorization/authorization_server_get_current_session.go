package grpcauthorization

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *AuthorizationServer) GetCurrentSession(ctx context.Context, req *protopkg.GetCurrentSessionRequest) (*protopkg.GetCurrentSessionResponse, error) {
	sessionIDStr, err := middlewarepkg.GetSessionID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "not authenticated")
	}
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid session ID in token")
	}

	session, err := s.database.SelectSessionByID(sessionID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "session not found")
		}
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &protopkg.GetCurrentSessionResponse{
			Session: &protopkg.Session{
				SessionId: session.ID.String(),
				UserAgent: session.UserAgent,
				IpAddress: session.IpAddress,
				CreatedAt: timestamppb.New(session.CreatedAt),
				UpdatedAt: timestamppb.New(session.UpdatedAt),
			},
		},
		nil
}
