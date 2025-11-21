package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/google/uuid"
	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommentServer) Create(ctx context.Context, request *protopkg.CreateCommentRequest) (*protopkg.CreateCommentResponse, error) {
	var parentCommentUUID *uuid.UUID
	if request.ParentCommentId != "" {
		UUID, err := uuid.Parse(request.ParentCommentId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parent_comment_id")
		}

		parentCommentUUID = &UUID
	}

	postUUID, err := uuid.Parse(request.PostId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid community_id") // This seems like a typo, should probably be "invalid post_id"
	}

	_, err = s.db.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
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

	comment := &ormpkg.Comment{
		ParentCommentID: parentCommentUUID,
		PostID:          postUUID,
		AuthorID:        userID,
		Content:         request.Content,
	}

	err = s.db.InsertComment(comment)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.CreateCommentResponse{}, nil
}
