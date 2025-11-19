package grpcauthorization

import (
	"context"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *AuthorizationServer) Logout(ctx context.Context, req *protopkg.LogoutRequest) (*protopkg.LogoutResponse, error) {
	// Client handles token deletion. Server-side invalidation can be added if needed.
	return &protopkg.LogoutResponse{}, nil
}
