package usergrpc

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stormhead-org/backend/internal/lib"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *UserServer) ListPosts(ctx context.Context, req *protopkg.ListUserPostsRequest) (*protopkg.ListUserPostsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID")
	}

	limit := int(req.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	posts, err := s.db.SelectPostsWithPagination(userID.String(), limit+1, req.Cursor)
	if err != nil {
		s.log.Error("failed to list user posts", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, lib.HandleError(err)
	}

	var nextCursor string
	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
		nextCursor = posts[len(posts)-1].ID.String()
	}

	protoPosts := make([]*protopkg.Post, len(posts))
	for i, post := range posts {
		var structContent *structpb.Struct
		if len(post.Content) > 0 {
			if err := json.Unmarshal(post.Content, &structContent); err != nil {
				s.log.Error("failed to unmarshal content from JSON for user post", zap.Error(err))
				return nil, status.Errorf(codes.Internal, "failed to process user post content")
			}
		}

		protoPosts[i] = &protopkg.Post{
			Id:            post.ID.String(),
			CommunityId:   post.CommunityID.String(),
			CommunityName: post.Community.Name,
			AuthorId:      post.AuthorID.String(),
			AuthorName:    post.Author.Name,
			Title:         post.Title,
			Content:       structContent,
			Status:        protopkg.PostStatus(post.Status),
			LikeCount:     int32(post.LikeCount),
			CreatedAt:     timestamppb.New(post.CreatedAt),
			UpdatedAt:     timestamppb.New(post.UpdatedAt),
			PublishedAt: func() *timestamppb.Timestamp {
				if !post.PublishedAt.IsZero() {
					return timestamppb.New(post.PublishedAt)
				}
				return nil
			}(),
		}
	}

	return &protopkg.ListUserPostsResponse{
		Posts:      protoPosts,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
