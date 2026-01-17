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

// Mock MemoryService
type mockMemoryService struct {
	createErr    error
	getErr       error
	searchErr    error
	updateErr    error
	deleteErr    error
	addTagErr    error
	removeTagErr error

	memory    *models.Memory
	memories  []*models.Memory
	searchRes []*ports.MemorySearchResult
}

func (m *mockMemoryService) Create(ctx context.Context, content string) (*models.Memory, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) CreateWithEmbeddings(ctx context.Context, content string) (*models.Memory, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) CreateFromConversation(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) Search(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.memories, nil
}

func (m *mockMemoryService) SearchWithScores(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.searchRes, nil
}

func (m *mockMemoryService) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.memories, nil
}

func (m *mockMemoryService) Delete(ctx context.Context, id string) error {
	return m.deleteErr
}

func (m *mockMemoryService) SetImportance(ctx context.Context, id string, importance float32) (*models.Memory, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) SetConfidence(ctx context.Context, id string, confidence float32) (*models.Memory, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) SetUserRating(ctx context.Context, id string, rating int) (*models.Memory, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) AddTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	if m.addTagErr != nil {
		return nil, m.addTagErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) RemoveTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	if m.removeTagErr != nil {
		return nil, m.removeTagErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) RegenerateEmbeddings(ctx context.Context, id string) (*models.Memory, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) Pin(ctx context.Context, id string, pinned bool) (*models.Memory, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) Archive(ctx context.Context, id string) (*models.Memory, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.memory, nil
}

func (m *mockMemoryService) DeleteByConversationID(ctx context.Context, conversationID string) error {
	return nil
}

func (m *mockMemoryService) SearchWithThreshold(ctx context.Context, query string, threshold float32, limit int) ([]*models.Memory, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.memories, nil
}

func (m *mockMemoryService) SearchWithDynamicImportance(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.memories, nil
}

func (m *mockMemoryService) TrackUsage(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryService) GetUsageByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryService) GetUsageByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryService) Update(ctx context.Context, memory *models.Memory) error {
	return m.updateErr
}

// Tests for MemoryHandler.CreateMemory

func TestMemoryHandler_CreateMemory_Success(t *testing.T) {
	memory := models.NewMemory("amem_test123", "Test memory content")
	mockService := &mockMemoryService{memory: memory}
	handler := NewMemoryHandler(mockService)

	body := `{"content": "Test memory content"}`
	req := httptest.NewRequest("POST", "/api/v1/memories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.CreateMemory(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response MemoryResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Content != "Test memory content" {
		t.Errorf("expected content 'Test memory content', got %v", response.Content)
	}
}

