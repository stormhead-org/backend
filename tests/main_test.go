package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

var (
	pgContainer *postgres.PostgresContainer
	pgConnStr   string
)

// TestMain is the entry point for all tests.
func TestMain(m *testing.M) {
	ctx := context.Background()

	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	// Setup test database container
	err := setupTestDatabase(ctx)
	if err != nil {
		logger.Fatal("Failed to set up test database", zap.Error(err))
	}

	// Run all tests
	exitCode := m.Run()

	// Teardown test database container
	err = teardownTestDatabase(ctx)
	if err != nil {
		logger.Error("Failed to tear down test database", zap.Error(err))
	}

	os.Exit(exitCode)
}

func setupTestDatabase(ctx context.Context) error {
	var err error
	pgContainer, err = postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute)),
	)
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}

	pgConnStr, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get postgres connection string: %w", err)
	}

	// Set environment variable for tests to use
	os.Setenv("DATABASE_URL", pgConnStr)
	return nil
}

func teardownTestDatabase(ctx context.Context) error {
	if pgContainer != nil {
		return pgContainer.Terminate(ctx)
	}
	return nil
}
