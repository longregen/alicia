//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/domain/models"
)

func TestSyncFlow_StanzaIDTracking(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()

	conversationRepo := postgres.NewConversationRepository(db.Pool)
	idGen := id.NewGenerator()
	mockLiveKit := &mockLiveKitService{}

	conversationSvc := services.NewConversationService(conversationRepo, mockLiveKit, idGen)

	// Create a conversation
	conversation, err := conversationSvc.Create(ctx, "test-user", "Sync Test")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	// Initial state should have default stanza IDs
	if conversation.LastClientStanzaID != 0 {
		t.Errorf("expected initial client stanza ID 0, got %d", conversation.LastClientStanzaID)
	}
	if conversation.LastServerStanzaID != -1 {
		t.Errorf("expected initial server stanza ID -1, got %d", conversation.LastServerStanzaID)
	}

	// Test: Update client stanza ID
	conversation.UpdateLastClientStanzaID(5)
	err = conversationRepo.Update(ctx, conversation)
	if err != nil {
		t.Fatalf("failed to update conversation: %v", err)
	}

	retrieved, err := conversationRepo.GetByID(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("failed to retrieve conversation: %v", err)
	}

	if retrieved.LastClientStanzaID != 5 {
		t.Errorf("expected client stanza ID 5, got %d", retrieved.LastClientStanzaID)
	}

	// Test: Update server stanza ID (negative)
	conversation.UpdateLastServerStanzaID(-3)
	err = conversationRepo.Update(ctx, conversation)
	if err != nil {
		t.Fatalf("failed to update conversation: %v", err)
	}

	retrieved, err = conversationRepo.GetByID(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("failed to retrieve conversation: %v", err)
	}

	if retrieved.LastServerStanzaID != -3 {
		t.Errorf("expected server stanza ID -3, got %d", retrieved.LastServerStanzaID)
	}

	// Test: Ensure stanza IDs only increase
	conversation.UpdateLastClientStanzaID(3) // Lower than current (5)
	err = conversationRepo.Update(ctx, conversation)
	if err != nil {
		t.Fatalf("failed to update conversation: %v", err)
	}

	retrieved, err = conversationRepo.GetByID(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("failed to retrieve conversation: %v", err)
	}

	// Should still be 5, not 3
	if retrieved.LastClientStanzaID != 5 {
		t.Errorf("client stanza ID should not decrease, expected 5, got %d", retrieved.LastClientStanzaID)
	}
}

func TestSyncFlow_MessageSyncStatus(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	messageRepo := postgres.NewMessageRepository(db.Pool)

	// Create conversation
	conversation := fixtures.CreateConversation(ctx, t, "conv1", "Sync Test")

	// Test: Create a local (unsynced) message
	localMessage := models.NewLocalMessage("local123", conversation.ID, 1, models.MessageRoleUser, "Test message")

	err := messageRepo.Create(ctx, localMessage)
	if err != nil {
		t.Fatalf("failed to create local message: %v", err)
	}

	if localMessage.SyncStatus != models.SyncStatusPending {
		t.Errorf("expected sync status 'pending', got '%s'", localMessage.SyncStatus)
	}
	if localMessage.LocalID != "local123" {
		t.Errorf("expected local ID 'local123', got '%s'", localMessage.LocalID)
	}

	// Test: Mark message as synced with server ID
	localMessage.MarkAsSynced("msg_server456")
	err = messageRepo.Update(ctx, localMessage)
	if err != nil {
		t.Fatalf("failed to update message: %v", err)
	}

	synced, err := messageRepo.GetByID(ctx, localMessage.ID)
	if err != nil {
		t.Fatalf("failed to retrieve synced message: %v", err)
	}

	if synced.SyncStatus != models.SyncStatusSynced {
		t.Errorf("expected sync status 'synced', got '%s'", synced.SyncStatus)
	}
	if synced.ServerID != "msg_server456" {
		t.Errorf("expected server ID 'msg_server456', got '%s'", synced.ServerID)
	}
	if synced.SyncedAt == nil {
		t.Error("synced_at should be set")
	}

	// Test: Mark message as conflict
	localMessage.MarkAsConflict()
	err = messageRepo.Update(ctx, localMessage)
	if err != nil {
		t.Fatalf("failed to update message to conflict: %v", err)
	}

	conflicted, err := messageRepo.GetByID(ctx, localMessage.ID)
	if err != nil {
		t.Fatalf("failed to retrieve conflicted message: %v", err)
	}

	if conflicted.SyncStatus != models.SyncStatusConflict {
		t.Errorf("expected sync status 'conflict', got '%s'", conflicted.SyncStatus)
	}
}

