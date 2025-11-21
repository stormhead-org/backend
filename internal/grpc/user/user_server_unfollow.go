package usergrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *UserServer) Unfollow(ctx context.Context, request *protopkg.UnfollowRequest) (*protopkg.UnfollowResponse, error) {
	user, err := s.db.SelectUserByID(request.UserId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("user not found", zap.String("user_id", request.UserId))
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

	follower, err := s.db.SelectFollowerByID(
		user.ID.String(),
		userID.String(),
	)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug(
			"user not followed",
			zap.String("user_id", user.ID.String()),
			zap.String("follower_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "not followed")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	err = s.db.DeleteFollower(follower)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UnfollowResponse{}, nil
}
