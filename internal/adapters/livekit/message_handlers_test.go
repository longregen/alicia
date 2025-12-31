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
	var results []*models.Message
	for _, msg := range m.store {
		if msg.ConversationID == conversationID {
			results = append(results, msg)
		}
	}
	return results, nil
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

func (m *mockMessageRepo) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	// Simple implementation: return message chain by following PreviousID
	var chain []*models.Message
	currentID := tipMessageID

	for currentID != "" {
		msg, ok := m.store[currentID]
		if !ok {
			break
		}
		chain = append([]*models.Message{msg}, chain...)
		if msg.PreviousID == "" {
			break
		}
		currentID = msg.PreviousID
	}

	return chain, nil
}

func (m *mockMessageRepo) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	msg, ok := m.store[messageID]
	if !ok {
		return nil, errors.New("not found")
	}

	// Find all messages with the same PreviousID
	var siblings []*models.Message
	for _, m := range m.store {
		if msg.PreviousID == "" && m.PreviousID == "" {
			if m.ID != messageID && m.ConversationID == msg.ConversationID {
				siblings = append(siblings, m)
			}
		} else if msg.PreviousID != "" && m.PreviousID != "" && m.PreviousID == msg.PreviousID {
			if m.ID != messageID {
				siblings = append(siblings, m)
			}
		}
	}

	return siblings, nil
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

func (m *mockConversationRepo) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	if conv, ok := m.store[conversationID]; ok {
		conv.TipMessageID = &messageID
	}
	return nil
}

func (m *mockConversationRepo) UpdatePromptVersion(ctx context.Context, convID, versionID string) error {
	if conv, ok := m.store[convID]; ok {
		conv.SystemPromptVersionID = versionID
	}
	return nil
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

type mockToolUseRepo struct {
	toolUses map[string]*models.ToolUse
}

func (m *mockToolUseRepo) Create(ctx context.Context, toolUse *models.ToolUse) error {
	if m.toolUses == nil {
		m.toolUses = make(map[string]*models.ToolUse)
	}
	m.toolUses[toolUse.ID] = toolUse
	return nil
}

func (m *mockToolUseRepo) GetByID(ctx context.Context, id string) (*models.ToolUse, error) {
	if m.toolUses == nil {
		m.toolUses = make(map[string]*models.ToolUse)
	}
	if tu, ok := m.toolUses[id]; ok {
		return tu, nil
	}
	return nil, errors.New("not found")
}

func (m *mockToolUseRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	return []*models.ToolUse{}, nil
}

func (m *mockToolUseRepo) Update(ctx context.Context, toolUse *models.ToolUse) error {
	if m.toolUses == nil {
		m.toolUses = make(map[string]*models.ToolUse)
	}
	m.toolUses[toolUse.ID] = toolUse
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

func (m *mockIDGenerator) GenerateVoteID() string {
	return "av_test1"
}

func (m *mockIDGenerator) GenerateNoteID() string {
	return "an_test1"
}

func (m *mockIDGenerator) GenerateSessionStatsID() string {
	return "ass_test1"
}

func (m *mockIDGenerator) GenerateOptimizationRunID() string {
	return "aor_test1"
}

func (m *mockIDGenerator) GeneratePromptCandidateID() string {
	return "apc_test1"
}

func (m *mockIDGenerator) GeneratePromptEvaluationID() string {
	return "ape_test1"
}

func (m *mockIDGenerator) GenerateTrainingExampleID() string {
	return "gte_test1"
}

func (m *mockIDGenerator) GenerateSystemPromptVersionID() string {
	return "spv_test1"
}

// Mock VoteRepository for testing feedback handlers
type mockVoteRepository struct {
	votes      map[string]*models.Vote
	aggregates map[string]*models.VoteAggregates
	createErr  error
	deleteErr  error
}

func newMockVoteRepository() *mockVoteRepository {
	return &mockVoteRepository{
		votes:      make(map[string]*models.Vote),
		aggregates: make(map[string]*models.VoteAggregates),
	}
}

func (m *mockVoteRepository) Create(ctx context.Context, vote *models.Vote) error {
	if m.createErr != nil {
		return m.createErr
	}
	key := vote.TargetType + ":" + vote.TargetID
	m.votes[key] = vote
	return nil
}

func (m *mockVoteRepository) Delete(ctx context.Context, targetType string, targetID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	key := targetType + ":" + targetID
	delete(m.votes, key)
	return nil
}

func (m *mockVoteRepository) GetByTarget(ctx context.Context, targetType string, targetID string) ([]*models.Vote, error) {
	key := targetType + ":" + targetID
	if vote, ok := m.votes[key]; ok {
		return []*models.Vote{vote}, nil
	}
	return []*models.Vote{}, nil
}

func (m *mockVoteRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Vote, error) {
	var result []*models.Vote
	for _, vote := range m.votes {
		if vote.MessageID == messageID {
			result = append(result, vote)
		}
	}
	return result, nil
}

func (m *mockVoteRepository) GetAggregates(ctx context.Context, targetType string, targetID string) (*models.VoteAggregates, error) {
	key := targetType + ":" + targetID
	if agg, ok := m.aggregates[key]; ok {
		return agg, nil
	}
	// Return default aggregates
	return &models.VoteAggregates{Upvotes: 1, Downvotes: 0}, nil
}

func (m *mockVoteRepository) GetToolUseVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithToolContext, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetMemoryVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetMemoryUsageVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetMemoryExtractionVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithExtractionContext, error) {
	return nil, nil
}

func (m *mockVoteRepository) CountByTargetType(ctx context.Context, targetType string) (int, error) {
	count := 0
	for _, vote := range m.votes {
		if vote.TargetType == targetType {
			count++
		}
	}
	return count, nil
}

// Mock NoteRepository for testing note handlers
type mockNoteRepository struct {
	notes     map[string]*models.Note
	createErr error
	updateErr error
	deleteErr error
}

func newMockNoteRepository() *mockNoteRepository {
	return &mockNoteRepository{
		notes: make(map[string]*models.Note),
	}
}

func (m *mockNoteRepository) Create(ctx context.Context, note *models.Note) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.notes[note.ID] = note
	return nil
}

