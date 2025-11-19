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

func (s *CommunityServer) Join(ctx context.Context, req *protopkg.JoinCommunityRequest) (*protopkg.JoinCommunityResponse, error) {
	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	communityUUID, err := uuid.Parse(req.CommunityId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid community id")
	}

	_, err = s.db.SelectCommunityByID(req.CommunityId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	_, err = s.db.SelectCommunityUser(req.CommunityId, userID.String())
	if err != gorm.ErrRecordNotFound {
		if err == nil {
			// Idempotency: if user is already a member, return success
			return &protopkg.JoinCommunityResponse{}, nil
		}
		s.log.Error("error checking community membership", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	communityUser := &orm.CommunityUser{
		CommunityID: communityUUID,
		UserID:      userID,
	}

	if err := s.db.InsertCommunityUser(communityUser); err != nil {
		s.log.Error("error inserting community user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not join community")
	}

	// Assign "@everyone" role
	everyoneRole, err := s.db.SelectRoleByName("@everyone", &communityUUID)
	if err != nil {
		s.log.Error("could not find @everyone role for community", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not find role")
	}
	userRole := &orm.UserRole{
		UserID: userID,
		RoleID: everyoneRole.ID,
	}
	if err := s.db.InsertUserRole(userRole); err != nil {
		s.log.Error("failed to assign @everyone role to user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not assign role")
	}
	return &protopkg.JoinCommunityResponse{}, nil
}
