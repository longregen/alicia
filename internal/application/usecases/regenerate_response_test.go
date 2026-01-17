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

// Mock implementations specific to RegenerateResponse tests

// mockMessageRepoForRegenerate is a specialized mock for regenerate response tests
type mockMessageRepoForRegenerate struct {
	mu              sync.RWMutex
	store           map[string]*models.Message
	sequenceNumbers map[string]int
	getByIDErr      error
	deleteErr       error
}

func newMockMessageRepoForRegenerate() *mockMessageRepoForRegenerate {
	return &mockMessageRepoForRegenerate{
		store:           make(map[string]*models.Message),
		sequenceNumbers: make(map[string]int),
	}
}

func (m *mockMessageRepoForRegenerate) copyMessage(msg *models.Message) *models.Message {
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

func (m *mockMessageRepoForRegenerate) Create(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *mockMessageRepoForRegenerate) GetByID(ctx context.Context, id string) (*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if msg, ok := m.store[id]; ok {
		return m.copyMessage(msg), nil
	}
	return nil, nil // Return nil, nil for not found (as per the use case expectation)
}

func (m *mockMessageRepoForRegenerate) Update(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[msg.ID]; !ok {
		return errors.New("not found")
	}
	m.store[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *mockMessageRepoForRegenerate) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.store, id)
	return nil
}

func (m *mockMessageRepoForRegenerate) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForRegenerate) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	messages := []*models.Message{}
	for _, msg := range m.store {
		if msg.ConversationID == conversationID {
			messages = append(messages, m.copyMessage(msg))
		}
	}
	return messages, nil
}

func (m *mockMessageRepoForRegenerate) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	num := m.sequenceNumbers[conversationID]
	m.sequenceNumbers[conversationID] = num + 1
	return num, nil
}

func (m *mockMessageRepoForRegenerate) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForRegenerate) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var chain []*models.Message
	currentID := tipMessageID

	for currentID != "" {
		msg, ok := m.store[currentID]
		if !ok {
			break
		}
		chain = append([]*models.Message{m.copyMessage(msg)}, chain...)
		if msg.PreviousID == "" {
			break
		}
		currentID = msg.PreviousID
	}

	return chain, nil
}

func (m *mockMessageRepoForRegenerate) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForRegenerate) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForRegenerate) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, errors.New("not found")
}

func (m *mockMessageRepoForRegenerate) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForRegenerate) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepoForRegenerate) DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error {
	return nil
}

// mockIDGeneratorForRegenerate is a simple mock for ID generation
type mockIDGeneratorForRegenerate struct {
	messageCounter int
}

func newMockIDGeneratorForRegenerate() *mockIDGeneratorForRegenerate {
	return &mockIDGeneratorForRegenerate{}
}

func (m *mockIDGeneratorForRegenerate) GenerateConversationID() string {
	return "conv_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateMessageID() string {
	m.messageCounter++
	return "msg_new_" + string(rune('0'+m.messageCounter))
}

func (m *mockIDGeneratorForRegenerate) GenerateSentenceID() string {
	return "sent_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateAudioID() string {
	return "audio_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateMemoryID() string {
	return "mem_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateMemoryUsageID() string {
	return "mu_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateToolID() string {
	return "tool_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateToolUseID() string {
	return "tu_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateReasoningStepID() string {
	return "rs_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateCommentaryID() string {
	return "comm_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateMetaID() string {
	return "meta_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateMCPServerID() string {
	return "amcp_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateVoteID() string {
	return "av_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateNoteID() string {
	return "an_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateSessionStatsID() string {
	return "ass_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateOptimizationRunID() string {
	return "aor_test1"
}

func (m *mockIDGeneratorForRegenerate) GeneratePromptCandidateID() string {
	return "apc_test1"
}

func (m *mockIDGeneratorForRegenerate) GeneratePromptEvaluationID() string {
	return "ape_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateTrainingExampleID() string {
	return "gte_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateSystemPromptVersionID() string {
	return "spv_test1"
}

func (m *mockIDGeneratorForRegenerate) GenerateRequestID() string {
	return "areq_test1"
}

// mockConversationRepoForRegenerate is a simple mock for conversation repository
type mockConversationRepoForRegenerate struct {
	store map[string]*models.Conversation
}