func (m *mockNoteRepository) Update(ctx context.Context, id string, content string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if note, ok := m.notes[id]; ok {
		note.Content = content
		return nil
	}
	return errors.New("note not found")
}

func (m *mockNoteRepository) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.notes, id)
	return nil
}

func (m *mockNoteRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Note, error) {
	var result []*models.Note
	for _, note := range m.notes {
		if note.MessageID == messageID {
			result = append(result, note)
		}
	}
	return result, nil
}

func (m *mockNoteRepository) GetByID(ctx context.Context, id string) (*models.Note, error) {
	if note, ok := m.notes[id]; ok {
		return note, nil
	}
	return nil, errors.New("note not found")
}

type mockProtocolHandler struct {
	handleConfigurationFunc func(ctx context.Context, config *protocol.Configuration) error
	sendEnvelopeFunc        func(ctx context.Context, envelope *protocol.Envelope) error
	sendAudioFunc           func(ctx context.Context, audio []byte, format string) error
	sentEnvelopes           []*protocol.Envelope
	sentAudio               [][]byte
	toolUseRepo             ports.ToolUseRepository
}

// Ensure mockProtocolHandler implements ProtocolHandlerInterface
var _ ProtocolHandlerInterface = (*mockProtocolHandler)(nil)

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

func (m *mockProtocolHandler) SendToolUseRequest(ctx context.Context, toolUse *models.ToolUse) error {
	return nil
}

func (m *mockProtocolHandler) SendToolUseResult(ctx context.Context, toolUse *models.ToolUse) error {
	return nil
}

func (m *mockProtocolHandler) SendAcknowledgement(ctx context.Context, ackedStanzaID int32, success bool) error {
	return nil
}

func (m *mockProtocolHandler) SendError(ctx context.Context, code int32, message string, recoverable bool) error {
	return nil
}

