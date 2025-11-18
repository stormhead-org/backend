package lib

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Paginatable defines the interface for models that can be paginated.
// The model must have an ID and a CreatedAt field.
type Paginatable interface {
	GetID() uuid.UUID
	GetCreatedAt() time.Time
}

// Paginate applies cursor-based keyset pagination to a GORM query.
// It orders by `created_at DESC` and `id DESC`.
// The cursor is the ID of the last item from the previous page.
func Paginate[T Paginatable](db *gorm.DB, query *gorm.DB, cursor string, limit int) (*gorm.DB, error) {
	if cursor == "" {
		return query.Limit(limit), nil
	}

	var cursorModel T
	err := db.Model(&cursorModel).Where("id = ?", cursor).First(&cursorModel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// If cursor is not found, return no results
			return query.Where("1 = 0"), nil
		}
		return nil, err
	}

	paginatedQuery := query.Where(
		"(created_at < ?) OR (created_at = ? AND id < ?)",
		cursorModel.GetCreatedAt(),
		cursorModel.GetCreatedAt(),
		cursorModel.GetID(),
	).Limit(limit)

	return paginatedQuery, nil
}
