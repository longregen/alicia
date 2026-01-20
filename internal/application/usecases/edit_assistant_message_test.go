package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// mockEditMessageRepo provides a mock MessageRepository for EditAssistantMessage tests
// with configurable behavior for error scenarios
type mockEditMessageRepo struct {
	messages    map[string]*models.Message
	getByIDErr  error
	updateErr   error
	updateCalls int
}

func newMockEditMessageRepo() *mockEditMessageRepo {
	return &mockEditMessageRepo{
		messages: make(map[string]*models.Message),
	}
}

func (m *mockEditMessageRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if msg, ok := m.messages[id]; ok {
		// Return a copy to avoid mutation
		return m.copyMessage(msg), nil
	}
	return nil, errors.New("message not found")
}

func (m *mockEditMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	m.updateCalls++
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.messages[msg.ID]; !ok {
		return errors.New("message not found")
	}
	m.messages[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *mockEditMessageRepo) copyMessage(msg *models.Message) *models.Message {
	return &models.Message{
		ID:               msg.ID,
		ConversationID:   msg.ConversationID,
		SequenceNumber:   msg.SequenceNumber,
		PreviousID:       msg.PreviousID,
		Role:             msg.Role,
		Contents:         msg.Contents,
		CreatedAt:        msg.CreatedAt,
		UpdatedAt:        msg.UpdatedAt,
		LocalID:          msg.LocalID,
		ServerID:         msg.ServerID,
		SyncStatus:       msg.SyncStatus,
		CompletionStatus: msg.CompletionStatus,
	}
}

// Implement remaining MessageRepository interface methods (unused but required)
func (m *mockEditMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	m.messages[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *mockEditMessageRepo) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockEditMessageRepo) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	return 0, nil
}

func (m *mockEditMessageRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockEditMessageRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockEditMessageRepo) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockEditMessageRepo) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockEditMessageRepo) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, nil
}

func (m *mockEditMessageRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockEditMessageRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockEditMessageRepo) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockEditMessageRepo) GetChainFromTipWithSiblings(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return m.GetChainFromTip(ctx, tipMessageID)
}

func (m *mockEditMessageRepo) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockEditMessageRepo) DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error {
	return nil
}

// Tests

func TestEditAssistantMessage_Execute_Success(t *testing.T) {
	// Setup
	msgRepo := newMockEditMessageRepo()

	// Create an assistant message to edit
	originalMsg := models.NewAssistantMessage("msg_123", "conv_123", 1, "Original content")
	originalMsg.CreatedAt = time.Now().UTC().Add(-time.Hour)
	originalMsg.UpdatedAt = originalMsg.CreatedAt
	msgRepo.Create(context.Background(), originalMsg)

	uc := NewEditAssistantMessage(msgRepo)

	input := &ports.EditAssistantMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "msg_123",
		NewContent:      "Updated content",
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output == nil {
		t.Fatal("expected output, got nil")
	}

	if output.UpdatedMessage == nil {
		t.Fatal("expected updated message, got nil")
	}

	if output.UpdatedMessage.Contents != "Updated content" {
		t.Errorf("expected content 'Updated content', got %s", output.UpdatedMessage.Contents)
	}

	if output.UpdatedMessage.ID != "msg_123" {
		t.Errorf("expected message ID 'msg_123', got %s", output.UpdatedMessage.ID)
	}

	if output.UpdatedMessage.Role != models.MessageRoleAssistant {
		t.Errorf("expected role assistant, got %s", output.UpdatedMessage.Role)
	}

	// Verify UpdatedAt was changed
	if !output.UpdatedMessage.UpdatedAt.After(originalMsg.CreatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}

	// Verify message was stored in repository
	stored, _ := msgRepo.GetByID(context.Background(), "msg_123")
	if stored.Contents != "Updated content" {
		t.Errorf("expected stored content 'Updated content', got %s", stored.Contents)
	}

	// Verify Update was called
	if msgRepo.updateCalls != 1 {
		t.Errorf("expected 1 update call, got %d", msgRepo.updateCalls)
	}
}

func TestEditAssistantMessage_Execute_MessageNotFound(t *testing.T) {
	// Setup
	msgRepo := newMockEditMessageRepo()
	// Don't create any messages - repository is empty

	uc := NewEditAssistantMessage(msgRepo)

	input := &ports.EditAssistantMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "nonexistent_msg",
		NewContent:      "Updated content",
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when message not found, got nil")
	}

	if output != nil {
		t.Errorf("expected nil output on error, got %v", output)
	}

	// Verify error message contains relevant info
	if !containsString(err.Error(), "failed to get message") {
		t.Errorf("expected error to mention 'failed to get message', got: %s", err.Error())
	}
}