func newMockConversationRepoForRegenerate() *mockConversationRepoForRegenerate {
	return &mockConversationRepoForRegenerate{
		store: make(map[string]*models.Conversation),
	}
}

func (m *mockConversationRepoForRegenerate) Create(ctx context.Context, c *models.Conversation) error {
	m.store[c.ID] = c
	return nil
}

func (m *mockConversationRepoForRegenerate) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	if c, ok := m.store[id]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (m *mockConversationRepoForRegenerate) Update(ctx context.Context, c *models.Conversation) error {
	if _, ok := m.store[c.ID]; !ok {
		return errors.New("not found")
	}
	m.store[c.ID] = c
	return nil
}

func (m *mockConversationRepoForRegenerate) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

func (m *mockConversationRepoForRegenerate) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepoForRegenerate) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepoForRegenerate) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	return nil, errors.New("not found")
}

func (m *mockConversationRepoForRegenerate) UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error {
	return nil
}

func (m *mockConversationRepoForRegenerate) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	delete(m.store, id)
	return nil
}

func (m *mockConversationRepoForRegenerate) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	if c, ok := m.store[id]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (m *mockConversationRepoForRegenerate) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepoForRegenerate) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepoForRegenerate) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	if c, ok := m.store[conversationID]; ok {
		c.TipMessageID = &messageID
		return nil
	}
	return errors.New("not found")
}

func (m *mockConversationRepoForRegenerate) UpdatePromptVersion(ctx context.Context, convID, versionID string) error {
	return nil
}

// mockLLMServiceForRegenerate is a mock LLM service for regenerate tests
type mockLLMServiceForRegenerate struct {
	chatResponse           *ports.LLMResponse
	chatError              error
	streamChannel          chan ports.LLMStreamChunk
	streamError            error
	streamWithToolsChannel chan ports.LLMStreamChunk
}

func newMockLLMServiceForRegenerate() *mockLLMServiceForRegenerate {
	return &mockLLMServiceForRegenerate{}
}

func (m *mockLLMServiceForRegenerate) Chat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	if m.chatError != nil {
		return nil, m.chatError
	}
	if m.chatResponse != nil {
		return m.chatResponse, nil
	}
	return &ports.LLMResponse{
		Content: "Regenerated response",
	}, nil
}

func (m *mockLLMServiceForRegenerate) ChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	if m.chatError != nil {
		return nil, m.chatError
	}
	if m.chatResponse != nil {
		return m.chatResponse, nil
	}
	return &ports.LLMResponse{
		Content: "Regenerated response with tools",
	}, nil
}

func (m *mockLLMServiceForRegenerate) ChatStream(ctx context.Context, messages []ports.LLMMessage) (<-chan ports.LLMStreamChunk, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}
	if m.streamChannel != nil {
		return m.streamChannel, nil
	}
	ch := make(chan ports.LLMStreamChunk, 10)
	go func() {
		defer close(ch)
		ch <- ports.LLMStreamChunk{Content: "Streaming ", Done: false}
		ch <- ports.LLMStreamChunk{Content: "response", Done: false}
		ch <- ports.LLMStreamChunk{Done: true}
	}()
	return ch, nil
}

func (m *mockLLMServiceForRegenerate) ChatStreamWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (<-chan ports.LLMStreamChunk, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}
	if m.streamWithToolsChannel != nil {
		return m.streamWithToolsChannel, nil
	}
	ch := make(chan ports.LLMStreamChunk, 10)
	go func() {
		defer close(ch)
		ch <- ports.LLMStreamChunk{Content: "Streaming with tools", Done: false}
		ch <- ports.LLMStreamChunk{Done: true}
	}()
	return ch, nil
}

// Helper function to create a GenerateResponse use case with mocks for regenerate tests
func createGenerateResponseForRegenerate(
	msgRepo ports.MessageRepository,
	convRepo ports.ConversationRepository,
	llmService ports.LLMService,
	idGen ports.IDGenerator,
) *GenerateResponse {
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	toolService := newMockToolService()
	txManager := &mockTransactionManager{}

	return NewGenerateResponse(
		msgRepo,
		sentRepo,
		toolUseRepo,
		toolRepo,
		reasoningRepo,
		convRepo,
		llmService,
		toolService,
		nil, // No memory service
		nil, // No prompt version service
		idGen,
		txManager,
		nil, // No broadcaster
	)
}

