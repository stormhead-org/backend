package orm

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Community struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	OwnerID     uuid.UUID
	Owner       User
	Slug        string
	Name        string
	Description string
	Rules       string
	IsBanned    bool
	BanReason   string
	MemberCount int       `gorm:"default:0"`
	PostCount   int       `gorm:"default:0"`
	Reputation  int       `gorm:"default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (c *Community) TableName() string {
	return "community"
}

func (c *Community) GetID() uuid.UUID {
	return c.ID
}

func (c *Community) BeforeCreate(transaction *gorm.DB) error {
	c.ID = uuid.New()
	return nil
}

func (c *PostgresClient) SelectCommunityByID(id string) (*Community, error) {
	var community Community
	tx := c.database.
		Select([]string{
			"id",
			"owner_id",
			"slug",
			"name",
			"description",
			"rules",
			"is_banned",
			"ban_reason",
			"member_count",
			"post_count",
			"reputation",
			"created_at",
			"updated_at",
		}).
		Where("id = ?", id).
		First(&community)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &community, nil
}

func (c *PostgresClient) SelectCommunityBySlug(slug string) (*Community, error) {
	var community Community
	tx := c.database.
		Select([]string{
			"id",
			"owner_id",
			"slug",
			"name",
			"description",
			"rules",
			"is_banned",
			"ban_reason",
			"member_count",
			"post_count",
			"reputation",
			"created_at",
			"updated_at",
		}).
		Where("slug = ?", slug).
		First(&community)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &community, nil
}

func (c *PostgresClient) SelectCommunityByName(name string) (*Community, error) {
	var community Community
	tx := c.database.
		Select([]string{
			"id",
			"owner_id",
			"slug",
			"name",
			"description",
			"rules",
			"is_banned",
			"ban_reason",
			"member_count",
			"post_count",
			"reputation",
			"created_at",
			"updated_at",
		}).
		Where("name = ?", name).
		First(&community)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &community, nil
}

func (c *PostgresClient) SelectCommunitiesWithPagination(owner_id string, limit int, cursor string) ([]*Community, error) {
	var communities []*Community
	query := c.database.
		Select([]string{
			"id",
			"owner_id",
			"slug",
			"name",
			"description",
			"rules",
			"is_banned",
			"ban_reason",
			"member_count",
			"post_count",
			"reputation",
			"created_at",
			"updated_at",
		}).
		Order("created_at DESC")

	if owner_id != "" {
		query = query.Where("owner_id = ?", owner_id)
	}

	if cursor != "" {
		var cursorCommunity Community
		tx := c.database.
			Where("id = ?", cursor).
			First(&cursorCommunity)

		if tx.Error != nil {
			return nil, tx.Error
		}

		query = query.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursorCommunity.CreatedAt,
			cursorCommunity.CreatedAt,
			cursorCommunity.ID,
		)
	}

	tx := query.Limit(limit).Find(&communities)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return communities, nil
}

func (c *PostgresClient) InsertCommunity(community *Community) error {
	transaction := c.database.Create(community)
	return transaction.Error
}

func (c *PostgresClient) UpdateCommunity(community *Community) error {
	tx := c.database.Model(community).Updates(community)
	return tx.Error
}

func (c *PostgresClient) DeleteCommunity(community *Community) error {
	tx := c.database.Delete(community)
	return tx.Error
}

func (c *PostgresClient) CountPostLikesInCommunity(communityID uuid.UUID) (int64, error) {
	var count int64
	tx := c.database.Model(&PostLike{}).
		Joins("JOIN post ON post.id = post_like.post_id").
		Where("post.community_id = ?", communityID).
		Count(&count)
	return count, tx.Error
}

func (c *PostgresClient) CountCommentsInCommunity(communityID uuid.UUID) (int64, error) {
	var count int64
	tx := c.database.Model(&Comment{}).
		Joins("JOIN post ON post.id = comment.post_id").
		Where("post.community_id = ?", communityID).
		Count(&count)
	return count, tx.Error
}