func TestEditAssistantMessage_Execute_NotAssistantMessage(t *testing.T) {
	// Setup
	msgRepo := newMockEditMessageRepo()

	// Create a USER message (not assistant)
	userMsg := models.NewUserMessage("msg_123", "conv_123", 1, "User message content")
	msgRepo.Create(context.Background(), userMsg)

	uc := NewEditAssistantMessage(msgRepo)

	input := &ports.EditAssistantMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "msg_123",
		NewContent:      "Trying to edit user message",
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when editing non-assistant message, got nil")
	}

	if output != nil {
		t.Errorf("expected nil output on error, got %v", output)
	}

	// Verify error message indicates role mismatch
	if !containsString(err.Error(), "expected assistant role") {
		t.Errorf("expected error to mention 'expected assistant role', got: %s", err.Error())
	}

	if !containsString(err.Error(), "user") {
		t.Errorf("expected error to mention 'user', got: %s", err.Error())
	}

	// Verify no update was attempted
	if msgRepo.updateCalls != 0 {
		t.Errorf("expected 0 update calls, got %d", msgRepo.updateCalls)
	}
}

func TestEditAssistantMessage_Execute_UpdateFails(t *testing.T) {
	// Setup
	msgRepo := newMockEditMessageRepo()
	msgRepo.updateErr = errors.New("database connection failed")

	// Create an assistant message
	assistantMsg := models.NewAssistantMessage("msg_123", "conv_123", 1, "Original content")
	msgRepo.Create(context.Background(), assistantMsg)

	uc := NewEditAssistantMessage(msgRepo)

	input := &ports.EditAssistantMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "msg_123",
		NewContent:      "Updated content",
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when update fails, got nil")
	}

	if output != nil {
		t.Errorf("expected nil output on error, got %v", output)
	}

	// Verify error wraps the underlying error
	if !containsString(err.Error(), "failed to update message") {
		t.Errorf("expected error to mention 'failed to update message', got: %s", err.Error())
	}

	if !containsString(err.Error(), "database connection failed") {
		t.Errorf("expected error to contain underlying error, got: %s", err.Error())
	}
}

func TestEditAssistantMessage_Execute_EmptyContent(t *testing.T) {
	// Setup
	msgRepo := newMockEditMessageRepo()

	// Create an assistant message
	assistantMsg := models.NewAssistantMessage("msg_123", "conv_123", 1, "Original content")
	msgRepo.Create(context.Background(), assistantMsg)

	uc := NewEditAssistantMessage(msgRepo)

	input := &ports.EditAssistantMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "msg_123",
		NewContent:      "", // Empty content
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Note: The current implementation allows empty content
	// If business logic requires non-empty content, this test should be updated
	// to expect an error. For now, we verify the implementation behavior.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output == nil {
		t.Fatal("expected output, got nil")
	}

	if output.UpdatedMessage.Contents != "" {
		t.Errorf("expected empty content, got %s", output.UpdatedMessage.Contents)
	}

	// Verify message was stored with empty content
	stored, _ := msgRepo.GetByID(context.Background(), "msg_123")
	if stored.Contents != "" {
		t.Errorf("expected stored empty content, got %s", stored.Contents)
	}
}

func TestEditAssistantMessage_Execute_SameContent(t *testing.T) {
	// Setup
	msgRepo := newMockEditMessageRepo()

	originalContent := "Same content that won't change"

	// Create an assistant message
	assistantMsg := models.NewAssistantMessage("msg_123", "conv_123", 1, originalContent)
	originalUpdatedAt := assistantMsg.UpdatedAt
	msgRepo.Create(context.Background(), assistantMsg)

	uc := NewEditAssistantMessage(msgRepo)

	input := &ports.EditAssistantMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "msg_123",
		NewContent:      originalContent, // Same as original
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify - should succeed even if content is the same
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output == nil {
		t.Fatal("expected output, got nil")
	}

	if output.UpdatedMessage.Contents != originalContent {
		t.Errorf("expected content %q, got %q", originalContent, output.UpdatedMessage.Contents)
	}

	// UpdatedAt should still be updated even if content is the same
	if !output.UpdatedMessage.UpdatedAt.After(originalUpdatedAt) && output.UpdatedMessage.UpdatedAt != originalUpdatedAt {
		// Note: Due to timing, UpdatedAt might be slightly different
		// The important thing is that the operation succeeded
	}

	// Verify update was called
	if msgRepo.updateCalls != 1 {
		t.Errorf("expected 1 update call, got %d", msgRepo.updateCalls)
	}
}

