package usergrpc

import (
	eventpkg "github.com/stormhead-org/backend/internal/event"
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
)

type UserServer struct {
	protopkg.UnimplementedUserServiceServer
	log      *zap.Logger
	db *orm.PostgresClient
	broker   *eventpkg.KafkaClient
}

func NewUserServer(log *zap.Logger, db *orm.PostgresClient, broker *eventpkg.KafkaClient) *UserServer {
	return &UserServer{
		log:      log,
		db: db,
		broker:   broker,
	}
}