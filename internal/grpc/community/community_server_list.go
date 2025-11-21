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

	sortBy := ""
	if req.SortBy == protopkg.ListCommunitiesRequest_POPULARITY {
		sortBy = "popularity"
	}

	communities, err := s.db.SelectCommunitiesWithPagination(
		"",
		int(req.Limit)+1,
		req.Cursor,
		sortBy,
		req.IncludeBanned,
		false, // Do not include archived communities in general list
	)
	if err != nil {
		s.log.Error("internal error listing communities", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	var nextCursor string
	var hasMore bool
	if len(communities) > int(req.Limit) {
		nextCursor = communities[req.Limit-1].ID.String()
		communities = communities[:req.Limit]
		hasMore = true
	}

	protoCommunities := make([]*protopkg.Community, len(communities))
	for i, community := range communities {
		var archivedAt *timestamppb.Timestamp
		if community.ArchivedAt != nil {
			archivedAt = timestamppb.New(*community.ArchivedAt)
		}

		protoCommunities[i] = &protopkg.Community{
			Id:          community.ID.String(),
			OwnerId:     community.OwnerID.String(),
			OwnerName:   community.Owner.Name, // Populated by Preload
			Slug:        community.Slug,
			Name:        community.Name,
			Description: community.Description,
			Rules:       community.Rules,
			MemberCount: int32(community.MemberCount),
			PostCount:   int32(community.PostCount),
			Reputation:  int32(community.Reputation), // Casting to int32 as per proto definition
			IsBanned:    community.IsBanned,
			IsArchived:  community.IsArchived,
			ArchivedAt:  archivedAt,
			CreatedAt:   timestamppb.New(community.CreatedAt),
			UpdatedAt:   timestamppb.New(community.UpdatedAt),
		}
	}

	return &protopkg.ListCommunitiesResponse{
		Communities: protoCommunities,
		NextCursor:  nextCursor,
		HasMore:     hasMore,
	}, nil
}