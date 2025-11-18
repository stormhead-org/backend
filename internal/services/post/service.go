package post

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/stormhead-org/backend/internal/lib"
	"github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/orm"
	"github.com/stormhead-org/backend/internal/services"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type PostServiceImpl struct {
	db  *orm.PostgresClient
	log *zap.Logger
}

func NewPostService(db *orm.PostgresClient, log *zap.Logger) services.PostService {
	return &PostServiceImpl{
		db:  db,
		log: log,
	}
}

func (s *PostServiceImpl) CreatePost(ctx context.Context, communityIDStr string, title string, content json.RawMessage) (*orm.Post, error) {
	communityUUID, err := uuid.Parse(communityIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid community_id")
	}

	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	// Check if community exists and user is a member
	_, err = s.db.SelectCommunityUser(communityIDStr, userID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.PermissionDenied, "user is not a member of the community")
		}
		s.log.Error("error checking community membership", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	post := &orm.Post{
		CommunityID: communityUUID,
		AuthorID:    userID,
		Title:       title,
		Content:     content,
		Status:      int(orm.PostStatusDraft),
	}

	if err := s.db.InsertPost(post); err != nil {
		s.log.Error("error inserting post", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create post")
	}
	return post, nil
}

func (s *PostServiceImpl) GetPost(ctx context.Context, postID string) (*orm.Post, error) {
	post, err := s.db.SelectPostByID(postID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}
	return post, nil
}

func (s *PostServiceImpl) UpdatePost(ctx context.Context, postID string, title *string, content json.RawMessage) (*orm.Post, error) {
	post, err := s.db.SelectPostByID(postID)
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

	if title != nil {
		post.Title = *title
	}
	if content != nil {
		post.Content = content
	}

	if err := s.db.UpdatePost(post); err != nil {
		s.log.Error("error updating post", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not update post")
	}
	return post, nil
}

func (s *PostServiceImpl) DeletePost(ctx context.Context, postID string) error {
	post, err := s.db.SelectPostByID(postID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	if post.AuthorID != userID {
		// TODO: Add moderator check (Phase 5)
		return status.Errorf(codes.PermissionDenied, "not an author")
	}

	if err := s.db.DeletePost(post); err != nil {
		s.log.Error("error deleting post", zap.Error(err))
		return status.Errorf(codes.Internal, "could not delete post")
	}
	return nil
}

func (s *PostServiceImpl) PublishPost(ctx context.Context, postID string) error {
	post, err := s.db.SelectPostByID(postID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	if post.AuthorID != userID {
		return status.Errorf(codes.PermissionDenied, "not an owner")
	}

	post.Status = int(orm.PostStatusPublished)
	post.PublishedAt = time.Now()

	if err := s.db.UpdatePost(post); err != nil {
		s.log.Error("error publishing post", zap.Error(err))
		return status.Errorf(codes.Internal, "could not publish post")
	}
	return nil
}

func (s *PostServiceImpl) UnpublishPost(ctx context.Context, postID string) error {
	post, err := s.db.SelectPostByID(postID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	if post.AuthorID != userID {
		return status.Errorf(codes.PermissionDenied, "not an owner")
	}

	post.Status = int(orm.PostStatusDraft)

	if err := s.db.UpdatePost(post); err != nil {
		s.log.Error("error unpublishing post", zap.Error(err))
		return status.Errorf(codes.Internal, "could not unpublish post")
	}
	return nil
}

func (s *PostServiceImpl) LikePost(ctx context.Context, postID string) error {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	post, err := s.db.SelectPostByID(postID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	_, err = s.db.SelectPostLikeByID(postID, userID.String())
	if err != gorm.ErrRecordNotFound {
		if err == nil {
			// Idempotency: already liked, return success
			return nil
		}
		s.log.Error("error checking post like", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	postLike := &orm.PostLike{
		PostID: post.ID,
		UserID: userID,
	}
	if err := s.db.InsertPostLike(postLike); err != nil {
		s.log.Error("error inserting post like", zap.Error(err))
		return status.Errorf(codes.Internal, "could not like post")
	}

	// This should be a transaction
	post.LikeCount++
	if err := s.db.UpdatePost(post); err != nil {
		s.log.Error("error updating post like count", zap.Error(err))
		// TODO: Maybe try to revert the like?
		return status.Errorf(codes.Internal, "could not update like count")
	}

	author, err := s.db.SelectUserByID(post.AuthorID.String())
	if err != nil {
		s.log.Error("could not find author to update reputation", zap.Error(err))
		return nil
	}
	reputation, err := lib.CalculateUserReputation(s.db, author)
	if err != nil {
		s.log.Error("failed to calculate author reputation", zap.Error(err))
		return nil
	}
	author.Reputation = int64(reputation)
	if err := s.db.UpdateUser(author); err != nil {
		s.log.Error("failed to update author reputation", zap.Error(err))
	}
	return nil
}

func (s *PostServiceImpl) UnlikePost(ctx context.Context, postID string) error {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	post, err := s.db.SelectPostByID(postID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	postLike, err := s.db.SelectPostLikeByID(postID, userID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Idempotency: not liked, return success
			return nil
		}
		s.log.Error("error checking post like", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	if err := s.db.DeletePostLike(postLike); err != nil {
		s.log.Error("error deleting post like", zap.Error(err))
		return status.Errorf(codes.Internal, "could not unlike post")
	}

	// This should be a transaction
	if post.LikeCount > 0 {
		post.LikeCount--
	}
	if err := s.db.UpdatePost(post); err != nil {
		s.log.Error("error updating post like count", zap.Error(err))
		return status.Errorf(codes.Internal, "could not update like count")
	}
	
	author, err := s.db.SelectUserByID(post.AuthorID.String())
	if err != nil {
		s.log.Error("could not find author to update reputation", zap.Error(err))
		return nil
	}
	reputation, err := lib.CalculateUserReputation(s.db, author)
	if err != nil {
		s.log.Error("failed to calculate author reputation", zap.Error(err))
		return nil
	}
	author.Reputation = int64(reputation)
	if err := s.db.UpdateUser(author); err != nil {
		s.log.Error("failed to update author reputation", zap.Error(err))
	}
	return nil
}

func (s *PostServiceImpl) CreateBookmark(ctx context.Context, postID string) error {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	_, err = s.db.SelectPostByID(postID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	_, err = s.db.SelectBookmarkByID(postID, userID.String())
	if err != gorm.ErrRecordNotFound {
		if err == nil {
			// Idempotency: already bookmarked, return success
			return nil
		}
		s.log.Error("error checking bookmark", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	bookmark := &orm.Bookmark{
		PostID: uuid.MustParse(postID),
		UserID: userID,
	}
	if err := s.db.InsertBookmark(bookmark); err != nil {
		s.log.Error("error inserting bookmark", zap.Error(err))
		return status.Errorf(codes.Internal, "could not create bookmark")
	}
	return nil
}

func (s *PostServiceImpl) DeleteBookmark(ctx context.Context, postID string) error {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	bookmark, err := s.db.SelectBookmarkByID(postID, userID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Idempotency: not bookmarked, return success
			return nil
		}
		s.log.Error("error checking bookmark", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	if err := s.db.DeleteBookmark(bookmark); err != nil {
		s.log.Error("error deleting bookmark", zap.Error(err))
		return status.Errorf(codes.Internal, "could not delete bookmark")
	}
	return nil
}
