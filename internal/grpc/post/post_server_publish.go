package postgrpc

import (
	"context"
	"time"

	"github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (s *PostServer) Publish(ctx context.Context, request *protopkg.PublishPostRequest) (*protopkg.PublishPostResponse, error) {
	post, err := s.db.SelectPostByID(request.PostId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	if post.AuthorID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "not an owner")
	}

	post.Status = int(orm.PostStatusPublished)
	post.PublishedAt = time.Now()

	if err := s.db.UpdatePost(post); err != nil {
		s.log.Error("error publishing post", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not publish post")
	}
	return &protopkg.PublishPostResponse{}, nil
}
