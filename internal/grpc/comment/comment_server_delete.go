package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommentServer) Delete(ctx context.Context, request *protopkg.DeleteCommentRequest) (*protopkg.DeleteCommentResponse, error) {
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

	err = s.db.DeleteComment(comment)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.DeleteCommentResponse{}, nil
}
