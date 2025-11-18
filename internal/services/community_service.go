package services

import (
	"context"

	"github.com/stormhead-org/backend/internal/orm"
)

// CommunityService defines the interface for community-related operations.
type CommunityService interface {
	CreateCommunity(ctx context.Context, slug, name, description, rules string) (*orm.Community, error)
	GetCommunity(ctx context.Context, communityID string) (*orm.Community, error)
	UpdateCommunity(ctx context.Context, communityID string, name, description, rules *string) (*orm.Community, error)
	DeleteCommunity(ctx context.Context, communityID string) error
	ListCommunities(ctx context.Context, cursor string, limit int) ([]*orm.Community, string, error)
	// ListUserCommunities(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]orm.Community, string, error)
	JoinCommunity(ctx context.Context, communityID string) error
	LeaveCommunity(ctx context.Context, communityID string) error
	// BanCommunity(ctx context.Context, communityID uuid.UUID, reason string) error
	// UnbanCommunity(ctx context.Context, communityID uuid.UUID) error
	// TransferOwnership(ctx context.Context, communityID, newOwnerID uuid.UUID) error
}