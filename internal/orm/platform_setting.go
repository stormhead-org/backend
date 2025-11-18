package orm

import "github.com/google/uuid"

type PlatformSetting struct {
	ID              int        `gorm:"primaryKey"`
	PlatformOwnerID *uuid.UUID `gorm:"type:uuid"`
}

func (PlatformSetting) TableName() string {
	return "platform_settings"
}