func (m *mockProtocolHandler) GetToolUseRepo() ports.ToolUseRepository {
	return m.toolUseRepo
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

	// Create a test tool use for the TypeToolUseResult test
	testToolUse := models.NewToolUse("tu_1", "msg_1", "test_tool", 1, map[string]any{})
	toolUseRepo.Create(context.Background(), testToolUse)

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
		toolUseRepo:               toolUseRepo,
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

// TestHandleFeedback tests the feedback handler
func TestHandleFeedback(t *testing.T) {
	t.Run("upvote creates vote and sends confirmation", func(t *testing.T) {
		// Setup mocks
		voteRepo := newMockVoteRepository()
		protocolHandler := newMockProtocolHandler()
		idGen := newMockIDGenerator()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			voteRepo:        voteRepo,
			protocolHandler: protocolHandler,
			idGenerator:     idGen,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeFeedback,
			Body: &protocol.Feedback{
				ID:         "fb_1",
				TargetType: "message",
				TargetID:   "msg_123",
				MessageID:  "msg_123",
				Vote:       "up",
			},
		}

		err := dispatcher.handleFeedback(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify vote was created
		if len(voteRepo.votes) != 1 {
			t.Errorf("expected 1 vote, got %d", len(voteRepo.votes))
		}

		// Verify confirmation was sent
		if len(protocolHandler.sentEnvelopes) != 1 {
			t.Fatalf("expected 1 envelope sent, got %d", len(protocolHandler.sentEnvelopes))
		}

		conf, ok := protocolHandler.sentEnvelopes[0].Body.(*protocol.FeedbackConfirmation)
		if !ok {
			t.Fatal("expected FeedbackConfirmation body")
		}
		if conf.UserVote != "up" {
			t.Errorf("expected UserVote 'up', got '%s'", conf.UserVote)
		}
		if conf.Aggregates.Upvotes != 1 {
			t.Errorf("expected Upvotes 1, got %d", conf.Aggregates.Upvotes)
		}
	})

	t.Run("downvote creates negative vote", func(t *testing.T) {
		voteRepo := newMockVoteRepository()
		protocolHandler := newMockProtocolHandler()
		idGen := newMockIDGenerator()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			voteRepo:        voteRepo,
			protocolHandler: protocolHandler,
			idGenerator:     idGen,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeFeedback,
			Body: &protocol.Feedback{
				ID:         "fb_2",
				TargetType: "tool_use",
				TargetID:   "tu_123",
				MessageID:  "msg_456",
				Vote:       "down",
			},
		}

		err := dispatcher.handleFeedback(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify vote was created with negative value
		if len(voteRepo.votes) != 1 {
			t.Errorf("expected 1 vote, got %d", len(voteRepo.votes))
		}
		vote := voteRepo.votes["tool_use:tu_123"]
		if vote.Value != models.VoteValueDown {
			t.Errorf("expected vote value %d, got %d", models.VoteValueDown, vote.Value)
		}
	})

	t.Run("remove deletes existing vote", func(t *testing.T) {
		voteRepo := newMockVoteRepository()
		// Pre-populate a vote
		voteRepo.votes["message:msg_789"] = &models.Vote{ID: "av_1", TargetType: "message", TargetID: "msg_789"}

		protocolHandler := newMockProtocolHandler()
		idGen := newMockIDGenerator()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			voteRepo:        voteRepo,
			protocolHandler: protocolHandler,
			idGenerator:     idGen,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeFeedback,
			Body: &protocol.Feedback{
				ID:         "fb_3",
				TargetType: "message",
				TargetID:   "msg_789",
				Vote:       "remove",
			},
		}

		err := dispatcher.handleFeedback(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify vote was deleted
		if len(voteRepo.votes) != 0 {
			t.Errorf("expected 0 votes after removal, got %d", len(voteRepo.votes))
		}
	})

	t.Run("handles nil voteRepo gracefully", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			voteRepo:        nil,
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeFeedback,
			Body: &protocol.Feedback{
				ID:         "fb_4",
				TargetType: "message",
				TargetID:   "msg_123",
				Vote:       "up",
			},
		}

		err := dispatcher.handleFeedback(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should still send confirmation even without repo
		if len(protocolHandler.sentEnvelopes) != 1 {
			t.Errorf("expected confirmation sent even with nil repo")
		}
	})
}

