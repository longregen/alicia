package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Helper function to add user context to requests
func addUserContext(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	return req.WithContext(ctx)
}

// setTestUserID is an alias for addUserContext for compatibility
func setTestUserID(req *http.Request, userID string) *http.Request {
	return addUserContext(req, userID)
}

// setURLParam adds a URL parameter to the request context (chi router style)
func setURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// MockIDGenerator is a mock ID generator for testing
type MockIDGenerator struct {
	counter int
}

func (m *MockIDGenerator) nextID(prefix string) string {
	m.counter++
	return fmt.Sprintf("%s_test_%d", prefix, m.counter)
}

func (m *MockIDGenerator) GenerateConversationID() string        { return m.nextID("conv") }
func (m *MockIDGenerator) GenerateMessageID() string             { return m.nextID("msg") }
func (m *MockIDGenerator) GenerateSentenceID() string            { return m.nextID("sent") }
func (m *MockIDGenerator) GenerateAudioID() string               { return m.nextID("audio") }
func (m *MockIDGenerator) GenerateMemoryID() string              { return m.nextID("mem") }
func (m *MockIDGenerator) GenerateMemoryUsageID() string         { return m.nextID("memuse") }
func (m *MockIDGenerator) GenerateToolID() string                { return m.nextID("tool") }
func (m *MockIDGenerator) GenerateToolUseID() string             { return m.nextID("tooluse") }
func (m *MockIDGenerator) GenerateReasoningStepID() string       { return m.nextID("reason") }
func (m *MockIDGenerator) GenerateCommentaryID() string          { return m.nextID("comment") }
func (m *MockIDGenerator) GenerateMetaID() string                { return m.nextID("meta") }
func (m *MockIDGenerator) GenerateMCPServerID() string           { return m.nextID("mcp") }
func (m *MockIDGenerator) GenerateVoteID() string                { return m.nextID("vote") }
func (m *MockIDGenerator) GenerateNoteID() string                { return m.nextID("note") }
func (m *MockIDGenerator) GenerateSessionStatsID() string        { return m.nextID("stats") }
func (m *MockIDGenerator) GenerateOptimizationRunID() string     { return m.nextID("opt") }
func (m *MockIDGenerator) GeneratePromptCandidateID() string     { return m.nextID("cand") }
func (m *MockIDGenerator) GeneratePromptEvaluationID() string    { return m.nextID("eval") }
func (m *MockIDGenerator) GenerateSystemPromptVersionID() string { return m.nextID("spv") }
func (m *MockIDGenerator) GenerateTrainingExampleID() string     { return m.nextID("tex") }

// MockVoteRepository is a mock vote repository for testing
type MockVoteRepository struct {
	votes map[string]*models.Vote
}

func (m *MockVoteRepository) Create(ctx context.Context, vote *models.Vote) error {
	if m.votes == nil {
		m.votes = make(map[string]*models.Vote)
	}
	m.votes[vote.TargetID] = vote
	return nil
}

func (m *MockVoteRepository) Delete(ctx context.Context, targetType string, targetID string) error {
	if m.votes != nil {
		delete(m.votes, targetID)
	}
	return nil
}

func (m *MockVoteRepository) GetByTarget(ctx context.Context, targetType string, targetID string) ([]*models.Vote, error) {
	if m.votes == nil {
		return nil, nil
	}
	if v, ok := m.votes[targetID]; ok {
		return []*models.Vote{v}, nil
	}
	return nil, nil
}

func (m *MockVoteRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Vote, error) {
	return nil, nil
}

func (m *MockVoteRepository) GetAggregates(ctx context.Context, targetType string, targetID string) (*models.VoteAggregates, error) {
	return &models.VoteAggregates{
		TargetType: targetType,
		TargetID:   targetID,
		Upvotes:    0,
		Downvotes:  0,
	}, nil
}

func (m *MockVoteRepository) CountByTargetType(ctx context.Context, targetType string) (int, error) {
	return 0, nil
}

func (m *MockVoteRepository) GetMemoryExtractionVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithExtractionContext, error) {
	return nil, nil
}

func (m *MockVoteRepository) GetToolUseVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithToolContext, error) {
	return nil, nil
}

func (m *MockVoteRepository) GetMemoryVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	return nil, nil
}

func (m *MockVoteRepository) GetMemoryUsageVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	return nil, nil
}
