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

// Mock implementations

type mockMessageRepo struct {
	mu              sync.RWMutex
	store           map[string]*models.Message
	sequenceNumbers map[string]int
}

func newMockMessageRepo() *mockMessageRepo {
	return &mockMessageRepo{
		store:           make(map[string]*models.Message),
		sequenceNumbers: make(map[string]int),
	}
}

// copyMessage creates a deep copy of a message to avoid race conditions
// Must be called while holding the read lock
func (m *mockMessageRepo) copyMessage(msg *models.Message) *models.Message {
	return m.copyMessageUnsafe(msg)
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Store a copy to avoid race conditions
	m.store[msg.ID] = m.copyMessageUnsafe(msg)
	return nil
}

func (m *mockMessageRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if msg, ok := m.store[id]; ok {
		return m.copyMessage(msg), nil
	}
	return nil, errors.New("not found")
}

func (m *mockMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[msg.ID]; !ok {
		return errors.New("not found")
	}
	// Store a copy of the message to avoid race conditions
	// when the caller modifies the message after Update returns
	m.store[msg.ID] = m.copyMessageUnsafe(msg)
	return nil
}

// copyMessageUnsafe creates a deep copy without locking (must be called while holding a lock)
func (m *mockMessageRepo) copyMessageUnsafe(msg *models.Message) *models.Message {
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

func (m *mockMessageRepo) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
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

func (m *mockMessageRepo) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	num := m.sequenceNumbers[conversationID]
	m.sequenceNumbers[conversationID] = num + 1
	return num, nil
}

func (m *mockMessageRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	messages := []*models.Message{}
	for _, msg := range m.store {
		if (msg.CompletionStatus == models.CompletionStatusStreaming || msg.CompletionStatus == models.CompletionStatusPending) && msg.CreatedAt.Before(olderThan) {
			messages = append(messages, m.copyMessage(msg))
		}
	}
	return messages, nil
}

func (m *mockMessageRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	messages := []*models.Message{}
	for _, msg := range m.store {
		if msg.ConversationID == conversationID && (msg.CompletionStatus == models.CompletionStatusStreaming || msg.CompletionStatus == models.CompletionStatusPending) && msg.CreatedAt.Before(olderThan) {
			messages = append(messages, m.copyMessage(msg))
		}
	}
	return messages, nil
}

func (m *mockMessageRepo) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Simple implementation: return message chain by following PreviousID
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

func (m *mockMessageRepo) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
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
		sCopy := *s
		return &sCopy, nil
	}
	return nil, errors.New("not found")
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

func (m *mockSentenceRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.Sentence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sentences := []*models.Sentence{}
	for _, s := range m.store {
		if s.MessageID == messageID {
			sCopy := *s
			sentences = append(sentences, &sCopy)
		}
	}
	return sentences, nil
}

func (m *mockSentenceRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
	return nil
}

func (m *mockSentenceRepo) GetNextSequenceNumber(ctx context.Context, messageID string) (int, error) {
	return 0, nil
}

func (m *mockSentenceRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Sentence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Note: Sentences don't have ConversationID directly, so returning all incomplete sentences for now
	// A complete implementation would need to join with messages to filter by conversation
	sentences := []*models.Sentence{}
	for _, s := range m.store {
		if (s.CompletionStatus == models.CompletionStatusStreaming || s.CompletionStatus == models.CompletionStatusPending) && s.CreatedAt.Before(olderThan) {
			sCopy := *s
			sentences = append(sentences, &sCopy)
		}
	}
	return sentences, nil
}

func (m *mockSentenceRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Sentence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sentences := []*models.Sentence{}
	for _, s := range m.store {
		if (s.CompletionStatus == models.CompletionStatusStreaming || s.CompletionStatus == models.CompletionStatusPending) && s.CreatedAt.Before(olderThan) {
			sCopy := *s
			sentences = append(sentences, &sCopy)
		}
	}
	return sentences, nil
}

type mockToolUseRepo struct {
	store map[string]*models.ToolUse
}

func newMockToolUseRepo() *mockToolUseRepo {
	return &mockToolUseRepo{
		store: make(map[string]*models.ToolUse),
	}
}

func (m *mockToolUseRepo) Create(ctx context.Context, tu *models.ToolUse) error {
	m.store[tu.ID] = tu
	return nil
}

