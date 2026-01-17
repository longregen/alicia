package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Test-specific mocks for EditUserMessage

// editMockMessageRepo provides extended mock implementation for edit tests
type editMockMessageRepo struct {
	*mockMessageRepo
	getByIDError             error
	updateError              error
	deleteAfterSequenceError error
	getAfterSequenceError    error
	getAfterSequenceResult   []*models.Message
}

func newEditMockMessageRepo() *editMockMessageRepo {
	return &editMockMessageRepo{
		mockMessageRepo: newMockMessageRepo(),
	}
}

func (m *editMockMessageRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	return m.mockMessageRepo.GetByID(ctx, id)
}

func (m *editMockMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	if m.updateError != nil {
		return m.updateError
	}
	return m.mockMessageRepo.Update(ctx, msg)
}

func (m *editMockMessageRepo) DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error {
	if m.deleteAfterSequenceError != nil {
		return m.deleteAfterSequenceError
	}
	return m.mockMessageRepo.DeleteAfterSequence(ctx, conversationID, afterSequence)
}

func (m *editMockMessageRepo) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	if m.getAfterSequenceError != nil {
		return nil, m.getAfterSequenceError
	}
	if m.getAfterSequenceResult != nil {
		return m.getAfterSequenceResult, nil
	}
	return m.mockMessageRepo.GetAfterSequence(ctx, conversationID, afterSequence)
}

// editMockConversationRepo provides extended mock implementation for edit tests
type editMockConversationRepo struct {
	*mockConversationRepo
	updateTipError error
}

func newEditMockConversationRepo() *editMockConversationRepo {
	return &editMockConversationRepo{
		mockConversationRepo: newMockConversationRepo(),
	}
}

func (m *editMockConversationRepo) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	if m.updateTipError != nil {
		return m.updateTipError
	}
	return m.mockConversationRepo.UpdateTip(ctx, conversationID, messageID)
}

// Helper function to create a test EditUserMessage use case
func createEditUserMessageUC(
	msgRepo ports.MessageRepository,
	convRepo ports.ConversationRepository,
	memService ports.MemoryService,
	generateUC *GenerateResponse,
	idGen ports.IDGenerator,
	txManager ports.TransactionManager,
) *EditUserMessage {
	return NewEditUserMessage(
		msgRepo,
		convRepo,
		memService,
		generateUC,
		idGen,
		txManager,
	)
}

func TestEditUserMessage_Execute_Success(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Create an assistant message that will be deleted
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "Assistant response")
	msgRepo.Create(context.Background(), assistantMsg)

	// Set up downstream messages to be returned
	msgRepo.getAfterSequenceResult = []*models.Message{assistantMsg}

	// Create a real GenerateResponse use case with mocked dependencies
	llmService := newMockLLMService()
	llmService.chatResponse = &ports.LLMResponse{Content: "New response to edited message"}

	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		llmService,
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the message was updated
	if output.UpdatedMessage == nil {
		t.Fatal("expected updated message to be returned")
	}

	if output.UpdatedMessage.Contents != "Edited content" {
		t.Errorf("expected content 'Edited content', got '%s'", output.UpdatedMessage.Contents)
	}

	// Verify deleted count
	if output.DeletedCount != 1 {
		t.Errorf("expected deleted count 1, got %d", output.DeletedCount)
	}

	// Verify assistant message was generated
	if output.AssistantMessage == nil {
		t.Fatal("expected assistant message to be generated")
	}

	if output.AssistantMessage.Contents != "New response to edited message" {
		t.Errorf("expected response content 'New response to edited message', got '%s'", output.AssistantMessage.Contents)
	}

	// Verify message stored in repo has updated content
	storedMsg, _ := msgRepo.GetByID(context.Background(), "user_msg_1")
	if storedMsg.Contents != "Edited content" {
		t.Errorf("stored message content not updated, got '%s'", storedMsg.Contents)
	}
}

func TestEditUserMessage_Execute_MessageNotFound(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Set up message not found error
	msgRepo.getByIDError = errors.New("not found")

	// Create a dummy GenerateResponse use case (won't be called)
	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		newMockLLMService(),
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "nonexistent_msg",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when message not found, got nil")
	}

	expectedErr := "failed to get target message: not found"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%v'", expectedErr, err)
	}
}

