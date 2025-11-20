package postgrpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (s *PostServer) CreateBookmark(ctx context.Context, request *protopkg.CreateBookmarkRequest) (*protopkg.CreateBookmarkResponse, error) {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	_, err = s.db.SelectPostByID(request.PostId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	_, err = s.db.SelectBookmarkByID(request.PostId, userID.String())
	if err != gorm.ErrRecordNotFound {
		if err == nil {
			// Idempotency: already bookmarked, return success
			return &protopkg.CreateBookmarkResponse{}, nil
		}
		s.log.Error("error checking bookmark", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	bookmark := &orm.Bookmark{
		PostID: uuid.MustParse(request.PostId),
		UserID: userID,
	}
	if err := s.db.InsertBookmark(bookmark); err != nil {
		s.log.Error("error inserting bookmark", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create bookmark")
	}
	return &protopkg.CreateBookmarkResponse{}, nil
}
