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
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Mock LiveKitService
type mockLiveKitService struct {
	createRoomErr    error
	getRoomErr       error
	generateTokenErr error
	room             *ports.LiveKitRoom
	token            *ports.LiveKitToken
}

func (m *mockLiveKitService) CreateRoom(ctx context.Context, name string) (*ports.LiveKitRoom, error) {
	if m.createRoomErr != nil {
		return nil, m.createRoomErr
	}
	if m.room == nil {
		return &ports.LiveKitRoom{Name: name}, nil
	}
	return m.room, nil
}

func (m *mockLiveKitService) GetRoom(ctx context.Context, name string) (*ports.LiveKitRoom, error) {
	if m.getRoomErr != nil {
		return nil, m.getRoomErr
	}
	if m.room == nil {
		return &ports.LiveKitRoom{Name: name}, nil
	}
	return m.room, nil
}

func (m *mockLiveKitService) GenerateToken(ctx context.Context, roomName, participantID, participantName string) (*ports.LiveKitToken, error) {
	if m.generateTokenErr != nil {
		return nil, m.generateTokenErr
	}
	if m.token == nil {
		return &ports.LiveKitToken{
			Token:     "test-token",
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		}, nil
	}
	return m.token, nil
}

func (m *mockLiveKitService) DeleteRoom(ctx context.Context, name string) error {
	return nil
}

func (m *mockLiveKitService) ListParticipants(ctx context.Context, roomName string) ([]*ports.LiveKitParticipant, error) {
	return nil, nil
}

func (m *mockLiveKitService) SendData(ctx context.Context, roomName string, data []byte, participantIDs []string) error {
	return nil
}

// Tests for TokenHandler.Generate

func TestTokenHandler_Generate_Success(t *testing.T) {
	convRepo := newMockConversationRepo()
	lkService := &mockLiveKitService{}

	// Create test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	convRepo.conversations["ac_test123"] = conv

	handler := NewTokenHandler(convRepo, lkService)

	body := `{"participant_id": "participant-1", "participant_name": "Test User"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Generate(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response dto.GenerateTokenResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Token != "test-token" {
		t.Errorf("expected token 'test-token', got %v", response.Token)
	}

	if response.ParticipantID != "participant-1" {
		t.Errorf("expected participant_id 'participant-1', got %v", response.ParticipantID)
	}
}

func TestTokenHandler_Generate_ExistingRoom(t *testing.T) {
	convRepo := newMockConversationRepo()
	lkService := &mockLiveKitService{}

	// Create test conversation with existing room
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conv.SetLiveKitRoom("existing-room")
	convRepo.conversations["ac_test123"] = conv

	handler := NewTokenHandler(convRepo, lkService)

	body := `{"participant_id": "participant-1"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Generate(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response dto.GenerateTokenResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.RoomName != "existing-room" {
		t.Errorf("expected room_name 'existing-room', got %v", response.RoomName)
	}
}

func TestTokenHandler_Generate_MissingParticipantID(t *testing.T) {
	convRepo := newMockConversationRepo()
	lkService := &mockLiveKitService{}
	handler := NewTokenHandler(convRepo, lkService)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Generate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestTokenHandler_Generate_ConversationNotFound(t *testing.T) {
	convRepo := newMockConversationRepo()
	convRepo.getErr = pgx.ErrNoRows
	lkService := &mockLiveKitService{}
	handler := NewTokenHandler(convRepo, lkService)

	body := `{"participant_id": "participant-1"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/nonexistent/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Generate(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestTokenHandler_Generate_InactiveConversation(t *testing.T) {
	convRepo := newMockConversationRepo()
	lkService := &mockLiveKitService{}

	// Create archived conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conv.Archive()
	convRepo.conversations["ac_test123"] = conv

	handler := NewTokenHandler(convRepo, lkService)

	body := `{"participant_id": "participant-1"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Generate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestTokenHandler_Generate_CreateRoomError(t *testing.T) {
	convRepo := newMockConversationRepo()
	lkService := &mockLiveKitService{
		createRoomErr: errors.New("create error"),
		getRoomErr:    errors.New("get error"),
	}

	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	convRepo.conversations["ac_test123"] = conv

	handler := NewTokenHandler(convRepo, lkService)

	body := `{"participant_id": "participant-1"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Generate(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestTokenHandler_Generate_GenerateTokenError(t *testing.T) {
	convRepo := newMockConversationRepo()
	lkService := &mockLiveKitService{
		generateTokenErr: errors.New("token error"),
	}

	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	convRepo.conversations["ac_test123"] = conv

	handler := NewTokenHandler(convRepo, lkService)

	body := `{"participant_id": "participant-1"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Generate(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestTokenHandler_Generate_DefaultParticipantName(t *testing.T) {
	convRepo := newMockConversationRepo()
	lkService := &mockLiveKitService{}

	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	convRepo.conversations["ac_test123"] = conv

	handler := NewTokenHandler(convRepo, lkService)

	// No participant_name provided
	body := `{"participant_id": "participant-1"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Generate(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Should use participant_id as default name
	var response dto.GenerateTokenResponse
	json.NewDecoder(rr.Body).Decode(&response)

	if response.ParticipantID != "participant-1" {
		t.Errorf("expected participant_id 'participant-1', got %v", response.ParticipantID)
	}
}

func TestTokenHandler_Generate_NoAuth(t *testing.T) {
	convRepo := newMockConversationRepo()
	lkService := &mockLiveKitService{}
	handler := NewTokenHandler(convRepo, lkService)

	body := `{"participant_id": "participant-1"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Generate(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}