func TestEditUserMessage_Execute_NotUserMessage(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create an assistant message (not a user message)
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 0, models.MessageRoleAssistant, "Assistant content")
	msgRepo.Create(context.Background(), assistantMsg)

	// Create a dummy GenerateResponse use case (won't be called)
	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		newMockLLMService(),
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "asst_msg_1",
		NewContent:      "Trying to edit assistant message",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when trying to edit assistant message, got nil")
	}

	expectedErr := "cannot edit message: expected user message but got assistant"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%v'", expectedErr, err)
	}
}

func TestEditUserMessage_Execute_UpdateFails(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Set up update error
	msgRepo.updateError = errors.New("database connection failed")

	// Create a dummy GenerateResponse use case (won't be called due to update failure)
	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		newMockLLMService(),
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when update fails, got nil")
	}

	expectedErr := "failed to update message: database connection failed"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%v'", expectedErr, err)
	}
}

func TestEditUserMessage_Execute_DeleteDownstreamFails(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Set up delete after sequence error
	msgRepo.deleteAfterSequenceError = errors.New("delete operation failed")

	// Create a dummy GenerateResponse use case (won't be called due to delete failure)
	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		newMockLLMService(),
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when delete downstream fails, got nil")
	}

	expectedErr := "failed to delete downstream messages: delete operation failed"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%v'", expectedErr, err)
	}
}

func TestEditUserMessage_Execute_GenerateResponseFails(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Set up LLM to fail
	llmService := newMockLLMService()
	llmService.chatError = errors.New("LLM service unavailable")

	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		llmService,
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when generate response fails, got nil")
	}

	// Verify error contains expected substring
	if !containsSubstring(err.Error(), "failed to generate response") {
		t.Errorf("expected error containing 'failed to generate response', got '%v'", err)
	}
}

func TestEditUserMessage_Execute_Streaming(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Set up streaming LLM service
	llmService := newMockLLMService()
	streamCh := make(chan ports.LLMStreamChunk, 10)
	llmService.streamChannel = streamCh

	go func() {
		defer close(streamCh)
		streamCh <- ports.LLMStreamChunk{Content: "Streaming ", Done: false}
		streamCh <- ports.LLMStreamChunk{Content: "response", Done: false}
		streamCh <- ports.LLMStreamChunk{Done: true}
	}()

	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		llmService,
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
		EnableStreaming: true,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify stream channel is provided
	if output.StreamChannel == nil {
		t.Fatal("expected stream channel to be provided for streaming mode")
	}

	// Verify updated message is returned
	if output.UpdatedMessage == nil {
		t.Fatal("expected updated message to be returned")
	}

	if output.UpdatedMessage.Contents != "Edited content" {
		t.Errorf("expected content 'Edited content', got '%s'", output.UpdatedMessage.Contents)
	}

	// Consume stream
	chunks := []string{}
	for chunk := range output.StreamChannel {
		if chunk.Error != nil {
			t.Fatalf("unexpected stream error: %v", chunk.Error)
		}
		if chunk.Text != "" {
			chunks = append(chunks, chunk.Text)
		}
	}

	if len(chunks) == 0 {
		t.Error("expected to receive stream chunks")
	}
}

func TestEditUserMessage_Execute_MemoryRetrieval(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Configure memory service to return relevant memories
	memorySearchCalled := false
	memService.searchWithScoresFunc = func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
		memorySearchCalled = true
		// Verify the search is using the new content
		if query != "What is my favorite color?" {
			t.Errorf("expected search query 'What is my favorite color?', got '%s'", query)
		}
		if threshold != 0.7 {
			t.Errorf("expected threshold 0.7, got %f", threshold)
		}
		if limit != 5 {
			t.Errorf("expected limit 5, got %d", limit)
		}
		mem := models.NewMemory("mem_1", "User's favorite color is blue")
		return []*ports.MemorySearchResult{
			{Memory: mem, Similarity: 0.95},
		}, nil
	}

	// Set up LLM service
	llmService := newMockLLMService()
	llmService.chatResponse = &ports.LLMResponse{Content: "Your favorite color is blue!"}

	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		llmService,
		newMockToolService(),
		memService,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "What is my favorite color?",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify memory search was called
	if !memorySearchCalled {
		t.Error("expected memory search to be called for new content")
	}

	// Verify relevant memories are returned in output
	if len(output.RelevantMemories) != 1 {
		t.Errorf("expected 1 relevant memory, got %d", len(output.RelevantMemories))
	}

	if output.RelevantMemories[0].Content != "User's favorite color is blue" {
		t.Errorf("expected memory content 'User's favorite color is blue', got '%s'", output.RelevantMemories[0].Content)
	}
}

