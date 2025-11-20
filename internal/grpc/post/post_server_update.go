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

func (s *PostServer) Update(ctx context.Context, request *protopkg.UpdatePostRequest) (*protopkg.UpdatePostResponse, error) {
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

	if request.Title != "" {
		post.Title = request.Title
	}

	if request.Content != nil {
		jsonBytes, err := request.Content.MarshalJSON()
		if err != nil {
			s.log.Error("failed to marshal content to JSON", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to process content")
		}
		post.Content = jsonBytes
	}

	if err := s.db.UpdatePost(post); err != nil {
		s.log.Error("error updating post", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not update post")
	}
	return &protopkg.UpdatePostResponse{}, nil
}
