package services

import (
	"context"
	"encoding/json"

	"github.com/stormhead-org/backend/internal/orm"
)

// PostService defines the interface for post-related operations.
type PostService interface {
	CreatePost(ctx context.Context, communityIDStr string, title string, content json.RawMessage) (*orm.Post, error)
	GetPost(ctx context.Context, postID string) (*orm.Post, error)
	UpdatePost(ctx context.Context, postID string, title *string, content json.RawMessage) (*orm.Post, error)
	DeletePost(ctx context.Context, postID string) error
	PublishPost(ctx context.Context, postID string) error
	UnpublishPost(ctx context.Context, postID string) error
	// ListUserPosts(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]orm.Post, string, error)
	LikePost(ctx context.Context, postID string) error
	UnlikePost(ctx context.Context, postID string) error
	CreateBookmark(ctx context.Context, postID string) error
	DeleteBookmark(ctx context.Context, postID string) error
	// ListBookmarks(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]orm.Post, string, error)
}