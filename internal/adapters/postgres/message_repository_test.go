package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/domain/models"
)

func TestMessageRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation first
	conv := models.NewConversation("ac_msg_test1", "test-user", "Message Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create message
	msg := models.NewMessage("msg_test1", conv.ID, 1, models.MessageRoleUser, "Hello, world!")

	err := repo.Create(context.Background(), msg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify retrieval
	retrieved, err := repo.GetByID(context.Background(), msg.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != msg.ID {
		t.Errorf("expected ID %s, got %s", msg.ID, retrieved.ID)
	}
	if retrieved.Contents != "Hello, world!" {
		t.Errorf("expected contents 'Hello, world!', got %s", retrieved.Contents)
	}
	if retrieved.Role != models.MessageRoleUser {
		t.Errorf("expected role user, got %s", retrieved.Role)
	}
}

func TestMessageRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)

	_, err := repo.GetByID(context.Background(), "nonexistent")
	if err != pgx.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestMessageRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_update1", "test-user", "Update Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create message
	msg := models.NewMessage("msg_update1", conv.ID, 1, models.MessageRoleAssistant, "Original")
	if err := repo.Create(context.Background(), msg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update contents
	msg.Contents = "Updated content"
	msg.CompletionStatus = models.CompletionStatusCompleted

	if err := repo.Update(context.Background(), msg); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetByID(context.Background(), msg.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Contents != "Updated content" {
		t.Errorf("expected updated content, got %s", retrieved.Contents)
	}
	if retrieved.CompletionStatus != models.CompletionStatusCompleted {
		t.Errorf("expected completion status completed, got %s", retrieved.CompletionStatus)
	}
}

func TestMessageRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_delete1", "test-user", "Delete Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create message
	msg := models.NewMessage("msg_delete1", conv.ID, 1, models.MessageRoleUser, "To delete")
	if err := repo.Create(context.Background(), msg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete it
	if err := repo.Delete(context.Background(), msg.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should not be retrievable
	_, err := repo.GetByID(context.Background(), msg.ID)
	if err != pgx.ErrNoRows {
		t.Errorf("expected message to be not found after deletion")
	}
}

func TestMessageRepository_GetByConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_getconv1", "test-user", "Get Conversation Messages")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create multiple messages
	msg1 := models.NewMessage("msg_conv1", conv.ID, 1, models.MessageRoleUser, "First")
	msg2 := models.NewMessage("msg_conv2", conv.ID, 2, models.MessageRoleAssistant, "Second")
	msg3 := models.NewMessage("msg_conv3", conv.ID, 3, models.MessageRoleUser, "Third")

	for _, msg := range []*models.Message{msg1, msg2, msg3} {
		if err := repo.Create(context.Background(), msg); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Get all messages for conversation
	messages, err := repo.GetByConversation(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetByConversation failed: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}

	// Should be ordered by sequence number
	if messages[0].SequenceNumber != 1 {
		t.Errorf("expected first message sequence 1, got %d", messages[0].SequenceNumber)
	}
	if messages[2].SequenceNumber != 3 {
		t.Errorf("expected third message sequence 3, got %d", messages[2].SequenceNumber)
	}
}

func TestMessageRepository_GetLatestByConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_latest1", "test-user", "Latest Messages")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create 5 messages
	for i := 1; i <= 5; i++ {
		msg := models.NewMessage(
			fmt.Sprintf("msg_latest%d", i),
			conv.ID,
			i,
			models.MessageRoleUser,
			fmt.Sprintf("Message %d", i),
		)
		if err := repo.Create(context.Background(), msg); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Get latest 3 messages
	messages, err := repo.GetLatestByConversation(context.Background(), conv.ID, 3)
	if err != nil {
		t.Fatalf("GetLatestByConversation failed: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}

	// Should be in ascending order (3, 4, 5)
	if messages[0].SequenceNumber != 3 {
		t.Errorf("expected first message sequence 3, got %d", messages[0].SequenceNumber)
	}
	if messages[2].SequenceNumber != 5 {
		t.Errorf("expected last message sequence 5, got %d", messages[2].SequenceNumber)
	}
}

func TestMessageRepository_GetNextSequenceNumber(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_seq1", "test-user", "Sequence Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// First sequence should be 1
	seq1, err := repo.GetNextSequenceNumber(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetNextSequenceNumber failed: %v", err)
	}
	if seq1 != 1 {
		t.Errorf("expected first sequence to be 1, got %d", seq1)
	}

	// Create a message
	msg := models.NewMessage("msg_seq1", conv.ID, seq1, models.MessageRoleUser, "First")
	if err := repo.Create(context.Background(), msg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Next sequence should be 2
	seq2, err := repo.GetNextSequenceNumber(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetNextSequenceNumber failed: %v", err)
	}
	if seq2 != 2 {
		t.Errorf("expected second sequence to be 2, got %d", seq2)
	}
}

func TestMessageRepository_GetNextSequenceNumber_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)
	txMgr := NewTransactionManager(pool)

	// Create conversation with unique ID using timestamp
	convID := fmt.Sprintf("ac_msg_concurrent_%d", time.Now().UnixNano())
	conv := models.NewConversation(convID, "test-user", "Concurrent Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Try to get sequence numbers concurrently
	numGoroutines := 10
	seqChan := make(chan int, numGoroutines)
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			var seq int
			// Wrap both getting sequence and creating message in a single transaction
			// This ensures the advisory lock is held until the message is created
			err := txMgr.WithTransaction(context.Background(), func(txCtx context.Context) error {
				var err error
				seq, err = repo.GetNextSequenceNumber(txCtx, conv.ID)
				if err != nil {
					return err
				}

				// Create a message with this sequence number to actually reserve it
				// This must happen in the same transaction to hold the advisory lock
				msg := models.NewMessage(fmt.Sprintf("msg_concurrent_%d_%d_%d", id, seq, time.Now().UnixNano()), conv.ID, seq, models.MessageRoleUser, fmt.Sprintf("Message %d", id))
				if err := repo.Create(txCtx, msg); err != nil {
					return err
				}

				return nil
			})

			if err != nil {
				errChan <- err
				return
			}

			seqChan <- seq
		}(i)
	}

	// Collect results
	sequences := make(map[int]bool)
	for i := 0; i < numGoroutines; i++ {
		select {
		case seq := <-seqChan:
			if sequences[seq] {
				t.Errorf("duplicate sequence number %d", seq)
			}
			sequences[seq] = true
		case err := <-errChan:
			t.Errorf("error getting sequence: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for sequence numbers")
		}
	}

	// All sequences should be unique and sequential
	if len(sequences) != numGoroutines {
		t.Errorf("expected %d unique sequences, got %d", numGoroutines, len(sequences))
	}
}

func TestMessageRepository_GetAfterSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_after1", "test-user", "After Sequence Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create messages 1-5
	for i := 1; i <= 5; i++ {
		msg := models.NewMessage(
			fmt.Sprintf("msg_after%d", i),
			conv.ID,
			i,
			models.MessageRoleUser,
			fmt.Sprintf("Message %d", i),
		)
		if err := repo.Create(context.Background(), msg); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Get messages after sequence 2
	messages, err := repo.GetAfterSequence(context.Background(), conv.ID, 2)
	if err != nil {
		t.Fatalf("GetAfterSequence failed: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("expected 3 messages after sequence 2, got %d", len(messages))
	}

	// Should start with sequence 3
	if messages[0].SequenceNumber != 3 {
		t.Errorf("expected first message sequence 3, got %d", messages[0].SequenceNumber)
	}
}

func TestMessageRepository_GetPendingSync(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_pending1", "test-user", "Pending Sync Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create pending message
	pending := models.NewLocalMessage("local_123", conv.ID, 1, models.MessageRoleUser, "Pending")
	pending.CompletionStatus = models.CompletionStatusCompleted
	if err := repo.Create(context.Background(), pending); err != nil {
		t.Fatalf("Create pending failed: %v", err)
	}

	// Create synced message
	synced := models.NewMessage("msg_synced1", conv.ID, 2, models.MessageRoleUser, "Synced")
	synced.SyncStatus = models.SyncStatusSynced
	if err := repo.Create(context.Background(), synced); err != nil {
		t.Fatalf("Create synced failed: %v", err)
	}

	// Get pending messages
	pendingMessages, err := repo.GetPendingSync(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetPendingSync failed: %v", err)
	}

	if len(pendingMessages) != 1 {
		t.Errorf("expected 1 pending message, got %d", len(pendingMessages))
	}

	if pendingMessages[0].ID != pending.ID {
		t.Errorf("expected pending message ID %s, got %s", pending.ID, pendingMessages[0].ID)
	}
}

func TestMessageRepository_GetByLocalID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_local1", "test-user", "Local ID Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create message with local ID
	msg := models.NewLocalMessage("local_456", conv.ID, 1, models.MessageRoleUser, "Local message")
	msg.CompletionStatus = models.CompletionStatusCompleted
	if err := repo.Create(context.Background(), msg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Retrieve by local ID
	retrieved, err := repo.GetByLocalID(context.Background(), "local_456")
	if err != nil {
		t.Fatalf("GetByLocalID failed: %v", err)
	}

	if retrieved.LocalID != "local_456" {
		t.Errorf("expected local ID local_456, got %s", retrieved.LocalID)
	}
}

func TestMessageRepository_GetIncompleteOlderThan(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_incomplete1", "test-user", "Incomplete Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create old incomplete message
	oldIncomplete := models.NewMessage("msg_incomplete_old", conv.ID, 1, models.MessageRoleAssistant, "Old")
	oldIncomplete.CompletionStatus = models.CompletionStatusStreaming
	oldIncomplete.CreatedAt = time.Now().Add(-2 * time.Hour)
	if err := repo.Create(context.Background(), oldIncomplete); err != nil {
		t.Fatalf("Create old incomplete failed: %v", err)
	}

	// Create recent incomplete message
	recentIncomplete := models.NewMessage("msg_incomplete_recent", conv.ID, 2, models.MessageRoleAssistant, "Recent")
	recentIncomplete.CompletionStatus = models.CompletionStatusPending
	if err := repo.Create(context.Background(), recentIncomplete); err != nil {
		t.Fatalf("Create recent incomplete failed: %v", err)
	}

	// Get incomplete older than 1 hour
	cutoff := time.Now().Add(-1 * time.Hour)
	messages, err := repo.GetIncompleteOlderThan(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("GetIncompleteOlderThan failed: %v", err)
	}

	// Should only contain old message
	found := false
	for _, msg := range messages {
		if msg.ID == oldIncomplete.ID {
			found = true
		}
		if msg.ID == recentIncomplete.ID {
			t.Error("recent incomplete message should not be in results")
		}
	}

	if !found {
		t.Error("old incomplete message not found")
	}
}

func TestMessageRepository_GetIncompleteByConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create two conversations
	conv1 := models.NewConversation("ac_msg_incomplete_conv1", "test-user", "Conversation 1")
	conv2 := models.NewConversation("ac_msg_incomplete_conv2", "test-user", "Conversation 2")
	for _, c := range []*models.Conversation{conv1, conv2} {
		if err := convRepo.Create(context.Background(), c); err != nil {
			t.Fatalf("Failed to create conversation: %v", err)
		}
	}

	// Create old incomplete in conv1
	msg1 := models.NewMessage("msg_incomplete_c1", conv1.ID, 1, models.MessageRoleAssistant, "Conv1")
	msg1.CompletionStatus = models.CompletionStatusFailed
	msg1.CreatedAt = time.Now().Add(-2 * time.Hour)
	if err := repo.Create(context.Background(), msg1); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create old incomplete in conv2
	msg2 := models.NewMessage("msg_incomplete_c2", conv2.ID, 1, models.MessageRoleAssistant, "Conv2")
	msg2.CompletionStatus = models.CompletionStatusStreaming
	msg2.CreatedAt = time.Now().Add(-2 * time.Hour)
	if err := repo.Create(context.Background(), msg2); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get incomplete for conv1 only
	cutoff := time.Now().Add(-1 * time.Hour)
	messages, err := repo.GetIncompleteByConversation(context.Background(), conv1.ID, cutoff)
	if err != nil {
		t.Fatalf("GetIncompleteByConversation failed: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	if messages[0].ID != msg1.ID {
		t.Errorf("expected message from conv1, got %s", messages[0].ID)
	}
}

func TestMessageRepository_GetByLocalID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)

	_, err := repo.GetByLocalID(context.Background(), "nonexistent_local_id")
	if err != pgx.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestMessageRepository_EmptyConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create empty conversation
	conv := models.NewConversation("ac_msg_empty1", "test-user", "Empty Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Get messages from empty conversation
	messages, err := repo.GetByConversation(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetByConversation failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 messages in empty conversation, got %d", len(messages))
	}
}

func TestMessageRepository_GetLatestByConversation_LimitGreaterThanTotal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_limit1", "test-user", "Limit Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create 3 messages
	for i := 1; i <= 3; i++ {
		msg := models.NewMessage(
			fmt.Sprintf("msg_limit%d", i),
			conv.ID,
			i,
			models.MessageRoleUser,
			fmt.Sprintf("Message %d", i),
		)
		if err := repo.Create(context.Background(), msg); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Request 10 messages (more than available)
	messages, err := repo.GetLatestByConversation(context.Background(), conv.ID, 10)
	if err != nil {
		t.Fatalf("GetLatestByConversation failed: %v", err)
	}

	// Should return all 3 messages
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}
}

func TestMessageRepository_UpdateNonexistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation for valid foreign key
	conv := models.NewConversation("ac_msg_nonexist1", "test-user", "Test")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Try to update nonexistent message
	msg := models.NewMessage("nonexistent_msg", conv.ID, 1, models.MessageRoleUser, "Test")
	err := repo.Update(context.Background(), msg)

	// Should not return error (UPDATE succeeds with 0 rows affected)
	if err != nil {
		t.Errorf("Update of nonexistent message returned error: %v", err)
	}
}

func TestMessageRepository_DeleteNonexistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)

	// Try to delete nonexistent message
	err := repo.Delete(context.Background(), "nonexistent_msg")

	// Should not return error (UPDATE succeeds with 0 rows affected)
	if err != nil {
		t.Errorf("Delete of nonexistent message returned error: %v", err)
	}
}

func TestMessageRepository_GetAfterSequence_NoResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_after_empty1", "test-user", "After Empty")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create messages 1-3
	for i := 1; i <= 3; i++ {
		msg := models.NewMessage(
			fmt.Sprintf("msg_after_empty%d", i),
			conv.ID,
			i,
			models.MessageRoleUser,
			fmt.Sprintf("Message %d", i),
		)
		if err := repo.Create(context.Background(), msg); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Get messages after sequence 100 (beyond all messages)
	messages, err := repo.GetAfterSequence(context.Background(), conv.ID, 100)
	if err != nil {
		t.Fatalf("GetAfterSequence failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 messages after sequence 100, got %d", len(messages))
	}
}

func TestMessageRepository_GetPendingSync_NoResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_msg_pending_empty1", "test-user", "Pending Empty")
	if err := convRepo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Create only synced messages
	msg := models.NewMessage("msg_synced_only", conv.ID, 1, models.MessageRoleUser, "Synced")
	msg.SyncStatus = models.SyncStatusSynced
	if err := repo.Create(context.Background(), msg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get pending messages
	messages, err := repo.GetPendingSync(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetPendingSync failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 pending messages, got %d", len(messages))
	}
}
