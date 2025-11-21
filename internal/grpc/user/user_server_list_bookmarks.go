package usergrpc

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stormhead-org/backend/internal/lib"
	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *UserServer) ListBookmarks(ctx context.Context, req *protopkg.ListBookmarksRequest) (*protopkg.ListBookmarksResponse, error) {
	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("cannot get user from context", zap.Error(err))
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	limit := int(req.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	bookmarks, err := s.db.SelectBookmarksWithPagination(userID.String(), limit+1, req.Cursor)
	if err != nil {
		s.log.Error("failed to list bookmarks", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, lib.HandleError(err)
	}

	var nextCursor string
	hasMore := len(bookmarks) > limit
	if hasMore {
		bookmarks = bookmarks[:limit]
		nextCursor = bookmarks[len(bookmarks)-1].ID.String()
	}

	protoPosts := make([]*protopkg.Post, len(bookmarks))
	for i, bookmark := range bookmarks {
		var structContent *structpb.Struct
		if len(bookmark.Post.Content) > 0 {
			if err := json.Unmarshal(bookmark.Post.Content, &structContent); err != nil {
				s.log.Error("failed to unmarshal content from JSON for bookmarked post", zap.Error(err))
				return nil, status.Errorf(codes.Internal, "failed to process bookmarked post content")
			}
		}

		protoPosts[i] = &protopkg.Post{
			Id:            bookmark.Post.ID.String(),
			CommunityId:   bookmark.Post.CommunityID.String(),
			CommunityName: bookmark.Post.Community.Name,
			AuthorId:      bookmark.Post.AuthorID.String(),
			AuthorName:    bookmark.Post.Author.Name,
			Title:         bookmark.Post.Title,
			Content:       structContent,
			Status:        protopkg.PostStatus(bookmark.Post.Status),
			LikeCount:     int32(bookmark.Post.LikeCount),
			CreatedAt:     timestamppb.New(bookmark.Post.CreatedAt),
			UpdatedAt:     timestamppb.New(bookmark.Post.UpdatedAt),
			PublishedAt: func() *timestamppb.Timestamp {
				if !bookmark.Post.PublishedAt.IsZero() {
					return timestamppb.New(bookmark.Post.PublishedAt)
				}
				return nil
			}(),
		}
	}

	return &protopkg.ListBookmarksResponse{
		Posts:      protoPosts,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