func TestSyncFlow_PendingMessagesRetrieval(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	messageRepo := postgres.NewMessageRepository(db.Pool)

	// Create conversation
	conversation := fixtures.CreateConversation(ctx, t, "conv1", "Sync Test")

	// Create synced message
	syncedMsg := models.NewMessage("msg1", conversation.ID, 1, models.MessageRoleUser, "Synced message")
	err := messageRepo.Create(ctx, syncedMsg)
	if err != nil {
		t.Fatalf("failed to create synced message: %v", err)
	}

	// Create pending messages
	pendingMsg1 := models.NewLocalMessage("local1", conversation.ID, 2, models.MessageRoleUser, "Pending 1")
	err = messageRepo.Create(ctx, pendingMsg1)
	if err != nil {
		t.Fatalf("failed to create pending message 1: %v", err)
	}

	pendingMsg2 := models.NewLocalMessage("local2", conversation.ID, 3, models.MessageRoleUser, "Pending 2")
	err = messageRepo.Create(ctx, pendingMsg2)
	if err != nil {
		t.Fatalf("failed to create pending message 2: %v", err)
	}

	// Test: Get pending messages for conversation
	pendingMessages, err := messageRepo.ListPendingSync(ctx, conversation.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list pending messages: %v", err)
	}

	if len(pendingMessages) != 2 {
		t.Errorf("expected 2 pending messages, got %d", len(pendingMessages))
	}

	// Verify all are pending
	for _, msg := range pendingMessages {
		if msg.SyncStatus != models.SyncStatusPending {
			t.Errorf("expected pending status, got '%s'", msg.SyncStatus)
		}
	}
}

func TestSyncFlow_MessageSequenceIntegrity(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()

	conversationRepo := postgres.NewConversationRepository(db.Pool)
	messageRepo := postgres.NewMessageRepository(db.Pool)
	idGen := id.NewGenerator()
	mockLiveKit := &mockLiveKitService{}

	conversationSvc := services.NewConversationService(conversationRepo, mockLiveKit, idGen)
	messageSvc := services.NewMessageService(messageRepo, idGen)

	// Create conversation
	conversation, err := conversationSvc.Create(ctx, "test-user", "Sequence Test")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	// Create messages in order
	msg1, err := messageSvc.Create(ctx, &services.CreateMessageInput{
		ConversationID: conversation.ID,
		Role:           models.MessageRoleUser,
		Contents:       "Message 1",
	})
	if err != nil {
		t.Fatalf("failed to create message 1: %v", err)
	}

	msg2, err := messageSvc.Create(ctx, &services.CreateMessageInput{
		ConversationID: conversation.ID,
		Role:           models.MessageRoleAssistant,
		Contents:       "Message 2",
	})
	if err != nil {
		t.Fatalf("failed to create message 2: %v", err)
	}

	msg3, err := messageSvc.Create(ctx, &services.CreateMessageInput{
		ConversationID: conversation.ID,
		Role:           models.MessageRoleUser,
		Contents:       "Message 3",
	})
	if err != nil {
		t.Fatalf("failed to create message 3: %v", err)
	}

	// Verify sequence numbers are sequential
	if msg1.SequenceNumber != 1 {
		t.Errorf("expected sequence 1, got %d", msg1.SequenceNumber)
	}
	if msg2.SequenceNumber != 2 {
		t.Errorf("expected sequence 2, got %d", msg2.SequenceNumber)
	}
	if msg3.SequenceNumber != 3 {
		t.Errorf("expected sequence 3, got %d", msg3.SequenceNumber)
	}

	// Test: List messages in order
	messages, err := messageRepo.ListByConversation(ctx, conversation.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}

	// Verify ordering
	for i, msg := range messages {
		expectedSeq := i + 1
		if msg.SequenceNumber != expectedSeq {
			t.Errorf("message at index %d has sequence %d, expected %d", i, msg.SequenceNumber, expectedSeq)
		}
	}

	// Test: Get messages after a specific sequence number (for reconnection)
	messagesAfter1, err := messageRepo.ListAfterSequence(ctx, conversation.ID, 1, 100)
	if err != nil {
		t.Fatalf("failed to list messages after sequence 1: %v", err)
	}

	if len(messagesAfter1) != 2 {
		t.Errorf("expected 2 messages after sequence 1, got %d", len(messagesAfter1))
	}

	if messagesAfter1[0].SequenceNumber != 2 || messagesAfter1[1].SequenceNumber != 3 {
		t.Error("messages after sequence 1 should be messages 2 and 3")
	}
}

