// Package db provides database connection helpers used across services.
package db

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	URL      string
	Timezone string
}

// Connect creates a traced connection pool.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	tz := cfg.Timezone
	if tz == "" {
		tz = "UTC"
	}
	poolConfig.ConnConfig.RuntimeParams["timezone"] = tz
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer(otelpgx.WithTrimSQLInSpanName())

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// ConnectSimple creates a traced connection pool with default settings.
func ConnectSimple(ctx context.Context, url string) (*pgxpool.Pool, error) {
	return Connect(ctx, Config{URL: url})
}

// MustConnect panics on error. Useful for main() initialization.
func MustConnect(ctx context.Context, cfg Config) *pgxpool.Pool {
	pool, err := Connect(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("database connection failed: %v", err))
	}
	return pool
}
