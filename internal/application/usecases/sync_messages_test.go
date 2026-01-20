package usecases

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// syncMockMessageRepo is a mock MessageRepository with configurable behavior for sync testing
type syncMockMessageRepo struct {
	mu    sync.RWMutex
	store map[string]*models.Message

	// Configurable behaviors
	getByLocalIDFunc func(ctx context.Context, localID string) (*models.Message, error)
	createFunc       func(ctx context.Context, msg *models.Message) error
	updateFunc       func(ctx context.Context, msg *models.Message) error
}

func newSyncMockMessageRepo() *syncMockMessageRepo {
	return &syncMockMessageRepo{
		store: make(map[string]*models.Message),
	}
}

func (m *syncMockMessageRepo) copyMessage(msg *models.Message) *models.Message {
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

func (m *syncMockMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, msg)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *syncMockMessageRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if msg, ok := m.store[id]; ok {
		return m.copyMessage(msg), nil
	}
	return nil, domain.ErrMessageNotFound
}

func (m *syncMockMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, msg)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[msg.ID]; !ok {
		return domain.ErrMessageNotFound
	}
	m.store[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *syncMockMessageRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
	return nil
}

func (m *syncMockMessageRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var messages []*models.Message
	for _, msg := range m.store {
		if msg.ConversationID == conversationID {
			messages = append(messages, m.copyMessage(msg))
		}
	}
	return messages, nil
}

func (m *syncMockMessageRepo) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *syncMockMessageRepo) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	return 0, nil
}

func (m *syncMockMessageRepo) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *syncMockMessageRepo) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *syncMockMessageRepo) GetChainFromTipWithSiblings(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return m.GetChainFromTip(ctx, tipMessageID)
}

func (m *syncMockMessageRepo) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *syncMockMessageRepo) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *syncMockMessageRepo) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	if m.getByLocalIDFunc != nil {
		return m.getByLocalIDFunc(ctx, localID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, msg := range m.store {
		if msg.LocalID == localID {
			return m.copyMessage(msg), nil
		}
	}
	return nil, domain.ErrMessageNotFound
}

func (m *syncMockMessageRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *syncMockMessageRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *syncMockMessageRepo) DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error {
	return nil
}

// AddMessage adds a message to the mock store (for test setup)
func (m *syncMockMessageRepo) AddMessage(msg *models.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[msg.ID] = m.copyMessage(msg)
}

// syncMockConversationRepo is a mock ConversationRepository for sync testing
type syncMockConversationRepo struct {
	mu    sync.RWMutex
	store map[string]*models.Conversation

	// Configurable behaviors
	getByIDFunc func(ctx context.Context, id string) (*models.Conversation, error)
}

func newSyncMockConversationRepo() *syncMockConversationRepo {
	return &syncMockConversationRepo{
		store: make(map[string]*models.Conversation),
	}
}

func (m *syncMockConversationRepo) Create(ctx context.Context, c *models.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[c.ID] = c
	return nil
}

func (m *syncMockConversationRepo) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if c, ok := m.store[id]; ok {
		return c, nil
	}
	return nil, domain.ErrConversationNotFound
}

func (m *syncMockConversationRepo) Update(ctx context.Context, c *models.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[c.ID]; !ok {
		return domain.ErrConversationNotFound
	}
	m.store[c.ID] = c
	return nil
}

func (m *syncMockConversationRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
	return nil
}

func (m *syncMockConversationRepo) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *syncMockConversationRepo) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *syncMockConversationRepo) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	return nil, domain.ErrConversationNotFound
}

func (m *syncMockConversationRepo) UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error {
	return nil
}

func (m *syncMockConversationRepo) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	return nil
}

func (m *syncMockConversationRepo) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	return nil, domain.ErrConversationNotFound
}

func (m *syncMockConversationRepo) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *syncMockConversationRepo) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *syncMockConversationRepo) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.store[conversationID]; ok {
		c.TipMessageID = &messageID
		return nil
	}
	return domain.ErrConversationNotFound
}

func (m *syncMockConversationRepo) UpdatePromptVersion(ctx context.Context, convID, versionID string) error {
	return nil
}

// syncMockIDGenerator is a mock IDGenerator for sync testing
type syncMockIDGenerator struct {
	mu             sync.Mutex
	messageCounter int
}

