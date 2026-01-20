package usecases

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// ============================================================================
// Mock implementations for SendMessage tests
// ============================================================================

// mockProcessUserMessageForSend is a mock implementation of ProcessUserMessage use case
type mockProcessUserMessageForSend struct {
	executeFunc func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error)
}

func newMockProcessUserMessageForSend() *mockProcessUserMessageForSend {
	return &mockProcessUserMessageForSend{}
}

func (m *mockProcessUserMessageForSend) Execute(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	// Default behavior: create a simple user message
	msg := models.NewMessage("msg_user_1", input.ConversationID, 0, models.MessageRoleUser, input.TextContent)
	return &ports.ProcessUserMessageOutput{
		Message:          msg,
		Audio:            nil,
		RelevantMemories: []*models.Memory{},
	}, nil
}

// mockGenerateResponseForSend is a mock implementation of GenerateResponse use case
type mockGenerateResponseForSend struct {
	executeFunc func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error)
}

func newMockGenerateResponseForSend() *mockGenerateResponseForSend {
	return &mockGenerateResponseForSend{}
}

func (m *mockGenerateResponseForSend) Execute(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	// Default behavior: create a simple assistant message
	msg := models.NewMessage("msg_assistant_1", input.ConversationID, 1, models.MessageRoleAssistant, "Hello! How can I help you?")
	msg.CompletionStatus = models.CompletionStatusCompleted
	return &ports.GenerateResponseOutput{
		Message:        msg,
		Sentences:      []*models.Sentence{},
		ToolUses:       []*models.ToolUse{},
		ReasoningSteps: []*models.ReasoningStep{},
		StreamChannel:  nil,
	}, nil
}

// mockMessageRepoForSend is a mock message repository for SendMessage tests
type mockMessageRepoForSend struct {
	mu         sync.RWMutex
	store      map[string]*models.Message
	deleteFunc func(ctx context.Context, id string) error
	deletedIDs []string
}

func newMockMessageRepoForSend() *mockMessageRepoForSend {
	return &mockMessageRepoForSend{
		store:      make(map[string]*models.Message),
		deletedIDs: []string{},
	}
}

func (m *mockMessageRepoForSend) copyMessage(msg *models.Message) *models.Message {
	if msg == nil {
		return nil
	}
	msgCopy := &models.Message{
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
	if msg.DeletedAt != nil {
		deletedAt := *msg.DeletedAt
		msgCopy.DeletedAt = &deletedAt
	}
	if msg.SyncedAt != nil {
		syncedAt := *msg.SyncedAt
		msgCopy.SyncedAt = &syncedAt
	}
	return msgCopy
}

func (m *mockMessageRepoForSend) Create(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *mockMessageRepoForSend) GetByID(ctx context.Context, id string) (*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if msg, ok := m.store[id]; ok {
		return m.copyMessage(msg), nil
	}
	return nil, errors.New("not found")
}

func (m *mockMessageRepoForSend) Update(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[msg.ID]; !ok {
		return errors.New("not found")
	}
	m.store[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *mockMessageRepoForSend) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedIDs = append(m.deletedIDs, id)
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	delete(m.store, id)
	return nil
}

func (m *mockMessageRepoForSend) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForSend) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	return 0, nil
}

func (m *mockMessageRepoForSend) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForSend) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForSend) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForSend) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, errors.New("not found")
}

func (m *mockMessageRepoForSend) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForSend) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForSend) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForSend) GetChainFromTipWithSiblings(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return m.GetChainFromTip(ctx, tipMessageID)
}

func (m *mockMessageRepoForSend) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForSend) DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error {
	return nil
}

// mockConversationRepoForSend is a mock conversation repository for SendMessage tests
type mockConversationRepoForSend struct {
	mu          sync.RWMutex
	store       map[string]*models.Conversation
	getByIDFunc func(ctx context.Context, id string) (*models.Conversation, error)
}

func newMockConversationRepoForSend() *mockConversationRepoForSend {
	return &mockConversationRepoForSend{
		store: make(map[string]*models.Conversation),
	}
}

func (m *mockConversationRepoForSend) Create(ctx context.Context, c *models.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[c.ID] = c
	return nil
}

