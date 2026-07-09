package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// findRepoRoot walks up the directory tree until it finds go.mod
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		dir = parent
	}
}

// SetupTestDB starts an ephemeral PostgreSQL container, runs schema migrations,
// and returns an active *sql.DB connection. It automatically registers cleanup.
func SetupTestDB(t *testing.T) (*sql.DB, error) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("gencon"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Duration(time.Second)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Ensure connection is active
	if pingErr := db.PingContext(ctx); pingErr != nil {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", pingErr)
	}

	// Register cleanup on test completion
	t.Cleanup(func() {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
	})

	// Find repo root to locate migration files
	root, err := findRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find repo root: %w", err)
	}

	migrationFiles := []string{
		filepath.Join(root, "internal", "postgres", "schema.sql"),
		filepath.Join(root, "internal", "postgres", "party_mode.sql"),
		filepath.Join(root, "internal", "postgres", "flexible_blocks.sql"),
		filepath.Join(root, "internal", "postgres", "cluster_key_update.sql"),
		filepath.Join(root, "internal", "postgres", "remove_org_trigger.sql"),
	}

	for _, file := range migrationFiles {
		// #nosec G304
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file, err)
		}
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			return nil, fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return db, nil
}