func newSyncMockIDGenerator() *syncMockIDGenerator {
	return &syncMockIDGenerator{}
}

func (m *syncMockIDGenerator) GenerateConversationID() string { return "conv_test" }
func (m *syncMockIDGenerator) GenerateMessageID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageCounter++
	return "msg_server_" + string(rune('0'+m.messageCounter))
}
func (m *syncMockIDGenerator) GenerateSentenceID() string            { return "sent_test" }
func (m *syncMockIDGenerator) GenerateAudioID() string               { return "audio_test" }
func (m *syncMockIDGenerator) GenerateMemoryID() string              { return "mem_test" }
func (m *syncMockIDGenerator) GenerateMemoryUsageID() string         { return "mu_test" }
func (m *syncMockIDGenerator) GenerateToolID() string                { return "tool_test" }
func (m *syncMockIDGenerator) GenerateToolUseID() string             { return "tu_test" }
func (m *syncMockIDGenerator) GenerateReasoningStepID() string       { return "rs_test" }
func (m *syncMockIDGenerator) GenerateCommentaryID() string          { return "comm_test" }
func (m *syncMockIDGenerator) GenerateMetaID() string                { return "meta_test" }
func (m *syncMockIDGenerator) GenerateMCPServerID() string           { return "mcp_test" }
func (m *syncMockIDGenerator) GenerateVoteID() string                { return "vote_test" }
func (m *syncMockIDGenerator) GenerateNoteID() string                { return "note_test" }
func (m *syncMockIDGenerator) GenerateSessionStatsID() string        { return "ss_test" }
func (m *syncMockIDGenerator) GenerateOptimizationRunID() string     { return "or_test" }
func (m *syncMockIDGenerator) GeneratePromptCandidateID() string     { return "pc_test" }
func (m *syncMockIDGenerator) GeneratePromptEvaluationID() string    { return "pe_test" }
func (m *syncMockIDGenerator) GenerateTrainingExampleID() string     { return "te_test" }
func (m *syncMockIDGenerator) GenerateSystemPromptVersionID() string { return "spv_test" }
func (m *syncMockIDGenerator) GenerateRequestID() string             { return "areq_test" }

// syncMockTransactionManager is a mock TransactionManager for sync testing
type syncMockTransactionManager struct {
	shouldRollback bool
	rollbackCalled bool
}

func newSyncMockTransactionManager() *syncMockTransactionManager {
	return &syncMockTransactionManager{}
}

func (m *syncMockTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	err := fn(ctx)
	if err != nil {
		m.rollbackCalled = true
	}
	return err
}

// Test cases

func TestSyncMessages_Execute_Success(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	// Create input with multiple messages
	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:        "local_msg_1",
				SequenceNumber: 1,
				PreviousID:     "",
				Role:           "user",
				Contents:       "Hello, this is my first message",
				CreatedAt:      time.Now().Add(-2 * time.Hour),
				UpdatedAt:      time.Now().Add(-2 * time.Hour),
			},
			{
				LocalID:        "local_msg_2",
				SequenceNumber: 2,
				PreviousID:     "local_msg_1",
				Role:           "assistant",
				Contents:       "Hello! How can I help you today?",
				CreatedAt:      time.Now().Add(-1 * time.Hour),
				UpdatedAt:      time.Now().Add(-1 * time.Hour),
			},
			{
				LocalID:        "local_msg_3",
				SequenceNumber: 3,
				PreviousID:     "local_msg_2",
				Role:           "user",
				Contents:       "I have a question about Go programming",
				CreatedAt:      time.Now().Add(-30 * time.Minute),
				UpdatedAt:      time.Now().Add(-30 * time.Minute),
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify output
	if output == nil {
		t.Fatal("expected output, got nil")
	}

	// Verify all messages were synced
	if len(output.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(output.Results))
	}

	for i, result := range output.Results {
		if result.Status != "synced" {
			t.Errorf("result[%d]: expected status 'synced', got '%s'", i, result.Status)
		}
		if result.ServerID == "" {
			t.Errorf("result[%d]: expected ServerID to be set", i)
		}
		if result.Message == nil {
			t.Errorf("result[%d]: expected Message to be set", i)
		}
	}

	// Verify SyncedAt is set
	if output.SyncedAt.IsZero() {
		t.Error("expected SyncedAt to be set")
	}

	// Verify messages were stored in repository
	messages, _ := msgRepo.GetByConversation(context.Background(), "conv_123")
	if len(messages) != 3 {
		t.Errorf("expected 3 messages in repo, got %d", len(messages))
	}
}

