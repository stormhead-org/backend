package postgrpc

import (
	"context"

	"github.com/stormhead-org/backend/internal/lib"
	"github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (s *PostServer) Unlike(ctx context.Context, request *protopkg.UnlikePostRequest) (*protopkg.UnlikePostResponse, error) {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	post, err := s.db.SelectPostByID(request.PostId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	postLike, err := s.db.SelectPostLikeByID(request.PostId, userID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Idempotency: not liked, return success
			return &protopkg.UnlikePostResponse{}, nil
		}
		s.log.Error("error checking post like", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	if err := s.db.DeletePostLike(postLike); err != nil {
		s.log.Error("error deleting post like", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not unlike post")
	}

	// This should be a transaction
	if post.LikeCount > 0 {
		post.LikeCount--
	}
	if err := s.db.UpdatePost(post); err != nil {
		s.log.Error("error updating post like count", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not update like count")
	}

	author, err := s.db.SelectUserByID(post.AuthorID.String())
	if err != nil {
		s.log.Error("could not find author to update reputation", zap.Error(err))
		return &protopkg.UnlikePostResponse{}, nil
	}
	reputation, err := lib.CalculateUserReputation(s.db, author)
	if err != nil {
		s.log.Error("failed to calculate author reputation", zap.Error(err))
		return &protopkg.UnlikePostResponse{}, nil
	}
	author.Reputation = int64(reputation)
	if err := s.db.UpdateUser(author); err != nil {
		s.log.Error("failed to update author reputation", zap.Error(err))
	}
	return &protopkg.UnlikePostResponse{}, nil
}
