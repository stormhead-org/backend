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

func (s *CommunityServer) TransferOwnership(ctx context.Context, req *protopkg.TransferCommunityOwnershipRequest) (*protopkg.TransferCommunityOwnershipResponse, error) {
	communityID, err := uuid.Parse(req.CommunityId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid community ID")
	}

	newOwnerID, err := uuid.Parse(req.NewOwnerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid new owner ID")
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

	// Only the current owner can transfer ownership
	if community.OwnerID != currentUserID {
		return nil, status.Errorf(codes.PermissionDenied, "only the owner can transfer community ownership")
	}

	// New owner must be a valid existing user
	_, err = s.db.SelectUserByID(newOwnerID.String())
	if err != nil {
		s.log.Error("new owner user not found", zap.Error(err), zap.String("new_owner_id", newOwnerID.String()))
		return nil, status.Errorf(codes.InvalidArgument, "new owner does not exist")
	}

	community.OwnerID = newOwnerID
	if err := s.db.UpdateCommunity(community); err != nil {
		s.log.Error("failed to transfer community ownership", zap.Error(err), zap.String("community_id", communityID.String()))
		return nil, lib.HandleError(err)
	}

	// TODO: Add event for ownership transfer for potential notifications or audit logs

	return &protopkg.TransferCommunityOwnershipResponse{}, nil
}