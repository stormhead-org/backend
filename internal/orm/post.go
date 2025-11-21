package orm

import (
	"encoding/json"
	"errors" // Added import
	"time"

	"github.com/google/uuid"
	"github.com/stormhead-org/backend/internal/lib"
	"gorm.io/gorm"
)

type PostStatus int

const (
	PostStatusDraft PostStatus = iota
	PostStatusPublished
)

type Post struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	CommunityID uuid.UUID
	Community   Community `gorm:"foreignKey:CommunityID"`
	AuthorID    uuid.UUID
	Author      User `gorm:"foreignKey:AuthorID"`
	Title       string
	Content     json.RawMessage `gorm:"type:jsonb"`
	Status      int
	LikeCount   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt time.Time
}

func (c *Post) TableName() string {
	return "post"
}

func (p *Post) ValidateContent() error {
	if len(p.Content) > 0 {
		// Check for valid JSON
		if !json.Valid(p.Content) {
			return gorm.ErrInvalidData
		}

		// Check for size limit (1MB)
		if len(p.Content) > 1024*1024 { // 1MB
			return errors.New("post content exceeds 1MB limit")
		}
	}
	return nil
}

func (p *Post) BeforeCreate(transaction *gorm.DB) error {
	p.ID = uuid.New()
	if err := p.ValidateContent(); err != nil {
		return err
	}
	return nil
}

func (p *Post) BeforeUpdate(transaction *gorm.DB) error {
	if err := p.ValidateContent(); err != nil {
		return err
	}
	return nil
}

func (p Post) GetID() uuid.UUID {
	return p.ID
}

func (p Post) GetCreatedAt() time.Time {
	return p.CreatedAt
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
		Order("created_at DESC, id DESC")

	if author_id != "" {
		query = query.Where("author_id = ?", author_id)
	}

	paginatedQuery, err := lib.Paginate[Post](c.database, query, cursor, limit)
	if err != nil {
		return nil, err
	}

	tx := paginatedQuery.Find(&posts)
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
