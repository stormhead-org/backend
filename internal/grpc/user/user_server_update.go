package usergrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *UserServer) UpdateProfile(ctx context.Context, request *protopkg.UpdateProfileRequest) (*protopkg.UpdateProfileResponse, error) {
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

	user.Description = request.Description

	err = s.db.UpdateUser(user)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UpdateProfileResponse{}, nil
}
