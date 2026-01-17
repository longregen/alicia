package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Mock VoteRepository
type mockVoteRepo struct {
	votes      map[string]*models.Vote
	createErr  error
	getErr     error
	deleteErr  error
	aggregates *models.VoteAggregates
}

func newMockVoteRepo() *mockVoteRepo {
	return &mockVoteRepo{
		votes: make(map[string]*models.Vote),
		aggregates: &models.VoteAggregates{
			Upvotes:   0,
			Downvotes: 0,
		},
	}
}

func (m *mockVoteRepo) Create(ctx context.Context, vote *models.Vote) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.votes[vote.ID] = vote
	return nil
}

func (m *mockVoteRepo) Delete(ctx context.Context, targetType string, targetID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for id, vote := range m.votes {
		if vote.TargetID == targetID && vote.TargetType == targetType {
			delete(m.votes, id)
			break
		}
	}
	return nil
}

func (m *mockVoteRepo) GetAggregates(ctx context.Context, targetType string, targetID string) (*models.VoteAggregates, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.aggregates, nil
}

func (m *mockVoteRepo) GetByTarget(ctx context.Context, targetType string, targetID string) ([]*models.Vote, error) {
	var votes []*models.Vote
	for _, vote := range m.votes {
		if vote.TargetType == targetType && vote.TargetID == targetID {
			votes = append(votes, vote)
		}
	}
	return votes, nil
}

func (m *mockVoteRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.Vote, error) {
	return m.GetByTarget(ctx, "message", messageID)
}

func (m *mockVoteRepo) CountByTargetType(ctx context.Context, targetType string) (int, error) {
	return 0, nil
}

func (m *mockVoteRepo) GetMemoryExtractionVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithExtractionContext, error) {
	return nil, nil
}

func (m *mockVoteRepo) GetToolUseVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithToolContext, error) {
	return nil, nil
}

func (m *mockVoteRepo) GetMemoryVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	return nil, nil
}

func (m *mockVoteRepo) GetMemoryUsageVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	return nil, nil
}

// Tests for VoteHandler.VoteOnMessage

func TestVoteHandler_VoteOnMessage_UpvoteSuccess(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 1, Downvotes: 0}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"vote": "up"}`
	req := httptest.NewRequest("POST", "/api/v1/messages/am_test123/vote", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.VoteOnMessage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Upvotes != 1 {
		t.Errorf("expected 1 upvote, got %d", response.Upvotes)
	}

	if *response.UserVote != "up" {
		t.Errorf("expected user vote 'up', got %v", *response.UserVote)
	}
}

func TestVoteHandler_VoteOnMessage_DownvoteSuccess(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 0, Downvotes: 1}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"vote": "down"}`
	req := httptest.NewRequest("POST", "/api/v1/messages/am_test123/vote", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.VoteOnMessage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Downvotes != 1 {
		t.Errorf("expected 1 downvote, got %d", response.Downvotes)
	}
}

func TestVoteHandler_VoteOnMessage_InvalidVote(t *testing.T) {
	voteRepo := newMockVoteRepo()
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"vote": "invalid"}`
	req := httptest.NewRequest("POST", "/api/v1/messages/am_test123/vote", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.VoteOnMessage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// Tests for VoteHandler.RemoveMessageVote

func TestVoteHandler_RemoveMessageVote_Success(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 0, Downvotes: 0}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	req := httptest.NewRequest("DELETE", "/api/v1/messages/am_test123/vote", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.RemoveMessageVote(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.UserVote != nil {
		t.Errorf("expected user vote to be nil, got %v", response.UserVote)
	}
}

// Tests for VoteHandler.GetMessageVotes

func TestVoteHandler_GetMessageVotes_Success(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 5, Downvotes: 2}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	req := httptest.NewRequest("GET", "/api/v1/messages/am_test123/votes", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetMessageVotes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Upvotes != 5 {
		t.Errorf("expected 5 upvotes, got %d", response.Upvotes)
	}

	if response.Downvotes != 2 {
		t.Errorf("expected 2 downvotes, got %d", response.Downvotes)
	}
}

// Tests for VoteHandler.VoteOnToolUse

func TestVoteHandler_VoteOnToolUse_Success(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 1, Downvotes: 0}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"vote": "up"}`
	req := httptest.NewRequest("POST", "/api/v1/tool-uses/atu_test123/vote", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "atu_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.VoteOnToolUse(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TargetType != "tool_use" {
		t.Errorf("expected target_type 'tool_use', got %v", response.TargetType)
	}
}

