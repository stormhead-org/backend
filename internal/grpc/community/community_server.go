package communitygrpc

import (
	"github.com/stormhead-org/backend/internal/orm"
	"go.uber.org/zap"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type CommunityServer struct {
	protopkg.UnimplementedCommunityServiceServer
	log *zap.Logger
	db  *orm.PostgresClient
}

func NewCommunityServer(log *zap.Logger, db *orm.PostgresClient) *CommunityServer {
	return &CommunityServer{
		log: log,
		db:  db,
	}
}

// ... Other methods (Ban, Unban, TransferOwnership) would follow the same pattern
