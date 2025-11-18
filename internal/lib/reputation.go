package lib

import "github.com/google/uuid"

// Reputationable is an interface for entities that have a reputation.
type Reputationable interface {
	GetID() uuid.UUID
}

// ReputationStore defines the methods for accessing reputation-related data.
type ReputationStore interface {
	CountPostLikesByAuthor(authorID uuid.UUID) (int64, error)
	CountCommentLikesByAuthor(authorID uuid.UUID) (int64, error)
	CountPostLikesInCommunity(communityID uuid.UUID) (int64, error)
	CountCommentsInCommunity(communityID uuid.UUID) (int64, error)
}

// CalculateUserReputation calculates the reputation for a user.
func CalculateUserReputation(store ReputationStore, user Reputationable) (float64, error) {
	postLikes, err := store.CountPostLikesByAuthor(user.GetID())
	if err != nil {
		return 0, err
	}

	commentLikes, err := store.CountCommentLikesByAuthor(user.GetID())
	if err != nil {
		return 0, err
	}

	return float64(postLikes + commentLikes), nil
}

// CalculateCommunityReputation calculates the reputation for a community.
func CalculateCommunityReputation(store ReputationStore, community Reputationable) (float64, error) {
	postLikes, err := store.CountPostLikesInCommunity(community.GetID())
	if err != nil {
		return 0, err
	}

	comments, err := store.CountCommentsInCommunity(community.GetID())
	if err != nil {
		return 0, err
	}

	return float64(postLikes) + (float64(comments) * 0.1), nil
}
