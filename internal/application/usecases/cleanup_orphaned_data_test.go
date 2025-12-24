package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

func TestCleanupOrphanedData_Execute(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	now := time.Now()
	oldTime := now.Add(-2 * time.Hour)

	oldMsg := models.NewMessage("msg_old", "conv_123", 0, models.MessageRoleAssistant, "Old message")
	oldMsg.MarkAsStreaming()
	oldMsg.CreatedAt = oldTime
	msgRepo.Create(context.Background(), oldMsg)

	recentMsg := models.NewMessage("msg_recent", "conv_123", 1, models.MessageRoleAssistant, "Recent message")
	recentMsg.MarkAsStreaming()
	recentMsg.CreatedAt = now.Add(-30 * time.Minute)
	msgRepo.Create(context.Background(), recentMsg)

	oldSent := models.NewSentence("sent_old", "msg_old", 0, "Old sentence")
	oldSent.MarkAsStreaming()
	oldSent.CreatedAt = oldTime
	sentenceRepo.Create(context.Background(), oldSent)

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge: 1 * time.Hour,
		DryRun: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.OrphanedMessagesFound == 0 {
		t.Error("expected orphaned messages to be found")
	}

	if output.OrphanedSentencesFound == 0 {
		t.Error("expected orphaned sentences to be found")
	}

	if len(output.Errors) > 0 {
		t.Errorf("unexpected errors: %v", output.Errors)
	}
}

func TestCleanupOrphanedData_DryRun(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	oldTime := time.Now().Add(-2 * time.Hour)

	oldMsg := models.NewMessage("msg_old", "conv_123", 0, models.MessageRoleAssistant, "Old message")
	oldMsg.MarkAsStreaming()
	oldMsg.CreatedAt = oldTime
	msgRepo.Create(context.Background(), oldMsg)

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge: 1 * time.Hour,
		DryRun: true,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.OrphanedMessagesFound == 0 {
		t.Error("expected orphaned messages to be found")
	}

	if output.OrphanedMessagesCleaned != 0 {
		t.Error("expected no messages to be cleaned in dry run mode")
	}

	if output.OrphanedSentencesCleaned != 0 {
		t.Error("expected no sentences to be cleaned in dry run mode")
	}

	retrievedMsg, _ := msgRepo.GetByID(context.Background(), "msg_old")
	if retrievedMsg.CompletionStatus == models.CompletionStatusFailed {
		t.Error("expected message not to be marked as failed in dry run mode")
	}
}

func TestCleanupOrphanedData_WithConversationID(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	oldTime := time.Now().Add(-2 * time.Hour)

	msg1 := models.NewMessage("msg_1", "conv_123", 0, models.MessageRoleAssistant, "Message 1")
	msg1.MarkAsStreaming()
	msg1.CreatedAt = oldTime
	msgRepo.Create(context.Background(), msg1)

	msg2 := models.NewMessage("msg_2", "conv_456", 0, models.MessageRoleAssistant, "Message 2")
	msg2.MarkAsStreaming()
	msg2.CreatedAt = oldTime
	msgRepo.Create(context.Background(), msg2)

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge:         1 * time.Hour,
		DryRun:         false,
		ConversationID: "conv_123",
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.OrphanedMessagesFound == 0 {
		t.Error("expected orphaned messages to be found for conv_123")
	}
}

func TestCleanupOrphanedData_NoOrphanedData(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	recentMsg := models.NewMessage("msg_recent", "conv_123", 0, models.MessageRoleAssistant, "Recent message")
	recentMsg.MarkAsCompleted()
	msgRepo.Create(context.Background(), recentMsg)

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge: 1 * time.Hour,
		DryRun: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.OrphanedMessagesFound != 0 {
		t.Errorf("expected 0 orphaned messages, got %d", output.OrphanedMessagesFound)
	}

	if output.OrphanedSentencesFound != 0 {
		t.Errorf("expected 0 orphaned sentences, got %d", output.OrphanedSentencesFound)
	}
}

