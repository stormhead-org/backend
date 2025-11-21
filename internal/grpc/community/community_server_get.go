package communitygrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/stormhead-org/backend/internal/lib"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommunityServer) Get(ctx context.Context, req *protopkg.GetCommunityRequest) (*protopkg.GetCommunityResponse, error) {
	community, err := s.db.SelectCommunityByID(req.CommunityId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	reputation, err := lib.CalculateCommunityReputation(s.db, community)
	if err != nil {
		s.log.Error("failed to calculate community reputation", zap.Error(err), zap.String("community_id", req.CommunityId))
		// Do not fail the request if reputation calculation fails, just log it.
	}
	community.Reputation = int64(reputation) // Changed here

	return &protopkg.GetCommunityResponse{
		Community: &protopkg.Community{
			Id:          community.ID.String(),
			OwnerId:     community.OwnerID.String(),
			Slug:        community.Slug,
			Name:        community.Name,
			Description: community.Description,
			Rules:       community.Rules,
			Reputation:  int32(community.Reputation),
			CreatedAt:   timestamppb.New(community.CreatedAt),
			UpdatedAt:   timestamppb.New(community.UpdatedAt),
		},
	}, nil
}