// Test cases

func TestRegenerateResponse_Execute_Success(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Hello, how are you?")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message to regenerate
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "I'm doing well!")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Configure mock LLM response
	llmService.chatResponse = &ports.LLMResponse{
		Content: "I'm doing great, thanks for asking!",
	}

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.DeletedMessageID != "asst_msg_1" {
		t.Errorf("expected DeletedMessageID asst_msg_1, got %s", output.DeletedMessageID)
	}

	if output.NewMessage == nil {
		t.Fatal("expected NewMessage to be set")
	}

	if output.NewMessage.Contents != "I'm doing great, thanks for asking!" {
		t.Errorf("expected new content, got %s", output.NewMessage.Contents)
	}

	// Verify old message was deleted
	deletedMsg, _ := msgRepo.GetByID(context.Background(), "asst_msg_1")
	if deletedMsg != nil {
		t.Error("expected old message to be deleted")
	}
}

func TestRegenerateResponse_Execute_MessageNotFound(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "nonexistent_msg",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	_, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when message not found")
	}

	if !stringContains(err.Error(), "message not found") {
		t.Errorf("expected 'message not found' error, got: %v", err)
	}
}

func TestRegenerateResponse_Execute_NotAssistantMessage(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create a user message (not assistant)
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "user_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	_, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when target is not assistant message")
	}

	if !stringContains(err.Error(), "not an assistant message") {
		t.Errorf("expected 'not an assistant message' error, got: %v", err)
	}
}

func TestRegenerateResponse_Execute_NoPreviousMessage(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create assistant message without PreviousID
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 0, models.MessageRoleAssistant, "Hello!")
	// Note: PreviousID is empty string by default
	msgRepo.Create(context.Background(), assistantMsg)

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	_, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when no previous message reference")
	}

	if !stringContains(err.Error(), "no previous message reference") {
		t.Errorf("expected 'no previous message reference' error, got: %v", err)
	}
}

func TestRegenerateResponse_Execute_PreviousMessageNotFound(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create assistant message with PreviousID pointing to non-existent message
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "Hello!")
	assistantMsg.PreviousID = "deleted_user_msg"
	msgRepo.Create(context.Background(), assistantMsg)

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	_, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when previous user message not found")
	}

	if !stringContains(err.Error(), "previous user message not found") {
		t.Errorf("expected 'previous user message not found' error, got: %v", err)
	}
}

func TestRegenerateResponse_Execute_PreviousMessageNotUserRole(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create a system message as the previous message
	systemMsg := models.NewMessage("system_msg_1", "conv_123", 0, models.MessageRoleSystem, "You are a helpful assistant.")
	msgRepo.Create(context.Background(), systemMsg)

	// Create assistant message with PreviousID pointing to system message
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "Hello!")
	assistantMsg.PreviousID = "system_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	_, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when previous message is not user role")
	}

	if !stringContains(err.Error(), "previous message is not a user message") {
		t.Errorf("expected 'previous message is not a user message' error, got: %v", err)
	}
}

func TestRegenerateResponse_Execute_DeleteFails(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "Hi there!")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Configure delete to fail
	msgRepo.deleteErr = errors.New("database connection lost")

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	_, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when delete fails")
	}

	if !stringContains(err.Error(), "failed to delete target message") {
		t.Errorf("expected 'failed to delete target message' error, got: %v", err)
	}
}

func TestRegenerateResponse_Execute_GenerateResponseFails(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "Hi there!")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Configure LLM to fail
	llmService.chatError = errors.New("LLM service unavailable")

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	_, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when generate fails")
	}

	if !stringContains(err.Error(), "failed to generate new response") {
		t.Errorf("expected 'failed to generate new response' error, got: %v", err)
	}

	// Verify old message was still deleted (operation already happened)
	deletedMsg, _ := msgRepo.GetByID(context.Background(), "asst_msg_1")
	if deletedMsg != nil {
		t.Error("expected old message to be deleted even when generation fails")
	}
}

