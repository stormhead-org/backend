package community

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/stormhead-org/backend/internal/lib"
	"github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/orm"
	"github.com/stormhead-org/backend/internal/services"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type CommunityServiceImpl struct {
	db  *orm.PostgresClient
	log *zap.Logger
}

func NewCommunityService(db *orm.PostgresClient, log *zap.Logger) services.CommunityService {
	return &CommunityServiceImpl{
		db:  db,
		log: log,
	}
}

func (s *CommunityServiceImpl) CreateCommunity(ctx context.Context, slug, name, description, rules string) (*orm.Community, error) {
	// TODO: Implement proper validation functions
	// err := ValidateCommunitySlug(slug)
	// if err != nil {
	// 	return nil, status.Errorf(codes.InvalidArgument, "slug not match conditions")
	// }
	// err = ValidateCommunityName(name)
	// if err != nil {
	// 	return nil, status.Errorf(codes.InvalidArgument, "name not match conditions")
	// }

	_, err := s.db.SelectCommunityBySlug(slug)
	if err != gorm.ErrRecordNotFound {
		if err == nil {
			return nil, status.Errorf(codes.AlreadyExists, "slug already exists")
		}
		s.log.Error("error selecting community by slug", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not check slug")
	}

	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error getting user from context", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	community := &orm.Community{
		OwnerID:     userID,
		Slug:        slug,
		Name:        name,
		Description: description,
		Rules:       rules,
	}

	if err := s.db.InsertCommunity(community); err != nil {
		s.log.Error("internal error inserting community", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create community")
	}

	// Create and assign @everyone role
	everyoneRole := &orm.Role{
		Name:        "@everyone",
		CommunityID: &community.ID,
		Type:        "community",
		Permissions: json.RawMessage(`{}`),
	}
	if err := s.db.InsertRole(everyoneRole); err != nil {
		s.log.Error("failed to create @everyone role for community", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create role")
	}
	userRole := &orm.UserRole{
		UserID: userID,
		RoleID: everyoneRole.ID,
	}
	if err := s.db.InsertUserRole(userRole); err != nil {
		s.log.Error("failed to assign @everyone role to creator", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not assign role")
	}

	s.log.Info("community created",
		zap.String("id", community.ID.String()),
		zap.String("owner_id", community.OwnerID.String()),
		zap.String("name", community.Name),
	)
	return community, nil
}

func (s *CommunityServiceImpl) GetCommunity(ctx context.Context, communityID string) (*orm.Community, error) {
	community, err := s.db.SelectCommunityByID(communityID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	reputation, err := lib.CalculateCommunityReputation(s.db, community)
	if err != nil {
		s.log.Error("failed to calculate community reputation", zap.Error(err), zap.String("community_id", communityID))
		// Do not fail the request if reputation calculation fails, just log it.
	}
	community.Reputation = int(reputation)

	return community, nil
}

func (s *CommunityServiceImpl) UpdateCommunity(ctx context.Context, communityID string, name, description, rules *string) (*orm.Community, error) {
	community, err := s.db.SelectCommunityByID(communityID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	// TODO: Replace with proper permission check (Phase 5)
	if community.OwnerID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "not an owner")
	}

	if name != nil {
		community.Name = *name
	}
	if description != nil {
		community.Description = *description
	}
	if rules != nil {
		community.Rules = *rules
	}

	if err := s.db.UpdateCommunity(community); err != nil {
		s.log.Error("internal error updating community", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not update community")
	}
	return community, nil
}

func (s *CommunityServiceImpl) DeleteCommunity(ctx context.Context, communityID string) error {
	community, err := s.db.SelectCommunityByID(communityID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	// TODO: Replace with proper permission check (Phase 5)
	if community.OwnerID != userID {
		return status.Errorf(codes.PermissionDenied, "not an owner")
	}

	if err := s.db.DeleteCommunity(community); err != nil {
		s.log.Error("internal error deleting community", zap.Error(err))
		return status.Errorf(codes.Internal, "could not delete community")
	}
	return nil
}

func (s *CommunityServiceImpl) ListCommunities(ctx context.Context, cursor string, limit int) ([]*orm.Community, string, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	communities, err := s.db.SelectCommunitiesWithPagination("", limit+1, cursor)
	if err != nil {
		s.log.Error("internal error listing communities", zap.Error(err))
		return nil, "", status.Errorf(codes.Internal, "database error")
	}

	var nextCursor string
	if len(communities) > limit {
		nextCursor = communities[limit-1].ID.String()
		communities = communities[:limit]
	}

	return communities, nextCursor, nil
}

func (s *CommunityServiceImpl) JoinCommunity(ctx context.Context, communityID string) error {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	communityUUID, err := uuid.Parse(communityID)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid community id")
	}

	_, err = s.db.SelectCommunityByID(communityID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	_, err = s.db.SelectCommunityUser(communityID, userID.String())
	if err != gorm.ErrRecordNotFound {
		if err == nil {
			// Idempotency: if user is already a member, return success
			return nil
		}
		s.log.Error("error checking community membership", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	communityUser := &orm.CommunityUser{
		CommunityID: communityUUID,
		UserID:      userID,
	}

	if err := s.db.InsertCommunityUser(communityUser); err != nil {
		s.log.Error("error inserting community user", zap.Error(err))
		return status.Errorf(codes.Internal, "could not join community")
	}

	// Assign "@everyone" role
	everyoneRole, err := s.db.SelectRoleByName("@everyone", &communityUUID)
	if err != nil {
		s.log.Error("could not find @everyone role for community", zap.Error(err))
		return status.Errorf(codes.Internal, "could not find role")
	}
	userRole := &orm.UserRole{
		UserID: userID,
		RoleID: everyoneRole.ID,
	}
	if err := s.db.InsertUserRole(userRole); err != nil {
		s.log.Error("failed to assign @everyone role to user", zap.Error(err))
		return status.Errorf(codes.Internal, "could not assign role")
	}

	return nil
}

func (s *CommunityServiceImpl) LeaveCommunity(ctx context.Context, communityID string) error {
    userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get user from context")
	}

	communityUUID, err := uuid.Parse(communityID)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid community id")
	}

    community, err := s.db.SelectCommunityByID(communityID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

    if community.OwnerID == userID {
        return status.Errorf(codes.PermissionDenied, "owner cannot leave the community, transfer ownership first")
    }

	communityUser, err := s.db.SelectCommunityUser(communityID, userID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Idempotency: if user is not a member, return success
			return nil
		}
		s.log.Error("error checking community membership", zap.Error(err))
		return status.Errorf(codes.Internal, "database error")
	}

	if err := s.db.DeleteCommunityUser(communityUser); err != nil {
		s.log.Error("error deleting community user", zap.Error(err))
		return status.Errorf(codes.Internal, "could not leave community")
	}

    // Remove "@everyone" role
	everyoneRole, err := s.db.SelectRoleByName("@everyone", &communityUUID)
	if err != nil {
		s.log.Error("could not find @everyone role for community", zap.Error(err))
		return status.Errorf(codes.Internal, "could not find role")
	}
	userRole := &orm.UserRole{
		UserID: userID,
		RoleID: everyoneRole.ID,
	}
	if err := s.db.DeleteUserRole(userRole); err != nil {
		s.log.Error("failed to remove @everyone role from user", zap.Error(err))
		return status.Errorf(codes.Internal, "could not remove role")
	}

	return nil
}

// ... (Ban, Unban, TransferOwnership etc. to be implemented)