func TestMemoryHandler_CreateMemory_WithTags(t *testing.T) {
	memory := models.NewMemory("amem_test123", "Test memory content")
	memory.Tags = []string{"tag1", "tag2"}

	mockService := &mockMemoryService{memory: memory}
	handler := NewMemoryHandler(mockService)

	body := `{"content": "Test memory content", "tags": ["tag1", "tag2"]}`
	req := httptest.NewRequest("POST", "/api/v1/memories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.CreateMemory(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}
}

func TestMemoryHandler_CreateMemory_EmptyContent(t *testing.T) {
	mockService := &mockMemoryService{}
	handler := NewMemoryHandler(mockService)

	body := `{"content": ""}`
	req := httptest.NewRequest("POST", "/api/v1/memories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.CreateMemory(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestMemoryHandler_CreateMemory_ServiceError(t *testing.T) {
	mockService := &mockMemoryService{createErr: errors.New("service error")}
	handler := NewMemoryHandler(mockService)

	body := `{"content": "Test memory content"}`
	req := httptest.NewRequest("POST", "/api/v1/memories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.CreateMemory(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

// Tests for MemoryHandler.GetMemory

func TestMemoryHandler_GetMemory_Success(t *testing.T) {
	memory := models.NewMemory("amem_test123", "Test memory content")
	mockService := &mockMemoryService{memory: memory}
	handler := NewMemoryHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/memories/amem_test123", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetMemory(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response MemoryResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != "amem_test123" {
		t.Errorf("expected id 'amem_test123', got %v", response.ID)
	}
}

func TestMemoryHandler_GetMemory_NotFound(t *testing.T) {
	mockService := &mockMemoryService{getErr: errors.New("not found")}
	handler := NewMemoryHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/memories/nonexistent", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetMemory(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// Tests for MemoryHandler.ListMemories

func TestMemoryHandler_ListMemories_Success(t *testing.T) {
	mem1 := models.NewMemory("amem_1", "Memory 1")
	mem2 := models.NewMemory("amem_2", "Memory 2")

	mockService := &mockMemoryService{
		memories: []*models.Memory{mem1, mem2},
	}
	handler := NewMemoryHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/memories", nil)
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.ListMemories(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response MemoryListResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
}

func TestMemoryHandler_ListMemories_WithTags(t *testing.T) {
	mem1 := models.NewMemory("amem_1", "Memory 1")
	mem1.Tags = []string{"important"}

	mockService := &mockMemoryService{
		memories: []*models.Memory{mem1},
	}
	handler := NewMemoryHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/memories?tags=important", nil)
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.ListMemories(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// Tests for MemoryHandler.SearchMemories

func TestMemoryHandler_SearchMemories_Success(t *testing.T) {
	mem := models.NewMemory("amem_1", "Test memory")
	searchRes := []*ports.MemorySearchResult{
		{Memory: mem, Similarity: 0.95},
	}

	mockService := &mockMemoryService{searchRes: searchRes}
	handler := NewMemoryHandler(mockService)

	body := `{"query": "test query", "limit": 10}`
	req := httptest.NewRequest("POST", "/api/v1/memories/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.SearchMemories(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response SearchResultsResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(response.Results))
	}

	if response.Results[0].Similarity != 0.95 {
		t.Errorf("expected similarity 0.95, got %f", response.Results[0].Similarity)
	}
}

func TestMemoryHandler_SearchMemories_EmptyQuery(t *testing.T) {
	mockService := &mockMemoryService{}
	handler := NewMemoryHandler(mockService)

	body := `{"query": ""}`
	req := httptest.NewRequest("POST", "/api/v1/memories/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.SearchMemories(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// Tests for MemoryHandler.UpdateMemory

func TestMemoryHandler_UpdateMemory_Success(t *testing.T) {
	memory := models.NewMemory("amem_test123", "Updated content")
	mockService := &mockMemoryService{memory: memory}
	handler := NewMemoryHandler(mockService)

	content := "Updated content"
	body := `{"content": "Updated content"}`
	req := httptest.NewRequest("PUT", "/api/v1/memories/amem_test123", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.UpdateMemory(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response MemoryResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Content != content {
		t.Errorf("expected content '%s', got %v", content, response.Content)
	}
}

func TestMemoryHandler_UpdateMemory_NotFound(t *testing.T) {
	mockService := &mockMemoryService{getErr: errors.New("not found")}
	handler := NewMemoryHandler(mockService)

	body := `{"content": "Updated content"}`
	req := httptest.NewRequest("PUT", "/api/v1/memories/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.UpdateMemory(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// Tests for MemoryHandler.DeleteMemory

func TestMemoryHandler_DeleteMemory_Success(t *testing.T) {
	mockService := &mockMemoryService{}
	handler := NewMemoryHandler(mockService)

	req := httptest.NewRequest("DELETE", "/api/v1/memories/amem_test123", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DeleteMemory(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}
}

func TestMemoryHandler_DeleteMemory_ServiceError(t *testing.T) {
	mockService := &mockMemoryService{deleteErr: errors.New("delete error")}
	handler := NewMemoryHandler(mockService)

	req := httptest.NewRequest("DELETE", "/api/v1/memories/amem_test123", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DeleteMemory(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

// Tests for MemoryHandler.AddTag

func TestMemoryHandler_AddTag_Success(t *testing.T) {
	memory := models.NewMemory("amem_test123", "Test memory")
	memory.Tags = []string{"tag1"}

	mockService := &mockMemoryService{memory: memory}
	handler := NewMemoryHandler(mockService)

	body := `{"tag": "tag1"}`
	req := httptest.NewRequest("POST", "/api/v1/memories/amem_test123/tags", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.AddTag(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestMemoryHandler_AddTag_EmptyTag(t *testing.T) {
	mockService := &mockMemoryService{}
	handler := NewMemoryHandler(mockService)

	body := `{"tag": ""}`
	req := httptest.NewRequest("POST", "/api/v1/memories/amem_test123/tags", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.AddTag(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// Tests for MemoryHandler.RemoveTag

func TestMemoryHandler_RemoveTag_Success(t *testing.T) {
	memory := models.NewMemory("amem_test123", "Test memory")
	mockService := &mockMemoryService{memory: memory}
	handler := NewMemoryHandler(mockService)

	req := httptest.NewRequest("DELETE", "/api/v1/memories/amem_test123/tags/tag1", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	rctx.URLParams.Add("tag", "tag1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.RemoveTag(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// Tests for MemoryHandler.PinMemory

func TestMemoryHandler_PinMemory_Success(t *testing.T) {
	memory := models.NewMemory("amem_test123", "Test memory")
	memory.Pinned = true

	mockService := &mockMemoryService{memory: memory}
	handler := NewMemoryHandler(mockService)

	body := `{"pinned": true}`
	req := httptest.NewRequest("POST", "/api/v1/memories/amem_test123/pin", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.PinMemory(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response MemoryResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Pinned {
		t.Error("expected memory to be pinned")
	}
}

// Tests for MemoryHandler.ArchiveMemory

func TestMemoryHandler_ArchiveMemory_Success(t *testing.T) {
	memory := models.NewMemory("amem_test123", "Test memory")
	memory.Archived = true

	mockService := &mockMemoryService{memory: memory}
	handler := NewMemoryHandler(mockService)

	req := httptest.NewRequest("POST", "/api/v1/memories/amem_test123/archive", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "amem_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ArchiveMemory(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response MemoryResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Archived {
		t.Error("expected memory to be archived")
	}
}
