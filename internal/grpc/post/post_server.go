package postgrpc

import (
	"github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	"go.uber.org/zap"
)

type PostServer struct {
	protopkg.UnimplementedPostServiceServer
	log *zap.Logger
	db  *orm.PostgresClient
}

func NewPostServer(log *zap.Logger, db *orm.PostgresClient) *PostServer {
	return &PostServer{
		log: log,
		db:  db,
	}
}