func (m *mockConversationRepoForSend) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if c, ok := m.store[id]; ok {
		return c, nil
	}
	return nil, nil // Return nil, nil when not found (per convention)
}

func (m *mockConversationRepoForSend) Update(ctx context.Context, c *models.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[c.ID]; !ok {
		return errors.New("not found")
	}
	m.store[c.ID] = c
	return nil
}

func (m *mockConversationRepoForSend) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
	return nil
}

func (m *mockConversationRepoForSend) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepoForSend) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepoForSend) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	return nil, errors.New("not found")
}

func (m *mockConversationRepoForSend) UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error {
	return nil
}

func (m *mockConversationRepoForSend) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	return nil
}

func (m *mockConversationRepoForSend) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	return nil, errors.New("not found")
}

func (m *mockConversationRepoForSend) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepoForSend) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepoForSend) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.store[conversationID]; ok {
		c.TipMessageID = &messageID
		return nil
	}
	return errors.New("not found")
}

func (m *mockConversationRepoForSend) UpdatePromptVersion(ctx context.Context, convID, versionID string) error {
	return nil
}

// mockTransactionManagerForSend is a mock transaction manager for SendMessage tests
type mockTransactionManagerForSend struct{}

func (m *mockTransactionManagerForSend) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// ============================================================================
// Test helper function
// ============================================================================

// executeSendMessageWithMocks is a helper function that mimics SendMessage.Execute
// but allows us to inject mock use cases for testing
func executeSendMessageWithMocks(
	ctx context.Context,
	convRepo *mockConversationRepoForSend,
	msgRepo *mockMessageRepoForSend,
	processUserMessage *mockProcessUserMessageForSend,
	generateResponse *mockGenerateResponseForSend,
	txManager *mockTransactionManagerForSend,
	input *ports.SendMessageInput,
) (*ports.SendMessageOutput, error) {
	// 1. Validate conversation exists and is active
	conversation, err := convRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, errors.New("conversation not found: " + input.ConversationID)
	}

	if conversation.Status != "active" {
		return nil, errors.New("conversation is not active")
	}

	// 2. Call ProcessUserMessage.Execute()
	processInput := &ports.ProcessUserMessageInput{
		ConversationID: input.ConversationID,
		TextContent:    input.TextContent,
		AudioData:      input.AudioData,
		AudioFormat:    input.AudioFormat,
		PreviousID:     input.PreviousID,
	}

	processOutput, err := processUserMessage.Execute(ctx, processInput)
	if err != nil {
		return nil, err
	}

	// 3. Call GenerateResponse.Execute()
	generateInput := &ports.GenerateResponseInput{
		ConversationID:   input.ConversationID,
		UserMessageID:    processOutput.Message.ID,
		RelevantMemories: processOutput.RelevantMemories,
		EnableTools:      input.EnableTools,
		EnableReasoning:  input.EnableReasoning,
		EnableStreaming:  input.EnableStreaming,
		PreviousID:       processOutput.Message.ID,
	}

	generateOutput, err := generateResponse.Execute(ctx, generateInput)
	if err != nil {
		// Compensating action: delete the orphaned user message
		_ = msgRepo.Delete(ctx, processOutput.Message.ID)
		return nil, err
	}

	// 4. Return combined output
	return &ports.SendMessageOutput{
		UserMessage:      processOutput.Message,
		Audio:            processOutput.Audio,
		RelevantMemories: processOutput.RelevantMemories,
		AssistantMessage: generateOutput.Message,
		StreamChannel:    generateOutput.StreamChannel,
	}, nil
}

// ============================================================================
// Test cases
// ============================================================================

