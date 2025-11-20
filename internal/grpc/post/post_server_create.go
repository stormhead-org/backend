package postgrpc

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (s *PostServer) Create(ctx context.Context, request *protopkg.CreatePostRequest) (*protopkg.CreatePostResponse, error) {
	var content json.RawMessage
	if request.Content != nil {
		jsonBytes, err := request.Content.MarshalJSON()
		if err != nil {
			s.log.Error("failed to marshal content to JSON", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to process content")
		}
		content = jsonBytes
	}

	communityUUID, err := uuid.Parse(request.CommunityId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid community_id")
	}

	userID, err := middleware.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	// Check if community exists and user is a member
	_, err = s.db.SelectCommunityUser(request.CommunityId, userID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.PermissionDenied, "user is not a member of the community")
		}
		s.log.Error("error checking community membership", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	post := &orm.Post{
		CommunityID: communityUUID,
		AuthorID:    userID,
		Title:       request.Title,
		Content:     content,
		Status:      int(orm.PostStatusDraft),
	}

	if err := s.db.InsertPost(post); err != nil {
		s.log.Error("error inserting post", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create post")
	}

	return &protopkg.CreatePostResponse{}, nil
}