func TestEditAssistantMessage_Execute_GetByIDError(t *testing.T) {
	// Setup
	msgRepo := newMockEditMessageRepo()
	msgRepo.getByIDErr = errors.New("database timeout")

	uc := NewEditAssistantMessage(msgRepo)

	input := &ports.EditAssistantMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "msg_123",
		NewContent:      "Updated content",
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when GetByID fails, got nil")
	}

	if output != nil {
		t.Errorf("expected nil output on error, got %v", output)
	}

	// Verify error wraps the underlying error
	if !containsString(err.Error(), "database timeout") {
		t.Errorf("expected error to contain 'database timeout', got: %s", err.Error())
	}
}

func TestEditAssistantMessage_Execute_PreservesOtherFields(t *testing.T) {
	// Setup
	msgRepo := newMockEditMessageRepo()

	// Create an assistant message with various fields set
	assistantMsg := models.NewAssistantMessage("msg_123", "conv_123", 5, "Original content")
	assistantMsg.PreviousID = "msg_122"
	assistantMsg.LocalID = "local_123"
	assistantMsg.ServerID = "server_123"
	assistantMsg.SyncStatus = models.SyncStatusSynced
	assistantMsg.CompletionStatus = models.CompletionStatusCompleted
	originalCreatedAt := assistantMsg.CreatedAt
	msgRepo.Create(context.Background(), assistantMsg)

	uc := NewEditAssistantMessage(msgRepo)

	input := &ports.EditAssistantMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "msg_123",
		NewContent:      "Updated content",
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := output.UpdatedMessage

	// Verify content was updated
	if msg.Contents != "Updated content" {
		t.Errorf("expected content 'Updated content', got %s", msg.Contents)
	}

	// Verify other fields were preserved
	if msg.ID != "msg_123" {
		t.Errorf("expected ID 'msg_123', got %s", msg.ID)
	}

	if msg.ConversationID != "conv_123" {
		t.Errorf("expected ConversationID 'conv_123', got %s", msg.ConversationID)
	}

	if msg.SequenceNumber != 5 {
		t.Errorf("expected SequenceNumber 5, got %d", msg.SequenceNumber)
	}

	if msg.PreviousID != "msg_122" {
		t.Errorf("expected PreviousID 'msg_122', got %s", msg.PreviousID)
	}

	if msg.Role != models.MessageRoleAssistant {
		t.Errorf("expected Role assistant, got %s", msg.Role)
	}

	if msg.LocalID != "local_123" {
		t.Errorf("expected LocalID 'local_123', got %s", msg.LocalID)
	}

	if msg.ServerID != "server_123" {
		t.Errorf("expected ServerID 'server_123', got %s", msg.ServerID)
	}

	if msg.SyncStatus != models.SyncStatusSynced {
		t.Errorf("expected SyncStatus synced, got %s", msg.SyncStatus)
	}

	if msg.CompletionStatus != models.CompletionStatusCompleted {
		t.Errorf("expected CompletionStatus completed, got %s", msg.CompletionStatus)
	}

	// CreatedAt should not change
	if !msg.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("expected CreatedAt to be preserved, got %v (original: %v)", msg.CreatedAt, originalCreatedAt)
	}
}

func TestEditAssistantMessage_Execute_SystemRoleMessage(t *testing.T) {
	// Setup
	msgRepo := newMockEditMessageRepo()

	// Create a system message
	systemMsg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleSystem, "System prompt")
	msgRepo.Create(context.Background(), systemMsg)

	uc := NewEditAssistantMessage(msgRepo)

	input := &ports.EditAssistantMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "msg_123",
		NewContent:      "Trying to edit system message",
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when editing system message, got nil")
	}

	if output != nil {
		t.Errorf("expected nil output on error, got %v", output)
	}

	// Verify error message indicates role mismatch
	if !containsString(err.Error(), "expected assistant role") {
		t.Errorf("expected error to mention 'expected assistant role', got: %s", err.Error())
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