func TestRegenerateResponse_Execute_Streaming(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Hello, how are you?")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message to regenerate
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "I'm doing well!")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Create a stream channel with test data
	streamCh := make(chan ports.LLMStreamChunk, 10)
	llmService.streamChannel = streamCh

	go func() {
		defer close(streamCh)
		streamCh <- ports.LLMStreamChunk{Content: "I'm ", Done: false}
		streamCh <- ports.LLMStreamChunk{Content: "doing ", Done: false}
		streamCh <- ports.LLMStreamChunk{Content: "great!", Done: false}
		streamCh <- ports.LLMStreamChunk{Done: true}
	}()

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: true,
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.StreamChannel == nil {
		t.Fatal("expected StreamChannel to be set")
	}

	if output.DeletedMessageID != "asst_msg_1" {
		t.Errorf("expected DeletedMessageID asst_msg_1, got %s", output.DeletedMessageID)
	}

	// Consume stream chunks
	var chunks []string
	for chunk := range output.StreamChannel {
		if chunk.Error != nil {
			t.Fatalf("unexpected stream error: %v", chunk.Error)
		}
		if chunk.Text != "" {
			chunks = append(chunks, chunk.Text)
		}
	}

	if len(chunks) == 0 {
		t.Error("expected to receive some stream chunks")
	}
}

func TestRegenerateResponse_Execute_WithToolsEnabled(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "What is 2+2?")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "4")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Configure LLM response
	llmService.chatResponse = &ports.LLMResponse{
		Content: "The answer is 4",
	}

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     true,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.NewMessage.Contents != "The answer is 4" {
		t.Errorf("expected 'The answer is 4', got %s", output.NewMessage.Contents)
	}
}

func TestRegenerateResponse_Execute_WithReasoningEnabled(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "What is 2+2?")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "4")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Configure LLM response with reasoning
	llmService.chatResponse = &ports.LLMResponse{
		Content:   "The answer is 4",
		Reasoning: "Let me calculate 2+2...",
	}

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: true,
		EnableStreaming: false,
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.NewMessage.Contents != "The answer is 4" {
		t.Errorf("expected 'The answer is 4', got %s", output.NewMessage.Contents)
	}
}

func TestRegenerateResponse_Execute_GetByIDError(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Configure GetByID to return an error
	msgRepo.getByIDErr = errors.New("database error")

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	_, err := uc.Execute(context.Background(), input)

	// Verify
	if err == nil {
		t.Fatal("expected error when GetByID fails")
	}

	if !stringContains(err.Error(), "failed to get target message") {
		t.Errorf("expected 'failed to get target message' error, got: %v", err)
	}
}

func TestRegenerateResponse_Execute_PreservesConversationID(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	conversationID := "conv_specific_123"

	// Create conversation
	conv := models.NewConversation(conversationID, "", "")
	convRepo.Create(context.Background(), conv)

	// Create user message
	userMsg := models.NewMessage("user_msg_1", conversationID, 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message
	assistantMsg := models.NewMessage("asst_msg_1", conversationID, 1, models.MessageRoleAssistant, "Hi!")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Configure LLM response
	llmService.chatResponse = &ports.LLMResponse{
		Content: "Hello there!",
	}

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.NewMessage.ConversationID != conversationID {
		t.Errorf("expected ConversationID %s, got %s", conversationID, output.NewMessage.ConversationID)
	}
}

func TestRegenerateResponse_Execute_GeneratesNewMessageID(t *testing.T) {
	// Setup
	msgRepo := newMockMessageRepoForRegenerate()
	convRepo := newMockConversationRepoForRegenerate()
	llmService := newMockLLMServiceForRegenerate()
	idGen := newMockIDGeneratorForRegenerate()

	// Create conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message
	assistantMsg := models.NewMessage("asst_msg_1", "conv_123", 1, models.MessageRoleAssistant, "Hi!")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Configure LLM response
	llmService.chatResponse = &ports.LLMResponse{
		Content: "New response",
	}

	generateUC := createGenerateResponseForRegenerate(msgRepo, convRepo, llmService, idGen)
	uc := NewRegenerateResponse(msgRepo, convRepo, generateUC, idGen)

	input := &ports.RegenerateResponseInput{
		MessageID:       "asst_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify new ID is different from old ID
	if output.NewMessage.ID == output.DeletedMessageID {
		t.Error("expected new message ID to be different from deleted message ID")
	}

	// Verify new message ID was generated (should start with msg_new_)
	if output.NewMessage.ID == "" {
		t.Error("expected new message ID to be set")
	}
}

// stringContains is a helper function to check if a string contains a substring
// Named uniquely to avoid conflicts with other test files in the same package
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
