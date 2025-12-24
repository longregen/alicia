package livekit

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

// Mock implementations for testing message handlers

type mockMessageRepo struct {
	store           map[string]*models.Message
	sequenceNumbers map[string]int
}

func newMockMessageRepo() *mockMessageRepo {
	return &mockMessageRepo{
		store:           make(map[string]*models.Message),
		sequenceNumbers: make(map[string]int),
	}
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	m.store[msg.ID] = msg
	return nil
}

func (m *mockMessageRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	if msg, ok := m.store[id]; ok {
		return msg, nil
	}
	return nil, errors.New("not found")
}

func (m *mockMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	if _, ok := m.store[msg.ID]; !ok {
		return errors.New("not found")
	}
	m.store[msg.ID] = msg
	return nil
}

func (m *mockMessageRepo) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	messages := []*models.Message{}
	for _, msg := range m.store {
		if msg.ConversationID == conversationID {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

func (m *mockMessageRepo) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	num := m.sequenceNumbers[conversationID]
	m.sequenceNumbers[conversationID] = num + 1
	return num, nil
}

func (m *mockMessageRepo) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

func (m *mockMessageRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, errors.New("not found")
}

func (m *mockMessageRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

type mockConversationRepo struct {
	store map[string]*models.Conversation
}

func newMockConversationRepo() *mockConversationRepo {
	return &mockConversationRepo{
		store: make(map[string]*models.Conversation),
	}
}

func (m *mockConversationRepo) Create(ctx context.Context, conv *models.Conversation) error {
	m.store[conv.ID] = conv
	return nil
}

func (m *mockConversationRepo) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	if conv, ok := m.store[id]; ok {
		return conv, nil
	}
	return nil, errors.New("not found")
}

func (m *mockConversationRepo) Update(ctx context.Context, conv *models.Conversation) error {
	if _, ok := m.store[conv.ID]; !ok {
		return errors.New("not found")
	}
	m.store[conv.ID] = conv
	return nil
}

func (m *mockConversationRepo) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

func (m *mockConversationRepo) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepo) UpdateStanzaIDs(ctx context.Context, conversationID string, lastClientStanzaID, lastServerStanzaID int32) error {
	if conv, ok := m.store[conversationID]; ok {
		if lastClientStanzaID != 0 {
			conv.LastClientStanzaID = lastClientStanzaID
		}
		if lastServerStanzaID != 0 {
			conv.LastServerStanzaID = lastServerStanzaID
		}
	}
	return nil
}

func (m *mockConversationRepo) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	return nil, errors.New("not found")
}

func (m *mockConversationRepo) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepo) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	delete(m.store, id)
	return nil
}

func (m *mockConversationRepo) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	if conv, ok := m.store[id]; ok {
		return conv, nil
	}
	return nil, errors.New("not found")
}

func (m *mockConversationRepo) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepo) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

type mockSentenceRepo struct{}

func (m *mockSentenceRepo) Create(ctx context.Context, sentence *models.Sentence) error {
	return nil
}

func (m *mockSentenceRepo) GetByID(ctx context.Context, id string) (*models.Sentence, error) {
	return nil, errors.New("not found")
}

func (m *mockSentenceRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.Sentence, error) {
	return []*models.Sentence{}, nil
}

func (m *mockSentenceRepo) Update(ctx context.Context, sentence *models.Sentence) error {
	return nil
}

func (m *mockSentenceRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockSentenceRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Sentence, error) {
	return []*models.Sentence{}, nil
}

func (m *mockSentenceRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Sentence, error) {
	return []*models.Sentence{}, nil
}

func (m *mockSentenceRepo) GetNextSequenceNumber(ctx context.Context, messageID string) (int, error) {
	return 0, nil
}

type mockReasoningStepRepo struct{}

func (m *mockReasoningStepRepo) Create(ctx context.Context, step *models.ReasoningStep) error {
	return nil
}

func (m *mockReasoningStepRepo) GetByID(ctx context.Context, id string) (*models.ReasoningStep, error) {
	return nil, errors.New("not found")
}

func (m *mockReasoningStepRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.ReasoningStep, error) {
	return []*models.ReasoningStep{}, nil
}

func (m *mockReasoningStepRepo) Update(ctx context.Context, step *models.ReasoningStep) error {
	return nil
}

