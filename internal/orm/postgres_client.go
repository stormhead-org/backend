package orm

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresClient struct {
	database *gorm.DB
}

func NewPostgresClient(host string, port string, user string, password string) (*PostgresClient, error) {
	database, err := gorm.Open(
		postgres.Open(
			fmt.Sprintf(
				"host=%s port=%s user=%s password=%s sslmode=disable",
				host,
				port,
				user,
				password,
			),
		),
		&gorm.Config{},
	)
	if err != nil {
		return nil, err
	}

	rawDatabase, err := database.DB()
	if err != nil {
		return nil, err
	}

	rawDatabase.SetMaxOpenConns(1)
	rawDatabase.SetMaxIdleConns(1)
	rawDatabase.SetConnMaxIdleTime(5 * time.Second)

	return &PostgresClient{
		database: database,
	}, nil
}

func (c *PostgresClient) CountUsers() (int64, error) {
	var count int64
	if err := c.database.Model(&User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (c *PostgresClient) IsCommunityMember(communityID, userID string) (bool, error) {
	var count int64
	err := c.database.Model(&CommunityUser{}).
		Where("community_id = ? AND user_id = ?", communityID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *PostgresClient) SelectRoleByName(name string, communityID *uuid.UUID) (*Role, error) {
	var role Role
	query := c.database.Where("name = ?", name)
	if communityID != nil {
		query = query.Where("community_id = ?", *communityID)
	} else {
		query = query.Where("community_id IS NULL")
	}
	if err := query.First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (c *PostgresClient) InsertRole(role *Role) error {
	return c.database.Create(role).Error
}

func (c *PostgresClient) InsertUserRole(userRole *UserRole) error {
	return c.database.Create(userRole).Error
}

func (c *PostgresClient) DeleteSessionsByUserID(userID string) error {
	tx := c.database.Where("user_id = ?", userID).Delete(&Session{})
	return tx.Error
}

func (c *PostgresClient) UpdatePlatformOwner(userID uuid.UUID) error {
	return c.database.Model(&PlatformSetting{}).Where("id = ?", 1).Update("platform_owner_id", userID).Error
}