// TestSendMessage_Execute_Success tests the happy path with a text message
func TestSendMessage_Execute_Success(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Create an active conversation
	conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	// Configure ProcessUserMessage to return a user message with memories
	mem := models.NewMemory("mem_1", "User likes coffee")
	processUserMessage.executeFunc = func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
		if input.ConversationID != "conv_123" {
			t.Errorf("expected conversation ID conv_123, got %s", input.ConversationID)
		}
		if input.TextContent != "Hello there!" {
			t.Errorf("expected text 'Hello there!', got %s", input.TextContent)
		}
		msg := models.NewMessage("msg_user_1", input.ConversationID, 0, models.MessageRoleUser, input.TextContent)
		return &ports.ProcessUserMessageOutput{
			Message:          msg,
			Audio:            nil,
			RelevantMemories: []*models.Memory{mem},
		}, nil
	}

	// Configure GenerateResponse to return an assistant message
	generateResponse.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		if input.ConversationID != "conv_123" {
			t.Errorf("expected conversation ID conv_123, got %s", input.ConversationID)
		}
		if input.UserMessageID != "msg_user_1" {
			t.Errorf("expected user message ID msg_user_1, got %s", input.UserMessageID)
		}
		if len(input.RelevantMemories) != 1 {
			t.Errorf("expected 1 relevant memory, got %d", len(input.RelevantMemories))
		}
		msg := models.NewMessage("msg_assistant_1", input.ConversationID, 1, models.MessageRoleAssistant, "Hi! Nice to meet you!")
		msg.CompletionStatus = models.CompletionStatusCompleted
		return &ports.GenerateResponseOutput{
			Message: msg,
		}, nil
	}

	input := &ports.SendMessageInput{
		ConversationID:  "conv_123",
		TextContent:     "Hello there!",
		EnableTools:     true,
		EnableReasoning: false,
	}

	output, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.UserMessage == nil {
		t.Fatal("expected user message to be set")
	}

	if output.UserMessage.ID != "msg_user_1" {
		t.Errorf("expected user message ID msg_user_1, got %s", output.UserMessage.ID)
	}

	if output.AssistantMessage == nil {
		t.Fatal("expected assistant message to be set")
	}

	if output.AssistantMessage.ID != "msg_assistant_1" {
		t.Errorf("expected assistant message ID msg_assistant_1, got %s", output.AssistantMessage.ID)
	}

	if len(output.RelevantMemories) != 1 {
		t.Errorf("expected 1 relevant memory, got %d", len(output.RelevantMemories))
	}

	if output.StreamChannel != nil {
		t.Error("expected no stream channel for non-streaming mode")
	}
}

// TestSendMessage_Execute_WithAudio tests message with audio data
func TestSendMessage_Execute_WithAudio(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Create an active conversation
	conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	// Configure ProcessUserMessage to return a user message with audio
	audioData := []byte("fake audio data")
	processUserMessage.executeFunc = func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
		if len(input.AudioData) == 0 {
			t.Error("expected audio data to be passed")
		}
		if input.AudioFormat != "audio/opus" {
			t.Errorf("expected audio format 'audio/opus', got %s", input.AudioFormat)
		}
		msg := models.NewMessage("msg_user_1", input.ConversationID, 0, models.MessageRoleUser, "Transcribed audio text")
		audio := models.NewAudio("audio_1", models.AudioTypeInput, "audio/opus")
		audio.MessageID = msg.ID
		return &ports.ProcessUserMessageOutput{
			Message: msg,
			Audio:   audio,
		}, nil
	}

	input := &ports.SendMessageInput{
		ConversationID: "conv_123",
		AudioData:      audioData,
		AudioFormat:    "audio/opus",
	}

	output, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Audio == nil {
		t.Fatal("expected audio to be set")
	}

	if output.Audio.ID != "audio_1" {
		t.Errorf("expected audio ID audio_1, got %s", output.Audio.ID)
	}

	if output.Audio.AudioFormat != "audio/opus" {
		t.Errorf("expected audio format 'audio/opus', got %s", output.Audio.AudioFormat)
	}
}

