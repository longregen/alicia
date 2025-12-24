//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/domain/models"
)

func TestConversationFlow_CreateAndRetrieve(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()

	// Setup repositories and services
	conversationRepo := postgres.NewConversationRepository(db.Pool)
	messageRepo := postgres.NewMessageRepository(db.Pool)
	idGen := id.NewGenerator()

	// Create mock LiveKit service
	mockLiveKit := &mockLiveKitService{}
	conversationSvc := services.NewConversationService(conversationRepo, mockLiveKit, idGen)
	messageSvc := services.NewMessageService(messageRepo, idGen)

	// Test: Create a conversation
	conversation, err := conversationSvc.Create(ctx, "test-user", "Test Conversation")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	if conversation.ID == "" {
		t.Fatal("conversation ID should not be empty")
	}
	if conversation.Title != "Test Conversation" {
		t.Errorf("expected title 'Test Conversation', got '%s'", conversation.Title)
	}
	if conversation.Status != models.ConversationStatusActive {
		t.Errorf("expected status 'active', got '%s'", conversation.Status)
	}

	// Test: Retrieve the conversation
	retrieved, err := conversationSvc.GetByID(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("failed to retrieve conversation: %v", err)
	}

	if retrieved.ID != conversation.ID {
		t.Errorf("expected ID %s, got %s", conversation.ID, retrieved.ID)
	}
	if retrieved.Title != conversation.Title {
		t.Errorf("expected title %s, got %s", conversation.Title, retrieved.Title)
	}

	// Test: Add a user message
	userMsg, err := messageSvc.Create(ctx, &services.CreateMessageInput{
		ConversationID: conversation.ID,
		Role:           models.MessageRoleUser,
		Contents:       "Hello, Alicia!",
	})
	if err != nil {
		t.Fatalf("failed to create user message: %v", err)
	}

	if userMsg.ID == "" {
		t.Fatal("message ID should not be empty")
	}
	if userMsg.ConversationID != conversation.ID {
		t.Errorf("expected conversation ID %s, got %s", conversation.ID, userMsg.ConversationID)
	}
	if userMsg.Contents != "Hello, Alicia!" {
		t.Errorf("expected contents 'Hello, Alicia!', got '%s'", userMsg.Contents)
	}
	if userMsg.SequenceNumber != 1 {
		t.Errorf("expected sequence number 1, got %d", userMsg.SequenceNumber)
	}

	// Test: Add an assistant message
	assistantMsg, err := messageSvc.Create(ctx, &services.CreateMessageInput{
		ConversationID: conversation.ID,
		Role:           models.MessageRoleAssistant,
		Contents:       "Hello! How can I help you?",
	})
	if err != nil {
		t.Fatalf("failed to create assistant message: %v", err)
	}

	if assistantMsg.SequenceNumber != 2 {
		t.Errorf("expected sequence number 2, got %d", assistantMsg.SequenceNumber)
	}

	// Test: List messages in conversation
	messages, err := messageRepo.ListByConversation(ctx, conversation.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	// Verify message order (should be sorted by sequence number)
	if messages[0].SequenceNumber != 1 || messages[1].SequenceNumber != 2 {
		t.Error("messages not in correct sequence order")
	}
}

func TestConversationFlow_UpdateAndArchive(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()

	conversationRepo := postgres.NewConversationRepository(db.Pool)
	idGen := id.NewGenerator()
	mockLiveKit := &mockLiveKitService{}
	conversationSvc := services.NewConversationService(conversationRepo, mockLiveKit, idGen)

	// Create conversation
	conversation, err := conversationSvc.Create(ctx, "test-user", "Original Title")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	// Test: Update title
	updatedConv, err := conversationSvc.UpdateTitle(ctx, conversation.ID, "New Title")
	if err != nil {
		t.Fatalf("failed to update title: %v", err)
	}

	if updatedConv.Title != "New Title" {
		t.Errorf("expected title 'New Title', got '%s'", updatedConv.Title)
	}

	// Test: Archive conversation
	archivedConv, err := conversationSvc.Archive(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("failed to archive conversation: %v", err)
	}

	if archivedConv.Status != models.ConversationStatusArchived {
		t.Errorf("expected status 'archived', got '%s'", archivedConv.Status)
	}

	// Test: Unarchive conversation
	unarchivedConv, err := conversationSvc.Unarchive(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("failed to unarchive conversation: %v", err)
	}

	if unarchivedConv.Status != models.ConversationStatusActive {
		t.Errorf("expected status 'active', got '%s'", unarchivedConv.Status)
	}

	// Test: Delete conversation
	err = conversationSvc.Delete(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("failed to delete conversation: %v", err)
	}

	// Verify deletion (should return error when trying to retrieve)
	_, err = conversationSvc.GetByID(ctx, conversation.ID)
	if err == nil {
		t.Error("expected error when retrieving deleted conversation")
	}
}

func TestConversationFlow_ListActiveConversations(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()

	conversationRepo := postgres.NewConversationRepository(db.Pool)
	idGen := id.NewGenerator()
	mockLiveKit := &mockLiveKitService{}
	conversationSvc := services.NewConversationService(conversationRepo, mockLiveKit, idGen)

	// Create multiple conversations
	_, err := conversationSvc.Create(ctx, "test-user", "Active 1")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conv2, err := conversationSvc.Create(ctx, "test-user", "Active 2")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conv3, err := conversationSvc.Create(ctx, "test-user", "To Archive")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	// Archive one conversation
	_, err = conversationSvc.Archive(ctx, conv3.ID)
	if err != nil {
		t.Fatalf("failed to archive conversation: %v", err)
	}

	// Delete another conversation
	_, err = conversationSvc.Create(ctx, "test-user", "To Delete")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}
	err = conversationSvc.Delete(ctx, conv2.ID)
	if err != nil {
		t.Fatalf("failed to delete conversation: %v", err)
	}

	// List active conversations
	activeConversations, err := conversationSvc.ListActive(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to list active conversations: %v", err)
	}

	// Should only return active conversations (not archived or deleted)
	if len(activeConversations) != 2 {
		t.Errorf("expected 2 active conversations, got %d", len(activeConversations))
	}

	for _, conv := range activeConversations {
		if conv.Status != models.ConversationStatusActive {
			t.Errorf("expected active status, got %s", conv.Status)
		}
	}
}

// mockLiveKitService is a simple mock for testing
type mockLiveKitService struct{}

func (m *mockLiveKitService) CreateRoom(ctx context.Context, name string) (*mockRoom, error) {
	return &mockRoom{Name: name}, nil
}

func (m *mockLiveKitService) GetRoom(ctx context.Context, name string) (*mockRoom, error) {
	return &mockRoom{Name: name}, nil
}

func (m *mockLiveKitService) DeleteRoom(ctx context.Context, name string) error {
	return nil
}

func (m *mockLiveKitService) GenerateToken(ctx context.Context, roomName, participantID, participantName string) (*mockToken, error) {
	return &mockToken{Token: "mock-token"}, nil
}

type mockRoom struct {
	Name string
}

type mockToken struct {
	Token string
}
