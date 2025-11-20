package postgrpc

import (
	"context"

	"github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (s *PostServer) DeleteBookmark(ctx context.Context, request *protopkg.DeleteBookmarkRequest) (*protopkg.DeleteBookmarkResponse, error) {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	bookmark, err := s.db.SelectBookmarkByID(request.PostId, userID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Idempotency: not bookmarked, return success
			return &protopkg.DeleteBookmarkResponse{}, nil
		}
		s.log.Error("error checking bookmark", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	if err := s.db.DeleteBookmark(bookmark); err != nil {
		s.log.Error("error deleting bookmark", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not delete bookmark")
	}
	return &protopkg.DeleteBookmarkResponse{}, nil
}