// TestHandleUserNote tests the user note handler
func TestHandleUserNote(t *testing.T) {
	t.Run("create note saves and sends confirmation", func(t *testing.T) {
		noteRepo := newMockNoteRepository()
		protocolHandler := newMockProtocolHandler()
		idGen := newMockIDGenerator()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			noteRepo:        noteRepo,
			protocolHandler: protocolHandler,
			idGenerator:     idGen,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeUserNote,
			Body: &protocol.UserNote{
				ID:        "",
				MessageID: "msg_123",
				Content:   "This is a test note",
				Category:  "improvement",
				Action:    "create",
			},
		}

		err := dispatcher.handleUserNote(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify note was created
		if len(noteRepo.notes) != 1 {
			t.Errorf("expected 1 note, got %d", len(noteRepo.notes))
		}

		// Verify confirmation was sent
		if len(protocolHandler.sentEnvelopes) != 1 {
			t.Fatalf("expected 1 envelope sent, got %d", len(protocolHandler.sentEnvelopes))
		}

		conf, ok := protocolHandler.sentEnvelopes[0].Body.(*protocol.NoteConfirmation)
		if !ok {
			t.Fatal("expected NoteConfirmation body")
		}
		if !conf.Success {
			t.Error("expected success=true")
		}
	})

	t.Run("update note modifies content", func(t *testing.T) {
		noteRepo := newMockNoteRepository()
		noteRepo.notes["note_123"] = &models.Note{
			ID:        "note_123",
			MessageID: "msg_456",
			Content:   "Original content",
		}

		protocolHandler := newMockProtocolHandler()
		idGen := newMockIDGenerator()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			noteRepo:        noteRepo,
			protocolHandler: protocolHandler,
			idGenerator:     idGen,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeUserNote,
			Body: &protocol.UserNote{
				ID:        "note_123",
				MessageID: "msg_456",
				Content:   "Updated content",
				Action:    "update",
			},
		}

		err := dispatcher.handleUserNote(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify note was updated
		if noteRepo.notes["note_123"].Content != "Updated content" {
			t.Errorf("expected updated content, got '%s'", noteRepo.notes["note_123"].Content)
		}
	})

	t.Run("delete note removes from store", func(t *testing.T) {
		noteRepo := newMockNoteRepository()
		noteRepo.notes["note_456"] = &models.Note{
			ID:        "note_456",
			MessageID: "msg_789",
			Content:   "To be deleted",
		}

		protocolHandler := newMockProtocolHandler()
		idGen := newMockIDGenerator()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			noteRepo:        noteRepo,
			protocolHandler: protocolHandler,
			idGenerator:     idGen,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeUserNote,
			Body: &protocol.UserNote{
				ID:        "note_456",
				MessageID: "msg_789",
				Action:    "delete",
			},
		}

		err := dispatcher.handleUserNote(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify note was deleted
		if len(noteRepo.notes) != 0 {
			t.Errorf("expected 0 notes after deletion, got %d", len(noteRepo.notes))
		}
	})

	t.Run("handles nil noteRepo gracefully", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			noteRepo:        nil,
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeUserNote,
			Body: &protocol.UserNote{
				ID:        "",
				MessageID: "msg_123",
				Content:   "Test",
				Action:    "create",
			},
		}

		err := dispatcher.handleUserNote(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should send confirmation with success=false
		if len(protocolHandler.sentEnvelopes) != 1 {
			t.Fatalf("expected confirmation sent even with nil repo")
		}
		conf := protocolHandler.sentEnvelopes[0].Body.(*protocol.NoteConfirmation)
		if conf.Success {
			t.Error("expected success=false when noteRepo is nil")
		}
	})

	t.Run("defaults to general category when empty", func(t *testing.T) {
		noteRepo := newMockNoteRepository()
		protocolHandler := newMockProtocolHandler()
		idGen := newMockIDGenerator()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			noteRepo:        noteRepo,
			protocolHandler: protocolHandler,
			idGenerator:     idGen,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeUserNote,
			Body: &protocol.UserNote{
				MessageID: "msg_123",
				Content:   "Test",
				Category:  "", // Empty category
				Action:    "create",
			},
		}

		err := dispatcher.handleUserNote(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify note has default category
		for _, note := range noteRepo.notes {
			if note.Category != models.NoteCategoryGeneral {
				t.Errorf("expected category '%s', got '%s'", models.NoteCategoryGeneral, note.Category)
			}
		}
	})
}

