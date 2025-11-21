package grpc

import (
	"context"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *SearchServer) Search(context context.Context, request *protopkg.SearchRequest) (*protopkg.SearchResponse, error) {
	return &protopkg.SearchResponse{}, nil
}