func TestEditUserMessage_Execute_ConversationMismatch(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the user message belonging to a different conversation
	userMsg := models.NewMessage("user_msg_1", "conv_456", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Create a dummy GenerateResponse use case
	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		newMockLLMService(),
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123", // Different from the message's conversation
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when conversation ID doesn't match, got nil")
	}

	expectedErr := "message user_msg_1 does not belong to conversation conv_123"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%v'", expectedErr, err)
	}
}

func TestEditUserMessage_Execute_EmptyContent(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Create a dummy GenerateResponse use case
	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		newMockLLMService(),
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "", // Empty content
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when new content is empty, got nil")
	}

	expectedErr := "new content is required for user message edit"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%v'", expectedErr, err)
	}
}

func TestEditUserMessage_Execute_UpdateTipFails(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Set up update tip error
	convRepo.updateTipError = errors.New("failed to update conversation tip")

	// Create a dummy GenerateResponse use case
	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		newMockLLMService(),
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when update tip fails, got nil")
	}

	expectedErr := "failed to update conversation tip: failed to update conversation tip"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%v'", expectedErr, err)
	}
}

func TestEditUserMessage_Execute_MemoryRetrievalFails(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Configure memory service to fail
	memService.searchWithScoresFunc = func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
		return nil, errors.New("memory service unavailable")
	}

	// Set up LLM service (should still work despite memory failure)
	llmService := newMockLLMService()
	llmService.chatResponse = &ports.LLMResponse{Content: "Response without memory context"}

	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		llmService,
		newMockToolService(),
		memService,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
		EnableStreaming: false,
	}

	// Memory retrieval failure should not cause the entire operation to fail
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error (memory failure should be logged, not returned): %v", err)
	}

	// Verify operation succeeded despite memory failure
	if output.UpdatedMessage == nil {
		t.Fatal("expected updated message to be returned")
	}

	// Relevant memories should be empty due to failure
	if len(output.RelevantMemories) != 0 {
		t.Errorf("expected 0 relevant memories due to failure, got %d", len(output.RelevantMemories))
	}
}

func TestEditUserMessage_Execute_WithTools(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Create tool repository with enabled tools
	toolRepo := newMockToolRepo()
	tool := models.NewTool("tool_1", "calculator", "Calculate numbers", map[string]any{})
	tool.Enable()
	toolRepo.Create(context.Background(), tool)

	// Set up LLM service with tool response
	llmService := newMockLLMService()
	callCount := 0
	llmService.chatWithToolsFunc = func(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
		callCount++
		if callCount == 1 {
			return &ports.LLMResponse{
				Content: "Let me calculate",
				ToolCalls: []*ports.LLMToolCall{
					{Name: "calculator", Arguments: map[string]any{"expression": "2+2"}},
				},
			}, nil
		}
		return &ports.LLMResponse{Content: "The answer is 4"}, nil
	}

	toolService := newMockToolService()
	toolService.executeToolUseFunc = func(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
		tu := models.NewToolUse(toolUseID, "msg_test", "calculator", 0, map[string]any{"expression": "2+2"})
		tu.Complete("4")
		return tu, nil
	}

	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		toolRepo,
		newMockReasoningStepRepo(),
		convRepo,
		llmService,
		toolService,
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Calculate 2+2",
		EnableTools:     true,
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify updated message
	if output.UpdatedMessage == nil {
		t.Fatal("expected updated message")
	}

	// Verify assistant response
	if output.AssistantMessage == nil {
		t.Fatal("expected assistant message")
	}

	if output.AssistantMessage.Contents != "The answer is 4" {
		t.Errorf("expected 'The answer is 4', got '%s'", output.AssistantMessage.Contents)
	}
}

