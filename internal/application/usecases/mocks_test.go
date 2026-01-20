package usecases

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// ============================================================================
// Common Mock implementations shared across tests
// ============================================================================

// mockMessageRepo is a mock message repository for testing
type mockMessageRepo struct {
	mu       sync.RWMutex
	store    map[string]*models.Message
	seqNum   int
	messages []*models.Message
}

func newMockMessageRepo() *mockMessageRepo {
	return &mockMessageRepo{
		store:    make(map[string]*models.Message),
		messages: make([]*models.Message, 0),
	}
}

func (m *mockMessageRepo) copyMessage(msg *models.Message) *models.Message {
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

func (m *mockMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[msg.ID] = m.copyMessage(msg)
	m.messages = append(m.messages, m.copyMessage(msg))
	return nil
}

func (m *mockMessageRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if msg, ok := m.store[id]; ok {
		return m.copyMessage(msg), nil
	}
	return nil, nil
}

func (m *mockMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[msg.ID]; !ok {
		return errors.New("not found")
	}
	m.store[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *mockMessageRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
	return nil
}

func (m *mockMessageRepo) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Message
	for _, msg := range m.store {
		if msg.ConversationID == conversationID {
			result = append(result, m.copyMessage(msg))
		}
	}
	return result, nil
}

func (m *mockMessageRepo) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seqNum++
	return m.seqNum, nil
}

func (m *mockMessageRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Message
	for _, msg := range m.store {
		if msg.ConversationID == conversationID {
			result = append(result, m.copyMessage(msg))
		}
	}
	return result, nil
}

func (m *mockMessageRepo) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Message
	for _, msg := range m.store {
		if msg.ConversationID == conversationID && msg.SequenceNumber > afterSequence {
			result = append(result, m.copyMessage(msg))
		}
	}
	return result, nil
}

func (m *mockMessageRepo) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, errors.New("not found")
}

func (m *mockMessageRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Message
	for _, msg := range m.store {
		// Incomplete = anything not completed (streaming, pending, failed)
		isIncomplete := msg.CompletionStatus != models.CompletionStatusCompleted
		if isIncomplete && msg.CreatedAt.Before(olderThan) {
			result = append(result, m.copyMessage(msg))
		}
	}
	return result, nil
}

func (m *mockMessageRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Message
	for _, msg := range m.store {
		// Incomplete = anything not completed (streaming, pending, failed)
		isIncomplete := msg.CompletionStatus != models.CompletionStatusCompleted
		if msg.ConversationID == conversationID && isIncomplete && msg.CreatedAt.Before(olderThan) {
			result = append(result, m.copyMessage(msg))
		}
	}
	return result, nil
}

func (m *mockMessageRepo) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetChainFromTipWithSiblings(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	// For the mock, just delegate to GetChainFromTip - tests can override if needed
	return m.GetChainFromTip(ctx, tipMessageID)
}

func (m *mockMessageRepo) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, msg := range m.store {
		if msg.ConversationID == conversationID && msg.SequenceNumber > afterSequence {
			delete(m.store, id)
		}
	}
	return nil
}

// mockConversationRepo is a mock conversation repository for testing
type mockConversationRepo struct {
	mu    sync.RWMutex
	store map[string]*models.Conversation
}

func newMockConversationRepo() *mockConversationRepo {
	return &mockConversationRepo{
		store: make(map[string]*models.Conversation),
	}
}

func (m *mockConversationRepo) Create(ctx context.Context, c *models.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[c.ID] = c
	return nil
}

func (m *mockConversationRepo) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if c, ok := m.store[id]; ok {
		return c, nil
	}
	return nil, errors.New("conversation not found")
}

func (m *mockConversationRepo) Update(ctx context.Context, c *models.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[c.ID]; !ok {
		return errors.New("not found")
	}
	m.store[c.ID] = c
	return nil
}

func (m *mockConversationRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
	return nil
}

func (m *mockConversationRepo) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepo) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepo) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, conv := range m.store {
		if conv.LiveKitRoomName == roomName {
			return conv, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockConversationRepo) UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error {
	return nil
}

func (m *mockConversationRepo) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	return nil
}

func (m *mockConversationRepo) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	return nil, errors.New("not found")
}

func (m *mockConversationRepo) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepo) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *mockConversationRepo) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.store[conversationID]; ok {
		c.TipMessageID = &messageID
		return nil
	}
	return errors.New("not found")
}

