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
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	workerpkg "github.com/stormhead-org/backend/internal/worker"
)

var workerCommand = &cobra.Command{
	Use:   "worker",
	Short: "worker",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return workerCommandImpl()
	},
}

func workerCommandImpl() error {
	var logger interface{}
	if os.Getenv("DEBUG") == "1" {
		logger = zap.NewDevelopment
	} else {
		logger = zap.NewProduction
	}

	if os.Getenv("DEBUG") == "1" {
		godotenv.Load()
	}

	// Application
	application := fx.New(
		fx.NopLogger,
		fx.Provide(
			logger,

			// Kafka client
			func(logger *zap.Logger) (*eventpkg.KafkaClient, error) {
				kafkaHost := os.Getenv("KAFKA_HOST")
				if kafkaHost == "" {
					kafkaHost = "127.0.0.1"
				}

				kafkaPort := os.Getenv("KAFKA_PORT")
				if kafkaPort == "" {
					kafkaPort = "9092"
				}

				kafkaTopic := os.Getenv("KAFKA_TOPIC")
				if kafkaTopic == "" {
					kafkaTopic = "common"
				}

				kafkaGroup := os.Getenv("KAFKA_GROUP")
				if kafkaGroup == "" {
					kafkaGroup = "common"
				}

				kafkaClient, err := eventpkg.NewKafkaClient(
					kafkaHost,
					kafkaPort,
					kafkaTopic,
					kafkaGroup,
				)
				return kafkaClient, err
			},

			// Mail client
			func(logger *zap.Logger) (*clientpkg.MailClient, error) {
				smtpHost := os.Getenv("SMTP_HOST")
				if smtpHost == "" {
					smtpHost = "127.0.0.1"
				}

				smtpPort := os.Getenv("SMTP_PORT")
				if smtpPort == "" {
					smtpPort = "587"
				}

				smtpUser := os.Getenv("SMTP_USER")
				if smtpUser == "" {
					smtpUser = "user"
				}

				smtpPassword := os.Getenv("SMTP_PASSWORD")
				if smtpPassword == "" {
					smtpPassword = "password"
				}

				mailClient := clientpkg.NewMailClient(
					smtpHost,
					smtpPort,
					smtpUser,
					smtpPassword,
				)
				return mailClient, nil
			},

			func(lifecycle fx.Lifecycle, shutdowner fx.Shutdowner, logger *zap.Logger) (*ormpkg.PostgresClient, error) {
				postgresHost := os.Getenv("POSTGRES_HOST")
				if postgresHost == "" {
					postgresHost = "127.0.0.1"
				}

				postgresPort := os.Getenv("POSTGRES_PORT")
				if postgresPort == "" {
					postgresPort = "5432"
				}

				postgresUser := os.Getenv("POSTGRES_USER")
				if postgresUser == "" {
					postgresUser = "postgres"
				}

				postgresPassword := os.Getenv("POSTGRES_PASSWORD")
				if postgresPassword == "" {
					postgresPassword = "postgres"
				}

				client, err := ormpkg.NewPostgresClient(
					postgresHost,
					postgresPort,
					postgresUser,
					postgresPassword,
				)
				if err != nil {
					return nil, err
				}

				return client, err
			},

			// Application
			func(
				lifecycle fx.Lifecycle,
				shutdowner fx.Shutdowner,
				logger *zap.Logger,
				kafkaClient *eventpkg.KafkaClient,
				mailClient *clientpkg.MailClient,
				databaseClient *ormpkg.PostgresClient,
			) (*workerpkg.Worker, error) {
				verificationURL := os.Getenv("VERIFICATION_URL")
				if verificationURL == "" {
					verificationURL = "http://localhost:3000/verify-email"
				}
				passwordResetURL := os.Getenv("PASSWORD_RESET_URL")
				if passwordResetURL == "" {
					passwordResetURL = "http://localhost:3000/reset-password"
				}
				config := &workerpkg.Config{
					VerificationURL:   verificationURL,
					PasswordResetURL: passwordResetURL,
				}

				worker := workerpkg.NewWorker(logger, kafkaClient, mailClient, databaseClient, config)

				lifecycle.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						return worker.Start()
					},
					OnStop: func(ctx context.Context) error {
						return worker.Stop()
					},
				})

				return worker, nil
			},
		),
		fx.Invoke(
			func(*eventpkg.KafkaClient) {},
			func(*clientpkg.MailClient) {},
			func(*workerpkg.Worker) {},
		),
	)
	application.Run()

	err := application.Err()
	if err != nil {
		os.Exit(1)
	}

	return nil
}

func init() {
	rootCommand.AddCommand(workerCommand)
}
