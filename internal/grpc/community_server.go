package grpc

import (
	"context"

	"github.com/stormhead-org/backend/internal/services"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type CommunityServer struct {
	protopkg.UnimplementedCommunityServiceServer
	log              *zap.Logger
	communityService services.CommunityService
}

func NewCommunityServer(log *zap.Logger, communityService services.CommunityService) *CommunityServer {
	return &CommunityServer{
		log:              log,
		communityService: communityService,
	}
}

func (s *CommunityServer) Create(ctx context.Context, request *protopkg.CreateCommunityRequest) (*protopkg.CreateCommunityResponse, error) {
	community, err := s.communityService.CreateCommunity(ctx, request.Slug, request.Name, request.Description, request.Rules)
	if err != nil {
		// Errors from the service layer should already be gRPC status errors
		return nil, err
	}

	return &protopkg.CreateCommunityResponse{
		Community: &protopkg.Community{
			Id:          community.ID.String(),
			OwnerId:     community.OwnerID.String(),
			Slug:        community.Slug,
			Name:        community.Name,
			Description: community.Description,
			Rules:       community.Rules,
			CreatedAt:   timestamppb.New(community.CreatedAt),
			UpdatedAt:   timestamppb.New(community.UpdatedAt),
		},
	}, nil
}

func (s *CommunityServer) Get(ctx context.Context, request *protopkg.GetCommunityRequest) (*protopkg.GetCommunityResponse, error) {
	community, err := s.communityService.GetCommunity(ctx, request.CommunityId)
	if err != nil {
		return nil, err
	}

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

func (s *CommunityServer) Update(ctx context.Context, request *protopkg.UpdateCommunityRequest) (*protopkg.UpdateCommunityResponse, error) {
	community, err := s.communityService.UpdateCommunity(ctx, request.CommunityId, request.Name, request.Description, request.Rules)
	if err != nil {
		return nil, err
	}

	return &protopkg.UpdateCommunityResponse{
		Community: &protopkg.Community{
			Id:          community.ID.String(),
			OwnerId:     community.OwnerID.String(),
			Slug:        community.Slug,
			Name:        community.Name,
			Description: community.Description,
			Rules:       community.Rules,
			CreatedAt:   timestamppb.New(community.CreatedAt),
			UpdatedAt:   timestamppb.New(community.UpdatedAt),
		},
	}, nil
}

func (s *CommunityServer) Delete(ctx context.Context, request *protopkg.DeleteCommunityRequest) (*protopkg.DeleteCommunityResponse, error) {
	err := s.communityService.DeleteCommunity(ctx, request.CommunityId)
	if err != nil {
		return nil, err
	}
	return &protopkg.DeleteCommunityResponse{}, nil
}

func (s *CommunityServer) ListCommunities(ctx context.Context, request *protopkg.ListCommunitiesRequest) (*protopkg.ListCommunitiesResponse, error) {
	communities, nextCursor, err := s.communityService.ListCommunities(ctx, request.Cursor, int(request.Limit))
	if err != nil {
		return nil, err
	}

	protoCommunities := make([]*protopkg.Community, len(communities))
	for i, community := range communities {
		protoCommunities[i] = &protopkg.Community{
			Id:          community.ID.String(),
			OwnerId:     community.OwnerID.String(),
			Slug:        community.Slug,
			Name:        community.Name,
			Description: community.Description,
			CreatedAt:   timestamppb.New(community.CreatedAt),
			UpdatedAt:   timestamppb.New(community.UpdatedAt),
		}
	}

	return &protopkg.ListCommunitiesResponse{
		Communities: protoCommunities,
		NextCursor:  nextCursor,
		HasMore:     nextCursor != "",
	}, nil
}

func (s *CommunityServer) Join(ctx context.Context, request *protopkg.JoinCommunityRequest) (*protopkg.JoinCommunityResponse, error) {
	if err := s.communityService.JoinCommunity(ctx, request.CommunityId); err != nil {
		return nil, err
	}
	return &protopkg.JoinCommunityResponse{}, nil
}

func (s *CommunityServer) Leave(ctx context.Context, request *protopkg.LeaveCommunityRequest) (*protopkg.LeaveCommunityResponse, error) {
	if err := s.communityService.LeaveCommunity(ctx, request.CommunityId); err != nil {
		return nil, err
	}
	return &protopkg.LeaveCommunityResponse{}, nil
}

// ... Other methods (Ban, Unban, TransferOwnership) would follow the same pattern
