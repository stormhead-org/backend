package postgrpc

import (
	"context"

	"github.com/stormhead-org/backend/internal/lib"
	"github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (s *PostServer) Like(ctx context.Context, request *protopkg.LikePostRequest) (*protopkg.LikePostResponse, error) {
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

	_, err = s.db.SelectPostLikeByID(request.PostId, userID.String())
	if err != gorm.ErrRecordNotFound {
		if err == nil {
			// Idempotency: already liked, return success
			return &protopkg.LikePostResponse{}, nil
		}
		s.log.Error("error checking post like", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	postLike := &orm.PostLike{
		PostID: post.ID,
		UserID: userID,
	}
	if err := s.db.InsertPostLike(postLike); err != nil {
		s.log.Error("error inserting post like", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not like post")
	}

	// This should be a transaction
	post.LikeCount++
	if err := s.db.UpdatePost(post); err != nil {
		s.log.Error("error updating post like count", zap.Error(err))
		// TODO: Maybe try to revert the like?
		return nil, status.Errorf(codes.Internal, "could not update like count")
	}

	author, err := s.db.SelectUserByID(post.AuthorID.String())
	if err != nil {
		s.log.Error("could not find author to update reputation", zap.Error(err))
		return &protopkg.LikePostResponse{}, nil
	}
	reputation, err := lib.CalculateUserReputation(s.db, author)
	if err != nil {
		s.log.Error("failed to calculate author reputation", zap.Error(err))
		return &protopkg.LikePostResponse{}, nil
	}
	author.Reputation = int64(reputation)
	if err := s.db.UpdateUser(author); err != nil {
		s.log.Error("failed to update author reputation", zap.Error(err))
	}
	return &protopkg.LikePostResponse{}, nil
}