func (m *mockReasoningStepRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockReasoningStepRepo) GetNextSequenceNumber(ctx context.Context, messageID string) (int, error) {
	return 0, nil
}

type mockToolUseRepo struct{}

func (m *mockToolUseRepo) Create(ctx context.Context, toolUse *models.ToolUse) error {
	return nil
}

func (m *mockToolUseRepo) GetByID(ctx context.Context, id string) (*models.ToolUse, error) {
	return nil, errors.New("not found")
}

func (m *mockToolUseRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	return []*models.ToolUse{}, nil
}

func (m *mockToolUseRepo) Update(ctx context.Context, toolUse *models.ToolUse) error {
	return nil
}

func (m *mockToolUseRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockToolUseRepo) GetPending(ctx context.Context, maxAge int) ([]*models.ToolUse, error) {
	return []*models.ToolUse{}, nil
}

type mockMemoryUsageRepo struct{}

func (m *mockMemoryUsageRepo) Create(ctx context.Context, usage *models.MemoryUsage) error {
	return nil
}

func (m *mockMemoryUsageRepo) GetByID(ctx context.Context, id string) (*models.MemoryUsage, error) {
	return nil, errors.New("not found")
}

func (m *mockMemoryUsageRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryUsageRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMemoryUsageRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryUsageRepo) GetByMemory(ctx context.Context, memoryID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryUsageRepo) GetUsageStats(ctx context.Context, memoryID string) (*ports.MemoryUsageStats, error) {
	return &ports.MemoryUsageStats{}, nil
}

type mockCommentaryRepo struct{}

func (m *mockCommentaryRepo) Create(ctx context.Context, commentary *models.Commentary) error {
	return nil
}

func (m *mockCommentaryRepo) GetByID(ctx context.Context, id string) (*models.Commentary, error) {
	return nil, errors.New("not found")
}

func (m *mockCommentaryRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.Commentary, error) {
	return []*models.Commentary{}, nil
}

func (m *mockCommentaryRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockCommentaryRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.Commentary, error) {
	return []*models.Commentary{}, nil
}

// mockAgentSender is a mock implementation of AgentSender for testing
type mockAgentSender struct {
	sentData     [][]byte
	sentAudio    []audioData
	mu           sync.Mutex
	sendDataFunc func(ctx context.Context, data []byte) error
}

type audioData struct {
	data   []byte
	format string
}

func newMockAgentSender() *mockAgentSender {
	return &mockAgentSender{
		sentData:  [][]byte{},
		sentAudio: []audioData{},
	}
}

func (m *mockAgentSender) SendData(ctx context.Context, data []byte) error {
	if m.sendDataFunc != nil {
		return m.sendDataFunc(ctx, data)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentData = append(m.sentData, data)
	return nil
}

func (m *mockAgentSender) SendAudio(ctx context.Context, audio []byte, format string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentAudio = append(m.sentAudio, audioData{data: audio, format: format})
	return nil
}

type mockIDGenerator struct {
	messageCounter  int
	sentenceCounter int
	toolUseCounter  int
}

func newMockIDGenerator() *mockIDGenerator {
	return &mockIDGenerator{}
}

func (m *mockIDGenerator) GenerateConversationID() string {
	return "conv_test1"
}

func (m *mockIDGenerator) GenerateMessageID() string {
	m.messageCounter++
	return "msg_test" + string(rune('0'+m.messageCounter))
}

func (m *mockIDGenerator) GenerateSentenceID() string {
	m.sentenceCounter++
	return "sent_test" + string(rune('0'+m.sentenceCounter))
}

func (m *mockIDGenerator) GenerateAudioID() string {
	return "audio_test1"
}

func (m *mockIDGenerator) GenerateMemoryID() string {
	return "mem_test1"
}

func (m *mockIDGenerator) GenerateMemoryUsageID() string {
	return "mu_test1"
}

func (m *mockIDGenerator) GenerateToolID() string {
	return "tool_test1"
}

func (m *mockIDGenerator) GenerateToolUseID() string {
	m.toolUseCounter++
	return "tu_test" + string(rune('0'+m.toolUseCounter))
}

func (m *mockIDGenerator) GenerateReasoningStepID() string {
	return "rs_test1"
}

func (m *mockIDGenerator) GenerateCommentaryID() string {
	return "comm_test1"
}

func (m *mockIDGenerator) GenerateMetaID() string {
	return "meta_test1"
}

func (m *mockIDGenerator) GenerateMCPServerID() string {
	return "amcp_test1"
}

type mockProtocolHandler struct {
	handleConfigurationFunc func(ctx context.Context, config *protocol.Configuration) error
	sendEnvelopeFunc        func(ctx context.Context, envelope *protocol.Envelope) error
	sendAudioFunc           func(ctx context.Context, audio []byte, format string) error
	sentEnvelopes           []*protocol.Envelope
	sentAudio               [][]byte
}

func newMockProtocolHandler() *mockProtocolHandler {
	return &mockProtocolHandler{
		sentEnvelopes: []*protocol.Envelope{},
		sentAudio:     [][]byte{},
	}
}

func (m *mockProtocolHandler) HandleConfiguration(ctx context.Context, config *protocol.Configuration) error {
	if m.handleConfigurationFunc != nil {
		return m.handleConfigurationFunc(ctx, config)
	}
	return nil
}

func (m *mockProtocolHandler) SendEnvelope(ctx context.Context, envelope *protocol.Envelope) error {
	m.sentEnvelopes = append(m.sentEnvelopes, envelope)
	if m.sendEnvelopeFunc != nil {
		return m.sendEnvelopeFunc(ctx, envelope)
	}
	return nil
}

func (m *mockProtocolHandler) SendAudio(ctx context.Context, audio []byte, format string) error {
	m.sentAudio = append(m.sentAudio, audio)
	if m.sendAudioFunc != nil {
		return m.sendAudioFunc(ctx, audio, format)
	}
	return nil
}

type mockProcessUserMessageUseCase struct {
	executeFunc func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error)
}

