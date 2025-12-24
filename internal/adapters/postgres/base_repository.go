package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BaseRepository provides common functionality for all repository implementations.
// It manages the database pool and provides transaction-aware connection handling.
type BaseRepository struct {
	pool *pgxpool.Pool
}

// NewBaseRepository creates a new base repository with the given connection pool.
func NewBaseRepository(pool *pgxpool.Pool) BaseRepository {
	return BaseRepository{pool: pool}
}

// Pool returns the underlying connection pool.
// This is provided for repositories that need direct pool access,
// but most repositories should use conn() instead.
func (r *BaseRepository) Pool() *pgxpool.Pool {
	return r.pool
}

// conn returns the appropriate database connection based on the context.
// If the context contains a transaction, it returns the transaction.
// Otherwise, it returns the connection pool.
// This ensures repositories automatically participate in transactions when available.
func (r *BaseRepository) conn(ctx context.Context) interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
} {
	return GetConn(ctx, r.pool)
}
