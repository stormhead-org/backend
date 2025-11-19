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
	communitygrpcpkg "github.com/stormhead-org/backend/internal/grpc/community"
	jwtpkg "github.com/stormhead-org/backend/internal/jwt"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	"github.com/stormhead-org/backend/internal/services/community"
	"github.com/stormhead-org/backend/internal/services/post"
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
				if os.Getenv("DEBUG") == "1" {
					godotenv.Load()
				}
				jwtSecret := os.Getenv("JWT_SECRET")
				if jwtSecret == "" {
					jwtSecret = "123456"
				}
				return jwtpkg.NewJWT(jwtSecret), nil
			},

			// Clients
			func(logger *zap.Logger) (*ormpkg.PostgresClient, error) {
				if os.Getenv("DEBUG") == "1" {
					godotenv.Load()
				}
				return ormpkg.NewPostgresClient(
					os.Getenv("POSTGRES_HOST"),
					os.Getenv("POSTGRES_PORT"),
					os.Getenv("POSTGRES_USER"),
					os.Getenv("POSTGRES_PASSWORD"),
				)
			},
			func(logger *zap.Logger) (*eventpkg.KafkaClient, error) {
				if os.Getenv("DEBUG") == "1" {
					godotenv.Load()
				}
				return eventpkg.NewKafkaClient(
					os.Getenv("KAFKA_HOST"),
					os.Getenv("KAFKA_PORT"),
					os.Getenv("KAFKA_TOPIC"),
					os.Getenv("KAFKA_GROUP"),
				)
			},
			clientpkg.NewHIBPClient,

			// Services
			community.NewCommunityService,
			post.NewPostService,

			// gRPC Servers
			authorizationgrpcpkg.NewAuthorizationServer,
			communitygrpcpkg.NewCommunityServer,
			grpcpkg.NewPostServer,
			grpcpkg.NewCommentServer,
			grpcpkg.NewUserServer,

			// Main gRPC Server
			func(
				lc fx.Lifecycle,
				logger *zap.Logger,
				jwt *jwtpkg.JWT,
				db *ormpkg.PostgresClient,
				authServer *authorizationgrpcpkg.AuthorizationServer,
				communityServer *communitygrpcpkg.CommunityServer,
				postServer *grpcpkg.PostServer,
				commentServer *grpcpkg.CommentServer,
				userServer *grpcpkg.UserServer,
			) (*grpcpkg.GRPC, error) {
				grpcServer, err := grpcpkg.NewGRPC(
					logger,
					os.Getenv("GRPC_HOST"),
					os.Getenv("GRPC_PORT"),
					jwt,
					db,
					authServer,
					communityServer,
					postServer,
					commentServer,
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
