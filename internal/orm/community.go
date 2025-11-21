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
	IsArchived  bool       `gorm:"default:false"`
	ArchivedAt  *time.Time `gorm:"default:null"`
	MemberCount int        `gorm:"default:0"`
	PostCount   int        `gorm:"default:0"`
	Reputation  int64      `gorm:"default:0"`
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
		Preload("Owner").
		Select([]string{
			"community.id",
			"community.owner_id",
			"community.slug",
			"community.name",
			"community.description",
			"community.rules",
			"community.is_banned",
			"community.ban_reason",
			"community.is_archived",
			"community.archived_at",
			"community.member_count",
			"community.post_count",
			"community.reputation",
			"community.created_at",
			"community.updated_at",
		}).
		Where("community.id = ?", id).
		First(&community)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &community, nil
}

func (c *PostgresClient) SelectCommunityBySlug(slug string) (*Community, error) {
	var community Community
	tx := c.database.
		Preload("Owner").
		Select([]string{
			"community.id",
			"community.owner_id",
			"community.slug",
			"community.name",
			"community.description",
			"community.rules",
			"community.is_banned",
			"community.ban_reason",
			"community.is_archived",
			"community.archived_at",
			"community.member_count",
			"community.post_count",
			"community.reputation",
			"community.created_at",
			"community.updated_at",
		}).
		Where("community.slug = ?", slug).
		First(&community)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &community, nil
}

func (c *PostgresClient) SelectCommunityByName(name string) (*Community, error) {
	var community Community
	tx := c.database.
		Preload("Owner").
		Select([]string{
			"community.id",
			"community.owner_id",
			"community.slug",
			"community.name",
			"community.description",
			"community.rules",
			"community.is_banned",
			"community.ban_reason",
			"community.is_archived",
			"community.archived_at",
			"community.member_count",
			"community.post_count",
			"community.reputation",
			"community.created_at",
			"community.updated_at",
		}).
		Where("community.name = ?", name).
		First(&community)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &community, nil
}

func (c *PostgresClient) SelectCommunitiesWithPagination(ownerID string, limit int, cursor string, sortBy string, includeBanned bool, includeArchived bool) ([]*Community, error) {
	var communities []*Community
	query := c.database.
		Preload("Owner").
		Select([]string{
			"community.id",
			"community.owner_id",
			"community.slug",
			"community.name",
			"community.description",
			"community.rules",
			"community.is_banned",
			"community.ban_reason",
			"community.is_archived",
			"community.archived_at",
			"community.member_count",
			"community.post_count",
			"community.reputation",
			"community.created_at",
			"community.updated_at",
		})

	if ownerID != "" {
		query = query.Where("community.owner_id = ?", ownerID)
	}

	if !includeBanned {
		query = query.Where("community.is_banned = ?", false)
	}

	if !includeArchived {
		query = query.Where("community.is_archived = ?", false)
	}

	orderClause := "community.created_at DESC"
	if sortBy == "popularity" {
		orderClause = "community.member_count DESC, community.created_at DESC"
	}
	query = query.Order(orderClause)

	if cursor != "" {
		var cursorCommunity Community
		tx := c.database.
			Where("id = ?", cursor).
			First(&cursorCommunity)

		if tx.Error != nil {
			return nil, tx.Error
		}

		// Handle cursor for different sort orders
		if sortBy == "popularity" {
			query = query.Where(
				"(community.member_count < ?) OR (community.member_count = ? AND community.created_at < ?) OR (community.member_count = ? AND community.created_at = ? AND community.id < ?)",
				cursorCommunity.MemberCount,
				cursorCommunity.MemberCount,
				cursorCommunity.CreatedAt,
				cursorCommunity.MemberCount,
				cursorCommunity.CreatedAt,
				cursorCommunity.ID,
			)
		} else { // default to created_at
			query = query.Where(
				"(community.created_at < ?) OR (community.created_at = ? AND community.id < ?)",
				cursorCommunity.CreatedAt,
				cursorCommunity.CreatedAt,
				cursorCommunity.ID,
			)
		}
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