func (m *mockConversationRepo) UpdatePromptVersion(ctx context.Context, convID, versionID string) error {
	return nil
}

// mockSentenceRepo is a mock sentence repository for testing
type mockSentenceRepo struct {
	mu    sync.RWMutex
	store map[string]*models.Sentence
}

func newMockSentenceRepo() *mockSentenceRepo {
	return &mockSentenceRepo{
		store: make(map[string]*models.Sentence),
	}
}

func (m *mockSentenceRepo) Create(ctx context.Context, s *models.Sentence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[s.ID] = s
	return nil
}

func (m *mockSentenceRepo) GetByID(ctx context.Context, id string) (*models.Sentence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.store[id]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockSentenceRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.Sentence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Sentence
	for _, s := range m.store {
		if s.MessageID == messageID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSentenceRepo) Update(ctx context.Context, s *models.Sentence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[s.ID]; !ok {
		return errors.New("not found")
	}
	m.store[s.ID] = s
	return nil
}

func (m *mockSentenceRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
	return nil
}

func (m *mockSentenceRepo) GetNextSequenceNumber(ctx context.Context, messageID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	maxSeq := 0
	for _, s := range m.store {
		if s.MessageID == messageID && s.SequenceNumber > maxSeq {
			maxSeq = s.SequenceNumber
		}
	}
	return maxSeq + 1, nil
}

func (m *mockSentenceRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Sentence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Sentence
	for _, s := range m.store {
		// Incomplete = anything not completed (streaming, pending, failed)
		isIncomplete := s.CompletionStatus != models.CompletionStatusCompleted
		if isIncomplete && s.CreatedAt.Before(olderThan) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSentenceRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Sentence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Sentence
	for _, s := range m.store {
		// Incomplete = anything not completed (streaming, pending, failed)
		isIncomplete := s.CompletionStatus != models.CompletionStatusCompleted
		if isIncomplete && s.CreatedAt.Before(olderThan) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSentenceRepo) GetOrphanedSentences(ctx context.Context) ([]*models.Sentence, error) {
	return []*models.Sentence{}, nil
}

func (m *mockSentenceRepo) DeleteOrphanedSentences(ctx context.Context) (int, error) {
	return 0, nil
}

// mockIDGenerator is a mock ID generator for testing
type mockIDGenerator struct {
	mu      sync.Mutex
	counter int
}

func newMockIDGenerator() *mockIDGenerator {
	return &mockIDGenerator{counter: 0}
}

func (m *mockIDGenerator) GenerateConversationID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "conv_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateMessageID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "msg_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateSentenceID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "sent_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateAudioID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "audio_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateToolID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "tool_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateToolUseID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "tooluse_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateMemoryID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "mem_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateMemoryUsageID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "memusage_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateReasoningStepID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "reasoning_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GeneratePromptVersionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "promptver_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateCommentaryID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "commentary_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateMetaID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "meta_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateMCPServerID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "mcp_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateVoteID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "vote_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateNoteID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "note_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateSessionStatsID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "sessionstats_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateOptimizationRunID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "optrun_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GeneratePromptCandidateID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "promptcand_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GeneratePromptEvaluationID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "prompteval_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateTrainingExampleID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "trainex_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateSystemPromptVersionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "syspromptver_test_" + string(rune('0'+m.counter%10))
}

func (m *mockIDGenerator) GenerateRequestID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "request_test_" + string(rune('0'+m.counter%10))
}

// mockTransactionManager is a mock transaction manager for testing
type mockTransactionManager struct{}

func (m *mockTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// mockMemoryService is a mock memory service for testing
type mockMemoryService struct {
	memories             []*models.Memory
	searchWithScoresFunc func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error)
	trackUsageFunc       func(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error)
}

func newMockMemoryService() *mockMemoryService {
	return &mockMemoryService{
		memories: []*models.Memory{},
	}
}

func (m *mockMemoryService) Create(ctx context.Context, content string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) CreateWithEmbeddings(ctx context.Context, content string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) CreateFromConversation(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryService) Search(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	return m.memories, nil
}

func (m *mockMemoryService) SearchWithThreshold(ctx context.Context, query string, threshold float32, limit int) ([]*models.Memory, error) {
	return m.memories, nil
}

func (m *mockMemoryService) SearchWithDynamicImportance(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	return m.memories, nil
}

func (m *mockMemoryService) SearchWithScores(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
	if m.searchWithScoresFunc != nil {
		return m.searchWithScoresFunc(ctx, query, threshold, limit)
	}
	return []*ports.MemorySearchResult{}, nil
}

func (m *mockMemoryService) TrackUsage(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error) {
	if m.trackUsageFunc != nil {
		return m.trackUsageFunc(ctx, memoryID, conversationID, messageID, similarityScore)
	}
	return nil, nil
}

func (m *mockMemoryService) GetUsageByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryService) GetUsageByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryService) Update(ctx context.Context, memory *models.Memory) error {
	return nil
}

