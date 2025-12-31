package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamingMessageCleanup validates Bug #4: Browser refresh during streaming leaves partial messages
// This test simulates:
// 1. A message being created and marked as streaming
// 2. Browser refresh (simulated by NOT calling MarkAsCompleted)
// 3. Checking if GetIncompleteOlderThan can find and clean up the orphaned message
func TestStreamingMessageCleanup(t *testing.T) {
	pool := setupTestDB(t)

	ctx := context.Background()
	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	// Create a test conversation (using ac_test prefix for cleanup)
	conversationID := "ac_test_cleanup_1"
	conversation := models.NewConversation(conversationID, "test-user", "Test Cleanup Conversation")
	err := convRepo.Create(ctx, conversation)
	require.NoError(t, err, "Failed to create test conversation")

	// STEP 1: Simulate message creation during streaming
	message := &models.Message{
		ID:               "msg-streaming-1",
		ConversationID:   conversationID,
		SequenceNumber:   1,
		Role:             models.MessageRoleAssistant,
		Contents:         "Partial content...",
		CompletionStatus: models.CompletionStatusStreaming, // Marked as streaming
		SyncStatus:       models.SyncStatusSynced,
		CreatedAt:        time.Now().Add(-2 * time.Hour), // Created 2 hours ago
		UpdatedAt:        time.Now().Add(-2 * time.Hour),
	}

	err = repo.Create(ctx, message)
	require.NoError(t, err, "Failed to create streaming message")

	// STEP 2: Verify the message exists and is in streaming state
	fetchedMsg, err := repo.GetByID(ctx, message.ID)
	require.NoError(t, err, "Failed to fetch message")
	assert.Equal(t, models.CompletionStatusStreaming, fetchedMsg.CompletionStatus, "Message should be in streaming state")

	// STEP 3: Simulate browser refresh - the message is now orphaned
	// In reality, the browser closes and the defer block in generate_response.go never runs
	// So the message stays in 'streaming' state forever

	// STEP 4: Check if GetIncompleteOlderThan can find this orphaned message
	cutoffTime := time.Now().Add(-1 * time.Hour) // Messages older than 1 hour
	incompleteMessages, err := repo.GetIncompleteOlderThan(ctx, cutoffTime)
	require.NoError(t, err, "Failed to get incomplete messages")

	// BUG VALIDATION: Does the cleanup mechanism find the orphaned streaming message?
	assert.NotEmpty(t, incompleteMessages, "REAL BUG: GetIncompleteOlderThan should find orphaned streaming messages")

	if len(incompleteMessages) > 0 {
		found := false
		for _, msg := range incompleteMessages {
			if msg.ID == message.ID {
				found = true
				assert.Equal(t, models.CompletionStatusStreaming, msg.CompletionStatus)
				t.Logf("✓ Found orphaned streaming message: %s (created %v ago)", msg.ID, time.Since(msg.CreatedAt))
			}
		}
		assert.True(t, found, "The specific orphaned message should be in the results")
	}

	// STEP 5: Verify cleanup by conversation also works
	incompleteByConv, err := repo.GetIncompleteByConversation(ctx, conversationID, cutoffTime)
	require.NoError(t, err, "Failed to get incomplete messages by conversation")
	assert.NotEmpty(t, incompleteByConv, "GetIncompleteByConversation should also find the orphaned message")

	// STEP 6: Test that marking as failed works
	message.MarkAsFailed()
	err = repo.Update(ctx, message)
	require.NoError(t, err, "Failed to update message to failed state")

	// Verify the status changed
	updatedMsg, err := repo.GetByID(ctx, message.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CompletionStatusFailed, updatedMsg.CompletionStatus, "Message should now be marked as failed")
}

// TestStreamingMessageCleanup_NotCalledOnStartup validates Bug #4 part 2:
// Check if the cleanup is actually invoked anywhere (startup, background job, etc.)
func TestStreamingMessageCleanup_NotCalledOnStartup(t *testing.T) {
	// This test is more of a documentation test - it validates that
	// GetIncompleteOlderThan EXISTS but is NOT called automatically
	//
	// Based on code analysis:
	// 1. The cleanup function EXISTS in internal/application/usecases/cleanup_orphaned_data.go
	// 2. It's properly implemented with GetIncompleteOlderThan
	// 3. BUT there's NO code in cmd/alicia/serve.go that calls it on startup
	// 4. AND there's NO background goroutine running periodic cleanup
	//
	// This means orphaned streaming messages will remain in the database forever
	// unless manually cleaned up.

	t.Skip("This test documents the absence of automatic cleanup - see code comments")

	// To fix this bug, the cleanup should be called:
	// Option 1: On server startup in cmd/alicia/serve.go
	// Option 2: As a periodic background job (every 5-15 minutes)
	// Option 3: On new conversation start (cleanup old orphans)
}