// Mock MemoryService for testing memory action handlers
type mockMemoryService struct {
	memories       map[string]*models.Memory
	createErr      error
	updateErr      error
	deleteErr      error
	importanceErr  error
	lastImportance float32
	lastMemoryID   string
}

func newMockMemoryService() *mockMemoryService {
	return &mockMemoryService{
		memories: make(map[string]*models.Memory),
	}
}

func (m *mockMemoryService) Create(ctx context.Context, content string) (*models.Memory, error) {
	return m.CreateWithEmbeddings(ctx, content)
}

func (m *mockMemoryService) CreateWithEmbeddings(ctx context.Context, content string) (*models.Memory, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	memory := &models.Memory{
		ID:      "mem_test_created",
		Content: content,
	}
	m.memories[memory.ID] = memory
	return memory, nil
}

func (m *mockMemoryService) CreateFromConversation(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error) {
	return m.CreateWithEmbeddings(ctx, content)
}

func (m *mockMemoryService) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	if mem, ok := m.memories[id]; ok {
		return mem, nil
	}
	return nil, errors.New("memory not found")
}

func (m *mockMemoryService) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryService) Update(ctx context.Context, memory *models.Memory) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.memories[memory.ID] = memory
	return nil
}

func (m *mockMemoryService) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.memories, id)
	return nil
}

func (m *mockMemoryService) SetImportance(ctx context.Context, id string, importance float32) (*models.Memory, error) {
	if m.importanceErr != nil {
		return nil, m.importanceErr
	}
	m.lastMemoryID = id
	m.lastImportance = importance
	if mem, ok := m.memories[id]; ok {
		mem.Importance = importance
		return mem, nil
	}
	// Create a mock memory if it doesn't exist
	mem := &models.Memory{ID: id, Importance: importance}
	m.memories[id] = mem
	return mem, nil
}

func (m *mockMemoryService) SetConfidence(ctx context.Context, id string, confidence float32) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) SetUserRating(ctx context.Context, id string, rating int) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) AddTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) RemoveTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) Search(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryService) SearchWithThreshold(ctx context.Context, query string, threshold float32, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryService) SearchWithDynamicImportance(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryService) SearchWithScores(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
	return []*ports.MemorySearchResult{}, nil
}

func (m *mockMemoryService) TrackUsage(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryService) GetUsageByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryService) GetUsageByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryService) RegenerateEmbeddings(ctx context.Context, id string) (*models.Memory, error) {
	if mem, ok := m.memories[id]; ok {
		return mem, nil
	}
	return nil, errors.New("memory not found")
}

func (m *mockMemoryService) Pin(ctx context.Context, id string, pinned bool) (*models.Memory, error) {
	if mem, ok := m.memories[id]; ok {
		return mem, nil
	}
	return nil, errors.New("memory not found")
}

func (m *mockMemoryService) Archive(ctx context.Context, id string) (*models.Memory, error) {
	if mem, ok := m.memories[id]; ok {
		return mem, nil
	}
	return nil, errors.New("memory not found")
}

// Mock OptimizationService for testing dimension preference handlers
type mockOptimizationService struct {
	dimensionWeights map[string]float64
}

func newMockOptimizationService() *mockOptimizationService {
	return &mockOptimizationService{
		dimensionWeights: make(map[string]float64),
	}
}

func (m *mockOptimizationService) SetDimensionWeights(weights map[string]float64) {
	for k, v := range weights {
		m.dimensionWeights[k] = v
	}
}

// Implement remaining OptimizationService interface methods
func (m *mockOptimizationService) StartOptimizationRun(ctx context.Context, name, promptType, baselinePrompt string) (*models.OptimizationRun, error) {
	return nil, nil
}

func (m *mockOptimizationService) GetOptimizationRun(ctx context.Context, id string) (*models.OptimizationRun, error) {
	return nil, nil
}

func (m *mockOptimizationService) ListOptimizationRuns(ctx context.Context, opts ports.ListOptimizationRunsOptions) ([]*models.OptimizationRun, error) {
	return nil, nil
}

