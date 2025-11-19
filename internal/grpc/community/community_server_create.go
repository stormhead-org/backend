package communitygrpc

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommunityServer) Create(ctx context.Context, req *protopkg.CreateCommunityRequest) (*protopkg.CreateCommunityResponse, error) {
	// TODO: Implement proper validation functions
	// err := ValidateCommunitySlug(slug)
	// if err != nil {
	// 	return nil, status.Errorf(codes.InvalidArgument, "slug not match conditions")
	// }
	// err = ValidateCommunityName(name)
	// if err != nil {
	// 	return nil, status.Errorf(codes.InvalidArgument, "name not match conditions")
	// }

	_, err := s.db.SelectCommunityBySlug(req.Slug)
	if err != gorm.ErrRecordNotFound {
		if err == nil {
			return nil, status.Errorf(codes.AlreadyExists, "slug already exists")
		}
		s.log.Error("error selecting community by slug", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not check slug")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error getting user from context", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	community := &ormpkg.Community{
		OwnerID:     userID,
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Rules:       req.Rules,
	}

	if err := s.db.InsertCommunity(community); err != nil {
		s.log.Error("internal error inserting community", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create community")
	}

	// Create and assign @everyone role
	everyoneRole := &ormpkg.Role{
		Name:        "@everyone",
		CommunityID: &community.ID,
		Type:        "community",
		Permissions: json.RawMessage(`{}`),
	}
	if err := s.db.InsertRole(everyoneRole); err != nil {
		s.log.Error("failed to create @everyone role for community", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create role")
	}
	userRole := &ormpkg.UserRole{
		UserID: userID,
		RoleID: everyoneRole.ID,
	}
	if err := s.db.InsertUserRole(userRole); err != nil {
		s.log.Error("failed to assign @everyone role to creator", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not assign role")
	}

	s.log.Info("community created",
		zap.String("id", community.ID.String()),
		zap.String("owner_id", community.OwnerID.String()),
		zap.String("name", community.Name),
	)

	return &protopkg.CreateCommunityResponse{
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