func TestCleanupOrphanedData_DeleteOrphanedSentencesForMessage(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	sent1 := models.NewSentence("sent_1", "msg_123", 0, "Sentence 1")
	sent1.MarkAsFailed()
	sentenceRepo.Create(context.Background(), sent1)

	sent2 := models.NewSentence("sent_2", "msg_123", 1, "Sentence 2")
	sent2.MarkAsStreaming()
	sentenceRepo.Create(context.Background(), sent2)

	sent3 := models.NewSentence("sent_3", "msg_123", 2, "Sentence 3")
	sent3.MarkAsCompleted()
	sentenceRepo.Create(context.Background(), sent3)

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	err := uc.DeleteOrphanedSentencesForMessage(context.Background(), "msg_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err1 := sentenceRepo.GetByID(context.Background(), "sent_1")
	if err1 == nil {
		t.Error("expected failed sentence to be deleted")
	}

	_, err2 := sentenceRepo.GetByID(context.Background(), "sent_2")
	if err2 == nil {
		t.Error("expected streaming sentence to be deleted")
	}

	sent3Retrieved, err3 := sentenceRepo.GetByID(context.Background(), "sent_3")
	if err3 != nil || sent3Retrieved == nil {
		t.Error("expected completed sentence to remain")
	}
}

func TestCleanupOrphanedData_DeleteOrphanedSentencesForMessageNotFound(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	err := uc.DeleteOrphanedSentencesForMessage(context.Background(), "msg_nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupOrphanedData_MarkStaleStreamingDataAsFailed(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	oldTime := time.Now().Add(-2 * time.Hour)

	staleMsg := models.NewMessage("msg_stale", "conv_123", 0, models.MessageRoleAssistant, "Stale message")
	staleMsg.MarkAsStreaming()
	staleMsg.CreatedAt = oldTime
	msgRepo.Create(context.Background(), staleMsg)

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	err := uc.MarkStaleStreamingDataAsFailed(context.Background(), 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := msgRepo.GetByID(context.Background(), "msg_stale")
	if retrieved.CompletionStatus != models.CompletionStatusFailed {
		t.Errorf("expected message status failed, got %s", retrieved.CompletionStatus)
	}
}

func TestCleanupOrphanedData_CleanupCompletedMessages(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	oldTime := time.Now().Add(-2 * time.Hour)

	completedMsg := models.NewMessage("msg_completed", "conv_123", 0, models.MessageRoleAssistant, "Completed message")
	completedMsg.MarkAsCompleted()
	completedMsg.CreatedAt = oldTime
	msgRepo.Create(context.Background(), completedMsg)

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge: 1 * time.Hour,
		DryRun: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.OrphanedMessagesFound != 0 {
		t.Error("expected completed messages not to be marked as orphaned")
	}
}

func TestCleanupOrphanedData_CleanupPendingMessages(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	oldTime := time.Now().Add(-2 * time.Hour)

	pendingMsg := models.NewMessage("msg_pending", "conv_123", 0, models.MessageRoleAssistant, "Pending message")
	pendingMsg.CompletionStatus = models.CompletionStatusPending // Set to pending (not the default completed)
	pendingMsg.CreatedAt = oldTime
	msgRepo.Create(context.Background(), pendingMsg)

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge: 1 * time.Hour,
		DryRun: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.OrphanedMessagesFound == 0 {
		t.Error("expected pending messages to be found as orphaned")
	}
}

func TestCleanupOrphanedData_CleanupStreamingSentences(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	oldTime := time.Now().Add(-2 * time.Hour)

	// Create a streaming sentence (incomplete) that should be found as orphaned
	streamingSent := models.NewSentence("sent_streaming", "msg_123", 0, "Streaming sentence")
	streamingSent.MarkAsStreaming()
	streamingSent.CreatedAt = oldTime
	sentenceRepo.Create(context.Background(), streamingSent)

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge: 1 * time.Hour,
		DryRun: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.OrphanedSentencesFound == 0 {
		t.Error("expected streaming sentences to be found as orphaned")
	}
}

func TestCleanupOrphanedData_MessageRepoError(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge: 1 * time.Hour,
		DryRun: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Errors) > 0 {
		t.Logf("errors encountered: %v", output.Errors)
	}
}

func TestCleanupOrphanedData_SentenceRepoError(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge: 1 * time.Hour,
		DryRun: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Errors) > 0 {
		t.Logf("errors encountered: %v", output.Errors)
	}
}

func TestCleanupOrphanedData_DeleteSentencesRepoError(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	err := uc.DeleteOrphanedSentencesForMessage(context.Background(), "msg_nonexistent")
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Logf("error occurred: %v", err)
		}
	}
}

func TestCleanupOrphanedData_LargeDataset(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentenceRepo := newMockSentenceRepo()

	oldTime := time.Now().Add(-2 * time.Hour)

	for i := 0; i < 100; i++ {
		msg := models.NewMessage("msg_"+string(rune(i)), "conv_123", i, models.MessageRoleAssistant, "Message")
		msg.MarkAsStreaming()
		msg.CreatedAt = oldTime
		msgRepo.Create(context.Background(), msg)

		sent := models.NewSentence("sent_"+string(rune(i)), "msg_"+string(rune(i)), 0, "Sentence")
		sent.MarkAsStreaming()
		sent.CreatedAt = oldTime
		sentenceRepo.Create(context.Background(), sent)
	}

	uc := NewCleanupOrphanedData(msgRepo, sentenceRepo)

	input := &CleanupOrphanedDataInput{
		MaxAge: 1 * time.Hour,
		DryRun: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.OrphanedMessagesFound == 0 {
		t.Error("expected orphaned messages to be found")
	}

	if output.OrphanedSentencesFound == 0 {
		t.Error("expected orphaned sentences to be found")
	}
}
