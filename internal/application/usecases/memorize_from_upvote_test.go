package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// mockMessageRepository implements ports.MessageRepository for testing
type mockMessageRepository struct {
	messages       map[string]*models.Message
	byConversation map[string][]*models.Message
	getByIDError   error
	getByConvError error
}

func newMockMessageRepository() *mockMessageRepository {
	return &mockMessageRepository{
		messages:       make(map[string]*models.Message),
		byConversation: make(map[string][]*models.Message),
	}
}

func (m *mockMessageRepository) Create(ctx context.Context, message *models.Message) error {
	m.messages[message.ID] = message
	m.byConversation[message.ConversationID] = append(m.byConversation[message.ConversationID], message)
	return nil
}

func (m *mockMessageRepository) GetByID(ctx context.Context, id string) (*models.Message, error) {
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	msg, ok := m.messages[id]
	if !ok {
		return nil, errors.New("message not found")
	}
	return msg, nil
}

func (m *mockMessageRepository) Update(ctx context.Context, message *models.Message) error {
	m.messages[message.ID] = message
	return nil
}

func (m *mockMessageRepository) Delete(ctx context.Context, id string) error {
	delete(m.messages, id)
	return nil
}

func (m *mockMessageRepository) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	if m.getByConvError != nil {
		return nil, m.getByConvError
	}
	return m.byConversation[conversationID], nil
}

func (m *mockMessageRepository) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	if m.getByConvError != nil {
		return nil, m.getByConvError
	}
	msgs := m.byConversation[conversationID]
	if len(msgs) <= limit {
		return msgs, nil
	}
	return msgs[len(msgs)-limit:], nil
}

func (m *mockMessageRepository) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	return len(m.byConversation[conversationID]) + 1, nil
}

func (m *mockMessageRepository) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepository) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepository) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepository) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepository) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepository) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepository) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepository) DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error {
	return nil
}

// mockVoteRepository implements ports.VoteRepository for testing
type mockVoteRepository struct {
	aggregates        *models.VoteAggregates
	getAggregatesErr  error
}

func newMockVoteRepository() *mockVoteRepository {
	return &mockVoteRepository{
		aggregates: &models.VoteAggregates{
			Upvotes:   1,
			Downvotes: 0,
			NetScore:  1,
		},
	}
}

func (m *mockVoteRepository) Create(ctx context.Context, vote *models.Vote) error {
	return nil
}

func (m *mockVoteRepository) Delete(ctx context.Context, targetType string, targetID string) error {
	return nil
}

