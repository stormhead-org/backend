package grpc

import (
	"context"
	"encoding/json"

	"github.com/stormhead-org/backend/internal/services"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type PostServer struct {
	protopkg.UnimplementedPostServiceServer
	log         *zap.Logger
	postService services.PostService
}

func NewPostServer(log *zap.Logger, postService services.PostService) *PostServer {
	return &PostServer{
		log:         log,
		postService: postService,
	}
}

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

	_, err := s.postService.CreatePost(ctx, request.CommunityId, request.Title, content)
	if err != nil {
		return nil, err
	}

	return &protopkg.CreatePostResponse{}, nil
}

func (s *PostServer) Get(ctx context.Context, request *protopkg.GetPostRequest) (*protopkg.GetPostResponse, error) {
	post, err := s.postService.GetPost(ctx, request.PostId)
	if err != nil {
		return nil, err
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

func (s *PostServer) Update(ctx context.Context, request *protopkg.UpdatePostRequest) (*protopkg.UpdatePostResponse, error) {
	var content json.RawMessage
	if request.Content != nil {
		jsonBytes, err := request.Content.MarshalJSON()
		if err != nil {
			s.log.Error("failed to marshal content to JSON", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to process content")
		}
		content = jsonBytes
	}

	_, err := s.postService.UpdatePost(ctx, request.PostId, &request.Title, content)
	if err != nil {
		return nil, err
	}

	return &protopkg.UpdatePostResponse{}, nil
}

func (s *PostServer) Delete(ctx context.Context, request *protopkg.DeletePostRequest) (*protopkg.DeletePostResponse, error) {
	if err := s.postService.DeletePost(ctx, request.PostId); err != nil {
		return nil, err
	}
	return &protopkg.DeletePostResponse{}, nil
}

func (s *PostServer) Publish(ctx context.Context, request *protopkg.PublishPostRequest) (*protopkg.PublishPostResponse, error) {
	if err := s.postService.PublishPost(ctx, request.PostId); err != nil {
		return nil, err
	}
	return &protopkg.PublishPostResponse{}, nil
}

func (s *PostServer) Unpublish(ctx context.Context, request *protopkg.UnpublishPostRequest) (*protopkg.UnpublishPostResponse, error) {
	if err := s.postService.UnpublishPost(ctx, request.PostId); err != nil {
		return nil, err
	}
	return &protopkg.UnpublishPostResponse{}, nil
}

func (s *PostServer) Like(ctx context.Context, request *protopkg.LikePostRequest) (*protopkg.LikePostResponse, error) {
	if err := s.postService.LikePost(ctx, request.PostId); err != nil {
		return nil, err
	}
	return &protopkg.LikePostResponse{}, nil
}

func (s *PostServer) Unlike(ctx context.Context, request *protopkg.UnlikePostRequest) (*protopkg.UnlikePostResponse, error) {
	if err := s.postService.UnlikePost(ctx, request.PostId); err != nil {
		return nil, err
	}
	return &protopkg.UnlikePostResponse{}, nil
}

func (s *PostServer) CreateBookmark(ctx context.Context, request *protopkg.CreateBookmarkRequest) (*protopkg.CreateBookmarkResponse, error) {
	if err := s.postService.CreateBookmark(ctx, request.PostId); err != nil {
		return nil, err
	}
	return &protopkg.CreateBookmarkResponse{}, nil
}

func (s *PostServer) DeleteBookmark(ctx context.Context, request *protopkg.DeleteBookmarkRequest) (*protopkg.DeleteBookmarkResponse, error) {
	if err := s.postService.DeleteBookmark(ctx, request.PostId); err != nil {
		return nil, err
	}
	return &protopkg.DeleteBookmarkResponse{}, nil
}

// ... ListComments and ListBookmarks would be refactored similarly

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