func (m *mockToolUseRepo) GetByID(ctx context.Context, id string) (*models.ToolUse, error) {
	if tu, ok := m.store[id]; ok {
		return tu, nil
	}
	return nil, errors.New("not found")
}

func (m *mockToolUseRepo) Update(ctx context.Context, tu *models.ToolUse) error {
	if _, ok := m.store[tu.ID]; !ok {
		return errors.New("not found")
	}
	m.store[tu.ID] = tu
	return nil
}

func (m *mockToolUseRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	return []*models.ToolUse{}, nil
}

func (m *mockToolUseRepo) GetPending(ctx context.Context, limit int) ([]*models.ToolUse, error) {
	return []*models.ToolUse{}, nil
}

type mockToolRepo struct {
	store map[string]*models.Tool
}

func newMockToolRepo() *mockToolRepo {
	return &mockToolRepo{
		store: make(map[string]*models.Tool),
	}
}

func (m *mockToolRepo) Create(ctx context.Context, tool *models.Tool) error {
	m.store[tool.ID] = tool
	return nil
}

func (m *mockToolRepo) GetByID(ctx context.Context, id string) (*models.Tool, error) {
	if tool, ok := m.store[id]; ok {
		return tool, nil
	}
	return nil, errors.New("not found")
}

func (m *mockToolRepo) GetByName(ctx context.Context, name string) (*models.Tool, error) {
	for _, tool := range m.store {
		if tool.Name == name {
			return tool, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockToolRepo) Update(ctx context.Context, tool *models.Tool) error {
	if _, ok := m.store[tool.ID]; !ok {
		return errors.New("not found")
	}
	m.store[tool.ID] = tool
	return nil
}

func (m *mockToolRepo) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

func (m *mockToolRepo) ListEnabled(ctx context.Context) ([]*models.Tool, error) {
	tools := []*models.Tool{}
	for _, tool := range m.store {
		if tool.Enabled && tool.DeletedAt == nil {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

func (m *mockToolRepo) ListAll(ctx context.Context) ([]*models.Tool, error) {
	tools := []*models.Tool{}
	for _, tool := range m.store {
		if tool.DeletedAt == nil {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

type mockReasoningStepRepo struct {
	store map[string]*models.ReasoningStep
}

func newMockReasoningStepRepo() *mockReasoningStepRepo {
	return &mockReasoningStepRepo{
		store: make(map[string]*models.ReasoningStep),
	}
}

func (m *mockReasoningStepRepo) Create(ctx context.Context, step *models.ReasoningStep) error {
	m.store[step.ID] = step
	return nil
}

func (m *mockReasoningStepRepo) GetByID(ctx context.Context, id string) (*models.ReasoningStep, error) {
	if step, ok := m.store[id]; ok {
		return step, nil
	}
	return nil, errors.New("not found")
}

func (m *mockReasoningStepRepo) Update(ctx context.Context, step *models.ReasoningStep) error {
	if _, ok := m.store[step.ID]; !ok {
		return errors.New("not found")
	}
	m.store[step.ID] = step
	return nil
}

func (m *mockReasoningStepRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.ReasoningStep, error) {
	return []*models.ReasoningStep{}, nil
}

func (m *mockReasoningStepRepo) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

func (m *mockReasoningStepRepo) GetNextSequenceNumber(ctx context.Context, messageID string) (int, error) {
	return 0, nil
}

type mockConversationRepo struct {
	store map[string]*models.Conversation
}

func newMockConversationRepo() *mockConversationRepo {
	return &mockConversationRepo{
		store: make(map[string]*models.Conversation),
	}
}

func (m *mockConversationRepo) Create(ctx context.Context, c *models.Conversation) error {
	m.store[c.ID] = c
	return nil
}

func (m *mockConversationRepo) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	if c, ok := m.store[id]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (m *mockConversationRepo) Update(ctx context.Context, c *models.Conversation) error {
	if _, ok := m.store[c.ID]; !ok {
		return errors.New("not found")
	}
	m.store[c.ID] = c
	return nil
}

func (m *mockConversationRepo) Delete(ctx context.Context, id string) error {
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
	for _, c := range m.store {
		if c.LiveKitRoomName == roomName {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockConversationRepo) UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error {
	return nil
}

func (m *mockConversationRepo) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	delete(m.store, id)
	return nil
}

func (m *mockConversationRepo) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	if c, ok := m.store[id]; ok {
		return c, nil
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
	if c, ok := m.store[conversationID]; ok {
		c.TipMessageID = &messageID
		return nil
	}
	return errors.New("not found")
}

func (m *mockConversationRepo) UpdatePromptVersion(ctx context.Context, convID, versionID string) error {
	if c, ok := m.store[convID]; ok {
		c.SystemPromptVersionID = versionID
		return nil
	}
	return errors.New("not found")
}

type mockLLMService struct {
	chatResponse           *ports.LLMResponse
	chatError              error
	chatWithToolsResponse  *ports.LLMResponse
	chatWithToolsError     error
	chatWithToolsFunc      func(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error)
	streamChannel          chan ports.LLMStreamChunk
	streamError            error
	streamWithToolsChannel chan ports.LLMStreamChunk
	streamWithToolsError   error
}

func newMockLLMService() *mockLLMService {
	return &mockLLMService{}
}

func (m *mockLLMService) Chat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	if m.chatError != nil {
		return nil, m.chatError
	}
	if m.chatResponse != nil {
		return m.chatResponse, nil
	}
	return &ports.LLMResponse{
		Content: "This is a test response",
	}, nil
}

func (m *mockLLMService) ChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	if m.chatWithToolsFunc != nil {
		return m.chatWithToolsFunc(ctx, messages, tools)
	}
	if m.chatWithToolsError != nil {
		return nil, m.chatWithToolsError
	}
	if m.chatWithToolsResponse != nil {
		return m.chatWithToolsResponse, nil
	}
	return &ports.LLMResponse{
		Content: "Response with tools available",
	}, nil
}

func (m *mockLLMService) ChatStream(ctx context.Context, messages []ports.LLMMessage) (<-chan ports.LLMStreamChunk, error) {
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

func (m *mockLLMService) ChatStreamWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (<-chan ports.LLMStreamChunk, error) {
	if m.streamWithToolsError != nil {
		return nil, m.streamWithToolsError
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

type mockToolService struct {
	createToolUseFunc  func(ctx context.Context, messageID, toolName string, arguments map[string]any) (*models.ToolUse, error)
	executeToolUseFunc func(ctx context.Context, toolUseID string) (*models.ToolUse, error)
}

func newMockToolService() *mockToolService {
	return &mockToolService{}
}

func (m *mockToolService) RegisterTool(ctx context.Context, name, description string, schema map[string]any) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) EnsureTool(ctx context.Context, name, description string, schema map[string]any) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) RegisterExecutor(name string, executor func(ctx context.Context, arguments map[string]any) (any, error)) error {
	return nil
}

func (m *mockToolService) GetByID(ctx context.Context, id string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) GetByName(ctx context.Context, name string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) Update(ctx context.Context, tool *models.Tool) error {
	return nil
}

func (m *mockToolService) Enable(ctx context.Context, id string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) Disable(ctx context.Context, id string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockToolService) ListEnabled(ctx context.Context) ([]*models.Tool, error) {
	return []*models.Tool{}, nil
}

func (m *mockToolService) ListAll(ctx context.Context) ([]*models.Tool, error) {
	return []*models.Tool{}, nil
}

func (m *mockToolService) ExecuteTool(ctx context.Context, toolName string, arguments map[string]any) (any, error) {
	return "tool result", nil
}

func (m *mockToolService) CreateToolUse(ctx context.Context, messageID, toolName string, arguments map[string]any) (*models.ToolUse, error) {
	if m.createToolUseFunc != nil {
		return m.createToolUseFunc(ctx, messageID, toolName, arguments)
	}
	return models.NewToolUse("tu_test1", messageID, toolName, 0, arguments), nil
}

func (m *mockToolService) ExecuteToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	if m.executeToolUseFunc != nil {
		return m.executeToolUseFunc(ctx, toolUseID)
	}
	tu := models.NewToolUse(toolUseID, "msg_test", "test_tool", 0, map[string]any{})
	tu.Complete("tool result")
	return tu, nil
}

func (m *mockToolService) GetToolUseByID(ctx context.Context, id string) (*models.ToolUse, error) {
	return nil, nil
}

func (m *mockToolService) GetToolUsesByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	return []*models.ToolUse{}, nil
}

func (m *mockToolService) GetPendingToolUses(ctx context.Context, limit int) ([]*models.ToolUse, error) {
	return []*models.ToolUse{}, nil
}

func (m *mockToolService) CancelToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	return nil, nil
}

type mockMemoryService struct {
	searchWithScoresFunc func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error)
	trackUsageFunc       func(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error)
}

func newMockMemoryService() *mockMemoryService {
	return &mockMemoryService{}
}

func (m *mockMemoryService) Create(ctx context.Context, content string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) CreateWithEmbeddings(ctx context.Context, content string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) CreateFromConversation(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error) {
	return models.NewMemory("mem_test1", content), nil
}

func (m *mockMemoryService) GetByID(ctx context.Context, id string) (*models.Memory, error) {
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
	if m.searchWithScoresFunc != nil {
		return m.searchWithScoresFunc(ctx, query, threshold, limit)
	}
	return []*ports.MemorySearchResult{}, nil
}

func (m *mockMemoryService) TrackUsage(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error) {
	if m.trackUsageFunc != nil {
		return m.trackUsageFunc(ctx, memoryID, conversationID, messageID, similarityScore)
	}
	mu := models.NewMemoryUsage("mu_test1", conversationID, messageID, memoryID)
	mu.SimilarityScore = similarityScore
	return mu, nil
}

func (m *mockMemoryService) GetUsageByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryService) GetUsageByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryService) GetUsageByMemory(ctx context.Context, memoryID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryService) SetImportance(ctx context.Context, memoryID string, importance float32) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) SetConfidence(ctx context.Context, memoryID string, confidence float32) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) SetUserRating(ctx context.Context, id string, rating int) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) AddTag(ctx context.Context, memoryID, tag string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) RemoveTag(ctx context.Context, memoryID, tag string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryService) Update(ctx context.Context, memory *models.Memory) error {
	return nil
}

