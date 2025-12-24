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

	// Check if we're already in a transaction
	if GetTx(ctx) != nil {
		// Nested transaction - just execute the function
		return fn(ctx)
	}

	// Begin a new transaction
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Add transaction to context
	txCtx := context.WithValue(ctx, txKey, tx)

	// Ensure rollback on panic
	defer func() {
		if r := recover(); r != nil {
			// Rollback on panic
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("panic recovered: %v, rollback error: %w", r, rbErr)
			} else {
				err = fmt.Errorf("panic recovered in transaction: %v", r)
			}
		}
	}()

	// Execute the function
	err = fn(txCtx)
	if err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	// Commit the transaction
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
