package grpcauthorization

import (
	"github.com/stormhead-org/backend/internal/jwt"
	"go.uber.org/zap"

	clientpkg "github.com/stormhead-org/backend/internal/client"
	eventpkg "github.com/stormhead-org/backend/internal/event"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
)

type AuthorizationServer struct {
	protopkg.UnimplementedAuthorizationServiceServer
	log      *zap.Logger
	jwt      *jwt.JWT
	hibp     *clientpkg.HIBPClient
	database *ormpkg.PostgresClient
	broker   *eventpkg.KafkaClient
}

func NewAuthorizationServer(
	log *zap.Logger,
	jwt *jwt.JWT,
	hibp *clientpkg.HIBPClient,
	database *ormpkg.PostgresClient,
	broker *eventpkg.KafkaClient,
) *AuthorizationServer {
	return &AuthorizationServer{
		log:      log,
		jwt:      jwt,
		hibp:     hibp,
		database: database,
		broker:   broker,
	}
}