func TestSyncFlow_ReconnectionScenario(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	conversationRepo := postgres.NewConversationRepository(db.Pool)
	messageRepo := postgres.NewMessageRepository(db.Pool)

	// Simulate a conversation with messages
	conversation := fixtures.CreateConversation(ctx, t, "conv1", "Reconnection Test")

	// Create messages that were sent before disconnect
	fixtures.CreateMessage(ctx, t, "msg1", conversation.ID, models.MessageRoleUser, "Before disconnect 1", 1)
	fixtures.CreateMessage(ctx, t, "msg2", conversation.ID, models.MessageRoleAssistant, "Before disconnect 2", 2)
	fixtures.CreateMessage(ctx, t, "msg3", conversation.ID, models.MessageRoleUser, "Before disconnect 3", 3)

	// Client tracks last received stanza IDs before disconnect
	// Client stanza: 2 (client sent messages with stanzaId 1, 2)
	// Server stanza: -2 (server sent message with stanzaId -1, -2)
	conversation.UpdateLastClientStanzaID(2)
	conversation.UpdateLastServerStanzaID(-2)
	err := conversationRepo.Update(ctx, conversation)
	if err != nil {
		t.Fatalf("failed to update conversation: %v", err)
	}

	// Simulate reconnection: client needs messages after last known stanzas
	// Messages after client's last stanza
	messagesAfter2, err := messageRepo.ListAfterSequence(ctx, conversation.ID, 2, 100)
	if err != nil {
		t.Fatalf("failed to list messages after sequence 2: %v", err)
	}

	// Should get message 3 (the one after client's last acknowledged message)
	if len(messagesAfter2) != 1 {
		t.Errorf("expected 1 message after sequence 2, got %d", len(messagesAfter2))
	}
	if len(messagesAfter2) > 0 && messagesAfter2[0].SequenceNumber != 3 {
		t.Errorf("expected message 3, got sequence %d", messagesAfter2[0].SequenceNumber)
	}

	// Create new messages after reconnection
	fixtures.CreateMessage(ctx, t, "msg4", conversation.ID, models.MessageRoleAssistant, "After reconnect 1", 4)
	fixtures.CreateMessage(ctx, t, "msg5", conversation.ID, models.MessageRoleUser, "After reconnect 2", 5)

	// Get all messages to verify continuity
	allMessages, err := messageRepo.ListByConversation(ctx, conversation.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list all messages: %v", err)
	}

	if len(allMessages) != 5 {
		t.Errorf("expected 5 messages total, got %d", len(allMessages))
	}

	// Verify no gaps in sequence
	for i, msg := range allMessages {
		if msg.SequenceNumber != i+1 {
			t.Errorf("sequence gap detected: message at index %d has sequence %d", i, msg.SequenceNumber)
		}
	}
}
