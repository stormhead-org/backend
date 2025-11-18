package orm

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                uuid.UUID `gorm:"primaryKey"`
	Slug              string
	Name              string
	Description       string
	Email             string
	Password          string
	Salt              string
	VerificationToken string
	ResetToken        string
	ResetTokenExpiresAt *time.Time
	IsVerified        bool
	Reputation        int64
	LastActivity      time.Time
	Communities       []Community `gorm:"foreignKey:OwnerID"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	IsBanned          bool          `gorm:"default:false" json:"is_banned"`
	BanReason         string        `json:"ban_reason,omitempty"`
	Roles             []*Role       `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

// TableName returns the name of the table for the User model
func (u *User) TableName() string {
	return "user"
}

func (c *User) GetID() uuid.UUID {
	return c.ID
}

func (c *User) BeforeCreate(transaction *gorm.DB) error {
	c.ID = uuid.New()
	return nil
}

func (c *PostgresClient) SelectUserByID(ID string) (*User, error) {
	var user User
	tx := c.database.
		Select(
			[]string{
				"id",
				"slug",
				"name",
				"description",
				"email",
				"password",
				"salt",
				"verification_token",
				"reset_token",
				"is_verified",
				"reputation",
				"last_activity",
			},
		).
		Where("id = ?", ID).
		First(&user)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &user, nil
}

func (c *PostgresClient) SelectUserBySlug(slug string) (*User, error) {
	var user User
	tx := c.database.
		Select(
			[]string{
				"id",
				"slug",
				"name",
				"description",
				"email",
				"password",
				"salt",
				"verification_token",
				"reset_token",
				"is_verified",
				"reputation",
				"last_activity",
			},
		).
		Where("slug = ?", slug).
		First(&user)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &user, nil
}

func (c *PostgresClient) SelectUserByName(name string) (*User, error) {
	var user User
	tx := c.database.
		Select(
			[]string{
				"id",
				"name",
				"description",
				"email",
				"password",
				"salt",
				"verification_token",
				"reset_token",
				"is_verified",
				"reputation",
				"last_activity",
			},
		).
		Where("name = ?", name).
		First(&user)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &user, nil
}

func (c *PostgresClient) SelectUserByEmail(email string) (*User, error) {
	var user User
	tx := c.database.
		Select(
			[]string{
				"id",
				"slug",
				"name",
				"description",
				"email",
				"password",
				"salt",
				"verification_token",
				"reset_token",
				"is_verified",
				"reputation",
				"last_activity",
			},
		).
		Where("email = ?", email).
		First(&user)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &user, nil
}

func (c *PostgresClient) SelectUserByVerificationToken(verificationToken string) (*User, error) {
	var user User
	tx := c.database.
		Select(
			[]string{
				"id",
				"slug",
				"name",
				"description",
				"email",
				"password",
				"salt",
				"verification_token",
				"reset_token",
				"is_verified",
				"reputation",
				"last_activity",
			},
		).
		Where("verification_token = ?", verificationToken).
		First(&user)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &user, nil
}

func (c *PostgresClient) SelectUserByResetToken(resetToken string) (*User, error) {
	var user User
	tx := c.database.
		Select(
			[]string{
				"id",
				"slug",
				"name",
				"description",
				"email",
				"password",
				"salt",
				"verification_token",
				"reset_token",
				"reset_token_expires_at",
				"is_verified",
				"reputation",
				"last_activity",
			},
		).
		Where("reset_token = ?", resetToken).
		First(&user)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &user, nil
}

func (c *PostgresClient) InsertUser(user *User) error {
	tx := c.database.Create(user)
	return tx.Error
}

func (c *PostgresClient) UpdateUser(user *User) error {
	tx := c.database.Model(user).Updates(user)
	return tx.Error
}

func (c *PostgresClient) CountPostLikesByAuthor(authorID uuid.UUID) (int64, error) {
	var count int64
	tx := c.database.Model(&PostLike{}).
		Joins("JOIN post ON post.id = post_like.post_id").
		Where("post.author_id = ?", authorID).
		Count(&count)
	return count, tx.Error
}

func (c *PostgresClient) CountCommentLikesByAuthor(authorID uuid.UUID) (int64, error) {
	var count int64
	tx := c.database.Model(&CommentLike{}).
		Joins("JOIN comment ON comment.id = comment_like.comment_id").
		Where("comment.author_id = ?", authorID).
		Count(&count)
	return count, tx.Error
}

