package orm

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Bookmark struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	PostID    uuid.UUID
	Post      Post
	UserID    uuid.UUID
	User      User
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (b *Bookmark) TableName() string {
	return "bookmark"
}

func (b *Bookmark) BeforeCreate(transaction *gorm.DB) error {
	b.ID = uuid.New()
	return nil
}

func (c *PostgresClient) InsertBookmark(bookmark *Bookmark) error {
	return c.database.Create(bookmark).Error
}

func (c *PostgresClient) DeleteBookmark(bookmark *Bookmark) error {
	return c.database.Delete(bookmark).Error
}

func (c *PostgresClient) SelectBookmarkByID(postID, userID string) (*Bookmark, error) {
	var bookmark Bookmark
	tx := c.database.
		Where("post_id = ? AND user_id = ?", postID, userID).
		First(&bookmark)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &bookmark, nil
}

func (c *PostgresClient) SelectBookmarksWithPagination(userID string, limit int, cursor string) ([]*Bookmark, error) {
	var bookmarks []*Bookmark
	query := c.database.
		Preload("Post").
		Preload("Post.Community").
		Preload("Post.Author").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if cursor != "" {
		var cursorBookmark Bookmark
		tx := c.database.Where("id = ?", cursor).First(&cursorBookmark)
		if tx.Error != nil {
			return nil, tx.Error
		}
		query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", cursorBookmark.CreatedAt, cursorBookmark.CreatedAt, cursorBookmark.ID)
	}

	tx := query.Limit(limit).Find(&bookmarks)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return bookmarks, nil
}