package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/stormhead-org/backend/internal/orm"
)

// CommentService defines the interface for comment-related operations.
type CommentService interface {
	CreateComment(ctx context.Context, postID, authorID, parentID uuid.UUID, content string) (*orm.Comment, error)
	GetComment(ctx context.Context, commentID uuid.UUID) (*orm.Comment, error)
	UpdateComment(ctx context.Context, commentID uuid.UUID, content string) (*orm.Comment, error)
	DeleteComment(ctx context.Context, commentID uuid.UUID) error
	ListComments(ctx context.Context, postID uuid.UUID, cursor string, limit int) ([]orm.Comment, string, error)
	ListUserComments(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]orm.Comment, string, error)
	LikeComment(ctx context.Context, commentID, userID uuid.UUID) error
	UnlikeComment(ctx context.Context, commentID, userID uuid.UUID) error
}
