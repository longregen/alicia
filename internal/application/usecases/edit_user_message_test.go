package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Test-specific mocks for EditUserMessage

// editMockMessageRepo provides extended mock implementation for edit tests
type editMockMessageRepo struct {
	*mockMessageRepo
	getByIDError      error
	createError       error
	getSiblingsResult []*models.Message
	getSiblingsError  error
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

func (m *editMockMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	if m.createError != nil {
		return m.createError
	}
	return m.mockMessageRepo.Create(ctx, msg)
}

func (m *editMockMessageRepo) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	if m.getSiblingsError != nil {
		return nil, m.getSiblingsError
	}
	if m.getSiblingsResult != nil {
		return m.getSiblingsResult, nil
	}
	return m.mockMessageRepo.GetSiblings(ctx, messageID)
}

// editMockConversationRepo provides extended mock implementation for edit tests
type editMockConversationRepo struct {
	*mockConversationRepo
	updateTipError   error
	updateTipCalled  bool
	lastTipMessageID string
}

func newEditMockConversationRepo() *editMockConversationRepo {
	return &editMockConversationRepo{
		mockConversationRepo: newMockConversationRepo(),
	}
}

func (m *editMockConversationRepo) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	m.updateTipCalled = true
	m.lastTipMessageID = messageID
	if m.updateTipError != nil {
		return m.updateTipError
	}
	return m.mockConversationRepo.UpdateTip(ctx, conversationID, messageID)
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

	// Create the original user message with a previous_id
	userMsg := models.NewMessage("user_msg_1", "conv_123", 1, models.MessageRoleUser, "Original content")
	userMsg.PreviousID = "system_msg_0" // Has a parent message
	msgRepo.Create(context.Background(), userMsg)

	// Create an assistant message that follows (should NOT be deleted in new behavior)
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 2, models.MessageRoleAssistant, "Assistant response")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Set up siblings result (the original message is its own sibling)
	msgRepo.getSiblingsResult = []*models.Message{userMsg}

	// Create mock generate response use case
	generateUC := newMockGenerateResponseUseCase()
	generateUC.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		msg := models.NewMessage("msg_new_response", input.ConversationID, 2, models.MessageRoleAssistant, "New response to edited message")
		msg.CompletionStatus = models.CompletionStatusCompleted
		msg.PreviousID = input.UserMessageID // Should be the new user message
		return &ports.GenerateResponseOutput{
			Message:        msg,
			Sentences:      []*models.Sentence{},
			ToolUses:       []*models.ToolUse{},
			ReasoningSteps: []*models.ReasoningStep{},
		}, nil
	}

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

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

	// Verify a NEW message was created (not the original updated)
	if output.UpdatedMessage == nil {
		t.Fatal("expected new message to be returned")
	}

	// The returned message should have the new content
	if output.UpdatedMessage.Contents != "Edited content" {
		t.Errorf("expected content 'Edited content', got '%s'", output.UpdatedMessage.Contents)
	}

	// The returned message should be a NEW message (different ID from original)
	if output.UpdatedMessage.ID == "user_msg_1" {
		t.Error("expected a new message ID, got the original message ID")
	}

	// The new message should be a sibling (same PreviousID as original)
	if output.UpdatedMessage.PreviousID != userMsg.PreviousID {
		t.Errorf("expected PreviousID '%s' (sibling), got '%s'", userMsg.PreviousID, output.UpdatedMessage.PreviousID)
	}

	// DeletedCount should be 0 (branching preserves history)
	if output.DeletedCount != 0 {
		t.Errorf("expected deleted count 0 (branching), got %d", output.DeletedCount)
	}

	// Verify conversation tip was updated to the new message
	if !convRepo.updateTipCalled {
		t.Error("expected UpdateTip to be called")
	}
	if convRepo.lastTipMessageID != output.UpdatedMessage.ID {
		t.Errorf("expected tip to be updated to new message %s, got %s", output.UpdatedMessage.ID, convRepo.lastTipMessageID)
	}

	// Verify assistant message was generated
	if output.AssistantMessage == nil {
		t.Fatal("expected assistant message to be generated")
	}

	if output.AssistantMessage.Contents != "New response to edited message" {
		t.Errorf("expected response content 'New response to edited message', got '%s'", output.AssistantMessage.Contents)
	}

	// Verify the original message still exists and is unchanged
	originalMsg, err := msgRepo.GetByID(context.Background(), "user_msg_1")
	if err != nil {
		t.Fatalf("original message should still exist: %v", err)
	}
	if originalMsg.Contents != "Original content" {
		t.Errorf("original message content should be unchanged, got '%s'", originalMsg.Contents)
	}

	// Verify the downstream assistant message still exists
	downstreamMsg, err := msgRepo.GetByID(context.Background(), "asst_msg_1")
	if err != nil {
		t.Fatalf("downstream message should still exist: %v", err)
	}
	if downstreamMsg.Contents != "Assistant response" {
		t.Errorf("downstream message content should be unchanged, got '%s'", downstreamMsg.Contents)
	}
}

