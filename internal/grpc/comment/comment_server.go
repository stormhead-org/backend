package grpc

import (
	"go.uber.org/zap"

	eventpkg "github.com/stormhead-org/backend/internal/event"
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type CommentServer struct {
	protopkg.UnimplementedCommentServiceServer
	log      *zap.Logger
	db *orm.PostgresClient
	broker   *eventpkg.KafkaClient
}

func NewCommentServer(log *zap.Logger, db *orm.PostgresClient, broker *eventpkg.KafkaClient) *CommentServer {
	return &CommentServer{
		log:      log,
		db: db,
		broker:   broker,
	}
}
