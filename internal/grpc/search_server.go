package grpc

import (
	"context"

	"go.uber.org/zap"

	eventpkg "github.com/stormhead-org/backend/internal/event"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type SearchServer struct {
	protopkg.UnimplementedSearchServiceServer
	log      *zap.Logger
	database *ormpkg.PostgresClient
	broker   *eventpkg.KafkaClient
}

func NewSearchServer(log *zap.Logger, database *ormpkg.PostgresClient, broker *eventpkg.KafkaClient) *SearchServer {
	return &SearchServer{
		log:      log,
		database: database,
		broker:   broker,
	}
}

func (s *SearchServer) Search(context context.Context, request *protopkg.SearchRequest) (*protopkg.SearchResponse, error) {
	return &protopkg.SearchResponse{}, nil
}
