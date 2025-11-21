package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommentServer) Get(ctx context.Context, request *protopkg.GetCommentRequest) (*protopkg.GetCommentResponse, error) {
	comment, err := s.db.SelectCommentByID(request.CommentId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("comment not found", zap.String("comment_id", request.CommentId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	if comment.AuthorID != userID {
		s.log.Error("wrong comment ownership")
		return nil, status.Errorf(codes.PermissionDenied, "not an owner")
	}

	parentCommentID := ""
	if comment.ParentCommentID != nil {
		parentCommentID = comment.ParentCommentID.String()
	}

	return &protopkg.GetCommentResponse{
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
	}, nil
}