func TestSyncMessages_Execute_EmptyBatch(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	// Create input with empty messages array
	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages:       []ports.SyncMessageItem{},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify output
	if output == nil {
		t.Fatal("expected output, got nil")
	}

	// Verify empty results
	if len(output.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(output.Results))
	}

	// Verify SyncedAt is still set
	if output.SyncedAt.IsZero() {
		t.Error("expected SyncedAt to be set even for empty batch")
	}
}

func TestSyncMessages_Execute_ConversationNotFound(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Note: No conversation created

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "nonexistent_conv",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:  "local_msg_1",
				Role:     "user",
				Contents: "Hello",
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify error
	if err == nil {
		t.Fatal("expected error for nonexistent conversation, got nil")
	}

	// Verify error wraps the correct domain error
	if !errors.Is(err, domain.ErrConversationNotFound) {
		t.Errorf("expected error to wrap ErrConversationNotFound, got: %v", err)
	}

	// Verify no output
	if output != nil {
		t.Errorf("expected nil output, got: %v", output)
	}
}

func TestSyncMessages_Execute_DuplicateMessage(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	// Create an existing message with the same LocalID and content
	existingMsg := &models.Message{
		ID:               "msg_existing",
		LocalID:          "local_msg_1",
		ServerID:         "msg_existing",
		ConversationID:   "conv_123",
		SequenceNumber:   1,
		Role:             models.MessageRoleUser,
		Contents:         "Hello, this is my message",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now().Add(-1 * time.Hour),
		UpdatedAt:        time.Now().Add(-1 * time.Hour),
	}
	msgRepo.AddMessage(existingMsg)

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	// Try to sync a message with the same LocalID and same content (duplicate)
	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:        "local_msg_1",
				SequenceNumber: 1,
				Role:           "user",
				Contents:       "Hello, this is my message", // Same content as existing
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify output
	if output == nil {
		t.Fatal("expected output, got nil")
	}

	// Verify the result indicates "synced" (not "conflict" because content is same)
	if len(output.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(output.Results))
	}

	result := output.Results[0]
	if result.Status != "synced" {
		t.Errorf("expected status 'synced' for duplicate, got '%s'", result.Status)
	}

	// Verify it returns the existing server ID
	if result.ServerID != "msg_existing" {
		t.Errorf("expected ServerID 'msg_existing', got '%s'", result.ServerID)
	}

	// Verify the existing message is returned
	if result.Message == nil {
		t.Error("expected Message to be set")
	}
	if result.Message.ID != "msg_existing" {
		t.Errorf("expected existing message ID, got '%s'", result.Message.ID)
	}
}

func TestSyncMessages_Execute_ConflictDetection(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	// Create an existing message with the same LocalID but DIFFERENT content
	existingMsg := &models.Message{
		ID:               "msg_existing",
		LocalID:          "local_msg_1",
		ServerID:         "msg_existing",
		ConversationID:   "conv_123",
		SequenceNumber:   1,
		Role:             models.MessageRoleUser,
		Contents:         "Original content from server",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now().Add(-1 * time.Hour),
		UpdatedAt:        time.Now().Add(-1 * time.Hour),
	}
	msgRepo.AddMessage(existingMsg)

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	// Try to sync a message with the same LocalID but different content
	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:        "local_msg_1",
				SequenceNumber: 1,
				Role:           "user",
				Contents:       "Different content from client", // Different from existing
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify no error (conflicts are not errors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify output
	if output == nil {
		t.Fatal("expected output, got nil")
	}

	// Verify the result indicates conflict
	if len(output.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(output.Results))
	}

	result := output.Results[0]
	if result.Status != "conflict" {
		t.Errorf("expected status 'conflict', got '%s'", result.Status)
	}

	// Verify server ID is returned
	if result.ServerID != "msg_existing" {
		t.Errorf("expected ServerID 'msg_existing', got '%s'", result.ServerID)
	}

	// Verify the message was marked as conflict in the repository
	storedMsg, _ := msgRepo.GetByID(context.Background(), "msg_existing")
	if storedMsg == nil {
		t.Fatal("expected message to exist in repository")
	}
	if storedMsg.SyncStatus != models.SyncStatusConflict {
		t.Errorf("expected message SyncStatus to be 'conflict', got '%s'", storedMsg.SyncStatus)
	}
}

