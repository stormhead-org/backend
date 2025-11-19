package communitygrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommunityServer) ListCommunities(ctx context.Context, req *protopkg.ListCommunitiesRequest) (*protopkg.ListCommunitiesResponse, error) {
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 50
	}

	communities, err := s.db.SelectCommunitiesWithPagination("", int(req.Limit)+1, req.Cursor)
	if err != nil {
		s.log.Error("internal error listing communities", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	var nextCursor string
	if len(communities) > int(req.Limit) {
		nextCursor = communities[req.Limit-1].ID.String()
		communities = communities[:req.Limit]
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