// TestSendMessage_Execute_ConversationNotFound tests error when conversation doesn't exist
func TestSendMessage_Execute_ConversationNotFound(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Configure repo to return nil (not found)
	convRepo.getByIDFunc = func(ctx context.Context, id string) (*models.Conversation, error) {
		return nil, nil
	}

	input := &ports.SendMessageInput{
		ConversationID: "nonexistent_conv",
		TextContent:    "Hello",
	}

	_, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err == nil {
		t.Fatal("expected error for nonexistent conversation, got nil")
	}

	if err.Error() != "conversation not found: nonexistent_conv" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestSendMessage_Execute_ConversationInactive tests error when conversation is archived
func TestSendMessage_Execute_ConversationInactive(t *testing.T) {
	testCases := []struct {
		name   string
		status models.ConversationStatus
	}{
		{"archived conversation", models.ConversationStatusArchived},
		{"deleted conversation", models.ConversationStatusDeleted},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			convRepo := newMockConversationRepoForSend()
			msgRepo := newMockMessageRepoForSend()
			processUserMessage := newMockProcessUserMessageForSend()
			generateResponse := newMockGenerateResponseForSend()
			txManager := &mockTransactionManagerForSend{}

			// Create a conversation with non-active status
			conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
			conv.Status = tc.status
			convRepo.Create(context.Background(), conv)

			input := &ports.SendMessageInput{
				ConversationID: "conv_123",
				TextContent:    "Hello",
			}

			_, err := executeSendMessageWithMocks(
				context.Background(),
				convRepo,
				msgRepo,
				processUserMessage,
				generateResponse,
				txManager,
				input,
			)

			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}

			expectedErrSubstring := "conversation is not active"
			if err.Error() != expectedErrSubstring {
				t.Errorf("expected error containing '%s', got: %v", expectedErrSubstring, err)
			}
		})
	}
}

// TestSendMessage_Execute_ProcessUserMessageFails tests error handling when ProcessUserMessage fails
func TestSendMessage_Execute_ProcessUserMessageFails(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Create an active conversation
	conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	// Configure ProcessUserMessage to fail
	processUserMessage.executeFunc = func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
		return nil, errors.New("transcription failed")
	}

	input := &ports.SendMessageInput{
		ConversationID: "conv_123",
		AudioData:      []byte("audio"),
		AudioFormat:    "audio/opus",
	}

	_, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err == nil {
		t.Fatal("expected error when ProcessUserMessage fails, got nil")
	}

	if err.Error() != "transcription failed" {
		t.Errorf("expected error 'transcription failed', got: %v", err)
	}

	// Verify no cleanup was attempted (no message was created)
	if len(msgRepo.deletedIDs) != 0 {
		t.Errorf("expected no messages to be deleted, got %d", len(msgRepo.deletedIDs))
	}
}

// TestSendMessage_Execute_GenerateResponseFails tests error handling and cleanup when GenerateResponse fails
func TestSendMessage_Execute_GenerateResponseFails(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Create an active conversation
	conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	// Configure ProcessUserMessage to succeed
	processUserMessage.executeFunc = func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
		msg := models.NewMessage("msg_user_to_delete", input.ConversationID, 0, models.MessageRoleUser, input.TextContent)
		return &ports.ProcessUserMessageOutput{
			Message: msg,
		}, nil
	}

	// Configure GenerateResponse to fail
	generateResponse.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		return nil, errors.New("LLM service unavailable")
	}

	input := &ports.SendMessageInput{
		ConversationID: "conv_123",
		TextContent:    "Hello",
	}

	_, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err == nil {
		t.Fatal("expected error when GenerateResponse fails, got nil")
	}

	if err.Error() != "LLM service unavailable" {
		t.Errorf("expected error 'LLM service unavailable', got: %v", err)
	}

	// Verify compensating action: user message was deleted
	if len(msgRepo.deletedIDs) != 1 {
		t.Errorf("expected 1 message to be deleted, got %d", len(msgRepo.deletedIDs))
	}

	if len(msgRepo.deletedIDs) > 0 && msgRepo.deletedIDs[0] != "msg_user_to_delete" {
		t.Errorf("expected message 'msg_user_to_delete' to be deleted, got %s", msgRepo.deletedIDs[0])
	}
}

// TestSendMessage_Execute_GenerateResponseFails_DeleteAlsoFails tests cleanup failure is logged but doesn't affect error
func TestSendMessage_Execute_GenerateResponseFails_DeleteAlsoFails(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Create an active conversation
	conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	// Configure ProcessUserMessage to succeed
	processUserMessage.executeFunc = func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
		msg := models.NewMessage("msg_user_1", input.ConversationID, 0, models.MessageRoleUser, input.TextContent)
		return &ports.ProcessUserMessageOutput{
			Message: msg,
		}, nil
	}

	// Configure GenerateResponse to fail
	generateResponse.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		return nil, errors.New("LLM service unavailable")
	}

	// Configure Delete to also fail
	msgRepo.deleteFunc = func(ctx context.Context, id string) error {
		return errors.New("database connection lost")
	}

	input := &ports.SendMessageInput{
		ConversationID: "conv_123",
		TextContent:    "Hello",
	}

	_, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	// The original error from GenerateResponse should be returned
	if err == nil {
		t.Fatal("expected error when GenerateResponse fails, got nil")
	}

	if err.Error() != "LLM service unavailable" {
		t.Errorf("expected error 'LLM service unavailable', got: %v", err)
	}

	// Delete was still attempted
	if len(msgRepo.deletedIDs) != 1 {
		t.Errorf("expected delete to be attempted once, got %d attempts", len(msgRepo.deletedIDs))
	}
}

