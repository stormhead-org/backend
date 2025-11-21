package communitygrpc

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/stormhead-org/backend/internal/lib"
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
		return nil, lib.HandleError(err)
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	// TODO: Replace with proper permission check (Phase 5) - Current check is for owner only
	if community.OwnerID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "not an owner")
	}

	if req.Name != nil {
		trimmedName := strings.TrimSpace(*req.Name)
		if len(trimmedName) < 3 || len(trimmedName) > 50 {
			return nil, status.Errorf(codes.InvalidArgument, "community name must be between 3 and 50 characters")
		}
		existingCommunity, err := s.db.SelectCommunityByName(trimmedName)
		if err != nil && err != gorm.ErrRecordNotFound {
			s.log.Error("error checking community name uniqueness", zap.Error(err))
			return nil, lib.HandleError(err)
		}
		if existingCommunity != nil && existingCommunity.ID != community.ID {
			return nil, status.Errorf(codes.AlreadyExists, "community with this name already exists")
		}
		community.Name = trimmedName
	}

	if req.Description != nil {
		trimmedDescription := strings.TrimSpace(*req.Description)
		if len(trimmedDescription) > 500 {
			return nil, status.Errorf(codes.InvalidArgument, "community description cannot exceed 500 characters")
		}
		community.Description = trimmedDescription
	}

	if req.Rules != nil {
		trimmedRules := strings.TrimSpace(*req.Rules)
		if len(trimmedRules) > 1000 {
			return nil, status.Errorf(codes.InvalidArgument, "community rules cannot exceed 1000 characters")
		}
		community.Rules = trimmedRules
	}

	if err := s.db.UpdateCommunity(community); err != nil {
		s.log.Error("internal error updating community", zap.Error(err))
		return nil, lib.HandleError(err)
	}

	// Ensure OwnerName is populated for the response
	owner, err := s.db.SelectUserByID(community.OwnerID.String())
	if err != nil {
		s.log.Error("error selecting community owner", zap.Error(err))
		return nil, lib.HandleError(err)
	}

	return &protopkg.UpdateCommunityResponse{
		Community: &protopkg.Community{
			Id:          community.ID.String(),
			OwnerId:     community.OwnerID.String(),
			OwnerName:   owner.Name, // Populate OwnerName
			Slug:        community.Slug,
			Name:        community.Name,
			Description: community.Description,
			Rules:       community.Rules,
			MemberCount: int32(community.MemberCount), // Populate MemberCount
			PostCount:   int32(community.PostCount),   // Populate PostCount
			Reputation:  int32(community.Reputation),  // Populate Reputation (Changed here)
			IsBanned:    community.IsBanned,           // Populate IsBanned
			CreatedAt:   timestamppb.New(community.CreatedAt),
			UpdatedAt:   timestamppb.New(community.UpdatedAt),
		},
	}, nil
}