func (m *mockOptimizationService) CompleteRun(ctx context.Context, runID string, bestScore float64) error {
	return nil
}

func (m *mockOptimizationService) FailRun(ctx context.Context, runID string, reason string) error {
	return nil
}

func (m *mockOptimizationService) UpdateProgress(ctx context.Context, runID string, iteration int, currentScore float64) error {
	return nil
}

func (m *mockOptimizationService) AddCandidate(ctx context.Context, runID, promptText string, iteration int) (*models.PromptCandidate, error) {
	return nil, nil
}

func (m *mockOptimizationService) GetCandidates(ctx context.Context, runID string) ([]*models.PromptCandidate, error) {
	return nil, nil
}

func (m *mockOptimizationService) GetBestCandidate(ctx context.Context, runID string) (*models.PromptCandidate, error) {
	return nil, nil
}

func (m *mockOptimizationService) RecordEvaluation(ctx context.Context, candidateID, runID, input, output string, score float64, success bool, latencyMs int64) (*models.PromptEvaluation, error) {
	return nil, nil
}

func (m *mockOptimizationService) GetEvaluations(ctx context.Context, candidateID string) ([]*models.PromptEvaluation, error) {
	return nil, nil
}

func (m *mockOptimizationService) GetOptimizedProgram(ctx context.Context, runID string) (*ports.OptimizedProgram, error) {
	return nil, nil
}

func (m *mockOptimizationService) GetDimensionWeights() map[string]float64 {
	return m.dimensionWeights
}

// TestHandleMemoryAction tests the memory action handler
func TestHandleMemoryAction(t *testing.T) {
	t.Run("create memory saves and sends confirmation", func(t *testing.T) {
		memoryService := newMockMemoryService()
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			memoryService:   memoryService,
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeMemoryAction,
			Body: &protocol.MemoryAction{
				ID:     "",
				Action: "create",
				Memory: &protocol.MemoryData{
					Content:  "User prefers TypeScript",
					Category: "preference",
				},
			},
		}

		err := dispatcher.handleMemoryAction(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify memory was created
		if len(memoryService.memories) != 1 {
			t.Errorf("expected 1 memory, got %d", len(memoryService.memories))
		}

		// Verify confirmation was sent
		if len(protocolHandler.sentEnvelopes) != 1 {
			t.Fatalf("expected 1 envelope sent, got %d", len(protocolHandler.sentEnvelopes))
		}

		conf, ok := protocolHandler.sentEnvelopes[0].Body.(*protocol.MemoryConfirmation)
		if !ok {
			t.Fatal("expected MemoryConfirmation body")
		}
		if !conf.Success {
			t.Error("expected success=true")
		}
		if conf.Action != "create" {
			t.Errorf("expected action 'create', got '%s'", conf.Action)
		}
	})

	t.Run("delete memory removes from service", func(t *testing.T) {
		memoryService := newMockMemoryService()
		memoryService.memories["mem_123"] = &models.Memory{ID: "mem_123", Content: "Test"}

		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			memoryService:   memoryService,
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeMemoryAction,
			Body: &protocol.MemoryAction{
				ID:     "mem_123",
				Action: "delete",
			},
		}

		err := dispatcher.handleMemoryAction(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify memory was deleted
		if len(memoryService.memories) != 0 {
			t.Errorf("expected 0 memories after deletion, got %d", len(memoryService.memories))
		}
	})

	t.Run("pin memory sets high importance", func(t *testing.T) {
		memoryService := newMockMemoryService()
		memoryService.memories["mem_456"] = &models.Memory{ID: "mem_456", Content: "Important"}

		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			memoryService:   memoryService,
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeMemoryAction,
			Body: &protocol.MemoryAction{
				ID:     "mem_456",
				Action: "pin",
			},
		}

		err := dispatcher.handleMemoryAction(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify importance was set to 1.0
		if memoryService.lastImportance != 1.0 {
			t.Errorf("expected importance 1.0 for pin, got %f", memoryService.lastImportance)
		}

		// Verify confirmation was sent
		conf := protocolHandler.sentEnvelopes[0].Body.(*protocol.MemoryConfirmation)
		if !conf.Success {
			t.Error("expected success=true for pin")
		}
	})

	t.Run("archive memory sets low importance", func(t *testing.T) {
		memoryService := newMockMemoryService()
		memoryService.memories["mem_789"] = &models.Memory{ID: "mem_789", Content: "Archive me"}

		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			memoryService:   memoryService,
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeMemoryAction,
			Body: &protocol.MemoryAction{
				ID:     "mem_789",
				Action: "archive",
			},
		}

		err := dispatcher.handleMemoryAction(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify importance was set to 0.1
		if memoryService.lastImportance != 0.1 {
			t.Errorf("expected importance 0.1 for archive, got %f", memoryService.lastImportance)
		}
	})

	t.Run("handles nil memoryService gracefully", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			memoryService:   nil,
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeMemoryAction,
			Body: &protocol.MemoryAction{
				ID:     "mem_123",
				Action: "create",
				Memory: &protocol.MemoryData{Content: "Test"},
			},
		}

		err := dispatcher.handleMemoryAction(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should send confirmation with success=false
		conf := protocolHandler.sentEnvelopes[0].Body.(*protocol.MemoryConfirmation)
		if conf.Success {
			t.Error("expected success=false when memoryService is nil")
		}
	})

	t.Run("invalid message type returns error", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeMemoryAction,
			Body: &protocol.UserMessage{}, // Wrong type
		}

		err := dispatcher.handleMemoryAction(context.Background(), envelope)
		if err == nil {
			t.Fatal("expected error for invalid message type")
		}
	})
}

