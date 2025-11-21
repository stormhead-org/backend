package usergrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *UserServer) GetCurrent(ctx context.Context, request *protopkg.GetCurrentUserRequest) (*protopkg.GetCurrentUserResponse, error) {
	userID, err := middlewarepkg.GetUserID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	user, err := s.db.SelectUserByID(userID)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.GetCurrentUserResponse{
		User: &protopkg.CurrentUserProfile{
			Id:          user.ID.String(),
			Name:        user.Name,
			Description: user.Description,
			CreatedAt:   timestamppb.New(user.CreatedAt),
		},
	}, nil
}
