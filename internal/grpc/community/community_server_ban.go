package communitygrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"

	"github.com/stormhead-org/backend/internal/lib"
	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommunityServer) Ban(ctx context.Context, req *protopkg.BanCommunityRequest) (*protopkg.BanCommunityResponse, error) {
	communityID, err := uuid.Parse(req.CommunityId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid community ID")
	}

	community, err := s.db.SelectCommunityByID(communityID.String())
	if err != nil {
		s.log.Error("community not found", zap.Error(err), zap.String("community_id", communityID.String()))
		return nil, status.Errorf(codes.NotFound, "community not found")
	}

	currentUserID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	// TODO: Replace with proper permission check (Phase 7) - Current check is for owner only
	// Permission check for banning community
	if community.OwnerID != currentUserID {
		return nil, status.Errorf(codes.PermissionDenied, "only the owner can ban a community")
	}

	if community.IsBanned {
		return nil, status.Errorf(codes.AlreadyExists, "community is already banned")
	}

	community.IsBanned = true
	community.BanReason = req.Reason

	if err := s.db.UpdateCommunity(community); err != nil {
		s.log.Error("failed to ban community", zap.Error(err), zap.String("community_id", communityID.String()))
		return nil, lib.HandleError(err)
	}

	// TODO: Add event for community banned for potential notifications or audit logs

	return &protopkg.BanCommunityResponse{}, nil
}
