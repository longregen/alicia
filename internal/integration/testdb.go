//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestDB manages a test database instance
type TestDB struct {
	Pool *pgxpool.Pool
	DSN  string
}

// SetupTestDB creates a test database with migrations applied
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Get PostgreSQL connection from environment or use default
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "alicia")
	password := getEnv("POSTGRES_PASSWORD", "alicia")
	dbName := getEnv("POSTGRES_DB", "alicia_test")

	// Create database if it doesn't exist
	adminDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		user, password, host, port)

	db, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	// Drop and recreate database for clean state
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		t.Fatalf("failed to drop test database: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Connect to the test database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbName)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Run migrations
	if err := runMigrations(pool); err != nil {
		pool.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	testDB := &TestDB{
		Pool: pool,
		DSN:  dsn,
	}

	// Register cleanup
	t.Cleanup(func() {
		pool.Close()
	})

	return testDB
}

// runMigrations applies all migrations to the test database
func runMigrations(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// Read and execute migration files
	migrations := []string{
		"migrations/001_init.up.sql",
		"migrations/002_add_message_sync_tracking.up.sql",
		"migrations/003_add_stanza_id_tracking.up.sql",
		"migrations/004_add_completion_status.up.sql",
	}

	for _, migration := range migrations {
		content, err := os.ReadFile(migration)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration, err)
		}

		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration, err)
		}
	}

	return nil
}

// Clear removes all data from tables while preserving schema
func (db *TestDB) Clear(ctx context.Context) error {
	tables := []string{
		"alicia_meta",
		"alicia_user_conversation_commentaries",
		"alicia_reasoning_steps",
		"alicia_tool_uses",
		"alicia_tools",
		"alicia_memory_used",
		"alicia_memory",
		"alicia_audio",
		"alicia_sentences",
		"alicia_messages",
		"alicia_conversations",
	}

	for _, table := range tables {
		_, err := db.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	return nil
}

// GetConnection returns a new connection from the pool
func (db *TestDB) GetConnection(ctx context.Context) (*pgxpool.Conn, error) {
	return db.Pool.Acquire(ctx)
}

// WaitForReady waits for the database to be ready
func (db *TestDB) WaitForReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := db.Pool.Ping(ctx)
		if err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("database not ready after %v", timeout)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
