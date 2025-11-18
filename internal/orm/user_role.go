package orm

import "github.com/google/uuid"

type UserRole struct {
	UserID uuid.UUID `gorm:"type:uuid;primary_key" json:"user_id"`
	RoleID uuid.UUID `gorm:"type:uuid;primary_key" json:"role_id"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

func (c *PostgresClient) DeleteUserRole(userRole *UserRole) error {
	return c.database.Delete(userRole).Error
}
