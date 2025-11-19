package communitygrpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (s *CommunityServer) Leave(ctx context.Context, req *protopkg.LeaveCommunityRequest) (*protopkg.LeaveCommunityResponse, error) {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	communityUUID, err := uuid.Parse(req.CommunityId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid community id")
	}

	community, err := s.db.SelectCommunityByID(req.CommunityId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	if community.OwnerID == userID {
		return nil, status.Errorf(codes.PermissionDenied, "owner cannot leave the community, transfer ownership first")
	}

	communityUser, err := s.db.SelectCommunityUser(req.CommunityId, userID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Idempotency: if user is not a member, return success
			return &protopkg.LeaveCommunityResponse{}, nil
		}
		s.log.Error("error checking community membership", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	if err := s.db.DeleteCommunityUser(communityUser); err != nil {
		s.log.Error("error deleting community user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not leave community")
	}

	// Remove "@everyone" role
	everyoneRole, err := s.db.SelectRoleByName("@everyone", &communityUUID)
	if err != nil {
		s.log.Error("could not find @everyone role for community", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not find role")
	}
	userRole := &orm.UserRole{
		UserID: userID,
		RoleID: everyoneRole.ID,
	}
	if err := s.db.DeleteUserRole(userRole); err != nil {
		s.log.Error("failed to remove @everyone role from user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not remove role")
	}

	return &protopkg.LeaveCommunityResponse{}, nil
}
