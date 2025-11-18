package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/google/uuid"
	eventpkg "github.com/stormhead-org/backend/internal/event"
	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type CommentServer struct {
	protopkg.UnimplementedCommentServiceServer
	log      *zap.Logger
	database *ormpkg.PostgresClient
	broker   *eventpkg.KafkaClient
}

func NewCommentServer(log *zap.Logger, database *ormpkg.PostgresClient, broker *eventpkg.KafkaClient) *CommentServer {
	return &CommentServer{
		log:      log,
		database: database,
		broker:   broker,
	}
}

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
		return nil, status.Errorf(codes.InvalidArgument, "invalid community_id")
	}

	_, err = s.database.SelectPostByID(request.PostId)
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

	err = s.database.InsertComment(comment)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.CreateCommentResponse{}, nil
}

func (s *CommentServer) Get(ctx context.Context, request *protopkg.GetCommentRequest) (*protopkg.GetCommentResponse, error) {
	comment, err := s.database.SelectCommentByID(request.CommentId)
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

func (s *CommentServer) Update(ctx context.Context, request *protopkg.UpdateCommentRequest) (*protopkg.UpdateCommentResponse, error) {
	comment, err := s.database.SelectCommentByID(request.CommentId)
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

	comment.Content = request.Content

	err = s.database.UpdateComment(comment)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UpdateCommentResponse{}, nil
}

func (s *CommentServer) Delete(ctx context.Context, request *protopkg.DeleteCommentRequest) (*protopkg.DeleteCommentResponse, error) {
	comment, err := s.database.SelectCommentByID(request.CommentId)
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

	err = s.database.DeleteComment(comment)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.DeleteCommentResponse{}, nil
}

func (s *CommentServer) Like(ctx context.Context, request *protopkg.LikeCommentRequest) (*protopkg.LikeCommentResponse, error) {
	comment, err := s.database.SelectCommentByID(request.CommentId)
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

	_, err = s.database.SelectCommentLikeByID(
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

	err = s.database.InsertCommentLike(commentLike)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	comment.LikeCount += 1

	err = s.database.UpdateComment(comment)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.LikeCommentResponse{}, nil
}

func (s *CommentServer) Unlike(ctx context.Context, request *protopkg.UnlikeCommentRequest) (*protopkg.UnlikeCommentResponse, error) {
	comment, err := s.database.SelectCommentByID(request.CommentId)
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

	commentLike, err := s.database.SelectCommentLikeByID(
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

	err = s.database.DeleteCommentLike(commentLike)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	comment.LikeCount -= 1

	err = s.database.UpdateComment(comment)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UnlikeCommentResponse{}, nil
}

func (s *CommentServer) Stream(request *protopkg.StreamCommentRequest, stream protopkg.CommentService_StreamServer) error {
	return status.Errorf(codes.Unimplemented, "method Stream not implemented")
}
