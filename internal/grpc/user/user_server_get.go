package usergrpc

import (
	"context"

	"github.com/stormhead-org/backend/internal/lib"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *UserServer) Get(ctx context.Context, request *protopkg.GetUserRequest) (*protopkg.GetUserResponse, error) {
	user, err := s.db.SelectUserByID(request.UserId)
	if err != nil {
		s.log.Error("failed to select user by id", zap.Error(err), zap.String("user_id", request.UserId))
		return nil, lib.HandleError(err)
	}

	return &protopkg.GetUserResponse{
		User: &protopkg.UserProfile{
			Id:          user.ID.String(),
			Name:        user.Name,
			Description: user.Description,
			CreatedAt:   timestamppb.New(user.CreatedAt),
		},
	}, nil
}