// TestCompleteStreamingFlow validates the happy path
func TestCompleteStreamingFlow(t *testing.T) {
	pool := setupTestDB(t)

	ctx := context.Background()
	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	conversationID := "ac_test_complete"
	conversation := models.NewConversation(conversationID, "test-user", "Test Complete Conversation")
	err := convRepo.Create(ctx, conversation)
	require.NoError(t, err, "Failed to create test conversation")

	// Create a message and mark it as streaming
	message := &models.Message{
		ID:               "msg-complete-1",
		ConversationID:   conversationID,
		SequenceNumber:   1,
		Role:             models.MessageRoleAssistant,
		Contents:         "Complete content",
		CompletionStatus: models.CompletionStatusStreaming,
		SyncStatus:       models.SyncStatusSynced,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err = repo.Create(ctx, message)
	require.NoError(t, err)

	// Simulate successful streaming completion
	message.MarkAsCompleted()
	err = repo.Update(ctx, message)
	require.NoError(t, err)

	// Verify completed messages are NOT returned by GetIncompleteOlderThan
	cutoffTime := time.Now().Add(-1 * time.Hour)
	incompleteMessages, err := repo.GetIncompleteOlderThan(ctx, cutoffTime)
	require.NoError(t, err)

	for _, msg := range incompleteMessages {
		assert.NotEqual(t, message.ID, msg.ID, "Completed messages should not be in incomplete results")
	}
}

// TestMultipleStatusTypes validates that GetIncompleteOlderThan catches all incomplete states
func TestMultipleStatusTypes(t *testing.T) {
	pool := setupTestDB(t)

	ctx := context.Background()
	repo := NewMessageRepository(pool)
	convRepo := NewConversationRepository(pool)

	conversationID := "ac_test_multi"
	conversation := models.NewConversation(conversationID, "test-user", "Test Multi Conversation")
	err := convRepo.Create(ctx, conversation)
	require.NoError(t, err, "Failed to create test conversation")

	oldTime := time.Now().Add(-2 * time.Hour)

	// Create messages with different completion statuses
	testCases := []struct {
		id            string
		status        models.CompletionStatus
		shouldBeFound bool
	}{
		{"msg-pending", models.CompletionStatusPending, true},
		{"msg-streaming", models.CompletionStatusStreaming, true},
		{"msg-failed", models.CompletionStatusFailed, true},
		{"msg-completed", models.CompletionStatusCompleted, false},
	}

	for i, tc := range testCases {
		message := &models.Message{
			ID:               tc.id,
			ConversationID:   conversationID,
			SequenceNumber:   i + 1, // Different sequence numbers to avoid conflicts
			Role:             models.MessageRoleAssistant,
			Contents:         "Test content",
			CompletionStatus: tc.status,
			SyncStatus:       models.SyncStatusSynced,
			CreatedAt:        oldTime,
			UpdatedAt:        oldTime,
		}
		err := repo.Create(ctx, message)
		require.NoError(t, err)
	}

	// Get incomplete messages
	cutoffTime := time.Now().Add(-1 * time.Hour)
	incompleteMessages, err := repo.GetIncompleteOlderThan(ctx, cutoffTime)
	require.NoError(t, err)

	// Build a map of found message IDs
	foundIDs := make(map[string]bool)
	for _, msg := range incompleteMessages {
		foundIDs[msg.ID] = true
	}

	// Verify expectations
	for _, tc := range testCases {
		if tc.shouldBeFound {
			assert.True(t, foundIDs[tc.id], "Message %s with status %s should be found", tc.id, tc.status)
		} else {
			assert.False(t, foundIDs[tc.id], "Message %s with status %s should NOT be found", tc.id, tc.status)
		}
	}
}
