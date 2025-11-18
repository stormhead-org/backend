package fixtures

import (
	"time"

	"github.com/google/uuid"
)

// UserFixture represents a user for testing.
type UserFixture struct {
	ID         uuid.UUID
	Email      string
	Password   string
	IsVerified bool
	CreatedAt  time.Time
}

// CommunityFixture represents a community for testing.
type CommunityFixture struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatorID   uuid.UUID
	CreatedAt   time.Time
}

// PostFixture represents a post for testing.
type PostFixture struct {
	ID          uuid.UUID
	CommunityID uuid.UUID
	AuthorID    uuid.UUID
	Title       string
	Content     string
	CreatedAt   time.Time
}

// GetTestUser returns a standard user for use in tests.
func GetTestUser() UserFixture {
	return UserFixture{
		ID:         uuid.MustParse("c1f8e4d9-8b9a-4b7c-8c6f-4e2b0e1d7a3e"),
		Email:      "testuser@example.com",
		Password:   "password123",
		IsVerified: true,
		CreatedAt:  time.Now().Add(-24 * time.Hour),
	}
}

// GetTestCommunity returns a standard community for use in tests.
func GetTestCommunity(creatorID uuid.UUID) CommunityFixture {
	return CommunityFixture{
		ID:          uuid.MustParse("a1b2c3d4-e5f6-7890-1234-567890abcdef"),
		Name:        "Test Community",
		Description: "This is a test community.",
		CreatorID:   creatorID,
		CreatedAt:   time.Now().Add(-12 * time.Hour),
	}
}

// GetTestPost returns a standard post for use in tests.
func GetTestPost(communityID, authorID uuid.UUID) PostFixture {
	return PostFixture{
		ID:          uuid.MustParse("f1e2d3c4-b5a6-9876-5432-10fedcba9876"),
		CommunityID: communityID,
		AuthorID:    authorID,
		Title:       "Test Post Title",
		Content:     "This is the content of the test post.",
		CreatedAt:   time.Now().Add(-6 * time.Hour),
	}
}
