package usergrpc

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

func (s *UserServer) Follow(ctx context.Context, request *protopkg.FollowRequest) (*protopkg.FollowResponse, error) {
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

	if userID == user.ID {
		s.log.Debug("user following themself", zap.String("user_id", request.UserId))
		return nil, status.Errorf(codes.InvalidArgument, "following themself")
	}

	_, err = s.db.SelectFollowerByID(
		user.ID.String(),
		userID.String(),
	)
	if err == nil {
		s.log.Debug(
			"user already followed",
			zap.String("user_id", user.ID.String()),
			zap.String("follower_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "already followed")
	} else if err != gorm.ErrRecordNotFound {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	follower := &ormpkg.Follower{
		FollowerID: user.ID,
		UserID:     userID,
	}

	err = s.db.InsertFollower(follower)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.FollowResponse{}, nil
}
