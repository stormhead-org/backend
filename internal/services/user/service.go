package user

import (
	"go.uber.org/zap"

	clientpkg "github.com/stormhead-org/backend/internal/client"
	eventpkg "github.com/stormhead-org/backend/internal/event"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
)

type UserServiceImpl struct {
	log        *zap.Logger
	database   *ormpkg.PostgresClient
	broker     *eventpkg.KafkaClient
	hibpClient *clientpkg.HIBPClient
}
