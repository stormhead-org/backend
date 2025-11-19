package communitygrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommunityServer) Delete(ctx context.Context, req *protopkg.DeleteCommunityRequest) (*protopkg.DeleteCommunityResponse, error) {
	community, err := s.db.SelectCommunityByID(req.CommunityId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "community not found")
		}
		s.log.Error("error selecting community by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	// TODO: Replace with proper permission check (Phase 5)
	if community.OwnerID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "not an owner")
	}

	if err := s.db.DeleteCommunity(community); err != nil {
		s.log.Error("internal error deleting community", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not delete community")
	}

	return &protopkg.DeleteCommunityResponse{}, nil
}