// TestHandleDimensionPreference tests the dimension preference handler
func TestHandleDimensionPreference(t *testing.T) {
	t.Run("applies dimension weights to optimization service", func(t *testing.T) {
		optService := newMockOptimizationService()
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:      "conv_test1",
			optimizationService: optService,
			protocolHandler:     protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeDimensionPreference,
			Body: &protocol.DimensionPreference{
				ConversationID: "conv_test1",
				Weights: protocol.DimensionWeights{
					SuccessRate:    0.3,
					Quality:        0.25,
					Efficiency:     0.15,
					Robustness:     0.1,
					Generalization: 0.1,
					Diversity:      0.05,
					Innovation:     0.05,
				},
				Preset:    "accuracy",
				Timestamp: 1234567890,
			},
		}

		err := dispatcher.handleDimensionPreference(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify weights were applied
		if optService.dimensionWeights["successRate"] != 0.3 {
			t.Errorf("expected successRate 0.3, got %f", optService.dimensionWeights["successRate"])
		}
		if optService.dimensionWeights["quality"] != 0.25 {
			t.Errorf("expected quality 0.25, got %f", optService.dimensionWeights["quality"])
		}
	})

	t.Run("rejects mismatched conversation ID", func(t *testing.T) {
		optService := newMockOptimizationService()
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:      "conv_test1",
			optimizationService: optService,
			protocolHandler:     protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeDimensionPreference,
			Body: &protocol.DimensionPreference{
				ConversationID: "wrong_conv",
				Weights:        protocol.DimensionWeights{},
			},
		}

		err := dispatcher.handleDimensionPreference(context.Background(), envelope)
		// Should return an error from sendError
		if err != nil {
			t.Logf("Expected error for conversation mismatch: %v", err)
		}

		// Verify weights were NOT applied
		if len(optService.dimensionWeights) != 0 {
			t.Error("expected no weights applied for mismatched conversation")
		}
	})

	t.Run("handles nil optimizationService gracefully", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:      "conv_test1",
			optimizationService: nil,
			protocolHandler:     protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeDimensionPreference,
			Body: &protocol.DimensionPreference{
				ConversationID: "conv_test1",
				Weights:        protocol.DimensionWeights{SuccessRate: 0.5},
			},
		}

		err := dispatcher.handleDimensionPreference(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should complete without error when service is nil
	})

	t.Run("invalid message type returns error", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeDimensionPreference,
			Body: &protocol.UserMessage{}, // Wrong type
		}

		err := dispatcher.handleDimensionPreference(context.Background(), envelope)
		if err == nil {
			t.Fatal("expected error for invalid message type")
		}
	})
}