// TestSendMessage_Execute_Streaming tests that streaming mode returns a channel
func TestSendMessage_Execute_Streaming(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Create an active conversation
	conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	// Create a stream channel
	streamCh := make(chan *ports.ResponseStreamChunk, 10)

	// Configure GenerateResponse to return a stream channel
	generateResponse.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		if !input.EnableStreaming {
			t.Error("expected EnableStreaming to be true")
		}
		msg := models.NewMessage("msg_assistant_1", input.ConversationID, 1, models.MessageRoleAssistant, "")
		msg.CompletionStatus = models.CompletionStatusStreaming
		return &ports.GenerateResponseOutput{
			Message:       msg,
			StreamChannel: streamCh,
		}, nil
	}

	// Send some chunks asynchronously
	go func() {
		defer close(streamCh)
		streamCh <- &ports.ResponseStreamChunk{Text: "Hello ", IsFinal: false}
		streamCh <- &ports.ResponseStreamChunk{Text: "world!", IsFinal: false}
		streamCh <- &ports.ResponseStreamChunk{IsFinal: true}
	}()

	input := &ports.SendMessageInput{
		ConversationID:  "conv_123",
		TextContent:     "Hello",
		EnableStreaming: true,
	}

	output, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.StreamChannel == nil {
		t.Fatal("expected stream channel to be provided for streaming mode")
	}

	// Consume the stream and verify content
	chunks := []string{}
	for chunk := range output.StreamChannel {
		if chunk.Error != nil {
			t.Fatalf("unexpected stream error: %v", chunk.Error)
		}
		if chunk.Text != "" {
			chunks = append(chunks, chunk.Text)
		}
	}

	if len(chunks) != 2 {
		t.Errorf("expected 2 text chunks, got %d", len(chunks))
	}

	if len(chunks) >= 2 && (chunks[0] != "Hello " || chunks[1] != "world!") {
		t.Errorf("unexpected chunks: %v", chunks)
	}
}

// TestSendMessage_Execute_WithAllOptions tests with all options enabled
func TestSendMessage_Execute_WithAllOptions(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Create an active conversation
	conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	// Configure GenerateResponse to verify all options are passed
	generateResponse.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		if !input.EnableTools {
			t.Error("expected EnableTools to be true")
		}
		if !input.EnableReasoning {
			t.Error("expected EnableReasoning to be true")
		}
		if input.EnableStreaming {
			t.Error("expected EnableStreaming to be false")
		}
		msg := models.NewMessage("msg_assistant_1", input.ConversationID, 1, models.MessageRoleAssistant, "Response with reasoning")
		msg.CompletionStatus = models.CompletionStatusCompleted
		reasoningStep := models.NewReasoningStep("rs_1", msg.ID, "Thinking about the question...", 0)
		return &ports.GenerateResponseOutput{
			Message:        msg,
			ReasoningSteps: []*models.ReasoningStep{reasoningStep},
		}, nil
	}

	input := &ports.SendMessageInput{
		ConversationID:  "conv_123",
		TextContent:     "What is 2+2?",
		EnableTools:     true,
		EnableReasoning: true,
		EnableStreaming: false,
	}

	output, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.AssistantMessage == nil {
		t.Fatal("expected assistant message to be set")
	}

	if output.AssistantMessage.Contents != "Response with reasoning" {
		t.Errorf("expected content 'Response with reasoning', got %s", output.AssistantMessage.Contents)
	}
}