// Tests for VoteHandler.ToolUseQuickFeedback

func TestVoteHandler_ToolUseQuickFeedback_WrongTool(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 0, Downvotes: 1}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"feedback": "wrong_tool"}`
	req := httptest.NewRequest("POST", "/api/v1/tool-uses/atu_test123/quick-feedback", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "atu_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ToolUseQuickFeedback(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Downvotes != 1 {
		t.Errorf("expected 1 downvote (negative feedback), got %d", response.Downvotes)
	}
}

func TestVoteHandler_ToolUseQuickFeedback_Perfect(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 1, Downvotes: 0}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"feedback": "perfect"}`
	req := httptest.NewRequest("POST", "/api/v1/tool-uses/atu_test123/quick-feedback", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "atu_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ToolUseQuickFeedback(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Upvotes != 1 {
		t.Errorf("expected 1 upvote (positive feedback), got %d", response.Upvotes)
	}
}

func TestVoteHandler_ToolUseQuickFeedback_InvalidFeedback(t *testing.T) {
	voteRepo := newMockVoteRepo()
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"feedback": "invalid_feedback"}`
	req := httptest.NewRequest("POST", "/api/v1/tool-uses/atu_test123/quick-feedback", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "atu_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ToolUseQuickFeedback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// Tests for VoteHandler.VoteOnMemory

func TestVoteHandler_VoteOnMemory_Critical(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 1, Downvotes: 0}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"vote": "critical"}`
	req := httptest.NewRequest("POST", "/api/v1/memories/amem_test123/vote", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.VoteOnMemory(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TargetType != "memory" {
		t.Errorf("expected target_type 'memory', got %v", response.TargetType)
	}

	if response.Special == nil {
		t.Error("expected special votes map to be set")
	}
}

// Tests for VoteHandler.MemoryIrrelevanceReason

func TestVoteHandler_MemoryIrrelevanceReason_Success(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 0, Downvotes: 1}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"reason": "outdated"}`
	req := httptest.NewRequest("POST", "/api/v1/memories/amem_test123/irrelevance-reason", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.MemoryIrrelevanceReason(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Downvotes != 1 {
		t.Errorf("expected 1 downvote (irrelevance is negative), got %d", response.Downvotes)
	}
}

func TestVoteHandler_MemoryIrrelevanceReason_InvalidReason(t *testing.T) {
	voteRepo := newMockVoteRepo()
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"reason": "invalid_reason"}`
	req := httptest.NewRequest("POST", "/api/v1/memories/amem_test123/irrelevance-reason", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.MemoryIrrelevanceReason(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// Tests for VoteHandler.VoteOnReasoning

func TestVoteHandler_VoteOnReasoning_Success(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 1, Downvotes: 0}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"vote": "up"}`
	req := httptest.NewRequest("POST", "/api/v1/reasoning/ar_test123/vote", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ar_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.VoteOnReasoning(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TargetType != "reasoning" {
		t.Errorf("expected target_type 'reasoning', got %v", response.TargetType)
	}
}

// Tests for VoteHandler.ReasoningIssue

func TestVoteHandler_ReasoningIssue_Success(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.aggregates = &models.VoteAggregates{Upvotes: 0, Downvotes: 1}
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"issue": "incorrect_assumption"}`
	req := httptest.NewRequest("POST", "/api/v1/reasoning/ar_test123/issue", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ar_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ReasoningIssue(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response VoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Downvotes != 1 {
		t.Errorf("expected 1 downvote (issue is negative), got %d", response.Downvotes)
	}
}

func TestVoteHandler_ReasoningIssue_InvalidIssue(t *testing.T) {
	voteRepo := newMockVoteRepo()
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"issue": "invalid_issue"}`
	req := httptest.NewRequest("POST", "/api/v1/reasoning/ar_test123/issue", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ar_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ReasoningIssue(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// Tests for error conditions

func TestVoteHandler_CreateVote_RepositoryError(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.createErr = errors.New("database error")
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	body := `{"vote": "up"}`
	req := httptest.NewRequest("POST", "/api/v1/messages/am_test123/vote", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.VoteOnMessage(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestVoteHandler_GetAggregates_RepositoryError(t *testing.T) {
	voteRepo := newMockVoteRepo()
	voteRepo.getErr = errors.New("database error")
	idGen := newMockIDGenerator()
	handler := NewVoteHandler(voteRepo, idGen, nil)

	req := httptest.NewRequest("GET", "/api/v1/messages/am_test123/votes", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetMessageVotes(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}
