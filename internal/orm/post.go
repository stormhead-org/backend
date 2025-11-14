package orm

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Post struct {
	ID           uuid.UUID       `gorm:"primaryKey"`
	CommunityID  uuid.UUID
	Community    Community
	AuthorID     uuid.UUID
	Author       User
	Title        string
	Content      json.RawMessage `gorm:"type:jsonb"`
	Status       int
	LikeCount    int
	CommentCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	PublishedAt  time.Time
}

func (c *Post) TableName() string {
	return "post"
}

func (c *Post) BeforeCreate(transaction *gorm.DB) error {
	c.ID = uuid.New()
	return nil
}

func (c *PostgresClient) SelectPostByID(id string) (*Post, error) {
	var post Post
	tx := c.database.
		Select([]string{
			"id",
			"community_id",
			"author_id",
			"title",
			"content",
			"status",
			"like_count",
			"comment_count",
			"created_at",
			"updated_at",
			"published_at",
		}).
		Where("id = ?", id).
		Preload("Community").
		Preload("Author").
		First(&post)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &post, nil
}

func (c *PostgresClient) SelectPostsWithPagination(author_id string, limit int, cursor string) ([]*Post, error) {
	var posts []*Post
	query := c.database.
		Select([]string{
			"id",
			"community_id",
			"author_id",
			"title",
			"content",
			"status",
			"like_count",
			"comment_count",
			"created_at",
			"updated_at",
			"published_at",
		}).
		Preload("Community").
		Preload("Author").
		Order("created_at DESC")

	if author_id != "" {
		query = query.Where("author_id = ?", author_id)
	}

	if cursor != "" {
		var cursorPost Post
		tx := c.database.
			Where("id = ?", cursor).
			First(&cursorPost)

		if tx.Error != nil {
			return nil, tx.Error
		}

		query = query.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursorPost.CreatedAt,
			cursorPost.CreatedAt,
			cursorPost.ID,
		)
	}

	tx := query.Limit(limit).Find(&posts)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return posts, nil
}

func (c *PostgresClient) InsertPost(post *Post) error {
	transaction := c.database.Create(post)
	return transaction.Error
}

func (c *PostgresClient) UpdatePost(post *Post) error {
	tx := c.database.Model(post).Omit("Community").Omit("Author").Updates(post)
	return tx.Error
}

func (c *PostgresClient) DeletePost(post *Post) error {
	tx := c.database.Delete(post)
	return tx.Error
}
