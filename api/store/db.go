package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/shared/id"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

type txKey struct{}

func (s *Store) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if tx := txFromContext(ctx); tx != nil {
		return fn(ctx)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	ctx = context.WithValue(ctx, txKey{}, tx)

	if err := fn(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

func txFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey{}).(pgx.Tx)
	return tx
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (s *Store) conn(ctx context.Context) querier {
	if tx := txFromContext(ctx); tx != nil {
		return tx
	}
	return s.pool
}

// NewID generates a new ID with the given prefix (for custom prefixes)
func NewID(prefix string) string {
	return id.New(prefix)
}

// Re-export ID functions from internal/id for backward compatibility
var (
	NewConversationID      = id.NewConversation
	NewMessageID           = id.NewMessage
	NewMemoryID            = id.NewMemory
	NewMemoryUseID         = id.NewMemoryUse
	NewToolID              = id.NewTool
	NewToolUseID           = id.NewToolUse
	NewMCPServerID         = id.NewMCPServer
	NewMessageFeedbackID   = id.NewMessageFeedback
	NewToolUseFeedbackID   = id.NewToolUseFeedback
	NewMemoryUseFeedbackID = id.NewMemoryUseFeedback
)