func (m *mockMemoryService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMemoryService) DeleteByConversationID(ctx context.Context, conversationID string) error {
	return nil
}

func (m *mockMemoryService) SetImportance(ctx context.Context, id string, importance float32) (*models.Memory, error) {
	return nil, nil
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

func (m *mockMemoryService) RegenerateEmbeddings(ctx context.Context, id string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) Pin(ctx context.Context, id string, pinned bool) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) Archive(ctx context.Context, id string) (*models.Memory, error) {
	return nil, nil
}

// mockToolRepo is a mock tool repository for testing
type mockToolRepo struct {
	mu    sync.RWMutex
	store map[string]*models.Tool
}

func newMockToolRepo() *mockToolRepo {
	return &mockToolRepo{
		store: make(map[string]*models.Tool),
	}
}

func (m *mockToolRepo) Create(ctx context.Context, tool *models.Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[tool.ID] = tool
	return nil
}

func (m *mockToolRepo) GetByID(ctx context.Context, id string) (*models.Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if tool, ok := m.store[id]; ok {
		return tool, nil
	}
	return nil, errors.New("not found")
}

func (m *mockToolRepo) GetByName(ctx context.Context, name string) (*models.Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, tool := range m.store {
		if tool.Name == name {
			return tool, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockToolRepo) Update(ctx context.Context, tool *models.Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[tool.ID]; !ok {
		return errors.New("not found")
	}
	m.store[tool.ID] = tool
	return nil
}

func (m *mockToolRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
	return nil
}

func (m *mockToolRepo) ListEnabled(ctx context.Context) ([]*models.Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Tool
	for _, tool := range m.store {
		if tool.Enabled {
			result = append(result, tool)
		}
	}
	return result, nil
}

func (m *mockToolRepo) ListAll(ctx context.Context) ([]*models.Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Tool
	for _, tool := range m.store {
		result = append(result, tool)
	}
	return result, nil
}

// mockToolUseRepo is a mock tool use repository for testing
type mockToolUseRepo struct {
	mu    sync.RWMutex
	store map[string]*models.ToolUse
}

func newMockToolUseRepo() *mockToolUseRepo {
	return &mockToolUseRepo{
		store: make(map[string]*models.ToolUse),
	}
}

func (m *mockToolUseRepo) Create(ctx context.Context, toolUse *models.ToolUse) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[toolUse.ID] = toolUse
	return nil
}

func (m *mockToolUseRepo) GetByID(ctx context.Context, id string) (*models.ToolUse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if toolUse, ok := m.store[id]; ok {
		return toolUse, nil
	}
	return nil, errors.New("not found")
}

func (m *mockToolUseRepo) Update(ctx context.Context, toolUse *models.ToolUse) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[toolUse.ID]; !ok {
		return errors.New("not found")
	}
	m.store[toolUse.ID] = toolUse
	return nil
}

func (m *mockToolUseRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.ToolUse, 0)
	for _, toolUse := range m.store {
		if toolUse.MessageID == messageID {
			result = append(result, toolUse)
		}
	}
	return result, nil
}

func (m *mockToolUseRepo) GetPending(ctx context.Context, limit int) ([]*models.ToolUse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.ToolUse
	for _, toolUse := range m.store {
		if toolUse.Status == models.ToolStatusPending {
			result = append(result, toolUse)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// mockGenerateResponseUseCase is a mock implementation of ports.GenerateResponseUseCase
type mockGenerateResponseUseCase struct {
	executeFunc func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error)
}

func newMockGenerateResponseUseCase() *mockGenerateResponseUseCase {
	return &mockGenerateResponseUseCase{}
}

func (m *mockGenerateResponseUseCase) Execute(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
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
