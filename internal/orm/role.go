package orm

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string          `gorm:"type:varchar(255);not null" json:"name"`
	CommunityID *uuid.UUID      `gorm:"type:uuid" json:"community_id"`
	Color       string          `gorm:"type:varchar(7);not null;default:'#95a5a6'" json:"color"`
	Type        string          `gorm:"type:varchar(50);not null" json:"type"`
	Permissions json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"permissions"`
	CreatedAt   time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"not null;default:now()" json:"updated_at"`
	Community   *Community      `gorm:"foreignkey:CommunityID" json:"community,omitempty"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	r.ID = uuid.New()
	return
}
