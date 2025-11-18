package orm

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Session struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	UserID    uuid.UUID
	User      User
	UserAgent string
	IpAddress string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Session) TableName() string {
	return "session"
}

func (s *Session) BeforeCreate(transaction *gorm.DB) error {
	s.ID = uuid.New()
	return nil
}

func (c *PostgresClient) SelectSessionByID(ID string) (*Session, error) {
	var session Session
	tx := c.database.
		Select([]string{
			"id",
			"user_id",
			"user_agent",
			"ip_address",
			"created_at",
			"updated_at",
		}).
		Where("id = ?", ID).
		First(&session)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &session, nil
}

func (c *PostgresClient) SelectSessionsByUserID(userID string, cursor string, limit int) ([]*Session, error) {
	var sessions []*Session
	query := c.database.
		Select([]string{
			"id",
			"user_id",
			"user_agent",
			"ip_address",
			"created_at",
			"updated_at",
		}).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if cursor != "" {
		var cursorSession Session
		tx := c.database.
			Where("id = ?", cursor).
			First(&cursorSession)

		if tx.Error != nil {
			return nil, tx.Error
		}

		query = query.
			Where(
				"(created_at < ?) OR (created_at = ? AND id < ?)",
				cursorSession.CreatedAt,
				cursorSession.CreatedAt,
				cursorSession.ID,
			)
	}

	var tx *gorm.DB
	if limit != 0 {
		tx = query.Limit(limit).Find(&sessions)
	} else {
		tx = query.Find(&sessions)
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return sessions, nil
}

func (c *PostgresClient) InsertSession(session *Session) error {
	tx := c.database.Create(session)
	return tx.Error
}

func (c *PostgresClient) UpdateSession(session *Session) error {
	tx := c.database.Model(session).Updates(session)
	return tx.Error
}

func (c *PostgresClient) DeleteSession(session *Session) error {
	tx := c.database.Delete(session)
	return tx.Error
}

func (c *PostgresClient) DeleteSessions() error {
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	tx := c.database.
		Where("updated_at < ?", thirtyDaysAgo).
		Delete(&Session{})

	return tx.Error
}
