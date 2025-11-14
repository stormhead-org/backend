package grpc

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/google/uuid"
	eventpkg "github.com/stormhead-org/backend/internal/event"
	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type PostServer struct {
	protopkg.UnimplementedPostServiceServer
	log      *zap.Logger
	database *ormpkg.PostgresClient
	broker   *eventpkg.KafkaClient
}

func NewPostServer(log *zap.Logger, database *ormpkg.PostgresClient, broker *eventpkg.KafkaClient) *PostServer {
	return &PostServer{
		log:      log,
		database: database,
		broker:   broker,
	}
}

func (s *PostServer) Create(ctx context.Context, request *protopkg.CreatePostRequest) (*protopkg.CreatePostResponse, error) {
	communityUUID, err := uuid.Parse(request.CommunityId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid community_id")
	}

	_, err = s.database.SelectCommunityByID(request.CommunityId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("community not found", zap.String("community_id", request.CommunityId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}
	var content []byte
	if request.Content != nil {
		jsonBytes, err := request.Content.MarshalJSON()
		if err != nil {
			s.log.Error("failed to marshal content to JSON", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to process content")
		}
		content = jsonBytes
	}

	post := &ormpkg.Post{
		CommunityID: communityUUID,
		AuthorID:    userID,
		Title:       request.Title,
		Content:     content,
		Status:      0,
	}

	err = s.database.InsertPost(post)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.CreatePostResponse{}, nil
}

func (s *PostServer) Get(ctx context.Context, request *protopkg.GetPostRequest) (*protopkg.GetPostResponse, error) {
	post, err := s.database.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	// Преобразование json.RawMessage в google.protobuf.Struct
	var structContent *structpb.Struct
	if len(post.Content) > 0 {
		var contentInterface interface{}
		if err := json.Unmarshal(post.Content, &contentInterface); err != nil {
			s.log.Error("failed to unmarshal content from JSON", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to process content")
		}
		if contentStruct, ok := contentInterface.(map[string]interface{}); ok {
			structContent, err = structpb.NewStruct(contentStruct)
			if err != nil {
				s.log.Error("failed to create structpb struct from content", zap.Error(err))
				return nil, status.Errorf(codes.Internal, "failed to process content")
			}
		} else {
			structContent, err = structpb.NewStruct(map[string]interface{}{"value": contentInterface})
			if err != nil {
				s.log.Error("failed to create structpb struct from content", zap.Error(err))
				return nil, status.Errorf(codes.Internal, "failed to process content")
			}
		}
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
	post, err := s.database.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	if post.AuthorID != userID {
		s.log.Error("wrong post ownership")
		return nil, status.Errorf(codes.PermissionDenied, "not an owner")
	}

	post.Title = request.Title
	
	// Преобразование google.protobuf.Struct в json.RawMessage
	if request.Content != nil {
		jsonBytes, err := request.Content.MarshalJSON()
		if err != nil {
			s.log.Error("failed to marshal content to JSON", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to process content")
		}
		post.Content = jsonBytes
	}

	err = s.database.UpdatePost(post)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UpdatePostResponse{}, nil
}

func (s *PostServer) Delete(ctx context.Context, request *protopkg.DeletePostRequest) (*protopkg.DeletePostResponse, error) {
	post, err := s.database.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	if post.AuthorID != userID {
		s.log.Error("wrong post ownership")
		return nil, status.Errorf(codes.PermissionDenied, "not an owner")
	}

	err = s.database.DeletePost(post)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.DeletePostResponse{}, nil
}

func (s *PostServer) ListComments(ctx context.Context, request *protopkg.ListPostCommentsRequest) (*protopkg.ListPostCommentsResponse, error) {
	limit := int(request.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	comments, err := s.database.SelectCommentsWithPagination(request.PostId, "", limit+1, request.Cursor)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	hasMore := len(comments) > limit
	if hasMore {
		comments = comments[:limit]
	}

	var nextCursor string
	if hasMore && len(comments) > 0 {
		nextCursor = comments[len(comments)-1].ID.String()
	}

	result := make([]*protopkg.Comment, len(comments))
	for i, comment := range comments {
		parentCommentID := ""
		if comment.ParentCommentID != nil {
			parentCommentID = comment.ParentCommentID.String()
		}

		result[i] = &protopkg.Comment{
			Id:              comment.ID.String(),
			ParentCommentId: parentCommentID,
			PostId:          comment.PostID.String(),
			AuthorId:        comment.AuthorID.String(),
			AuthorName:      comment.Author.Name,
			Content:         comment.Content,
			CreatedAt:       timestamppb.New(comment.CreatedAt),
			UpdatedAt:       timestamppb.New(comment.UpdatedAt),
		}
	}

	return &protopkg.ListPostCommentsResponse{
		Comments:   result,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *PostServer) Publish(ctx context.Context, request *protopkg.PublishPostRequest) (*protopkg.PublishPostResponse, error) {
	post, err := s.database.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	if post.AuthorID != userID {
		s.log.Error("wrong post ownership")
		return nil, status.Errorf(codes.PermissionDenied, "not an owner")
	}

	post.Status = int(protopkg.PostStatus_POST_STATUS_PUBLISHED)
	post.PublishedAt = time.Now()

	err = s.database.UpdatePost(post)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.PublishPostResponse{}, nil
}

func (s *PostServer) Unpublish(ctx context.Context, request *protopkg.UnpublishPostRequest) (*protopkg.UnpublishPostResponse, error) {
	post, err := s.database.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	if post.AuthorID != userID {
		s.log.Error("wrong post ownership")
		return nil, status.Errorf(codes.PermissionDenied, "not an owner")
	}

	post.Status = int(protopkg.PostStatus_POST_STATUS_DRAFT)

	err = s.database.UpdatePost(post)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UnpublishPostResponse{}, nil
}

func (s *PostServer) Like(ctx context.Context, request *protopkg.LikePostRequest) (*protopkg.LikePostResponse, error) {
	post, err := s.database.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	_, err = s.database.SelectPostLikeByID(
		post.ID.String(),
		userID.String(),
	)
	if err == nil {
		s.log.Debug(
			"post already liked",
			zap.String("post_id", post.ID.String()),
			zap.String("user_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "already liked")
	} else if err != gorm.ErrRecordNotFound {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	postLike := &ormpkg.PostLike{
		PostID: post.ID,
		UserID: userID,
	}

	err = s.database.InsertPostLike(postLike)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	post.LikeCount += 1

	err = s.database.UpdatePost(post)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.LikePostResponse{}, nil
}

func (s *PostServer) Unlike(ctx context.Context, request *protopkg.UnlikePostRequest) (*protopkg.UnlikePostResponse, error) {
	post, err := s.database.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	postLike, err := s.database.SelectPostLikeByID(
		post.ID.String(),
		userID.String(),
	)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug(
			"post not liked",
			zap.String("post_id", post.ID.String()),
			zap.String("user_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "not liked")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	err = s.database.DeletePostLike(postLike)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	post.LikeCount -= 1

	err = s.database.UpdatePost(post)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UnlikePostResponse{}, nil
}

func (s *PostServer) CreateBookmark(ctx context.Context, request *protopkg.CreateBookmarkRequest) (*protopkg.CreateBookmarkResponse, error) {
	post, err := s.database.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	_, err = s.database.SelectBookmarkByID(
		post.ID.String(),
		userID.String(),
	)
	if err == nil {
		s.log.Debug(
			"post already bookmarked",
			zap.String("post_id", post.ID.String()),
			zap.String("user_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "already bookmarked")
	} else if err != gorm.ErrRecordNotFound {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	bookmark := &ormpkg.Bookmark{
		PostID: post.ID,
		UserID: userID,
	}

	err = s.database.InsertBookmark(bookmark)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.CreateBookmarkResponse{}, nil
}

func (s *PostServer) DeleteBookmark(ctx context.Context, request *protopkg.DeleteBookmarkRequest) (*protopkg.DeleteBookmarkResponse, error) {
	post, err := s.database.SelectPostByID(request.PostId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("post not found", zap.String("post_id", request.PostId))
		return nil, status.Errorf(codes.NotFound, "")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	userID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	postLike, err := s.database.SelectBookmarkByID(
		post.ID.String(),
		userID.String(),
	)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug(
			"post not bookmarked",
			zap.String("post_id", post.ID.String()),
			zap.String("user_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "not bookmarked")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	err = s.database.DeleteBookmark(postLike)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.DeleteBookmarkResponse{}, nil
}

func (s *PostServer) ListBookmarks(ctx context.Context, request *protopkg.ListBookmarksRequest) (*protopkg.ListBookmarksResponse, error) {
	limit := int(request.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	bookmarks, err := s.database.SelectBookmarksWithPagination(limit+1, request.Cursor)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	hasMore := len(bookmarks) > limit
	if hasMore {
		bookmarks = bookmarks[:limit]
	}

	var nextCursor string
	if hasMore && len(bookmarks) > 0 {
		nextCursor = bookmarks[len(bookmarks)-1].UserID.String()
	}

	posts := make([]*protopkg.Post, len(bookmarks))
	for i, bookmark := range bookmarks {
		posts[i] = &protopkg.Post{
			Id:            bookmark.Post.ID.String(),
			CommunityId:   bookmark.Post.CommunityID.String(),
			CommunityName: bookmark.Post.Community.Name,
			AuthorId:      bookmark.Post.AuthorID.String(),
			AuthorName:    bookmark.Post.Author.Name,
			Title:         bookmark.Post.Title,
			Content:       bookmark.Post.Content,
			Status:        protopkg.PostStatus(bookmark.Post.Status),
			CreatedAt:     timestamppb.New(bookmark.Post.CreatedAt),
			UpdatedAt:     timestamppb.New(bookmark.Post.UpdatedAt),
			PublishedAt:   timestamppb.New(bookmark.Post.PublishedAt),
		}
	}

	return &protopkg.ListBookmarksResponse{
		Posts:      posts,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
