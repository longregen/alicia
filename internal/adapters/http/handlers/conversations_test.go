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
	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/domain/models"
)

// Mock ConversationRepository
type mockConversationRepo struct {
	conversations map[string]*models.Conversation
	createErr     error
	getErr        error
	listErr       error
	deleteErr     error
}

func newMockConversationRepo() *mockConversationRepo {
	return &mockConversationRepo{
		conversations: make(map[string]*models.Conversation),
	}
}

func (m *mockConversationRepo) Create(ctx context.Context, c *models.Conversation) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.conversations[c.ID] = c
	return nil
}

func (m *mockConversationRepo) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	conv, ok := m.conversations[id]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return conv, nil
}

func (m *mockConversationRepo) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	return nil, nil
}

func (m *mockConversationRepo) Update(ctx context.Context, c *models.Conversation) error {
	return nil
}

func (m *mockConversationRepo) UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error {
	return nil
}

func (m *mockConversationRepo) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.conversations, id)
	return nil
}

func (m *mockConversationRepo) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	convs := make([]*models.Conversation, 0, len(m.conversations))
	for _, conv := range m.conversations {
		convs = append(convs, conv)
	}
	return convs, nil
}

func (m *mockConversationRepo) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	convs := make([]*models.Conversation, 0)
	for _, conv := range m.conversations {
		if conv.IsActive() {
			convs = append(convs, conv)
		}
	}
	return convs, nil
}

func (m *mockConversationRepo) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.conversations, id)
	return nil
}

func (m *mockConversationRepo) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	conv, ok := m.conversations[id]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return conv, nil
}

func (m *mockConversationRepo) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	convs := make([]*models.Conversation, 0, len(m.conversations))
	for _, conv := range m.conversations {
		convs = append(convs, conv)
	}
	return convs, nil
}

func (m *mockConversationRepo) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	convs := make([]*models.Conversation, 0)
	for _, conv := range m.conversations {
		if conv.IsActive() {
			convs = append(convs, conv)
		}
	}
	return convs, nil
}

// Mock IDGenerator
type mockIDGenerator struct {
	nextConversationID string
	nextMessageID      string
}

func newMockIDGenerator() *mockIDGenerator {
	return &mockIDGenerator{
		nextConversationID: "ac_test123",
		nextMessageID:      "am_test123",
	}
}

func (m *mockIDGenerator) GenerateConversationID() string {
	return m.nextConversationID
}

func (m *mockIDGenerator) GenerateMessageID() string {
	return m.nextMessageID
}

func (m *mockIDGenerator) GenerateSentenceID() string {
	return "ams_test123"
}

func (m *mockIDGenerator) GenerateAudioID() string {
	return "aa_test123"
}

func (m *mockIDGenerator) GenerateMemoryID() string {
	return "amem_test123"
}

func (m *mockIDGenerator) GenerateMemoryUsageID() string {
	return "amu_test123"
}

func (m *mockIDGenerator) GenerateToolID() string {
	return "at_test123"
}

func (m *mockIDGenerator) GenerateToolUseID() string {
	return "atu_test123"
}

func (m *mockIDGenerator) GenerateReasoningStepID() string {
	return "ar_test123"
}

func (m *mockIDGenerator) GenerateCommentaryID() string {
	return "aucc_test123"
}

func (m *mockIDGenerator) GenerateMetaID() string {
	return "amt_test123"
}

func (m *mockIDGenerator) GenerateMCPServerID() string {
	return "amcp_test123"
}

func (m *mockIDGenerator) GenerateVoteID() string {
	return "av_test123"
}

func (m *mockIDGenerator) GenerateNoteID() string {
	return "an_test123"
}

func (m *mockIDGenerator) GenerateSessionStatsID() string {
	return "ass_test123"
}

func (m *mockIDGenerator) GenerateOptimizationRunID() string {
	return "aor_test123"
}

func (m *mockIDGenerator) GeneratePromptCandidateID() string {
	return "apc_test123"
}

func (m *mockIDGenerator) GeneratePromptEvaluationID() string {
	return "ape_test123"
}

// Tests for ConversationsHandler.Create

