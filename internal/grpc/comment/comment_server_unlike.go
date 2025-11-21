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

func (s *CommentServer) Unlike(ctx context.Context, request *protopkg.UnlikeCommentRequest) (*protopkg.UnlikeCommentResponse, error) {
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

	commentLike, err := s.db.SelectCommentLikeByID(
		comment.ID.String(),
		userID.String(),
	)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug(
			"comment not liked",
			zap.String("comment_id", comment.ID.String()),
			zap.String("user_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "not liked")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	err = s.db.DeleteCommentLike(commentLike)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	comment.LikeCount -= 1

	err = s.db.UpdateComment(comment)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UnlikeCommentResponse{}, nil
}
