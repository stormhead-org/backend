package postgrpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stormhead-org/backend/internal/lib"
	middlewarepkg "github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *PostServer) CreatePost(ctx context.Context, req *protopkg.CreatePostRequest) (*protopkg.CreatePostResponse, error) {
	communityID, err := uuid.Parse(req.CommunityId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid community ID")
	}

	authorID, err := middlewarepkg.GetUserUUID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get user from context")
	}

	community, err := s.db.SelectCommunityByID(communityID.String())
	if err != nil {
		s.log.Error("community not found", zap.Error(err), zap.String("community_id", communityID.String()))
		return nil, status.Errorf(codes.NotFound, "community not found")
	}

	// Check if the user is a member of the community
	isMember, err := s.db.IsCommunityMember(communityID.String(), authorID.String()) // Corrected arguments
	if err != nil {
		s.log.Error("failed to check community membership", zap.Error(err), zap.String("community_id", communityID.String()), zap.String("author_id", authorID.String()))
		return nil, lib.HandleError(err)
	}
	if !isMember {
		return nil, status.Errorf(codes.PermissionDenied, "user is not a member of the community")
	}

	var contentRaw json.RawMessage
	if req.Content != nil {
		contentBytes, err := req.Content.MarshalJSON()
		if err != nil {
			s.log.Error("failed to marshal post content", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to process content")
		}
		contentRaw = json.RawMessage(contentBytes)
	}

	post := &orm.Post{
		CommunityID: communityID,
		AuthorID:    authorID,
		Title:       req.Title,
		Content:     contentRaw, // Changed this line
		Status:      int(protopkg.PostStatus_POST_STATUS_DRAFT),
		LikeCount:   0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.InsertPost(post); err != nil {
		s.log.Error("failed to create post", zap.Error(err))
		return nil, lib.HandleError(err)
	}

	// Update community post count
	community.PostCount++
	if err := s.db.UpdateCommunity(community); err != nil {
		s.log.Error("failed to update community post count", zap.Error(err), zap.String("community_id", communityID.String()))
		return nil, lib.HandleError(err)
	}

	// Update community reputation - T089
	communityReputation, err := lib.CalculateCommunityReputation(s.db, community)
	if err != nil {
		s.log.Error("failed to calculate community reputation", zap.Error(err), zap.String("community_id", communityID.String()))
		// Do not return error, as post creation was successful
	} else {
		community.Reputation = int64(communityReputation)
		if err := s.db.UpdateCommunity(community); err != nil {
			s.log.Error("failed to update community reputation", zap.Error(err), zap.String("community_id", communityID.String()))
			// Do not return error, as post creation was successful
		}
	}

	// TODO: Add event for new post creation

	// Re-fetch post to ensure all fields are populated for the response
	// This might not be strictly necessary if InsertPost populates everything,
	// but it's safer for related entities like Author/Community.
	fetchedPost, err := s.db.SelectPostByID(post.ID.String())
	if err != nil {
		s.log.Error("failed to fetch created post for response", zap.Error(err), zap.String("post_id", post.ID.String()))
		return nil, lib.HandleError(err)
	}

	var structContent *structpb.Struct
	if len(fetchedPost.Content) > 0 {
		if err := json.Unmarshal(fetchedPost.Content, &structContent); err != nil {
			s.log.Error("failed to unmarshal content from JSON", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to process content")
		}
	}

	return &protopkg.CreatePostResponse{
		Post: &protopkg.Post{ // Assumed CreatePostResponse now contains a 'Post' field
			Id:            fetchedPost.ID.String(),
			CommunityId:   fetchedPost.CommunityID.String(),
			CommunityName: fetchedPost.Community.Name, // Populated by Preload in SelectPostByID
			AuthorId:      fetchedPost.AuthorID.String(),
			AuthorName:    fetchedPost.Author.Name,    // Populated by Preload in SelectPostByID
			Title:         fetchedPost.Title,
			Content:       structContent, // Changed this line
			Status:        protopkg.PostStatus(fetchedPost.Status),
			LikeCount:     int32(fetchedPost.LikeCount),
			CreatedAt:     timestamppb.New(fetchedPost.CreatedAt),
			UpdatedAt:     timestamppb.New(fetchedPost.UpdatedAt),
			PublishedAt: func() *timestamppb.Timestamp {
				if !fetchedPost.PublishedAt.IsZero() {
					return timestamppb.New(fetchedPost.PublishedAt)
				}
				return nil
			}(),
		},
	}, nil
}
