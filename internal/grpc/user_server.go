package grpc

import (
	"context"
	"encoding/json" // Добавлен импорт json
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb" // Добавлен импорт structpb
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	eventpkg "github.com/stormhead-org/backend/internal/event"
	"github.com/stormhead-org/backend/internal/lib"
	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type UserServer struct {
	protopkg.UnimplementedUserServiceServer
	log      *zap.Logger
	database *ormpkg.PostgresClient
	broker   *eventpkg.KafkaClient
}

func NewUserServer(log *zap.Logger, database *ormpkg.PostgresClient, broker *eventpkg.KafkaClient) *UserServer {
	return &UserServer{
		log:      log,
		database: database,
		broker:   broker,
	}
}

func (s *UserServer) Get(ctx context.Context, request *protopkg.GetUserRequest) (*protopkg.GetUserResponse, error) {
	user, err := s.database.SelectUserByID(request.UserId)
	if err != nil {
		s.log.Error("failed to select user by id", zap.Error(err), zap.String("user_id", request.UserId))
		return nil, lib.HandleError(err)
	}

	return &protopkg.GetUserResponse{
		User: &protopkg.UserProfile{
			Id:          user.ID.String(),
			Name:        user.Name,
			Description: user.Description,
			CreatedAt:   timestamppb.New(user.CreatedAt),
		},
	}, nil
}

func (s *UserServer) GetCurrent(ctx context.Context, request *protopkg.GetCurrentUserRequest) (*protopkg.GetCurrentUserResponse, error) {
	userID, err := middlewarepkg.GetUserID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	user, err := s.database.SelectUserByID(userID)
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

func (s *UserServer) UpdateProfile(ctx context.Context, request *protopkg.UpdateProfileRequest) (*protopkg.UpdateProfileResponse, error) {
	userID, err := middlewarepkg.GetUserID(ctx)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	user, err := s.database.SelectUserByID(userID)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	user.Description = request.Description

	err = s.database.UpdateUser(user)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UpdateProfileResponse{}, nil
}

func (s *UserServer) GetStatistics(ctx context.Context, request *protopkg.GetUserStatisticsRequest) (*protopkg.GetUserStatisticsResponse, error) {
	user, err := s.database.SelectUserByID(request.UserId)
	if err != nil {
		s.log.Error("failed to select user by id", zap.Error(err), zap.String("user_id", request.UserId))
		return nil, lib.HandleError(err)
	}

	reputation, err := lib.CalculateUserReputation(s.database, user)
	if err != nil {
		s.log.Error("failed to calculate user reputation", zap.Error(err), zap.String("user_id", request.UserId))
		return nil, status.Errorf(codes.Internal, "failed to calculate reputation")
	}

	return &protopkg.GetUserStatisticsResponse{
		Statistics: &protopkg.UserStatistics{
			Reputation: reputation,
		},
	}, nil
}

func (s *UserServer) ListCommunities(ctx context.Context, request *protopkg.ListUserCommunitiesRequest) (*protopkg.ListUserCommunitiesResponse, error) {
	limit := int(request.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	communities, err := s.database.SelectCommunitiesWithPagination(request.UserId, limit+1, request.Cursor)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	hasMore := len(communities) > limit
	if hasMore {
		communities = communities[:limit]
	}

	var nextCursor string
	if hasMore && len(communities) > 0 {
		nextCursor = communities[len(communities)-1].ID.String()
	}

	result := make([]*protopkg.Community, len(communities))
	for i, community := range communities {
		result[i] = &protopkg.Community{
			Id:          community.ID.String(),
			OwnerId:     community.OwnerID.String(),
			Slug:        community.Slug,
			Name:        community.Name,
			Description: community.Description,
			CreatedAt:   timestamppb.New(community.CreatedAt),
			UpdatedAt:   timestamppb.New(community.UpdatedAt),
		}
	}

	return &protopkg.ListUserCommunitiesResponse{
		Communities: result,
		NextCursor:  nextCursor,
		HasMore:     hasMore,
	}, nil
}

func (s *UserServer) ListPosts(ctx context.Context, request *protopkg.ListUserPostsRequest) (*protopkg.ListUserPostsResponse, error) {
	limit := int(request.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	posts, err := s.database.SelectPostsWithPagination(request.UserId, limit+1, request.Cursor)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	var nextCursor string
	if hasMore && len(posts) > 0 {
		nextCursor = posts[len(posts)-1].ID.String()
	}

	result := make([]*protopkg.Post, len(posts))
	for i, post := range posts {
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

		result[i] = &protopkg.Post{
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
		}
	}

	return &protopkg.ListUserPostsResponse{
		Posts:      result,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *UserServer) ListComments(ctx context.Context, request *protopkg.ListUserCommentsRequest) (*protopkg.ListUserCommentsResponse, error) {
	limit := int(request.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	comments, err := s.database.SelectCommentsWithPagination("", request.UserId, limit+1, request.Cursor)
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

	result := make([]*protopkg.CommentWithPostInfo, len(comments))
	for i, comment := range comments {
		parentCommentID := ""
		if comment.ParentCommentID != nil {
			parentCommentID = comment.ParentCommentID.String()
		}

		result[i] = &protopkg.CommentWithPostInfo{
			PostId:    comment.Post.ID.String(),
			PostTitle: comment.Post.Title,
			Comment: &protopkg.Comment{
				Id:              comment.ID.String(),
				ParentCommentId: parentCommentID,
				PostId:          comment.PostID.String(),
				AuthorId:        comment.AuthorID.String(),
				AuthorName:      comment.Author.Name,
				Content:         comment.Content,
				CreatedAt:       timestamppb.New(comment.CreatedAt),
				UpdatedAt:       timestamppb.New(comment.UpdatedAt),
			},
		}
	}

	return &protopkg.ListUserCommentsResponse{
		Comments:   result,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *UserServer) Follow(ctx context.Context, request *protopkg.FollowRequest) (*protopkg.FollowResponse, error) {
	user, err := s.database.SelectUserByID(request.UserId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("user not found", zap.String("user_id", request.UserId))
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

	if userID == user.ID {
		s.log.Debug("user following themself", zap.String("user_id", request.UserId))
		return nil, status.Errorf(codes.InvalidArgument, "following themself")
	}

	_, err = s.database.SelectFollowerByID(
		user.ID.String(),
		userID.String(),
	)
	if err == nil {
		s.log.Debug(
			"user already followed",
			zap.String("user_id", user.ID.String()),
			zap.String("follower_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "already followed")
	} else if err != gorm.ErrRecordNotFound {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	follower := &ormpkg.Follower{
		FollowerID: user.ID,
		UserID:     userID,
	}

	err = s.database.InsertFollower(follower)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.FollowResponse{}, nil
}

func (s *UserServer) Unfollow(ctx context.Context, request *protopkg.UnfollowRequest) (*protopkg.UnfollowResponse, error) {
	user, err := s.database.SelectUserByID(request.UserId)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug("user not found", zap.String("user_id", request.UserId))
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

	follower, err := s.database.SelectFollowerByID(
		user.ID.String(),
		userID.String(),
	)
	if err == gorm.ErrRecordNotFound {
		s.log.Debug(
			"user not followed",
			zap.String("user_id", user.ID.String()),
			zap.String("follower_id", userID.String()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "not followed")
	}
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	err = s.database.DeleteFollower(follower)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	return &protopkg.UnfollowResponse{}, nil
}

func (s *UserServer) ListFollowers(ctx context.Context, request *protopkg.ListFollowersRequest) (*protopkg.ListFollowersResponse, error) {
	limit := int(request.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	followers, err := s.database.SelectFollowersWithPagination(request.UserId, "", limit+1, request.Cursor)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	hasMore := len(followers) > limit
	if hasMore {
		followers = followers[:limit]
	}

	var nextCursor string
	if hasMore && len(followers) > 0 {
		nextCursor = followers[len(followers)-1].UserID.String()
	}

	result := make([]*protopkg.UserProfile, len(followers))
	for i, follower := range followers {
		result[i] = &protopkg.UserProfile{
			Id:          follower.Follower.ID.String(),
			Name:        follower.Follower.Name,
			Description: follower.Follower.Description,
			CreatedAt:   timestamppb.New(follower.CreatedAt),
		}
	}

	return &protopkg.ListFollowersResponse{
		Users:      result,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *UserServer) ListFollowing(ctx context.Context, request *protopkg.ListFollowingRequest) (*protopkg.ListFollowingResponse, error) {
	limit := int(request.Limit)
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	followers, err := s.database.SelectFollowersWithPagination("", request.UserId, limit+1, request.Cursor)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "")
	}

	hasMore := len(followers) > limit
	if hasMore {
		followers = followers[:limit]
	}

	var nextCursor string
	if hasMore && len(followers) > 0 {
		nextCursor = followers[len(followers)-1].UserID.String()
	}

	result := make([]*protopkg.UserProfile, len(followers))
	for i, follower := range followers {
		result[i] = &protopkg.UserProfile{
			Id:          follower.Follower.ID.String(),
			Name:        follower.Follower.Name,
			Description: follower.Follower.Description,
			CreatedAt:   timestamppb.New(follower.CreatedAt),
		}
	}

	return &protopkg.ListFollowingResponse{
		Users:      result,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *UserServer) Heartbeat(ctx context.Context, request *protopkg.HeartbeatRequest) (*protopkg.HeartbeatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Heartbeat not implemented")
}
