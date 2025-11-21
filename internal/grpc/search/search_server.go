package grpc

import (
	"go.uber.org/zap"

	eventpkg "github.com/stormhead-org/backend/internal/event"
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type SearchServer struct {
	protopkg.UnimplementedSearchServiceServer
	log      *zap.Logger
	db *orm.PostgresClient
	broker   *eventpkg.KafkaClient
}

func NewSearchServer(log *zap.Logger, db *orm.PostgresClient, broker *eventpkg.KafkaClient) *SearchServer {
	return &SearchServer{
		log:      log,
		db: db,
		broker:   broker,
	}
}
