package usergrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *UserServer) ListComments(ctx context.Context, request *protopkg.ListUserCommentsRequest) (*protopkg.ListUserCommentsResponse, error) {
	limit := int(request.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	comments, err := s.db.SelectCommentsWithPagination("", request.UserId, limit+1, request.Cursor)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	hasMore := len(comments) > limit
	if hasMore {
		comments = comments[:limit]
	}

	var nextCursor string
	if hasMore && len(comments) > 0 {
		nextCursor = comments[len(comments)-1].ID.String()
	}

	result := make([]*protopkg.CommentWithPostInfo, len(comments))
	for i, comment := range comments {
		parentCommentID := ""
		if comment.ParentCommentID != nil {
			parentCommentID = comment.ParentCommentID.String()
		}

		result[i] = &protopkg.CommentWithPostInfo{
			PostId:    comment.Post.ID.String(),
			PostTitle: comment.Post.Title,
			Comment: &protopkg.Comment{
				Id:              comment.ID.String(),
				ParentCommentId: parentCommentID,
				PostId:          comment.PostID.String(),
				AuthorId:        comment.AuthorID.String(),
				AuthorName:      comment.Author.Name,
				Content:         comment.Content,
				CreatedAt:       timestamppb.New(comment.CreatedAt),
				UpdatedAt:       timestamppb.New(comment.UpdatedAt),
			},
		}
	}

	return &protopkg.ListUserCommentsResponse{
		Comments:   result,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