func TestEditUserMessage_Execute_GetAfterSequenceFails(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Set up GetAfterSequence to fail
	msgRepo.getAfterSequenceError = errors.New("failed to count downstream messages")

	// Create a dummy GenerateResponse use case
	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		newMockLLMService(),
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when GetAfterSequence fails, got nil")
	}

	expectedErr := "failed to get messages after sequence: failed to count downstream messages"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%v'", expectedErr, err)
	}
}

func TestEditUserMessage_Execute_MultipleDownstreamMessages(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Create multiple downstream messages
	assistantMsg1 := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "First response")
	userMsg2 := models.NewMessage("user_msg_2", "conv_123", 2, models.MessageRoleUser, "Follow-up question")
	assistantMsg2 := models.NewMessage("asst_msg_2", "conv_123", 3, models.MessageRoleAssistant, "Second response")
	msgRepo.Create(context.Background(), assistantMsg1)
	msgRepo.Create(context.Background(), userMsg2)
	msgRepo.Create(context.Background(), assistantMsg2)

	// Set up downstream messages to be returned
	msgRepo.getAfterSequenceResult = []*models.Message{assistantMsg1, userMsg2, assistantMsg2}

	// Set up LLM service
	llmService := newMockLLMService()
	llmService.chatResponse = &ports.LLMResponse{Content: "New response"}

	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		llmService,
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deleted count includes all downstream messages
	if output.DeletedCount != 3 {
		t.Errorf("expected deleted count 3, got %d", output.DeletedCount)
	}

	// Verify new response was generated
	if output.AssistantMessage == nil {
		t.Fatal("expected assistant message to be generated")
	}

	if output.AssistantMessage.Contents != "New response" {
		t.Errorf("expected 'New response', got '%s'", output.AssistantMessage.Contents)
	}
}

func TestEditUserMessage_Execute_NilMemoryService(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Set up LLM service
	llmService := newMockLLMService()
	llmService.chatResponse = &ports.LLMResponse{Content: "Response without memories"}

	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		llmService,
		newMockToolService(),
		nil, // No memory service
		nil,
		idGen,
		txManager,
		nil,
	)

	// Create use case with nil memory service
	uc := createEditUserMessageUC(msgRepo, convRepo, nil, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error with nil memory service: %v", err)
	}

	// Should succeed with empty memories
	if output.UpdatedMessage == nil {
		t.Fatal("expected updated message")
	}

	if output.RelevantMemories == nil {
		t.Log("relevant memories is nil as expected when memory service is nil")
	}
}

func TestEditUserMessage_Execute_UpdatesTimestamp(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message with a known timestamp
	originalTime := time.Now().Add(-1 * time.Hour).UTC()
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	userMsg.UpdatedAt = originalTime
	msgRepo.Create(context.Background(), userMsg)

	// Set up LLM service
	llmService := newMockLLMService()
	llmService.chatResponse = &ports.LLMResponse{Content: "Response"}

	generateUC := NewGenerateResponse(
		msgRepo,
		newMockSentenceRepo(),
		newMockToolUseRepo(),
		newMockToolRepo(),
		newMockReasoningStepRepo(),
		convRepo,
		llmService,
		newMockToolService(),
		nil,
		nil,
		idGen,
		txManager,
		nil,
	)

	uc := createEditUserMessageUC(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	beforeExecute := time.Now().UTC()

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the UpdatedAt timestamp was updated
	if output.UpdatedMessage.UpdatedAt.Before(beforeExecute) {
		t.Error("expected UpdatedAt to be updated to current time")
	}

	// Check the stored message in repo
	storedMsg, _ := msgRepo.GetByID(context.Background(), "user_msg_1")
	if storedMsg.UpdatedAt.Before(beforeExecute) {
		t.Error("stored message UpdatedAt should be updated")
	}
}

// Note: containsSubstring helper function is defined in edit_assistant_message_test.go