func TestEditUserMessage_Execute_CreatesSiblingBranch(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 1, models.MessageRoleUser, "Original content")
	userMsg.PreviousID = "parent_msg"
	msgRepo.Create(context.Background(), userMsg)

	// Set up siblings (original is the only sibling initially)
	msgRepo.getSiblingsResult = []*models.Message{userMsg}

	generateUC := newMockGenerateResponseUseCase()
	generateUC.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		msg := models.NewMessage("msg_response", input.ConversationID, 2, models.MessageRoleAssistant, "Response")
		return &ports.GenerateResponseOutput{Message: msg}, nil
	}

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

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

	// Verify the new message is a sibling (same PreviousID, same SequenceNumber)
	if output.UpdatedMessage.PreviousID != "parent_msg" {
		t.Errorf("expected sibling to have same PreviousID 'parent_msg', got '%s'", output.UpdatedMessage.PreviousID)
	}

	if output.UpdatedMessage.SequenceNumber != userMsg.SequenceNumber {
		t.Errorf("expected sibling to have same SequenceNumber %d, got %d", userMsg.SequenceNumber, output.UpdatedMessage.SequenceNumber)
	}

	// Verify the new message was stored in the repo
	storedNewMsg, err := msgRepo.GetByID(context.Background(), output.UpdatedMessage.ID)
	if err != nil {
		t.Fatalf("new message should be stored in repo: %v", err)
	}
	if storedNewMsg.Contents != "Edited content" {
		t.Errorf("stored new message content should be 'Edited content', got '%s'", storedNewMsg.Contents)
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

	generateUC := newMockGenerateResponseUseCase()

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "nonexistent_msg",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for nonexistent message")
	}
}

func TestEditUserMessage_Execute_NotUserMessage(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create an assistant message (not a user message)
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 0, models.MessageRoleAssistant, "Assistant content")
	msgRepo.Create(context.Background(), assistantMsg)

	generateUC := newMockGenerateResponseUseCase()

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "asst_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when trying to edit non-user message")
	}
}

func TestEditUserMessage_Execute_WrongConversation(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a user message in a different conversation
	userMsg := models.NewMessage("user_msg_1", "conv_456", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	generateUC := newMockGenerateResponseUseCase()

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123", // Different conversation
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when conversation ID doesn't match")
	}
}

func TestEditUserMessage_Execute_EmptyContent(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	generateUC := newMockGenerateResponseUseCase()

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "", // Empty content
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty new content")
	}
}

func TestEditUserMessage_Execute_SkipGeneration(t *testing.T) {
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
	userMsg.PreviousID = "parent_msg"
	msgRepo.Create(context.Background(), userMsg)

	// Set up siblings
	msgRepo.getSiblingsResult = []*models.Message{userMsg}

	generateUC := newMockGenerateResponseUseCase()
	generateCalled := false
	generateUC.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		generateCalled = true
		return nil, errors.New("should not be called")
	}

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
		SkipGeneration:  true, // Skip response generation
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify generate was NOT called
	if generateCalled {
		t.Error("expected generate response to NOT be called when SkipGeneration is true")
	}

	// Verify a new message was created
	if output.UpdatedMessage == nil {
		t.Fatal("expected new message to be returned")
	}

	if output.UpdatedMessage.Contents != "Edited content" {
		t.Errorf("expected content 'Edited content', got '%s'", output.UpdatedMessage.Contents)
	}

	// Verify DeletedCount is 0 (branching)
	if output.DeletedCount != 0 {
		t.Errorf("expected DeletedCount 0, got %d", output.DeletedCount)
	}

	// Verify no assistant message was generated
	if output.AssistantMessage != nil {
		t.Error("expected no assistant message when SkipGeneration is true")
	}

	// Verify the new message is a sibling
	if output.UpdatedMessage.PreviousID != "parent_msg" {
		t.Errorf("expected sibling with PreviousID 'parent_msg', got '%s'", output.UpdatedMessage.PreviousID)
	}
}

