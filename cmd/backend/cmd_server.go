package main

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"

	clientpkg "github.com/stormhead-org/backend/internal/client"
	eventpkg "github.com/stormhead-org/backend/internal/event"
	grpcpkg "github.com/stormhead-org/backend/internal/grpc"
	authorizationgrpcpkg "github.com/stormhead-org/backend/internal/grpc/authorization"
	commentgrpcpkg "github.com/stormhead-org/backend/internal/grpc/comment"
	communitygrpcpkg "github.com/stormhead-org/backend/internal/grpc/community"
	postgrpcpkg "github.com/stormhead-org/backend/internal/grpc/post"
	searchgrpcpkg "github.com/stormhead-org/backend/internal/grpc/search"
	usergrpcpkg "github.com/stormhead-org/backend/internal/grpc/user"
	jwtpkg "github.com/stormhead-org/backend/internal/jwt"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
)

var serverCommand = &cobra.Command{
	Use:   "server",
	Short: "server",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return serverCommandImpl()
	},
}

func serverCommandImpl() error {
	if os.Getenv("DEBUG") == "1" {
		godotenv.Load()
	}

	// Application
	application := fx.New(
		// fx.NopLogger,
		fx.Provide(
			// Logger
			func() *zap.Logger {
				if os.Getenv("DEBUG") == "1" {
					logger, _ := zap.NewDevelopment()
					return logger
				}
				logger, _ := zap.NewProduction()
				return logger
			},

			// Config/Secrets from .env
			func(logger *zap.Logger) (*jwtpkg.JWT, error) {
				jwtSecret := os.Getenv("JWT_SECRET")
				if jwtSecret == "" {
					jwtSecret = "123456"
				}
				return jwtpkg.NewJWT(jwtSecret), nil
			},

			// Clients
			func(logger *zap.Logger) (*ormpkg.PostgresClient, error) {
				return ormpkg.NewPostgresClient(
					os.Getenv("POSTGRES_HOST"),
					os.Getenv("POSTGRES_PORT"),
					os.Getenv("POSTGRES_USER"),
					os.Getenv("POSTGRES_PASSWORD"),
				)
			},
			func(logger *zap.Logger) (*eventpkg.KafkaClient, error) {
				return eventpkg.NewKafkaClient(
					os.Getenv("KAFKA_HOST"),
					os.Getenv("KAFKA_PORT"),
					os.Getenv("KAFKA_TOPIC"),
					os.Getenv("KAFKA_GROUP"),
				)
			},
			clientpkg.NewHIBPClient,

			// gRPC Servers
			authorizationgrpcpkg.NewAuthorizationServer,
			commentgrpcpkg.NewCommentServer,
			communitygrpcpkg.NewCommunityServer,
			postgrpcpkg.NewPostServer,
			usergrpcpkg.NewUserServer,
			searchgrpcpkg.NewSearchServer,

			// Main gRPC Server
			func(
				lc fx.Lifecycle,
				log *zap.Logger,
				jwt *jwtpkg.JWT,
				db *ormpkg.PostgresClient,
				authServer *authorizationgrpcpkg.AuthorizationServer,
				commentServer *commentgrpcpkg.CommentServer,
				communityServer *communitygrpcpkg.CommunityServer,
				postServer *postgrpcpkg.PostServer,
				searchServer *searchgrpcpkg.SearchServer,
				userServer *usergrpcpkg.UserServer,
			) (*grpcpkg.GRPC, error) {
				grpcServer, err := grpcpkg.NewGRPC(
					log,
					jwt,
					db,
					os.Getenv("GRPC_HOST"),
					os.Getenv("GRPC_PORT"),
					authServer,
					commentServer,
					communityServer,
					postServer,
					searchServer,
					userServer,
				)
				if err != nil {
					return nil, err
				}
				lc.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						return grpcServer.Start()
					},
					OnStop: func(ctx context.Context) error {
						return grpcServer.Stop()
					},
				})
				return grpcServer, nil
			},
		),
		fx.Invoke(func(*grpcpkg.GRPC) {}),
	)
	application.Run()

	err := application.Err()
	if err != nil {
		os.Exit(1)
	}

	return nil
}

func init() {
	rootCommand.AddCommand(serverCommand)
}

