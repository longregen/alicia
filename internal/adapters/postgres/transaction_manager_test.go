package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
)

func TestTransactionManager_Commit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	txMgr := NewTransactionManager(pool)
	convRepo := NewConversationRepository(pool)

	// Execute successful transaction
	conv := models.NewConversation("ac_tx_commit1", "test-user", "Transaction Commit Test")

	err := txMgr.WithTransaction(context.Background(), func(txCtx context.Context) error {
		return convRepo.Create(txCtx, conv)
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify conversation was committed
	retrieved, err := convRepo.GetByID(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if retrieved.ID != conv.ID {
		t.Error("conversation should be committed")
	}
}

func TestTransactionManager_Rollback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	txMgr := NewTransactionManager(pool)
	convRepo := NewConversationRepository(pool)

	// Execute transaction that returns error
	conv := models.NewConversation("ac_tx_rollback1", "test-user", "Transaction Rollback Test")
	testErr := errors.New("test error")

	err := txMgr.WithTransaction(context.Background(), func(txCtx context.Context) error {
		// Create conversation in transaction
		if err := convRepo.Create(txCtx, conv); err != nil {
			return err
		}
		// Return error to trigger rollback
		return testErr
	})

	if err != testErr {
		t.Fatalf("expected test error, got %v", err)
	}

	// Verify conversation was rolled back
	_, err = convRepo.GetByID(context.Background(), conv.ID)
	if err == nil {
		t.Error("conversation should have been rolled back")
	}
}

func TestTransactionManager_NestedTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	txMgr := NewTransactionManager(pool)
	convRepo := NewConversationRepository(pool)

	// Execute nested transactions
	conv1 := models.NewConversation("ac_tx_nested1", "test-user", "Nested 1")
	conv2 := models.NewConversation("ac_tx_nested2", "test-user", "Nested 2")

	err := txMgr.WithTransaction(context.Background(), func(txCtx context.Context) error {
		// Create first conversation
		if err := convRepo.Create(txCtx, conv1); err != nil {
			return err
		}

		// Nested transaction (should reuse existing)
		return txMgr.WithTransaction(txCtx, func(nestedCtx context.Context) error {
			return convRepo.Create(nestedCtx, conv2)
		})
	})

	if err != nil {
		t.Fatalf("Nested transaction failed: %v", err)
	}

	// Verify both conversations were committed
	if _, err := convRepo.GetByID(context.Background(), conv1.ID); err != nil {
		t.Error("first conversation should be committed")
	}
	if _, err := convRepo.GetByID(context.Background(), conv2.ID); err != nil {
		t.Error("second conversation should be committed")
	}
}

func TestTransactionManager_NestedRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	txMgr := NewTransactionManager(pool)
	convRepo := NewConversationRepository(pool)

	// Execute nested transactions with error
	conv1 := models.NewConversation("ac_tx_nested_rb1", "test-user", "Nested RB 1")
	conv2 := models.NewConversation("ac_tx_nested_rb2", "test-user", "Nested RB 2")
	testErr := errors.New("nested error")

	err := txMgr.WithTransaction(context.Background(), func(txCtx context.Context) error {
		// Create first conversation
		if err := convRepo.Create(txCtx, conv1); err != nil {
			return err
		}

		// Nested transaction that fails
		return txMgr.WithTransaction(txCtx, func(nestedCtx context.Context) error {
			if err := convRepo.Create(nestedCtx, conv2); err != nil {
				return err
			}
			return testErr
		})
	})

	if err != testErr {
		t.Fatalf("expected test error, got %v", err)
	}

	// Verify both conversations were rolled back
	if _, err := convRepo.GetByID(context.Background(), conv1.ID); err == nil {
		t.Error("first conversation should be rolled back")
	}
	if _, err := convRepo.GetByID(context.Background(), conv2.ID); err == nil {
		t.Error("second conversation should be rolled back")
	}
}

func TestTransactionManager_GetTx_NoTransaction(t *testing.T) {
	ctx := context.Background()

	tx := GetTx(ctx)
	if tx != nil {
		t.Error("expected nil transaction in empty context")
	}
}

func TestTransactionManager_GetTx_WithTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	txMgr := NewTransactionManager(pool)

	err := txMgr.WithTransaction(context.Background(), func(txCtx context.Context) error {
		tx := GetTx(txCtx)
		if tx == nil {
			t.Error("expected transaction in transaction context")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}
}

func TestTransactionManager_GetConn_Pool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	ctx := context.Background()
	conn := GetConn(ctx, pool)

	if conn == nil {
		t.Error("expected connection from pool")
	}
}

func TestTransactionManager_GetConn_Transaction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	txMgr := NewTransactionManager(pool)

	err := txMgr.WithTransaction(context.Background(), func(txCtx context.Context) error {
		conn := GetConn(txCtx, pool)
		if conn == nil {
			t.Error("expected connection from transaction")
		}

		// Verify it's the transaction, not the pool
		tx := GetTx(txCtx)
		if tx == nil {
			t.Error("expected transaction in context")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}
}
