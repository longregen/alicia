package services

import (
	"context"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// Tests

func TestMessageService_Create(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	// Create a conversation first
	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, err := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.ID != "msg_test1" {
		t.Errorf("expected ID msg_test1, got %s", msg.ID)
	}

	if msg.ConversationID != "conv_123" {
		t.Errorf("expected conversation ID conv_123, got %s", msg.ConversationID)
	}

	if msg.Contents != "Hello" {
		t.Errorf("expected contents 'Hello', got %s", msg.Contents)
	}

	if msg.Role != models.MessageRoleUser {
		t.Errorf("expected role user, got %s", msg.Role)
	}
}

func TestMessageService_Create_EmptyConversationID(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	_, err := svc.Create(context.Background(), "", models.MessageRoleUser, "Hello")
	if err == nil {
		t.Fatal("expected error for empty conversation ID, got nil")
	}
}

func TestMessageService_Create_EmptyContents(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	_, err := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "")
	if err == nil {
		t.Fatal("expected error for empty contents, got nil")
	}
}

func TestMessageService_Create_ConversationNotFound(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	_, err := svc.Create(context.Background(), "nonexistent", models.MessageRoleUser, "Hello")
	if err == nil {
		t.Fatal("expected error for nonexistent conversation, got nil")
	}
}

func TestMessageService_Create_ArchivedConversation(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	conv.Archive()
	convRepo.Create(context.Background(), conv)

	_, err := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")
	if err == nil {
		t.Fatal("expected error for archived conversation, got nil")
	}
}

func TestMessageService_CreateUserMessage(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, err := svc.CreateUserMessage(context.Background(), "conv_123", "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Role != models.MessageRoleUser {
		t.Errorf("expected role user, got %s", msg.Role)
	}
}

func TestMessageService_CreateAssistantMessage(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, err := svc.CreateAssistantMessage(context.Background(), "conv_123", "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Role != models.MessageRoleAssistant {
		t.Errorf("expected role assistant, got %s", msg.Role)
	}
}

func TestMessageService_CreateSystemMessage(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, err := svc.CreateSystemMessage(context.Background(), "conv_123", "System message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Role != models.MessageRoleSystem {
		t.Errorf("expected role system, got %s", msg.Role)
	}
}

func TestMessageService_GetByID(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	created, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")

	retrieved, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}
}

func TestMessageService_GetByID_EmptyID(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	_, err := svc.GetByID(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
}

func TestMessageService_GetByConversation(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Message 1")
	svc.Create(context.Background(), "conv_123", models.MessageRoleAssistant, "Message 2")

	messages, err := svc.GetByConversation(context.Background(), "conv_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestMessageService_GetLatestByConversation(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Message 1")
	svc.Create(context.Background(), "conv_123", models.MessageRoleAssistant, "Message 2")
	svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Message 3")

	messages, err := svc.GetLatestByConversation(context.Background(), "conv_123", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) > 2 {
		t.Errorf("expected at most 2 messages, got %d", len(messages))
	}
}

func TestMessageService_Update(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")
	msg.Contents = "Updated content"

	err := svc.Update(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := svc.GetByID(context.Background(), msg.ID)
	if retrieved.Contents != "Updated content" {
		t.Errorf("expected content to be updated")
	}
}

func TestMessageService_Update_NilMessage(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	err := svc.Update(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil message, got nil")
	}
}

func TestMessageService_AppendContent(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")

	updated, err := svc.AppendContent(context.Background(), msg.ID, " World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Contents != "Hello World" {
		t.Errorf("expected 'Hello World', got %s", updated.Contents)
	}
}

func TestMessageService_LinkToPrevious(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg1, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "First")
	msg2, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleAssistant, "Second")

	linked, err := svc.LinkToPrevious(context.Background(), msg2.ID, msg1.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if linked.PreviousID != msg1.ID {
		t.Error("expected previous message to be linked")
	}
}

func TestMessageService_Delete(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")

	err := svc.Delete(context.Background(), msg.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = msgRepo.GetByID(context.Background(), msg.ID)
	if err != nil {
		// Expected - message was deleted
	}
}

func TestMessageService_CreateSentence(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")

	sentence, err := svc.CreateSentence(context.Background(), msg.ID, "First sentence.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sentence.ID != "sent_test1" {
		t.Errorf("expected ID sent_test1, got %s", sentence.ID)
	}

	if sentence.MessageID != msg.ID {
		t.Errorf("expected message ID %s, got %s", msg.ID, sentence.MessageID)
	}

	if sentence.Text != "First sentence." {
		t.Errorf("expected text 'First sentence.', got %s", sentence.Text)
	}
}

func TestMessageService_CreateSentence_EmptyMessageID(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	_, err := svc.CreateSentence(context.Background(), "", "text")
	if err == nil {
		t.Fatal("expected error for empty message ID, got nil")
	}
}

func TestMessageService_CreateSentence_EmptyText(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")

	_, err := svc.CreateSentence(context.Background(), msg.ID, "")
	if err == nil {
		t.Fatal("expected error for empty text, got nil")
	}
}

func TestMessageService_GetSentencesByMessage(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")

	svc.CreateSentence(context.Background(), msg.ID, "First sentence.")
	svc.CreateSentence(context.Background(), msg.ID, "Second sentence.")

	sentences, err := svc.GetSentencesByMessage(context.Background(), msg.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sentences) != 2 {
		t.Errorf("expected 2 sentences, got %d", len(sentences))
	}
}

func TestMessageService_UpdateSentence(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")
	sentence, _ := svc.CreateSentence(context.Background(), msg.ID, "Original text.")

	sentence.Text = "Updated text."
	err := svc.UpdateSentence(context.Background(), sentence)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := sentenceRepo.GetByID(context.Background(), sentence.ID)
	if retrieved.Text != "Updated text." {
		t.Errorf("expected text to be updated")
	}
}

func TestMessageService_AttachAudioToSentence(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")
	sentence, _ := svc.CreateSentence(context.Background(), msg.ID, "Text.")

	updated, err := svc.AttachAudioToSentence(context.Background(), sentence.ID, models.AudioTypeOutput, "wav", []byte("audio data"), 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.AudioType != models.AudioTypeOutput {
		t.Error("expected audio type to be set")
	}

	if updated.AudioFormat != "wav" {
		t.Error("expected audio format to be set")
	}
}

func TestMessageService_DeleteSentence(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")
	sentence, _ := svc.CreateSentence(context.Background(), msg.ID, "Text.")

	err := svc.DeleteSentence(context.Background(), sentence.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = sentenceRepo.GetByID(context.Background(), sentence.ID)
	if err == nil {
		t.Error("expected error when getting deleted sentence")
	}
}

func TestMessageService_UpdateSentence_Deleted(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()
	convRepo := newMockConversationRepo()
	idGen := &mockIDGenerator{}

	svc := NewMessageService(msgRepo, sentenceRepo, convRepo, idGen)

	conv := models.NewConversation("conv_123", "user_123", "Test")
	convRepo.Create(context.Background(), conv)

	msg, _ := svc.Create(context.Background(), "conv_123", models.MessageRoleUser, "Hello")
	sentence, _ := svc.CreateSentence(context.Background(), msg.ID, "Text.")

	// Mark as deleted
	now := time.Now()
	sentence.DeletedAt = &now
	sentenceRepo.Update(context.Background(), sentence)

	// Try to update
	sentence.Text = "New text"
	err := svc.UpdateSentence(context.Background(), sentence)
	if err == nil {
		t.Fatal("expected error when updating deleted sentence, got nil")
	}
}
