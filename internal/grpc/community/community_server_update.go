package communitygrpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommunityServer) Update(ctx context.Context, req *protopkg.UpdateCommunityRequest) (*protopkg.UpdateCommunityResponse, error) {
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

	if req.Name != nil {
		community.Name = *req.Name
	}

	if req.Description != nil {
		community.Description = *req.Description
	}

	if req.Rules != nil {
		community.Rules = *req.Rules
	}

	if err := s.db.UpdateCommunity(community); err != nil {
		s.log.Error("internal error updating community", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not update community")
	}

	return &protopkg.UpdateCommunityResponse{
		Community: &protopkg.Community{
			Id:          community.ID.String(),
			OwnerId:     community.OwnerID.String(),
			Slug:        community.Slug,
			Name:        community.Name,
			Description: community.Description,
			Rules:       community.Rules,
			CreatedAt:   timestamppb.New(community.CreatedAt),
			UpdatedAt:   timestamppb.New(community.UpdatedAt),
		},
	}, nil
}
