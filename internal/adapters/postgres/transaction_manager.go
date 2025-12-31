package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// contextKey is a type for transaction context keys
type contextKey string

const txKey contextKey = "pgx_tx"

// TransactionManager implements the ports.TransactionManager interface
type TransactionManager struct {
	pool *pgxpool.Pool
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(pool *pgxpool.Pool) *TransactionManager {
	return &TransactionManager{pool: pool}
}

// WithTransaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	if GetTx(ctx) != nil {
		return fn(ctx)
	}

	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txKey, tx)

	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("panic recovered: %v, rollback error: %w", r, rbErr)
			} else {
				err = fmt.Errorf("panic recovered in transaction: %v", r)
			}
		}
	}()

	err = fn(txCtx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetTx retrieves the transaction from the context, if any
func GetTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey).(pgx.Tx); ok {
		return tx
	}
	return nil
}

// GetConn returns either the transaction or the pool based on context
// This is a helper for repositories to use the correct connection
func GetConn(ctx context.Context, pool *pgxpool.Pool) interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
} {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return pool
}
