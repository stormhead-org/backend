package usergrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stormhead-org/backend/internal/lib"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *UserServer) GetStatistics(ctx context.Context, request *protopkg.GetUserStatisticsRequest) (*protopkg.GetUserStatisticsResponse, error) {
	user, err := s.db.SelectUserByID(request.UserId)
	if err != nil {
		s.log.Error("failed to select user by id", zap.Error(err), zap.String("user_id", request.UserId))
		return nil, lib.HandleError(err)
	}

	reputation, err := lib.CalculateUserReputation(s.db, user)
	if err != nil {
		s.log.Error("failed to calculate user reputation", zap.Error(err), zap.String("user_id", request.UserId))
		return nil, status.Errorf(codes.Internal, "failed to calculate reputation")
	}

	return &protopkg.GetUserStatisticsResponse{
		Statistics: &protopkg.UserStatistics{
			Reputation: reputation,
		},
	}, nil
}
