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

func (s *PostServer) Delete(ctx context.Context, request *protopkg.DeletePostRequest) (*protopkg.DeletePostResponse, error) {
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
		// TODO: Add moderator check (Phase 5)
		return nil, status.Errorf(codes.PermissionDenied, "not an author")
	}

	if err := s.db.DeletePost(post); err != nil {
		s.log.Error("error deleting post", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not delete post")
	}
	return &protopkg.DeletePostResponse{}, nil
}
