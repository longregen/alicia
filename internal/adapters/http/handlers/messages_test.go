package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Mock MessageRepository
type mockMessageRepo struct {
	messages       map[string]*models.Message
	byConversation map[string][]*models.Message
	createErr      error
	getErr         error
	nextSeqNum     int
	seqErr         error
}

func newMockMessageRepo() *mockMessageRepo {
	return &mockMessageRepo{
		messages:       make(map[string]*models.Message),
		byConversation: make(map[string][]*models.Message),
		nextSeqNum:     1,
	}
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.messages[msg.ID] = msg
	m.byConversation[msg.ConversationID] = append(m.byConversation[msg.ConversationID], msg)
	return nil
}

func (m *mockMessageRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	msg, ok := m.messages[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return msg, nil
}

func (m *mockMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	return nil
}

func (m *mockMessageRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMessageRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.byConversation[conversationID], nil
}

func (m *mockMessageRepo) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	messages := m.byConversation[conversationID]
	if len(messages) == 0 {
		return []*models.Message{}, nil
	}
	if limit > len(messages) {
		limit = len(messages)
	}
	return messages[len(messages)-limit:], nil
}

func (m *mockMessageRepo) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	if m.seqErr != nil {
		return 0, m.seqErr
	}
	return m.nextSeqNum, nil
}

func (m *mockMessageRepo) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	messages := m.byConversation[conversationID]
	result := []*models.Message{}
	for _, msg := range messages {
		if msg.SequenceNumber > afterSequence {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (m *mockMessageRepo) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepo) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepo) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	for _, msg := range m.messages {
		if msg.LocalID == localID {
			return msg, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (m *mockMessageRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepo) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return nil, nil
}

// Mock GenerateResponseUseCase
type mockGenerateResponseUseCase struct {
	output *ports.GenerateResponseOutput
	err    error
}

func newMockGenerateResponseUseCase() *mockGenerateResponseUseCase {
	return &mockGenerateResponseUseCase{}
}

func (m *mockGenerateResponseUseCase) Execute(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.output, nil
}

// Tests for MessagesHandler.List

func TestMessagesHandler_List_Success(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	// Create test messages
	msg1 := models.NewUserMessage("am_1", "ac_test123", 1, "Hello")
	msg2 := models.NewAssistantMessage("am_2", "ac_test123", 2, "Hi there")
	messageRepo.byConversation["ac_test123"] = []*models.Message{msg1, msg2}

	req := httptest.NewRequest("GET", "/api/v1/conversations/ac_test123/messages", nil)
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["total"].(float64) != 2 {
		t.Errorf("expected total 2, got %v", response["total"])
	}
}

func TestMessagesHandler_List_ConversationNotFound(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	req := httptest.NewRequest("GET", "/api/v1/conversations/nonexistent/messages", nil)
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.List(rr, req)

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

func TestMessagesHandler_List_InactiveConversation(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	// Create an archived conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conv.Archive()
	conversationRepo.conversations["ac_test123"] = conv

	req := httptest.NewRequest("GET", "/api/v1/conversations/ac_test123/messages", nil)
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.List(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "conversation_inactive" {
		t.Errorf("expected error 'conversation_inactive', got %v", response["error"])
	}
}

func TestMessagesHandler_List_RepositoryError(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	messageRepo.getErr = errors.New("database error")
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	req := httptest.NewRequest("GET", "/api/v1/conversations/ac_test123/messages", nil)
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.List(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

// Tests for MessagesHandler.Send

func TestMessagesHandler_Send_Success(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	body := `{"contents": "Hello, Alicia!"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["id"] != "am_test123" {
		t.Errorf("expected id 'am_test123', got %v", response["id"])
	}

	if response["contents"] != "Hello, Alicia!" {
		t.Errorf("expected contents 'Hello, Alicia!', got %v", response["contents"])
	}

	if response["role"] != "user" {
		t.Errorf("expected role 'user', got %v", response["role"])
	}
}

func TestMessagesHandler_Send_MissingContents(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

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

func TestMessagesHandler_Send_ConversationNotFound(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	body := `{"contents": "Hello, Alicia!"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/nonexistent/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestMessagesHandler_Send_InactiveConversation(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	// Create an archived conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conv.Archive()
	conversationRepo.conversations["ac_test123"] = conv

	body := `{"contents": "Hello, Alicia!"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "conversation_inactive" {
		t.Errorf("expected error 'conversation_inactive', got %v", response["error"])
	}
}

func TestMessagesHandler_Send_InvalidJSON(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestMessagesHandler_Send_RepositoryError(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	messageRepo.createErr = errors.New("database error")
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	body := `{"contents": "Hello, Alicia!"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestMessagesHandler_Send_WithPreviousMessage(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil, nil)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	// Add a previous message
	prevMsg := models.NewUserMessage("am_prev", "ac_test123", 1, "First message")
	messageRepo.byConversation["ac_test123"] = []*models.Message{prevMsg}

	// Set conversation tip to the previous message
	tipID := "am_prev"
	conv.TipMessageID = &tipID

	body := `{"contents": "Second message"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Add user context
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["previous_id"] != "am_prev" {
		t.Errorf("expected previous_id 'am_prev', got %v", response["previous_id"])
	}
}
