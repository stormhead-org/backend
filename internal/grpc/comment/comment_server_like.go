package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommentServer) Like(ctx context.Context, request *protopkg.LikeCommentRequest) (*protopkg.LikeCommentResponse, error) {
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

	_, err = s.db.SelectCommentLikeByID(
		comment.ID.String(),
		userID.String(),
	)
	if err == nil {
		s.log.Debug(
			"comment already liked",
			zap.String("comment_id", comment.ID.String()),
			zap.String("user_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "already liked")
	} else if err != gorm.ErrRecordNotFound {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	commentLike := &ormpkg.CommentLike{
		CommentID: comment.ID,
		UserID:    userID,
	}

	err = s.db.InsertCommentLike(commentLike)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	comment.LikeCount += 1

	err = s.db.UpdateComment(comment)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.LikeCommentResponse{}, nil
}