func TestSyncMessages_Execute_DatabaseError(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	// Configure message repo to fail on Create
	dbError := errors.New("database connection lost")
	msgRepo.createFunc = func(ctx context.Context, msg *models.Message) error {
		return dbError
	}

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:        "local_msg_1",
				SequenceNumber: 1,
				Role:           "user",
				Contents:       "Hello",
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify error
	if err == nil {
		t.Fatal("expected error for database failure, got nil")
	}

	// Verify transaction was rolled back
	if !txManager.rollbackCalled {
		t.Error("expected transaction to be rolled back")
	}

	// Verify no output
	if output != nil {
		t.Errorf("expected nil output, got: %v", output)
	}
}

func TestSyncMessages_Execute_PartialFailure(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	// Configure message repo to fail on the second message
	callCount := 0
	msgRepo.createFunc = func(ctx context.Context, msg *models.Message) error {
		callCount++
		if callCount == 2 {
			return errors.New("database write failed for second message")
		}
		// Actually store the message for successful calls
		msgRepo.mu.Lock()
		defer msgRepo.mu.Unlock()
		msgRepo.store[msg.ID] = msgRepo.copyMessage(msg)
		return nil
	}

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:        "local_msg_1",
				SequenceNumber: 1,
				Role:           "user",
				Contents:       "First message - should succeed",
			},
			{
				LocalID:        "local_msg_2",
				SequenceNumber: 2,
				Role:           "user",
				Contents:       "Second message - will fail",
			},
			{
				LocalID:        "local_msg_3",
				SequenceNumber: 3,
				Role:           "user",
				Contents:       "Third message - after failure",
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify error occurred (partial failure triggers rollback)
	if err == nil {
		t.Fatal("expected error for partial failure, got nil")
	}

	// Verify transaction was rolled back
	if !txManager.rollbackCalled {
		t.Error("expected transaction to be rolled back on partial failure")
	}

	// Verify no output (transaction failed)
	if output != nil {
		t.Errorf("expected nil output on transaction failure, got: %v", output)
	}
}

func TestSyncMessages_Execute_NilInput(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	// Execute with nil input
	output, err := uc.Execute(context.Background(), nil)

	// Verify error
	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}

	if output != nil {
		t.Errorf("expected nil output, got: %v", output)
	}
}

func TestSyncMessages_Execute_EmptyConversationID(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "", // Empty conversation ID
		Messages: []ports.SyncMessageItem{
			{
				LocalID:  "local_msg_1",
				Role:     "user",
				Contents: "Hello",
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify error
	if err == nil {
		t.Fatal("expected error for empty conversation ID, got nil")
	}

	if output != nil {
		t.Errorf("expected nil output, got: %v", output)
	}
}

func TestSyncMessages_Execute_InvalidMessageMissingLocalID(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:  "", // Missing LocalID
				Role:     "user",
				Contents: "Hello",
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify no error (validation issues result in conflict status, not error)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify result indicates conflict due to validation
	if len(output.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(output.Results))
	}

	if output.Results[0].Status != "conflict" {
		t.Errorf("expected status 'conflict' for invalid message, got '%s'", output.Results[0].Status)
	}
}

func TestSyncMessages_Execute_InvalidMessageMissingRole(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:  "local_msg_1",
				Role:     "", // Missing Role
				Contents: "Hello",
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify no error (validation issues result in conflict status)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify result indicates conflict due to validation
	if len(output.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(output.Results))
	}

	if output.Results[0].Status != "conflict" {
		t.Errorf("expected status 'conflict' for message with missing role, got '%s'", output.Results[0].Status)
	}
}

func TestSyncMessages_Execute_GetByLocalIDError(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	// Configure GetByLocalID to return an unexpected error (not NotFound)
	dbError := errors.New("database query failed")
	msgRepo.getByLocalIDFunc = func(ctx context.Context, localID string) (*models.Message, error) {
		return nil, dbError
	}

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:  "local_msg_1",
				Role:     "user",
				Contents: "Hello",
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify error
	if err == nil {
		t.Fatal("expected error for database failure, got nil")
	}

	// Verify transaction rolled back
	if !txManager.rollbackCalled {
		t.Error("expected transaction to be rolled back")
	}

	if output != nil {
		t.Errorf("expected nil output, got: %v", output)
	}
}