func (m *mockProcessUserMessageUseCase) Execute(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return &ports.ProcessUserMessageOutput{
		Message: &models.Message{
			ID:             "msg_123",
			ConversationID: input.ConversationID,
			Contents:       input.TextContent,
			Role:           models.MessageRoleUser,
			SequenceNumber: 1,
		},
		RelevantMemories: []*models.Memory{},
	}, nil
}

type mockGenerateResponseUseCase struct {
	executeFunc func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error)
}

func (m *mockGenerateResponseUseCase) Execute(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return &ports.GenerateResponseOutput{
		Message: &models.Message{
			ID:             "msg_response_123",
			ConversationID: input.ConversationID,
			Contents:       "Test response",
			Role:           models.MessageRoleAssistant,
		},
	}, nil
}

type mockHandleToolUseCase struct {
	executeFunc func(ctx context.Context, input *ports.HandleToolInput) (*ports.HandleToolOutput, error)
}

func (m *mockHandleToolUseCase) Execute(ctx context.Context, input *ports.HandleToolInput) (*ports.HandleToolOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return &ports.HandleToolOutput{
		ToolUseID: input.ToolUseID,
		Success:   true,
		Result:    "tool result",
	}, nil
}

type mockResponseGenerationManager struct {
	registerGenerationFunc   func(messageID string, cancelFunc context.CancelFunc)
	unregisterGenerationFunc func(messageID string)
	cancelGenerationFunc     func(targetID string) error
	registerTTSFunc          func(targetID string, cancelFunc context.CancelFunc)
	unregisterTTSFunc        func(targetID string)
	cancelTTSFunc            func(targetID string) error
}

func newMockResponseGenerationManager() *mockResponseGenerationManager {
	return &mockResponseGenerationManager{}
}

func (m *mockResponseGenerationManager) RegisterGeneration(messageID string, cancelFunc context.CancelFunc) {
	if m.registerGenerationFunc != nil {
		m.registerGenerationFunc(messageID, cancelFunc)
	}
}

func (m *mockResponseGenerationManager) UnregisterGeneration(messageID string) {
	if m.unregisterGenerationFunc != nil {
		m.unregisterGenerationFunc(messageID)
	}
}

func (m *mockResponseGenerationManager) CancelGeneration(targetID string) error {
	if m.cancelGenerationFunc != nil {
		return m.cancelGenerationFunc(targetID)
	}
	return nil
}

func (m *mockResponseGenerationManager) RegisterTTS(targetID string, cancelFunc context.CancelFunc) {
	if m.registerTTSFunc != nil {
		m.registerTTSFunc(targetID, cancelFunc)
	}
}

func (m *mockResponseGenerationManager) UnregisterTTS(targetID string) {
	if m.unregisterTTSFunc != nil {
		m.unregisterTTSFunc(targetID)
	}
}

