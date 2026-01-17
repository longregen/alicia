package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// mockMessageRepoForUpvote implements ports.MessageRepository for testing
type mockMessageRepoForUpvote struct {
	messages       map[string]*models.Message
	byConversation map[string][]*models.Message
	getByIDError   error
	getByConvError error
}

func newMockMessageRepoForUpvote() *mockMessageRepoForUpvote {
	return &mockMessageRepoForUpvote{
		messages:       make(map[string]*models.Message),
		byConversation: make(map[string][]*models.Message),
	}
}

func (m *mockMessageRepoForUpvote) Create(ctx context.Context, message *models.Message) error {
	m.messages[message.ID] = message
	m.byConversation[message.ConversationID] = append(m.byConversation[message.ConversationID], message)
	return nil
}

func (m *mockMessageRepoForUpvote) GetByID(ctx context.Context, id string) (*models.Message, error) {
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	msg, ok := m.messages[id]
	if !ok {
		return nil, errors.New("message not found")
	}
	return msg, nil
}

func (m *mockMessageRepoForUpvote) Update(ctx context.Context, message *models.Message) error {
	m.messages[message.ID] = message
	return nil
}

func (m *mockMessageRepoForUpvote) Delete(ctx context.Context, id string) error {
	delete(m.messages, id)
	return nil
}

func (m *mockMessageRepoForUpvote) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	if m.getByConvError != nil {
		return nil, m.getByConvError
	}
	return m.byConversation[conversationID], nil
}

func (m *mockMessageRepoForUpvote) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	if m.getByConvError != nil {
		return nil, m.getByConvError
	}
	msgs := m.byConversation[conversationID]
	if len(msgs) <= limit {
		return msgs, nil
	}
	return msgs[len(msgs)-limit:], nil
}

func (m *mockMessageRepoForUpvote) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	return len(m.byConversation[conversationID]) + 1, nil
}

func (m *mockMessageRepoForUpvote) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForUpvote) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForUpvote) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForUpvote) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForUpvote) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForUpvote) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForUpvote) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForUpvote) DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error {
	return nil
}

// mockVoteRepoForUpvote implements ports.VoteRepository for testing
type mockVoteRepoForUpvote struct {
	aggregates       *models.VoteAggregates
	getAggregatesErr error
}

func newMockVoteRepoForUpvote() *mockVoteRepoForUpvote {
	return &mockVoteRepoForUpvote{
		aggregates: &models.VoteAggregates{
			Upvotes:   1,
			Downvotes: 0,
			NetScore:  1,
		},
	}
}

func (m *mockVoteRepoForUpvote) Create(ctx context.Context, vote *models.Vote) error {
	return nil
}

func (m *mockVoteRepoForUpvote) Delete(ctx context.Context, targetType string, targetID string) error {
	return nil
}

func (m *mockVoteRepoForUpvote) GetByTarget(ctx context.Context, targetType string, targetID string) ([]*models.Vote, error) {
	return nil, nil
}

func (m *mockVoteRepoForUpvote) GetByMessage(ctx context.Context, messageID string) ([]*models.Vote, error) {
	return nil, nil
}

func (m *mockVoteRepoForUpvote) GetAggregates(ctx context.Context, targetType string, targetID string) (*models.VoteAggregates, error) {
	if m.getAggregatesErr != nil {
		return nil, m.getAggregatesErr
	}
	return m.aggregates, nil
}

func (m *mockVoteRepoForUpvote) GetToolUseVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithToolContext, error) {
	return nil, nil
}

func (m *mockVoteRepoForUpvote) GetMemoryVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	return nil, nil
}

func (m *mockVoteRepoForUpvote) GetMemoryUsageVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	return nil, nil
}

func (m *mockVoteRepoForUpvote) GetMemoryExtractionVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithExtractionContext, error) {
	return nil, nil
}

func (m *mockVoteRepoForUpvote) CountByTargetType(ctx context.Context, targetType string) (int, error) {
	return 0, nil
}

func TestMemorizeFromUpvote_OnlyProcessesUpvotes(t *testing.T) {
	messageRepo := newMockMessageRepoForUpvote()
	voteRepo := newMockVoteRepoForUpvote()

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
	messageRepo := newMockMessageRepoForUpvote()
	voteRepo := newMockVoteRepoForUpvote()
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
	messageRepo := newMockMessageRepoForUpvote()
	voteRepo := newMockVoteRepoForUpvote()

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
	messageRepo := newMockMessageRepoForUpvote()
	messageRepo.getByIDError = errors.New("message not found")
	voteRepo := newMockVoteRepoForUpvote()

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
	messageRepo := newMockMessageRepoForUpvote()
	voteRepo := newMockVoteRepoForUpvote()
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
	messageRepo := newMockMessageRepoForUpvote()

	// Add test message with meaningful content
	msg := models.NewMessage("msg_123", "conv_456", 1, models.MessageRoleAssistant,
		"The user mentioned their favorite programming language is Rust and they work at Acme Corp in Seattle.")
	messageRepo.messages["msg_123"] = msg
	messageRepo.byConversation["conv_456"] = []*models.Message{msg}

	voteRepo := newMockVoteRepoForUpvote()

	// Create mock memory service that captures created memories
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	llmService.chatResponse = &ports.LLMResponse{
		Content: `{
			"extracted_facts": ["User's favorite programming language is Rust", "User works at Acme Corp in Seattle"],
			"importance_scores": [0.7, 0.8],
			"extraction_reasoning": "Extracted user preferences and work information"
		}`,
	}
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
	messageRepo := newMockMessageRepoForUpvote()

	// Add test messages
	msg1 := models.NewMessage("msg_1", "conv_456", 1, models.MessageRoleUser, "I'm working on a Go project")
	msg2 := models.NewMessage("msg_2", "conv_456", 2, models.MessageRoleAssistant, "Great! Go is excellent for backend services.")
	messageRepo.messages["msg_1"] = msg1
	messageRepo.messages["msg_2"] = msg2
	messageRepo.byConversation["conv_456"] = []*models.Message{msg1, msg2}

	voteRepo := newMockVoteRepoForUpvote()

	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	llmService.chatResponse = &ports.LLMResponse{
		Content: `{
			"extracted_facts": ["User is working on a Go project"],
			"importance_scores": [0.7],
			"extraction_reasoning": "Extracted project information"
		}`,
	}
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