func TestSyncMessages_Execute_UpdateConflictError(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	// Create an existing message with different content
	existingMsg := &models.Message{
		ID:               "msg_existing",
		LocalID:          "local_msg_1",
		ServerID:         "msg_existing",
		ConversationID:   "conv_123",
		SequenceNumber:   1,
		Role:             models.MessageRoleUser,
		Contents:         "Original content",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
	}
	msgRepo.AddMessage(existingMsg)

	// Configure Update to fail
	dbError := errors.New("update failed")
	msgRepo.updateFunc = func(ctx context.Context, msg *models.Message) error {
		return dbError
	}

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:  "local_msg_1",
				Role:     "user",
				Contents: "Different content", // Will trigger conflict update
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify error
	if err == nil {
		t.Fatal("expected error when update fails, got nil")
	}

	// Verify transaction rolled back
	if !txManager.rollbackCalled {
		t.Error("expected transaction to be rolled back")
	}

	if output != nil {
		t.Errorf("expected nil output, got: %v", output)
	}
}

func TestSyncMessages_Execute_DefaultTimestamps(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	// Create input without timestamps (zero values)
	beforeExec := time.Now().UTC()
	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:  "local_msg_1",
				Role:     "user",
				Contents: "Hello",
				// CreatedAt and UpdatedAt are zero values
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)
	afterExec := time.Now().UTC()

	// Verify no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify message has default timestamps
	if len(output.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(output.Results))
	}

	msg := output.Results[0].Message
	if msg == nil {
		t.Fatal("expected message to be set")
	}

	// Verify CreatedAt is set to a reasonable time (between before and after)
	if msg.CreatedAt.Before(beforeExec) || msg.CreatedAt.After(afterExec) {
		t.Errorf("expected CreatedAt to be between %v and %v, got %v", beforeExec, afterExec, msg.CreatedAt)
	}

	// Verify UpdatedAt defaults to CreatedAt when not provided
	if !msg.UpdatedAt.Equal(msg.CreatedAt) && msg.UpdatedAt.Before(msg.CreatedAt) {
		t.Errorf("expected UpdatedAt >= CreatedAt, got UpdatedAt=%v CreatedAt=%v", msg.UpdatedAt, msg.CreatedAt)
	}
}

func TestSyncMessages_Execute_PgxNotFoundError(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	// Configure GetByLocalID to return pgx.ErrNoRows (should be treated as not found)
	msgRepo.getByLocalIDFunc = func(ctx context.Context, localID string) (*models.Message, error) {
		return nil, pgx.ErrNoRows
	}

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:  "local_msg_1",
				Role:     "user",
				Contents: "Hello",
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify no error (pgx.ErrNoRows should be handled as "not found", creating new message)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify message was created
	if len(output.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(output.Results))
	}

	if output.Results[0].Status != "synced" {
		t.Errorf("expected status 'synced', got '%s'", output.Results[0].Status)
	}
}

