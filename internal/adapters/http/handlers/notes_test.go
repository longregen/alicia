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
)

// Mock NoteRepository
type mockNoteRepo struct {
	notes     map[string]*models.Note
	createErr error
	getErr    error
	updateErr error
	deleteErr error
}

func newMockNoteRepo() *mockNoteRepo {
	return &mockNoteRepo{
		notes: make(map[string]*models.Note),
	}
}

func (m *mockNoteRepo) Create(ctx context.Context, note *models.Note) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.notes[note.ID] = note
	return nil
}

func (m *mockNoteRepo) GetByID(ctx context.Context, id string) (*models.Note, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	note, ok := m.notes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return note, nil
}

func (m *mockNoteRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.Note, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	var notes []*models.Note
	for _, note := range m.notes {
		if note.MessageID == messageID {
			notes = append(notes, note)
		}
	}
	return notes, nil
}

func (m *mockNoteRepo) Update(ctx context.Context, id, content string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if note, ok := m.notes[id]; ok {
		note.Content = content
	}
	return nil
}

func (m *mockNoteRepo) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.notes, id)
	return nil
}

// Tests for NoteHandler.CreateMessageNote

func TestNoteHandler_CreateMessageNote_Success(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	body := `{"content": "Test note", "category": "improvement"}`
	req := httptest.NewRequest("POST", "/api/v1/messages/am_test123/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.CreateMessageNote(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response NoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Content != "Test note" {
		t.Errorf("expected content 'Test note', got %v", response.Content)
	}

	if response.Category != "improvement" {
		t.Errorf("expected category 'improvement', got %v", response.Category)
	}
}

func TestNoteHandler_CreateMessageNote_EmptyContent(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	body := `{"content": ""}`
	req := httptest.NewRequest("POST", "/api/v1/messages/am_test123/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.CreateMessageNote(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestNoteHandler_CreateMessageNote_InvalidCategory(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	body := `{"content": "Test note", "category": "invalid"}`
	req := httptest.NewRequest("POST", "/api/v1/messages/am_test123/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.CreateMessageNote(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestNoteHandler_CreateMessageNote_DefaultCategory(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	body := `{"content": "Test note"}`
	req := httptest.NewRequest("POST", "/api/v1/messages/am_test123/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.CreateMessageNote(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response NoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Category != models.NoteCategoryGeneral {
		t.Errorf("expected default category '%s', got %v", models.NoteCategoryGeneral, response.Category)
	}
}

// Tests for NoteHandler.GetMessageNotes

func TestNoteHandler_GetMessageNotes_Success(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()

	note1 := models.NewNote("an_1", "am_test123", "Note 1", "improvement")
	note2 := models.NewNote("an_2", "am_test123", "Note 2", "correction")
	noteRepo.notes["an_1"] = note1
	noteRepo.notes["an_2"] = note2

	handler := NewNoteHandler(noteRepo, idGen)

	req := httptest.NewRequest("GET", "/api/v1/messages/am_test123/notes", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetMessageNotes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response NoteListResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
}

func TestNoteHandler_GetMessageNotes_Empty(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	req := httptest.NewRequest("GET", "/api/v1/messages/am_test123/notes", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "am_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetMessageNotes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response NoteListResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Total != 0 {
		t.Errorf("expected total 0, got %d", response.Total)
	}
}

// Tests for NoteHandler.CreateToolUseNote

func TestNoteHandler_CreateToolUseNote_Success(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	body := `{"content": "Tool use note", "category": "correction"}`
	req := httptest.NewRequest("POST", "/api/v1/tool-uses/atu_test123/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "atu_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.CreateToolUseNote(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response NoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TargetType != "tool_use" {
		t.Errorf("expected target_type 'tool_use', got %v", response.TargetType)
	}
}

// Tests for NoteHandler.CreateReasoningNote

func TestNoteHandler_CreateReasoningNote_Success(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	body := `{"content": "Reasoning note", "category": "context"}`
	req := httptest.NewRequest("POST", "/api/v1/reasoning/ar_test123/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ar_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.CreateReasoningNote(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response NoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TargetType != "reasoning" {
		t.Errorf("expected target_type 'reasoning', got %v", response.TargetType)
	}
}

// Tests for NoteHandler.UpdateNote

func TestNoteHandler_UpdateNote_Success(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()

	note := models.NewNote("an_test123", "am_test123", "Original content", "general")
	noteRepo.notes["an_test123"] = note

	handler := NewNoteHandler(noteRepo, idGen)

	body := `{"content": "Updated content"}`
	req := httptest.NewRequest("PUT", "/api/v1/notes/an_test123", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "an_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.UpdateNote(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response NoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got %v", response.Content)
	}
}

func TestNoteHandler_UpdateNote_EmptyContent(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	body := `{"content": ""}`
	req := httptest.NewRequest("PUT", "/api/v1/notes/an_test123", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "an_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.UpdateNote(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestNoteHandler_UpdateNote_RepositoryError(t *testing.T) {
	noteRepo := newMockNoteRepo()
	noteRepo.updateErr = errors.New("update error")
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	body := `{"content": "Updated content"}`
	req := httptest.NewRequest("PUT", "/api/v1/notes/an_test123", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "an_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.UpdateNote(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

// Tests for NoteHandler.DeleteNote

func TestNoteHandler_DeleteNote_Success(t *testing.T) {
	noteRepo := newMockNoteRepo()
	idGen := newMockIDGenerator()

	note := models.NewNote("an_test123", "am_test123", "Test note", "general")
	noteRepo.notes["an_test123"] = note

	handler := NewNoteHandler(noteRepo, idGen)

	req := httptest.NewRequest("DELETE", "/api/v1/notes/an_test123", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "an_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DeleteNote(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}

	if _, exists := noteRepo.notes["an_test123"]; exists {
		t.Error("expected note to be deleted")
	}
}

func TestNoteHandler_DeleteNote_RepositoryError(t *testing.T) {
	noteRepo := newMockNoteRepo()
	noteRepo.deleteErr = errors.New("delete error")
	idGen := newMockIDGenerator()
	handler := NewNoteHandler(noteRepo, idGen)

	req := httptest.NewRequest("DELETE", "/api/v1/notes/an_test123", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "an_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DeleteNote(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}