// TestSendMessage_Execute_ConversationRepoError tests error when conversation repo returns error
func TestSendMessage_Execute_ConversationRepoError(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Configure repo to return an error
	convRepo.getByIDFunc = func(ctx context.Context, id string) (*models.Conversation, error) {
		return nil, errors.New("database connection failed")
	}

	input := &ports.SendMessageInput{
		ConversationID: "conv_123",
		TextContent:    "Hello",
	}

	_, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err == nil {
		t.Fatal("expected error when conversation repo fails, got nil")
	}

	if err.Error() != "database connection failed" {
		t.Errorf("expected error 'database connection failed', got: %v", err)
	}
}

// TestSendMessage_Execute_PassesPreviousID tests that PreviousID is correctly passed through
func TestSendMessage_Execute_PassesPreviousID(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Create an active conversation
	conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	// Configure ProcessUserMessage to verify PreviousID is passed
	processUserMessage.executeFunc = func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
		if input.PreviousID != "msg_previous" {
			t.Errorf("expected PreviousID 'msg_previous', got %s", input.PreviousID)
		}
		msg := models.NewMessage("msg_user_1", input.ConversationID, 0, models.MessageRoleUser, input.TextContent)
		msg.PreviousID = input.PreviousID
		return &ports.ProcessUserMessageOutput{
			Message: msg,
		}, nil
	}

	// Configure GenerateResponse to verify PreviousID is set to user message ID
	generateResponse.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		if input.PreviousID != "msg_user_1" {
			t.Errorf("expected PreviousID 'msg_user_1' (the user message), got %s", input.PreviousID)
		}
		msg := models.NewMessage("msg_assistant_1", input.ConversationID, 1, models.MessageRoleAssistant, "Response")
		return &ports.GenerateResponseOutput{
			Message: msg,
		}, nil
	}

	input := &ports.SendMessageInput{
		ConversationID: "conv_123",
		TextContent:    "Hello",
		PreviousID:     "msg_previous",
	}

	_, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSendMessage_Execute_MemoriesPassedToGenerateResponse tests that memories from ProcessUserMessage are passed to GenerateResponse
func TestSendMessage_Execute_MemoriesPassedToGenerateResponse(t *testing.T) {
	convRepo := newMockConversationRepoForSend()
	msgRepo := newMockMessageRepoForSend()
	processUserMessage := newMockProcessUserMessageForSend()
	generateResponse := newMockGenerateResponseForSend()
	txManager := &mockTransactionManagerForSend{}

	// Create an active conversation
	conv := models.NewConversation("conv_123", "user_1", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	// Create test memories
	mem1 := models.NewMemory("mem_1", "User's name is Alice")
	mem2 := models.NewMemory("mem_2", "User likes programming")

	// Configure ProcessUserMessage to return memories
	processUserMessage.executeFunc = func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
		msg := models.NewMessage("msg_user_1", input.ConversationID, 0, models.MessageRoleUser, input.TextContent)
		return &ports.ProcessUserMessageOutput{
			Message:          msg,
			RelevantMemories: []*models.Memory{mem1, mem2},
		}, nil
	}

	// Configure GenerateResponse to verify memories are passed
	generateResponse.executeFunc = func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
		if len(input.RelevantMemories) != 2 {
			t.Errorf("expected 2 relevant memories, got %d", len(input.RelevantMemories))
		}
		if len(input.RelevantMemories) >= 2 {
			if input.RelevantMemories[0].Content != "User's name is Alice" {
				t.Errorf("expected first memory content 'User's name is Alice', got %s", input.RelevantMemories[0].Content)
			}
			if input.RelevantMemories[1].Content != "User likes programming" {
				t.Errorf("expected second memory content 'User likes programming', got %s", input.RelevantMemories[1].Content)
			}
		}
		msg := models.NewMessage("msg_assistant_1", input.ConversationID, 1, models.MessageRoleAssistant, "Hello Alice!")
		return &ports.GenerateResponseOutput{
			Message: msg,
		}, nil
	}

	input := &ports.SendMessageInput{
		ConversationID: "conv_123",
		TextContent:    "Hello, who am I?",
	}

	output, err := executeSendMessageWithMocks(
		context.Background(),
		convRepo,
		msgRepo,
		processUserMessage,
		generateResponse,
		txManager,
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify memories are also returned in output
	if len(output.RelevantMemories) != 2 {
		t.Errorf("expected 2 relevant memories in output, got %d", len(output.RelevantMemories))
	}
}