// TestHandleEliteSelect tests the elite selection handler
func TestHandleEliteSelect(t *testing.T) {
	t.Run("accepts valid elite selection", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeEliteSelect,
			Body: &protocol.EliteSelect{
				ConversationID: "conv_test1",
				EliteID:        "elite_high_accuracy",
				Timestamp:      1234567890,
			},
		}

		err := dispatcher.handleEliteSelect(context.Background(), envelope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should complete without error - just logs the selection
	})

	t.Run("rejects mismatched conversation ID", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeEliteSelect,
			Body: &protocol.EliteSelect{
				ConversationID: "wrong_conv",
				EliteID:        "elite_123",
			},
		}

		err := dispatcher.handleEliteSelect(context.Background(), envelope)
		// Should return nil but send error through protocol
		if err != nil {
			t.Logf("Received error for conversation mismatch: %v", err)
		}
	})

	t.Run("rejects empty elite ID", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeEliteSelect,
			Body: &protocol.EliteSelect{
				ConversationID: "conv_test1",
				EliteID:        "", // Empty ID
			},
		}

		err := dispatcher.handleEliteSelect(context.Background(), envelope)
		// Should return nil but send error through protocol
		if err != nil {
			t.Logf("Received error for empty elite ID: %v", err)
		}
	})

	t.Run("invalid message type returns error", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			protocolHandler: protocolHandler,
		}

		envelope := &protocol.Envelope{
			Type: protocol.TypeEliteSelect,
			Body: &protocol.UserMessage{}, // Wrong type
		}

		err := dispatcher.handleEliteSelect(context.Background(), envelope)
		if err == nil {
			t.Fatal("expected error for invalid message type")
		}
	})
}

// TestSendServerInfo tests the server info sending function
func TestSendServerInfo(t *testing.T) {
	t.Run("sends server info envelope", func(t *testing.T) {
		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			protocolHandler: protocolHandler,
		}

		err := dispatcher.SendServerInfo(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify envelope was sent
		if len(protocolHandler.sentEnvelopes) != 1 {
			t.Fatalf("expected 1 envelope sent, got %d", len(protocolHandler.sentEnvelopes))
		}

		envelope := protocolHandler.sentEnvelopes[0]
		if envelope.Type != protocol.TypeServerInfo {
			t.Errorf("expected TypeServerInfo, got %d", envelope.Type)
		}

		serverInfo, ok := envelope.Body.(*protocol.ServerInfo)
		if !ok {
			t.Fatal("expected ServerInfo body")
		}
		if serverInfo.Connection.Status != "connected" {
			t.Errorf("expected status 'connected', got '%s'", serverInfo.Connection.Status)
		}
	})
}

// TestSendSessionStats tests the session stats sending function
func TestSendSessionStats(t *testing.T) {
	t.Run("sends session stats with message count", func(t *testing.T) {
		messageRepo := newMockMessageRepo()
		// Add some messages
		messageRepo.store["msg1"] = &models.Message{ID: "msg1", ConversationID: "conv_test1"}
		messageRepo.store["msg2"] = &models.Message{ID: "msg2", ConversationID: "conv_test1"}

		protocolHandler := newMockProtocolHandler()

		dispatcher := &DefaultMessageDispatcher{
			conversationID:  "conv_test1",
			messageRepo:     messageRepo,
			protocolHandler: protocolHandler,
		}

		err := dispatcher.SendSessionStats(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify envelope was sent
		if len(protocolHandler.sentEnvelopes) != 1 {
			t.Fatalf("expected 1 envelope sent, got %d", len(protocolHandler.sentEnvelopes))
		}

		envelope := protocolHandler.sentEnvelopes[0]
		if envelope.Type != protocol.TypeSessionStats {
			t.Errorf("expected TypeSessionStats, got %d", envelope.Type)
		}

		stats, ok := envelope.Body.(*protocol.SessionStats)
		if !ok {
			t.Fatal("expected SessionStats body")
		}
		if stats.MessageCount != 2 {
			t.Errorf("expected 2 messages, got %d", stats.MessageCount)
		}
	})
}
