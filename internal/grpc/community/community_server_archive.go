package communitygrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/stormhead-org/backend/internal/lib"
	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommunityServer) Archive(ctx context.Context, req *protopkg.ArchiveCommunityRequest) (*protopkg.ArchiveCommunityResponse, error) {
	if !req.Confirm {
		return nil, status.Errorf(codes.InvalidArgument, "confirmation is required to archive a community")
	}

	community, err := s.db.SelectCommunityByID(req.CommunityId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return nil, lib.HandleError(err)
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	// TODO: Replace with proper permission check (Phase 7) - Current check is for owner only
	// Permission check for archiving community (FR-225 modified to FR-Archive-Community)
	if community.OwnerID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "not an owner of the community")
	}

	if community.IsArchived {
		return nil, status.Errorf(codes.AlreadyExists, "community is already archived")
	}

	community.IsArchived = true
	now := timestamppb.Now().AsTime()
	community.ArchivedAt = &now
	community.Slug = community.Slug + "-" + community.ID.String() // Modify slug to free up original

	if err := s.db.UpdateCommunity(community); err != nil {
		s.log.Error("failed to archive community", zap.Error(err), zap.String("community_id", community.ID.String()))
		return nil, lib.HandleError(err)
	}

	return &protopkg.ArchiveCommunityResponse{}, nil
}