func TestConversationsHandler_Create_Success(t *testing.T) {
	repo := newMockConversationRepo()
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	body := `{"title": "Test Conversation"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["id"] != "ac_test123" {
		t.Errorf("expected id 'ac_test123', got %v", response["id"])
	}

	if response["title"] != "Test Conversation" {
		t.Errorf("expected title 'Test Conversation', got %v", response["title"])
	}
}

func TestConversationsHandler_Create_MissingTitle(t *testing.T) {
	repo := newMockConversationRepo()
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/conversations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "validation_error" {
		t.Errorf("expected error 'validation_error', got %v", response["error"])
	}
}

func TestConversationsHandler_Create_InvalidJSON(t *testing.T) {
	repo := newMockConversationRepo()
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/v1/conversations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestConversationsHandler_Create_RepositoryError(t *testing.T) {
	repo := newMockConversationRepo()
	repo.createErr = errors.New("database error")
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	body := `{"title": "Test Conversation"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

// Tests for ConversationsHandler.Get

func TestConversationsHandler_Get_Success(t *testing.T) {
	repo := newMockConversationRepo()
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	repo.conversations["ac_test123"] = conv

	req := httptest.NewRequest("GET", "/api/v1/conversations/ac_test123", nil)
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["id"] != "ac_test123" {
		t.Errorf("expected id 'ac_test123', got %v", response["id"])
	}
}

func TestConversationsHandler_Get_NotFound(t *testing.T) {
	repo := newMockConversationRepo()
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	req := httptest.NewRequest("GET", "/api/v1/conversations/nonexistent", nil)
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Get(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "not_found" {
		t.Errorf("expected error 'not_found', got %v", response["error"])
	}
}

// Tests for ConversationsHandler.List

func TestConversationsHandler_List_Success(t *testing.T) {
	repo := newMockConversationRepo()
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	// Create test conversations
	repo.conversations["ac_1"] = models.NewConversation("ac_1", "test-user", "Conversation 1")
	repo.conversations["ac_2"] = models.NewConversation("ac_2", "test-user", "Conversation 2")
	repo.conversations["ac_3"] = models.NewConversation("ac_3", "test-user", "Conversation 3")

	req := httptest.NewRequest("GET", "/api/v1/conversations", nil)
	req = addUserContext(req, "test-user")
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["total"].(float64) != 3 {
		t.Errorf("expected total 3, got %v", response["total"])
	}

	if response["limit"].(float64) != 50 {
		t.Errorf("expected limit 50, got %v", response["limit"])
	}

	if response["offset"].(float64) != 0 {
		t.Errorf("expected offset 0, got %v", response["offset"])
	}
}

func TestConversationsHandler_List_WithPagination(t *testing.T) {
	repo := newMockConversationRepo()
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	// Create test conversations
	repo.conversations["ac_1"] = models.NewConversation("ac_1", "test-user", "Conversation 1")

	req := httptest.NewRequest("GET", "/api/v1/conversations?limit=10&offset=5", nil)
	req = addUserContext(req, "test-user")
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["limit"].(float64) != 10 {
		t.Errorf("expected limit 10, got %v", response["limit"])
	}

	if response["offset"].(float64) != 5 {
		t.Errorf("expected offset 5, got %v", response["offset"])
	}
}

func TestConversationsHandler_List_ActiveOnly(t *testing.T) {
	repo := newMockConversationRepo()
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	// Create active and archived conversations
	activeConv := models.NewConversation("ac_1", "test-user", "Active Conversation")
	archivedConv := models.NewConversation("ac_2", "test-user", "Archived Conversation")
	archivedConv.Archive()

	repo.conversations["ac_1"] = activeConv
	repo.conversations["ac_2"] = archivedConv

	req := httptest.NewRequest("GET", "/api/v1/conversations?active=true", nil)
	req = addUserContext(req, "test-user")
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should only return 1 active conversation
	if response["total"].(float64) != 1 {
		t.Errorf("expected total 1, got %v", response["total"])
	}
}

// Tests for ConversationsHandler.Delete

func TestConversationsHandler_Delete_Success(t *testing.T) {
	repo := newMockConversationRepo()
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	repo.conversations["ac_test123"] = conv

	req := httptest.NewRequest("DELETE", "/api/v1/conversations/ac_test123", nil)
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}

	// Verify conversation was deleted
	if _, exists := repo.conversations["ac_test123"]; exists {
		t.Error("expected conversation to be deleted")
	}
}

func TestConversationsHandler_Delete_RepositoryError(t *testing.T) {
	repo := newMockConversationRepo()
	repo.deleteErr = errors.New("database error")
	idGen := newMockIDGenerator()
	handler := NewConversationsHandler(repo, idGen)

	req := httptest.NewRequest("DELETE", "/api/v1/conversations/ac_test123", nil)
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Delete(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}
