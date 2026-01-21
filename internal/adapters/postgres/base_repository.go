package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BaseRepository struct {
	pool *pgxpool.Pool
}

func NewBaseRepository(pool *pgxpool.Pool) BaseRepository {
	return BaseRepository{pool: pool}
}

func (r *BaseRepository) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *BaseRepository) conn(ctx context.Context) interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
} {
	return GetConn(ctx, r.pool)
}