func (m *mockVoteRepository) GetByTarget(ctx context.Context, targetType string, targetID string) ([]*models.Vote, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Vote, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetAggregates(ctx context.Context, targetType string, targetID string) (*models.VoteAggregates, error) {
	if m.getAggregatesErr != nil {
		return nil, m.getAggregatesErr
	}
	return m.aggregates, nil
}

func (m *mockVoteRepository) GetToolUseVotesWithContext(ctx context.Context, limit int) (interface{}, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetMemoryVotesWithContext(ctx context.Context, limit int) (interface{}, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetMemoryUsageVotesWithContext(ctx context.Context, limit int) (interface{}, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetMemoryExtractionVotesWithContext(ctx context.Context, limit int) (interface{}, error) {
	return nil, nil
}

func (m *mockVoteRepository) CountByTargetType(ctx context.Context, targetType string) (int, error) {
	return 0, nil
}

func TestMemorizeFromUpvote_OnlyProcessesUpvotes(t *testing.T) {
	messageRepo := newMockMessageRepository()
	voteRepo := newMockVoteRepository()

	// Create mock ExtractMemories with nil dependencies (won't be called for downvotes)
	extractMemories := NewExtractMemories(nil, nil, nil)

	uc := NewMemorizeFromUpvote(messageRepo, nil, voteRepo, extractMemories)

	// Test with downvote
	output, err := uc.Execute(context.Background(), &MemorizeFromUpvoteInput{
		TargetType: "message",
		TargetID:   "msg_123",
		Vote:       -1, // downvote
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.MemoriesCreated != 0 {
		t.Errorf("expected 0 memories created for downvote, got %d", output.MemoriesCreated)
	}

	if output.Reasoning != "Only upvotes trigger memory extraction" {
		t.Errorf("unexpected reasoning: %s", output.Reasoning)
	}
}

func TestMemorizeFromUpvote_RespectsMinUpvotesThreshold(t *testing.T) {
	messageRepo := newMockMessageRepository()
	voteRepo := newMockVoteRepository()
	voteRepo.aggregates = &models.VoteAggregates{
		Upvotes:   1,
		Downvotes: 1,
		NetScore:  0, // Below threshold
	}

	extractMemories := NewExtractMemories(nil, nil, nil)
	uc := NewMemorizeFromUpvote(messageRepo, nil, voteRepo, extractMemories)

	output, err := uc.Execute(context.Background(), &MemorizeFromUpvoteInput{
		TargetType: "message",
		TargetID:   "msg_123",
		Vote:       1,
		MinUpvotes: 1,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.MemoriesCreated != 0 {
		t.Errorf("expected 0 memories when below threshold, got %d", output.MemoriesCreated)
	}

	if output.Reasoning == "" {
		t.Error("expected reasoning to explain threshold not met")
	}
}

func TestMemorizeFromUpvote_HandlesUnsupportedTargetType(t *testing.T) {
	messageRepo := newMockMessageRepository()
	voteRepo := newMockVoteRepository()

	extractMemories := NewExtractMemories(nil, nil, nil)
	uc := NewMemorizeFromUpvote(messageRepo, nil, voteRepo, extractMemories)

	output, err := uc.Execute(context.Background(), &MemorizeFromUpvoteInput{
		TargetType: "tool_use", // Not supported for memory extraction
		TargetID:   "tu_123",
		Vote:       1,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.MemoriesCreated != 0 {
		t.Errorf("expected 0 memories for unsupported target type, got %d", output.MemoriesCreated)
	}

	if output.Reasoning == "" {
		t.Error("expected reasoning to explain unsupported target type")
	}
}

func TestMemorizeFromUpvote_HandlesMessageNotFound(t *testing.T) {
	messageRepo := newMockMessageRepository()
	messageRepo.getByIDError = errors.New("message not found")
	voteRepo := newMockVoteRepository()

	extractMemories := NewExtractMemories(nil, nil, nil)
	uc := NewMemorizeFromUpvote(messageRepo, nil, voteRepo, extractMemories)

	_, err := uc.Execute(context.Background(), &MemorizeFromUpvoteInput{
		TargetType: "message",
		TargetID:   "msg_nonexistent",
		Vote:       1,
	})

	if err == nil {
		t.Error("expected error when message not found")
	}
}

func TestMemorizeFromUpvote_HandlesVoteAggregatesError(t *testing.T) {
	messageRepo := newMockMessageRepository()
	voteRepo := newMockVoteRepository()
	voteRepo.getAggregatesErr = errors.New("database error")

	extractMemories := NewExtractMemories(nil, nil, nil)
	uc := NewMemorizeFromUpvote(messageRepo, nil, voteRepo, extractMemories)

	_, err := uc.Execute(context.Background(), &MemorizeFromUpvoteInput{
		TargetType: "message",
		TargetID:   "msg_123",
		Vote:       1,
	})

	if err == nil {
		t.Error("expected error when vote aggregates fail")
	}
}

func TestMemorizeFromUpvote_ExtractsFromMessage(t *testing.T) {
	messageRepo := newMockMessageRepository()

	// Add test message with meaningful content
	msg := models.NewMessage("msg_123", "conv_456", 1, models.MessageRoleAssistant,
		"The user mentioned their favorite programming language is Rust and they work at Acme Corp in Seattle.")
	messageRepo.messages["msg_123"] = msg
	messageRepo.byConversation["conv_456"] = []*models.Message{msg}

	voteRepo := newMockVoteRepository()

	// Create mock memory service that captures created memories
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	llmService.response = `{
		"extracted_facts": ["User's favorite programming language is Rust", "User works at Acme Corp in Seattle"],
		"importance_scores": [0.7, 0.8],
		"extraction_reasoning": "Extracted user preferences and work information"
	}`
	idGen := newMockIDGeneratorForExtract()

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromUpvote(messageRepo, nil, voteRepo, extractMemories)

	output, err := uc.Execute(context.Background(), &MemorizeFromUpvoteInput{
		TargetType: "message",
		TargetID:   "msg_123",
		Vote:       1,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.MemoriesCreated != 2 {
		t.Errorf("expected 2 memories created, got %d", output.MemoriesCreated)
	}
}

func TestMemorizeFromUpvote_ExtractsFromConversation(t *testing.T) {
	messageRepo := newMockMessageRepository()

	// Add test messages
	msg1 := models.NewMessage("msg_1", "conv_456", 1, models.MessageRoleUser, "I'm working on a Go project")
	msg2 := models.NewMessage("msg_2", "conv_456", 2, models.MessageRoleAssistant, "Great! Go is excellent for backend services.")
	messageRepo.messages["msg_1"] = msg1
	messageRepo.messages["msg_2"] = msg2
	messageRepo.byConversation["conv_456"] = []*models.Message{msg1, msg2}

	voteRepo := newMockVoteRepository()

	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	llmService.response = `{
		"extracted_facts": ["User is working on a Go project"],
		"importance_scores": [0.7],
		"extraction_reasoning": "Extracted project information"
	}`
	idGen := newMockIDGeneratorForExtract()

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromUpvote(messageRepo, nil, voteRepo, extractMemories)

	output, err := uc.Execute(context.Background(), &MemorizeFromUpvoteInput{
		TargetType: "conversation",
		TargetID:   "conv_456",
		Vote:       1,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.MemoriesCreated != 1 {
		t.Errorf("expected 1 memory created, got %d", output.MemoriesCreated)
	}
}
