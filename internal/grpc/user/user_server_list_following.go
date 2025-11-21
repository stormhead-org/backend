package usergrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *UserServer) ListFollowing(ctx context.Context, request *protopkg.ListFollowingRequest) (*protopkg.ListFollowingResponse, error) {
	limit := int(request.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	followers, err := s.db.SelectFollowersWithPagination("", request.UserId, limit+1, request.Cursor)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	hasMore := len(followers) > limit
	if hasMore {
		followers = followers[:limit]
	}

	var nextCursor string
	if hasMore && len(followers) > 0 {
		nextCursor = followers[len(followers)-1].UserID.String()
	}

	result := make([]*protopkg.UserProfile, len(followers))
	for i, follower := range followers {
		result[i] = &protopkg.UserProfile{
			Id:          follower.Follower.ID.String(),
			Name:        follower.Follower.Name,
			Description: follower.Follower.Description,
			CreatedAt:   timestamppb.New(follower.CreatedAt),
		}
	}

	return &protopkg.ListFollowingResponse{
		Users:      result,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