func TestSyncMessages_Execute_MixedResults(t *testing.T) {
	// Setup: Test a batch with new, duplicate, and conflict messages
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	// Create an existing message (will be duplicate when synced with same content)
	duplicateMsg := &models.Message{
		ID:               "msg_dup",
		LocalID:          "local_dup",
		ServerID:         "msg_dup",
		ConversationID:   "conv_123",
		SequenceNumber:   1,
		Role:             models.MessageRoleUser,
		Contents:         "Duplicate content",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
	}
	msgRepo.AddMessage(duplicateMsg)

	// Create another existing message (will conflict when synced with different content)
	conflictMsg := &models.Message{
		ID:               "msg_conflict",
		LocalID:          "local_conflict",
		ServerID:         "msg_conflict",
		ConversationID:   "conv_123",
		SequenceNumber:   2,
		Role:             models.MessageRoleUser,
		Contents:         "Original conflict content",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
	}
	msgRepo.AddMessage(conflictMsg)

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:        "local_new",
				SequenceNumber: 3,
				Role:           "user",
				Contents:       "Brand new message",
			},
			{
				LocalID:        "local_dup",
				SequenceNumber: 1,
				Role:           "user",
				Contents:       "Duplicate content", // Same as existing
			},
			{
				LocalID:        "local_conflict",
				SequenceNumber: 2,
				Role:           "user",
				Contents:       "Modified conflict content", // Different from existing
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all results
	if len(output.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(output.Results))
	}

	// Check each result type
	resultMap := make(map[string]ports.SyncedMessageResult)
	for _, r := range output.Results {
		resultMap[r.LocalID] = r
	}

	// New message should be synced
	if newResult, ok := resultMap["local_new"]; !ok {
		t.Error("missing result for local_new")
	} else if newResult.Status != "synced" {
		t.Errorf("expected local_new status 'synced', got '%s'", newResult.Status)
	}

	// Duplicate should be synced (same content)
	if dupResult, ok := resultMap["local_dup"]; !ok {
		t.Error("missing result for local_dup")
	} else if dupResult.Status != "synced" {
		t.Errorf("expected local_dup status 'synced', got '%s'", dupResult.Status)
	}

	// Conflict should be marked as conflict
	if conflictResult, ok := resultMap["local_conflict"]; !ok {
		t.Error("missing result for local_conflict")
	} else if conflictResult.Status != "conflict" {
		t.Errorf("expected local_conflict status 'conflict', got '%s'", conflictResult.Status)
	}
}

func TestSyncMessages_Execute_VerifyMessageFields(t *testing.T) {
	// Setup
	msgRepo := newSyncMockMessageRepo()
	convRepo := newSyncMockConversationRepo()
	idGen := newSyncMockIDGenerator()
	txManager := newSyncMockTransactionManager()

	// Create a conversation
	conv := models.NewConversation("conv_123", "user_1", "")
	convRepo.Create(context.Background(), conv)

	uc := NewSyncMessages(msgRepo, convRepo, idGen, txManager)

	createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)

	input := &ports.SyncMessagesInput{
		ConversationID: "conv_123",
		Messages: []ports.SyncMessageItem{
			{
				LocalID:        "local_msg_1",
				SequenceNumber: 5,
				PreviousID:     "prev_msg",
				Role:           "assistant",
				Contents:       "Test content",
				CreatedAt:      createdAt,
				UpdatedAt:      updatedAt,
			},
		},
	}

	// Execute
	output, err := uc.Execute(context.Background(), input)

	// Verify no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify message fields
	msg := output.Results[0].Message
	if msg == nil {
		t.Fatal("expected message to be set")
	}

	// Verify all fields are correctly set
	if msg.LocalID != "local_msg_1" {
		t.Errorf("expected LocalID 'local_msg_1', got '%s'", msg.LocalID)
	}
	if msg.SequenceNumber != 5 {
		t.Errorf("expected SequenceNumber 5, got %d", msg.SequenceNumber)
	}
	if msg.PreviousID != "prev_msg" {
		t.Errorf("expected PreviousID 'prev_msg', got '%s'", msg.PreviousID)
	}
	if msg.Role != models.MessageRoleAssistant {
		t.Errorf("expected Role 'assistant', got '%s'", msg.Role)
	}
	if msg.Contents != "Test content" {
		t.Errorf("expected Contents 'Test content', got '%s'", msg.Contents)
	}
	if msg.ConversationID != "conv_123" {
		t.Errorf("expected ConversationID 'conv_123', got '%s'", msg.ConversationID)
	}
	if msg.SyncStatus != models.SyncStatusSynced {
		t.Errorf("expected SyncStatus 'synced', got '%s'", msg.SyncStatus)
	}
	if msg.CompletionStatus != models.CompletionStatusCompleted {
		t.Errorf("expected CompletionStatus 'completed', got '%s'", msg.CompletionStatus)
	}
	if msg.ServerID == "" {
		t.Error("expected ServerID to be set")
	}
	if msg.SyncedAt == nil {
		t.Error("expected SyncedAt to be set")
	}
	if !msg.CreatedAt.Equal(createdAt) {
		t.Errorf("expected CreatedAt %v, got %v", createdAt, msg.CreatedAt)
	}
	if !msg.UpdatedAt.Equal(updatedAt) {
		t.Errorf("expected UpdatedAt %v, got %v", updatedAt, msg.UpdatedAt)
	}
}
