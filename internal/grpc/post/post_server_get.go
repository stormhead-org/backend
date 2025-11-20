package postgrpc

import (
	"context"
	"encoding/json"

	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func (s *PostServer) Get(ctx context.Context, request *protopkg.GetPostRequest) (*protopkg.GetPostResponse, error) {
	post, err := s.db.SelectPostByID(request.PostId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "post not found")
		}
		s.log.Error("error selecting post by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "database error")
	}

	structContent, err := rawContentToStruct(post.Content)
	if err != nil {
		s.log.Error("failed to convert raw content to struct", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to process content")
	}

	return &protopkg.GetPostResponse{
		Post: &protopkg.Post{
			Id:            post.ID.String(),
			CommunityId:   post.CommunityID.String(),
			CommunityName: post.Community.Name,
			AuthorId:      post.AuthorID.String(),
			AuthorName:    post.Author.Name,
			Title:         post.Title,
			Content:       structContent,
			Status:        protopkg.PostStatus(post.Status),
			CreatedAt:     timestamppb.New(post.CreatedAt),
			UpdatedAt:     timestamppb.New(post.UpdatedAt),
			PublishedAt:   timestamppb.New(post.PublishedAt),
		},
	}, nil
}

func rawContentToStruct(content json.RawMessage) (*structpb.Struct, error) {
	if len(content) == 0 {
		return nil, nil
	}
	var contentInterface interface{}
	if err := json.Unmarshal(content, &contentInterface); err != nil {
		return nil, err
	}
	if contentStruct, ok := contentInterface.(map[string]interface{}); ok {
		return structpb.NewStruct(contentStruct)
	}
	return structpb.NewStruct(map[string]interface{}{"value": contentInterface})
}