func (m *mockResponseGenerationManager) CancelTTS(targetID string) error {
	if m.cancelTTSFunc != nil {
		return m.cancelTTSFunc(targetID)
	}
	return nil
}

func (m *mockResponseGenerationManager) CleanupStaleGenerations(maxAge time.Duration) int {
	return 0
}

// Helper to create a test dispatcher with mocks
func createTestDispatcher() (*DefaultMessageDispatcher, *mockProtocolHandler, *mockMessageRepo) {
	mockProtocol := newMockProtocolHandler()
	messageRepo := newMockMessageRepo()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	handleToolUseCase := &mockHandleToolUseCase{}
	idGenerator := newMockIDGenerator()
	generationManager := newMockResponseGenerationManager()

	// Create a test conversation
	conversationID := "conv_test1"
	now := time.Now()
	conversation := &models.Conversation{
		ID:                 conversationID,
		Title:              "Test Conversation",
		Status:             models.ConversationStatusActive,
		LastClientStanzaID: 0,
		LastServerStanzaID: -1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	conversationRepo.Create(context.Background(), conversation)

	// Create a mock agent sender that always succeeds
	mockSender := newMockAgentSender()

	// Create ProtocolHandler using the mock sender
	protocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	dispatcher := &DefaultMessageDispatcher{
		protocolHandler:           protocolHandler,
		messageRepo:               messageRepo,
		processUserMessageUseCase: processUserMessageUseCase,
		generateResponseUseCase:   generateResponseUseCase,
		handleToolUseCase:         handleToolUseCase,
		idGenerator:               idGenerator,
		generationManager:         generationManager,
		conversationID:            conversationID,
	}

	return dispatcher, mockProtocol, messageRepo
}

// Tests for handleConfiguration

func TestHandleConfiguration_Success(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	envelope := &protocol.Envelope{
		Type: protocol.TypeConfiguration,
		Body: &protocol.Configuration{
			ConversationID:   "conv_test1",
			LastSequenceSeen: 0,
		},
	}

	err := dispatcher.handleConfiguration(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleConfiguration_InvalidMessageType(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	envelope := &protocol.Envelope{
		Type: protocol.TypeConfiguration,
		Body: &protocol.UserMessage{}, // Wrong type
	}

	err := dispatcher.handleConfiguration(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error for invalid message type, got nil")
	}
}

// Tests for handleUserMessage

func TestHandleUserMessage_Success(t *testing.T) {
	dispatcher, _, messageRepo := createTestDispatcher()

	userMsg := &protocol.UserMessage{
		ID:             "user_msg_1",
		ConversationID: "conv_test1",
		Content:        "Hello, world!",
		PreviousID:     "",
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeUserMessage,
		Body: userMsg,
	}

	err := dispatcher.handleUserMessage(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify message was processed
	msg, err := messageRepo.GetByID(context.Background(), "msg_123")
	if err == nil && msg != nil {
		if msg.Contents != "Hello, world!" {
			t.Errorf("expected content 'Hello, world!', got %s", msg.Contents)
		}
	}
}

func TestHandleUserMessage_ConversationMismatch(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	userMsg := &protocol.UserMessage{
		ID:             "user_msg_1",
		ConversationID: "wrong_conv_id",
		Content:        "Test message",
		PreviousID:     "",
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeUserMessage,
		Body: userMsg,
	}

	err := dispatcher.handleUserMessage(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error for conversation mismatch, got nil")
	}
}

func TestHandleUserMessage_InvalidMessageType(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	envelope := &protocol.Envelope{
		Type: protocol.TypeUserMessage,
		Body: &protocol.Configuration{}, // Wrong type
	}

	err := dispatcher.handleUserMessage(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error for invalid message type, got nil")
	}
}

func TestHandleUserMessage_ProcessUseCaseError(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	// Override process use case to return error
	dispatcher.processUserMessageUseCase = &mockProcessUserMessageUseCase{
		executeFunc: func(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
			return nil, errors.New("processing failed")
		},
	}

	userMsg := &protocol.UserMessage{
		ID:             "user_msg_1",
		ConversationID: "conv_test1",
		Content:        "Test message",
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeUserMessage,
		Body: userMsg,
	}

	err := dispatcher.handleUserMessage(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error from processing failure, got nil")
	}
}

// Tests for handleControlStop

func TestHandleControlStop_Success(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	cancelCalled := false
	dispatcher.generationManager = &mockResponseGenerationManager{
		cancelGenerationFunc: func(targetID string) error {
			cancelCalled = true
			if targetID != "msg_target_123" {
				t.Errorf("expected target ID msg_target_123, got %s", targetID)
			}
			return nil
		},
	}

	stopMsg := &protocol.ControlStop{
		ConversationID: "conv_test1",
		StopType:       protocol.StopTypeGeneration,
		TargetID:       "msg_target_123",
		Reason:         "User requested stop",
	}

	envelope := &protocol.Envelope{
		Type:     protocol.TypeControlStop,
		Body:     stopMsg,
		StanzaID: 5,
	}

	err := dispatcher.handleControlStop(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cancelCalled {
		t.Error("expected cancel generation to be called")
	}
}

func TestHandleControlStop_ConversationMismatch(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	stopMsg := &protocol.ControlStop{
		ConversationID: "wrong_conv_id",
		StopType:       protocol.StopTypeGeneration,
		TargetID:       "msg_target_123",
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeControlStop,
		Body: stopMsg,
	}

	err := dispatcher.handleControlStop(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error for conversation mismatch, got nil")
	}
}

func TestHandleControlStop_InvalidMessageType(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	envelope := &protocol.Envelope{
		Type: protocol.TypeControlStop,
		Body: &protocol.UserMessage{}, // Wrong type
	}

	err := dispatcher.handleControlStop(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error for invalid message type, got nil")
	}
}

func TestHandleControlStop_StopTypeSpeech(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	ttsCancelCalled := false
	dispatcher.generationManager = &mockResponseGenerationManager{
		cancelTTSFunc: func(targetID string) error {
			ttsCancelCalled = true
			if targetID != "msg_target_123" {
				t.Errorf("expected target ID msg_target_123, got %s", targetID)
			}
			return nil
		},
	}

	stopMsg := &protocol.ControlStop{
		ConversationID: "conv_test1",
		StopType:       protocol.StopTypeSpeech,
		TargetID:       "msg_target_123",
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeControlStop,
		Body: stopMsg,
	}

	err := dispatcher.handleControlStop(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ttsCancelCalled {
		t.Error("expected cancel TTS to be called")
	}
}

func TestHandleControlStop_StopTypeAll(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	generationCancelCalled := false
	ttsCancelCalled := false

	dispatcher.generationManager = &mockResponseGenerationManager{
		cancelGenerationFunc: func(targetID string) error {
			generationCancelCalled = true
			return nil
		},
		cancelTTSFunc: func(targetID string) error {
			ttsCancelCalled = true
			return nil
		},
	}

	stopMsg := &protocol.ControlStop{
		ConversationID: "conv_test1",
		StopType:       protocol.StopTypeAll,
		TargetID:       "msg_target_123",
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeControlStop,
		Body: stopMsg,
	}

	err := dispatcher.handleControlStop(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !generationCancelCalled {
		t.Error("expected cancel generation to be called")
	}

	if !ttsCancelCalled {
		t.Error("expected cancel TTS to be called")
	}
}

func TestHandleControlStop_UnknownStopType(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	stopMsg := &protocol.ControlStop{
		ConversationID: "conv_test1",
		StopType:       "unknown_type",
		TargetID:       "msg_target_123",
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeControlStop,
		Body: stopMsg,
	}

	err := dispatcher.handleControlStop(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error for unknown stop type, got nil")
	}
}

// Tests for handleAudioChunk

func TestHandleAudioData_Processing(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	// Mock ASR service
	mockASR := &mockASRService{
		transcribeFunc: func(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
			return &ports.ASRResult{
				Text:       "Hello world",
				Confidence: 0.95,
				Language:   "en-US",
			}, nil
		},
	}
	dispatcher.asrService = mockASR

	audioChunk := &protocol.AudioChunk{
		ConversationID: "conv_test1",
		Sequence:       1,
		Data:           []byte{0x01, 0x02, 0x03, 0x04},
		Format:         "pcm",
		DurationMs:     100,
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeAudioChunk,
		Body: audioChunk,
	}

	err := dispatcher.handleAudioChunk(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleAudioChunk_ASRError(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	mockASR := &mockASRService{
		transcribeFunc: func(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
			return nil, errors.New("ASR service failed")
		},
	}
	dispatcher.asrService = mockASR

	audioChunk := &protocol.AudioChunk{
		ConversationID: "conv_test1",
		Sequence:       1,
		Data:           []byte{0x01, 0x02},
		Format:         "pcm",
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeAudioChunk,
		Body: audioChunk,
	}

	err := dispatcher.handleAudioChunk(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error from ASR failure, got nil")
	}
}

func TestHandleAudioChunk_InvalidMessageType(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	envelope := &protocol.Envelope{
		Type: protocol.TypeAudioChunk,
		Body: &protocol.UserMessage{}, // Wrong type
	}

	err := dispatcher.handleAudioChunk(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error for invalid message type, got nil")
	}
}

// Tests for handleToolUseRequest

func TestHandleToolUseRequest_Success(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	toolReq := &protocol.ToolUseRequest{
		ID:             "tu_req_1",
		MessageID:      "msg_test1",
		ConversationID: "conv_test1",
		ToolName:       "calculator",
		Parameters:     map[string]any{"expression": "2+2"},
		TimeoutMs:      5000,
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeToolUseRequest,
		Body: toolReq,
	}

	err := dispatcher.handleToolUseRequest(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleToolUseRequest_ConversationMismatch(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	toolReq := &protocol.ToolUseRequest{
		ID:             "tu_req_1",
		MessageID:      "msg_test1",
		ConversationID: "wrong_conv_id",
		ToolName:       "calculator",
		Parameters:     map[string]any{},
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeToolUseRequest,
		Body: toolReq,
	}

	err := dispatcher.handleToolUseRequest(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error for conversation mismatch, got nil")
	}
}

func TestHandleToolUseRequest_ExecutionError(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	dispatcher.handleToolUseCase = &mockHandleToolUseCase{
		executeFunc: func(ctx context.Context, input *ports.HandleToolInput) (*ports.HandleToolOutput, error) {
			return nil, errors.New("tool execution failed")
		},
	}

	toolReq := &protocol.ToolUseRequest{
		ID:             "tu_req_1",
		MessageID:      "msg_test1",
		ConversationID: "conv_test1",
		ToolName:       "failing_tool",
		Parameters:     map[string]any{},
	}

	envelope := &protocol.Envelope{
		Type: protocol.TypeToolUseRequest,
		Body: toolReq,
	}

	err := dispatcher.handleToolUseRequest(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Mock ASR service for testing
type mockASRService struct {
	transcribeFunc func(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error)
}

func (m *mockASRService) Transcribe(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
	if m.transcribeFunc != nil {
		return m.transcribeFunc(ctx, audio, format)
	}
	return &ports.ASRResult{
		Text:       "test transcription",
		Confidence: 0.9,
		Language:   "en-US",
	}, nil
}

func (m *mockASRService) TranscribeStream(ctx context.Context, audioStream io.Reader, format string) (<-chan *ports.ASRResult, error) {
	// Not implemented for tests
	return nil, nil
}

// Tests for message dispatcher routing

func TestDispatchMessage_RouteToCorrectHandler(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	tests := []struct {
		name        string
		messageType protocol.MessageType
		body        any
		expectError bool
		errorCode   int32
	}{
		{
			name:        "Configuration message",
			messageType: protocol.TypeConfiguration,
			body: &protocol.Configuration{
				ConversationID:   "conv_test1",
				LastSequenceSeen: 0,
			},
			expectError: false,
		},
		{
			name:        "UserMessage",
			messageType: protocol.TypeUserMessage,
			body: &protocol.UserMessage{
				ID:             "user_msg_1",
				ConversationID: "conv_test1",
				Content:        "Test",
			},
			expectError: false,
		},
		{
			name:        "ControlStop",
			messageType: protocol.TypeControlStop,
			body: &protocol.ControlStop{
				ConversationID: "conv_test1",
				StopType:       protocol.StopTypeAll,
			},
			expectError: false,
		},
		{
			name:        "Invalid conversation ID",
			messageType: protocol.TypeUserMessage,
			body: &protocol.UserMessage{
				ID:             "user_msg_1",
				ConversationID: "wrong_conv",
				Content:        "Test",
			},
			expectError: true,
			errorCode:   protocol.ErrCodeConversationNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope := &protocol.Envelope{
				Type: tt.messageType,
				Body: tt.body,
			}

			err := dispatcher.DispatchMessage(context.Background(), envelope)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