func (m *mockMemoryService) Delete(ctx context.Context, id string) error {
	return nil
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

func (m *mockMemoryService) DeleteByConversationID(ctx context.Context, conversationID string) error {
	return nil
}

type mockIDGenerator struct {
	messageCounter     int
	sentenceCounter    int
	toolUseCounter     int
	reasoningCounter   int
	memoryCounter      int
	memoryUsageCounter int
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
	m.memoryCounter++
	return "mem_test" + string(rune('0'+m.memoryCounter))
}

func (m *mockIDGenerator) GenerateMemoryUsageID() string {
	m.memoryUsageCounter++
	return "mu_test" + string(rune('0'+m.memoryUsageCounter))
}

func (m *mockIDGenerator) GenerateToolID() string {
	return "tool_test1"
}

func (m *mockIDGenerator) GenerateToolUseID() string {
	m.toolUseCounter++
	return "tu_test" + string(rune('0'+m.toolUseCounter))
}

func (m *mockIDGenerator) GenerateReasoningStepID() string {
	m.reasoningCounter++
	return "rs_test" + string(rune('0'+m.reasoningCounter))
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

func (m *mockIDGenerator) GenerateRequestID() string {
	return "areq_test1"
}

type mockTransactionManager struct{}

func (m *mockTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// Tests

func TestGenerateResponse_BasicNonStreaming(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	convRepo := newMockConversationRepo()
	llmService := newMockLLMService()
	toolService := newMockToolService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation first
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	uc := NewGenerateResponse(
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

	// Create a user message first
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	input := &ports.GenerateResponseInput{
		ConversationID:  "conv_123",
		UserMessageID:   "user_msg_1",
		EnableStreaming: false,
		EnableTools:     false,
		EnableReasoning: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Message == nil {
		t.Fatal("expected message to be created")
	}

	if output.Message.Contents != "This is a test response" {
		t.Errorf("expected content 'This is a test response', got %s", output.Message.Contents)
	}

	if output.Message.Role != models.MessageRoleAssistant {
		t.Errorf("expected role assistant, got %s", output.Message.Role)
	}

	// Verify message was stored
	stored, _ := msgRepo.GetByID(context.Background(), output.Message.ID)
	if stored == nil {
		t.Error("message not stored")
	}
}

func TestGenerateResponse_WithReasoning(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	convRepo := newMockConversationRepo()
	llmService := newMockLLMService()
	toolService := newMockToolService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation first
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Configure LLM to return reasoning
	llmService.chatResponse = &ports.LLMResponse{
		Content:   "Response with reasoning",
		Reasoning: "This is my reasoning process",
	}

	uc := NewGenerateResponse(
		msgRepo,
		sentRepo,
		toolUseRepo,
		toolRepo,
		reasoningRepo,
		convRepo,
		llmService,
		toolService,
		nil,
		nil, // No prompt version service
		idGen,
		txManager,
		nil, // No broadcaster
	)

	input := &ports.GenerateResponseInput{
		ConversationID:  "conv_123",
		UserMessageID:   "user_msg_1",
		EnableStreaming: false,
		EnableTools:     false,
		EnableReasoning: true,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.ReasoningSteps) != 1 {
		t.Fatalf("expected 1 reasoning step, got %d", len(output.ReasoningSteps))
	}

	if output.ReasoningSteps[0].Content != "This is my reasoning process" {
		t.Errorf("expected reasoning content, got %s", output.ReasoningSteps[0].Content)
	}

	// Verify reasoning step was stored
	stored, _ := reasoningRepo.GetByID(context.Background(), output.ReasoningSteps[0].ID)
	if stored == nil {
		t.Error("reasoning step not stored")
	}
}

func TestGenerateResponse_WithMemoryRetrieval(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	convRepo := newMockConversationRepo()
	llmService := newMockLLMService()
	toolService := newMockToolService()
	memoryService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation first
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create a user message
	userMsg := models.NewMessage("user_msg_1", "conv_123", 0, models.MessageRoleUser, "What's my favorite color?")
	msgRepo.Create(context.Background(), userMsg)

	// Configure memory service to return relevant memories
	memoryService.searchWithScoresFunc = func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
		mem := models.NewMemory("mem_1", "User's favorite color is blue")
		return []*ports.MemorySearchResult{
			{Memory: mem, Similarity: 0.95},
		}, nil
	}

	// Track if memory usage was recorded
	memoryUsageTracked := false
	memoryService.trackUsageFunc = func(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error) {
		memoryUsageTracked = true
		if memoryID != "mem_1" {
			t.Errorf("expected memory ID mem_1, got %s", memoryID)
		}
		if similarityScore != 0.95 {
			t.Errorf("expected similarity 0.95, got %f", similarityScore)
		}
		mu := models.NewMemoryUsage("mu_1", conversationID, messageID, memoryID)
		mu.SimilarityScore = similarityScore
		return mu, nil
	}

	uc := NewGenerateResponse(
		msgRepo,
		sentRepo,
		toolUseRepo,
		toolRepo,
		reasoningRepo,
		convRepo,
		llmService,
		toolService,
		memoryService,
		nil, // No prompt version service
		idGen,
		txManager,
		nil, // No broadcaster
	)

	input := &ports.GenerateResponseInput{
		ConversationID:  "conv_123",
		UserMessageID:   "user_msg_1",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Message == nil {
		t.Fatal("expected message to be created")
	}

	if !memoryUsageTracked {
		t.Error("expected memory usage to be tracked")
	}
}

func TestGenerateResponse_WithToolExecution(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	convRepo := newMockConversationRepo()
	llmService := newMockLLMService()
	toolService := newMockToolService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation first
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create a tool
	tool := models.NewTool("tool_1", "calculator", "Calculate numbers", map[string]any{})
	tool.Enable()
	toolRepo.Create(context.Background(), tool)

	// Configure LLM to call tool, then provide final response
	callCount := 0
	llmService.chatWithToolsFunc = func(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
		callCount++
		if callCount == 1 {
			// First call: request tool use
			return &ports.LLMResponse{
				Content: "Let me calculate that",
				ToolCalls: []*ports.LLMToolCall{
					{Name: "calculator", Arguments: map[string]any{"expression": "2+2"}},
				},
			}, nil
		}
		// Second call: final response after tool execution
		return &ports.LLMResponse{
			Content: "The answer is 4",
		}, nil
	}

	// Configure tool service to execute tools
	toolService.createToolUseFunc = func(ctx context.Context, messageID, toolName string, arguments map[string]any) (*models.ToolUse, error) {
		tu := models.NewToolUse("tu_1", messageID, toolName, 0, arguments)
		return tu, nil
	}

	toolService.executeToolUseFunc = func(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
		tu := models.NewToolUse(toolUseID, "msg_test", "calculator", 0, map[string]any{"expression": "2+2"})
		tu.Complete("4")
		return tu, nil
	}

	uc := NewGenerateResponse(
		msgRepo,
		sentRepo,
		toolUseRepo,
		toolRepo,
		reasoningRepo,
		convRepo,
		llmService,
		toolService,
		nil,
		nil, // No prompt version service
		idGen,
		txManager,
		nil, // No broadcaster
	)

	input := &ports.GenerateResponseInput{
		ConversationID:  "conv_123",
		UserMessageID:   "user_msg_1",
		EnableStreaming: false,
		EnableTools:     true,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.ToolUses) != 1 {
		t.Fatalf("expected 1 tool use, got %d", len(output.ToolUses))
	}

	if output.Message.Contents != "The answer is 4" {
		t.Errorf("expected final response 'The answer is 4', got %s", output.Message.Contents)
	}
}

func TestGenerateResponse_LLMError(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	convRepo := newMockConversationRepo()
	llmService := newMockLLMService()
	toolService := newMockToolService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation first
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	llmService.chatError = errors.New("LLM service unavailable")

	uc := NewGenerateResponse(
		msgRepo,
		sentRepo,
		toolUseRepo,
		toolRepo,
		reasoningRepo,
		convRepo,
		llmService,
		toolService,
		nil,
		nil, // No prompt version service
		idGen,
		txManager,
		nil, // No broadcaster
	)

	input := &ports.GenerateResponseInput{
		ConversationID:  "conv_123",
		UserMessageID:   "user_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when LLM fails, got nil")
	}
}

func TestGenerateResponse_ToolExecutionError(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	convRepo := newMockConversationRepo()
	llmService := newMockLLMService()
	toolService := newMockToolService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation first
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create a tool
	tool := models.NewTool("tool_1", "faulty_tool", "A tool that fails", map[string]any{})
	tool.Enable()
	toolRepo.Create(context.Background(), tool)

	// Configure LLM to call tool
	callCount := 0
	llmService.chatWithToolsFunc = func(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
		callCount++
		if callCount == 1 {
			return &ports.LLMResponse{
				Content: "Let me use the tool",
				ToolCalls: []*ports.LLMToolCall{
					{Name: "faulty_tool", Arguments: map[string]any{}},
				},
			}, nil
		}
		// After tool error, LLM provides recovery response
		return &ports.LLMResponse{
			Content: "I encountered an error with the tool",
		}, nil
	}

	// Configure tool service to fail
	toolService.executeToolUseFunc = func(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
		return nil, errors.New("tool execution failed")
	}

	uc := NewGenerateResponse(
		msgRepo,
		sentRepo,
		toolUseRepo,
		toolRepo,
		reasoningRepo,
		convRepo,
		llmService,
		toolService,
		nil,
		nil, // No prompt version service
		idGen,
		txManager,
		nil, // No broadcaster
	)

	input := &ports.GenerateResponseInput{
		ConversationID:  "conv_123",
		UserMessageID:   "user_msg_1",
		EnableStreaming: false,
		EnableTools:     true,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still get a response despite tool failure
	if output.Message == nil {
		t.Fatal("expected message to be created")
	}

	if output.Message.Contents != "I encountered an error with the tool" {
		t.Errorf("expected recovery response, got %s", output.Message.Contents)
	}
}

func TestGenerateResponse_StreamingMode(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	convRepo := newMockConversationRepo()
	llmService := newMockLLMService()
	toolService := newMockToolService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation first
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Configure streaming channel
	streamCh := make(chan ports.LLMStreamChunk, 10)
	llmService.streamChannel = streamCh

	go func() {
		defer close(streamCh)
		streamCh <- ports.LLMStreamChunk{Content: "Hello ", Done: false}
		streamCh <- ports.LLMStreamChunk{Content: "world.", Done: false}
		streamCh <- ports.LLMStreamChunk{Done: true}
	}()

	uc := NewGenerateResponse(
		msgRepo,
		sentRepo,
		toolUseRepo,
		toolRepo,
		reasoningRepo,
		convRepo,
		llmService,
		toolService,
		nil,
		nil, // No prompt version service
		idGen,
		txManager,
		nil, // No broadcaster
	)

	input := &ports.GenerateResponseInput{
		ConversationID:  "conv_123",
		UserMessageID:   "user_msg_1",
		EnableStreaming: true,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.StreamChannel == nil {
		t.Fatal("expected stream channel to be provided")
	}

	// Get initial message state from repository (thread-safe)
	initialMsg, _ := msgRepo.GetByID(context.Background(), output.Message.ID)
	if initialMsg.CompletionStatus != models.CompletionStatusStreaming {
		t.Errorf("expected message status streaming, got %s", initialMsg.CompletionStatus)
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

	// Wait a bit for async processing to complete
	time.Sleep(100 * time.Millisecond)

	// Check final message state from repository (thread-safe)
	finalMsg, _ := msgRepo.GetByID(context.Background(), output.Message.ID)
	if finalMsg.CompletionStatus != models.CompletionStatusCompleted {
		t.Errorf("expected message status completed after stream, got %s", finalMsg.CompletionStatus)
	}

	if finalMsg.Contents != "Hello world." {
		t.Errorf("expected full content 'Hello world.', got %s", finalMsg.Contents)
	}
}

func TestGenerateResponse_StreamingError(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	convRepo := newMockConversationRepo()
	llmService := newMockLLMService()
	toolService := newMockToolService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation first
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	llmService.streamError = errors.New("streaming failed")

	uc := NewGenerateResponse(
		msgRepo,
		sentRepo,
		toolUseRepo,
		toolRepo,
		reasoningRepo,
		convRepo,
		llmService,
		toolService,
		nil,
		nil, // No prompt version service
		idGen,
		txManager,
		nil, // No broadcaster
	)

	input := &ports.GenerateResponseInput{
		ConversationID:  "conv_123",
		UserMessageID:   "user_msg_1",
		EnableStreaming: true,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when streaming fails to start, got nil")
	}
}

func TestGenerateResponse_WithPreGeneratedMessageID(t *testing.T) {
	msgRepo := newMockMessageRepo()
	sentRepo := newMockSentenceRepo()
	toolUseRepo := newMockToolUseRepo()
	toolRepo := newMockToolRepo()
	reasoningRepo := newMockReasoningStepRepo()
	convRepo := newMockConversationRepo()
	llmService := newMockLLMService()
	toolService := newMockToolService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation first
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	uc := NewGenerateResponse(
		msgRepo,
		sentRepo,
		toolUseRepo,
		toolRepo,
		reasoningRepo,
		convRepo,
		llmService,
		toolService,
		nil,
		nil, // No prompt version service
		idGen,
		txManager,
		nil, // No broadcaster
	)

	input := &ports.GenerateResponseInput{
		ConversationID:  "conv_123",
		UserMessageID:   "user_msg_1",
		MessageID:       "custom_msg_id",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Message.ID != "custom_msg_id" {
		t.Errorf("expected message ID custom_msg_id, got %s", output.Message.ID)
	}
}

func TestGenerateResponse_SentenceExtraction(t *testing.T) {
	uc := &GenerateResponse{}

	tests := []struct {
		name      string
		input     string
		maxLength int
		wantText  string
		wantRem   string
	}{
		{
			name:      "simple sentence",
			input:     "Hello world. More text",
			maxLength: 1000,
			wantText:  "Hello world.",
			wantRem:   "More text",
		},
		{
			name:      "question mark",
			input:     "How are you? I'm fine.",
			maxLength: 1000,
			wantText:  "How are you?",
			wantRem:   "I'm fine.",
		},
		{
			name:      "exclamation",
			input:     "Stop! Don't move.",
			maxLength: 1000,
			wantText:  "Stop!",
			wantRem:   "Don't move.",
		},
		{
			name:      "abbreviation followed by uppercase",
			input:     "Dr. Smith is here.",
			maxLength: 1000,
			wantText:  "Dr.",
			wantRem:   "Smith is here.",
		},
		{
			name:      "no sentence end",
			input:     "This has no ending",
			maxLength: 1000,
			wantText:  "",
			wantRem:   "This has no ending",
		},
		{
			name:      "force break on max length",
			input:     "This is a very long sentence without any punctuation",
			maxLength: 20,
			wantText:  "This is a very long",
			wantRem:   "sentence without any punctuation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotRem := uc.extractNextSentence(tt.input, tt.maxLength)
			if gotText != tt.wantText {
				t.Errorf("extractNextSentence() text = %q, want %q", gotText, tt.wantText)
			}
			if gotRem != tt.wantRem {
				t.Errorf("extractNextSentence() remaining = %q, want %q", gotRem, tt.wantRem)
			}
		})
	}
}