func TestEditUserMessage_Execute_CreateError(t *testing.T) {
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
	msgRepo.mockMessageRepo.Create(context.Background(), userMsg) // Use inner mock to avoid error

	// Set up siblings
	msgRepo.getSiblingsResult = []*models.Message{userMsg}

	// Set up create error (for the new sibling message)
	msgRepo.createError = errors.New("create failed")

	generateUC := newMockGenerateResponseUseCase()

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when create fails")
	}
}

func TestEditUserMessage_Execute_UpdateTipError(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.mockConversationRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Original content")
	msgRepo.Create(context.Background(), userMsg)

	// Set up siblings
	msgRepo.getSiblingsResult = []*models.Message{userMsg}

	// Set up update tip error
	convRepo.updateTipError = errors.New("update tip failed")

	generateUC := newMockGenerateResponseUseCase()

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when update tip fails")
	}
}

func TestEditUserMessage_Execute_PreservesOriginalMessage(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 1, models.MessageRoleUser, "Original content")
	userMsg.PreviousID = "system_msg"
	msgRepo.Create(context.Background(), userMsg)

	// Set up siblings
	msgRepo.getSiblingsResult = []*models.Message{userMsg}

	generateUC := newMockGenerateResponseUseCase()
	generateUC.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		msg := models.NewMessage("response_msg", input.ConversationID, 2, models.MessageRoleAssistant, "Response")
		return &ports.GenerateResponseOutput{Message: msg}, nil
	}

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the original message is UNCHANGED
	originalMsg, err := msgRepo.GetByID(context.Background(), "user_msg_1")
	if err != nil {
		t.Fatalf("original message should still exist: %v", err)
	}

	if originalMsg.Contents != "Original content" {
		t.Errorf("original message content should be 'Original content', got '%s'", originalMsg.Contents)
	}

	if originalMsg.PreviousID != "system_msg" {
		t.Errorf("original message PreviousID should be 'system_msg', got '%s'", originalMsg.PreviousID)
	}
}

func TestEditUserMessage_Execute_GenerateResponseUsesNewMessage(t *testing.T) {
	msgRepo := newEditMockMessageRepo()
	convRepo := newEditMockConversationRepo()
	memService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create the original user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 1, models.MessageRoleUser, "Original content")
	userMsg.PreviousID = "parent_msg"
	msgRepo.Create(context.Background(), userMsg)

	// Set up siblings
	msgRepo.getSiblingsResult = []*models.Message{userMsg}

	var capturedGenerateInput *ports.GenerateResponseInput
	generateUC := newMockGenerateResponseUseCase()
	generateUC.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		capturedGenerateInput = input
		msg := models.NewMessage("response_msg", input.ConversationID, 2, models.MessageRoleAssistant, "Response")
		return &ports.GenerateResponseOutput{Message: msg}, nil
	}

	uc := NewEditUserMessage(msgRepo, convRepo, memService, generateUC, idGen, txManager)

	input := &ports.EditUserMessageInput{
		ConversationID:  "conv_123",
		TargetMessageID: "user_msg_1",
		NewContent:      "Edited content",
		EnableStreaming: false,
		EnableTools:     true,
		EnableReasoning: true,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify GenerateResponse was called with the NEW message ID, not the original
	if capturedGenerateInput == nil {
		t.Fatal("expected GenerateResponse to be called")
	}

	if capturedGenerateInput.UserMessageID == "user_msg_1" {
		t.Error("GenerateResponse should use the NEW message ID, not the original")
	}

	if capturedGenerateInput.UserMessageID != output.UpdatedMessage.ID {
		t.Errorf("GenerateResponse UserMessageID should be %s, got %s", output.UpdatedMessage.ID, capturedGenerateInput.UserMessageID)
	}

	// Verify PreviousID is set to the new user message
	if capturedGenerateInput.PreviousID != output.UpdatedMessage.ID {
		t.Errorf("GenerateResponse PreviousID should be %s, got %s", output.UpdatedMessage.ID, capturedGenerateInput.PreviousID)
	}

	// Verify flags are passed through
	if !capturedGenerateInput.EnableTools {
		t.Error("EnableTools should be passed through")
	}
	if !capturedGenerateInput.EnableReasoning {
		t.Error("EnableReasoning should be passed through")
	}
}
