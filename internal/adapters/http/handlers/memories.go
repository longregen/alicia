package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type MemoryHandler struct {
	memoryService ports.MemoryService
}

func NewMemoryHandler(memoryService ports.MemoryService) *MemoryHandler {
	return &MemoryHandler{
		memoryService: memoryService,
	}
}

// CreateMemoryRequest represents a memory creation request
type CreateMemoryRequest struct {
	Content    string   `json:"content"`
	Tags       []string `json:"tags,omitempty"`
	Importance *float32 `json:"importance,omitempty"`
}

// UpdateMemoryRequest represents a memory update request
type UpdateMemoryRequest struct {
	Content    *string  `json:"content,omitempty"`
	Importance *float32 `json:"importance,omitempty"`
	Confidence *float32 `json:"confidence,omitempty"`
	UserRating *int     `json:"user_rating,omitempty"`
}

// AddTagRequest represents a tag addition request
type AddTagRequest struct {
	Tag string `json:"tag"`
}

// SearchMemoriesRequest represents a memory search request
type SearchMemoriesRequest struct {
	Query     string   `json:"query"`
	Limit     int      `json:"limit,omitempty"`
	Threshold *float32 `json:"threshold,omitempty"`
}

// MemoryResponse represents a memory in API responses
type MemoryResponse struct {
	ID         string   `json:"id"`
	Content    string   `json:"content"`
	Importance float32  `json:"importance"`
	Confidence float32  `json:"confidence"`
	UserRating *int     `json:"user_rating,omitempty"`
	Tags       []string `json:"tags"`
	SourceType string   `json:"source_type,omitempty"`
	Pinned     bool     `json:"pinned"`
	Archived   bool     `json:"archived"`
	CreatedAt  int64    `json:"created_at"`
	UpdatedAt  int64    `json:"updated_at"`
}

// MemorySearchResultResponse includes similarity score
type MemorySearchResultResponse struct {
	Memory     MemoryResponse `json:"memory"`
	Similarity float32        `json:"similarity"`
}

// MemoryListResponse represents a list of memories
type MemoryListResponse struct {
	Memories []MemoryResponse `json:"memories"`
	Total    int              `json:"total"`
}

// SearchResultsResponse represents search results with scores
type SearchResultsResponse struct {
	Results []MemorySearchResultResponse `json:"results"`
	Total   int                          `json:"total"`
}

func memoryToResponse(m *models.Memory) MemoryResponse {
	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}
	return MemoryResponse{
		ID:         m.ID,
		Content:    m.Content,
		Importance: m.Importance,
		Confidence: m.Confidence,
		UserRating: m.UserRating,
		Tags:       tags,
		SourceType: m.SourceType,
		Pinned:     m.Pinned,
		Archived:   m.Archived,
		CreatedAt:  m.CreatedAt.Unix(),
		UpdatedAt:  m.UpdatedAt.Unix(),
	}
}

// CreateMemory handles POST /api/v1/memories
func (h *MemoryHandler) CreateMemory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	req, ok := decodeJSON[CreateMemoryRequest](r, w)
	if !ok {
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Content) == "" {
		respondError(w, "validation_error", "Memory content is required", http.StatusBadRequest)
		return
	}

	// Create memory with embeddings for semantic search
	memory, err := h.memoryService.CreateWithEmbeddings(r.Context(), req.Content)
	if err != nil {
		respondError(w, "create_error", err.Error(), http.StatusInternalServerError)
		return
	}

	// Set optional fields
	if req.Importance != nil {
		memory, err = h.memoryService.SetImportance(r.Context(), memory.ID, *req.Importance)
		if err != nil {
			respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Add tags if provided
	for _, tag := range req.Tags {
		memory, err = h.memoryService.AddTag(r.Context(), memory.ID, tag)
		if err != nil {
			respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
			return
		}
	}

	respondJSON(w, memoryToResponse(memory), http.StatusCreated)
}

// ListMemories handles GET /api/v1/memories
func (h *MemoryHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	limit := parseIntQuery(r, "limit", 100)
	tags := r.URL.Query()["tags"]

	// Validate limit
	if limit > 500 {
		limit = 500
	}
	if limit < 1 {
		limit = 1
	}

	var memories []*models.Memory
	var err error

	if len(tags) > 0 {
		// Filter by tags
		memories, err = h.memoryService.GetByTags(r.Context(), tags, limit)
		if err != nil {
			log.Printf("[MemoryHandler.ListMemories] GetByTags failed: tags=%v, limit=%d, error=%v", tags, limit, err)
			respondError(w, "list_error", err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// No filtering - search returns all memories ordered by importance
		// Use a high-dimensional zero vector to get all memories
		memories, err = h.memoryService.Search(r.Context(), " ", limit)
		if err != nil {
			log.Printf("[MemoryHandler.ListMemories] Search failed: limit=%d, error=%v", limit, err)
			respondError(w, "list_error", err.Error(), http.StatusInternalServerError)
			return
		}
	}

	responses := make([]MemoryResponse, len(memories))
	for i, m := range memories {
		responses[i] = memoryToResponse(m)
	}

	respondJSON(w, &MemoryListResponse{
		Memories: responses,
		Total:    len(responses),
	}, http.StatusOK)
}

// GetMemory handles GET /api/v1/memories/{id}
func (h *MemoryHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	memory, err := h.memoryService.GetByID(r.Context(), memoryID)
	if err != nil {
		respondError(w, "not_found", "Memory not found", http.StatusNotFound)
		return
	}

	respondJSON(w, memoryToResponse(memory), http.StatusOK)
}

// UpdateMemory handles PUT /api/v1/memories/{id}
func (h *MemoryHandler) UpdateMemory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[UpdateMemoryRequest](r, w)
	if !ok {
		return
	}

	// Get existing memory
	memory, err := h.memoryService.GetByID(r.Context(), memoryID)
	if err != nil {
		respondError(w, "not_found", "Memory not found", http.StatusNotFound)
		return
	}

	// Update fields if provided
	if req.Content != nil && strings.TrimSpace(*req.Content) != "" {
		memory.Content = *req.Content
		// Regenerate embeddings when content changes
		memory, err = h.memoryService.RegenerateEmbeddings(r.Context(), memory.ID)
		if err != nil {
			respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if req.Importance != nil {
		memory, err = h.memoryService.SetImportance(r.Context(), memory.ID, *req.Importance)
		if err != nil {
			respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if req.Confidence != nil {
		memory, err = h.memoryService.SetConfidence(r.Context(), memory.ID, *req.Confidence)
		if err != nil {
			respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if req.UserRating != nil {
		memory, err = h.memoryService.SetUserRating(r.Context(), memory.ID, *req.UserRating)
		if err != nil {
			respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
			return
		}
	}

	respondJSON(w, memoryToResponse(memory), http.StatusOK)
}

// DeleteMemory handles DELETE /api/v1/memories/{id}
func (h *MemoryHandler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	err := h.memoryService.Delete(r.Context(), memoryID)
	if err != nil {
		if errors.Is(err, domain.ErrMemoryNotFound) {
			respondError(w, "not_found", "Memory not found", http.StatusNotFound)
			return
		}
		respondError(w, "delete_error", err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddTag handles POST /api/v1/memories/{id}/tags
func (h *MemoryHandler) AddTag(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[AddTagRequest](r, w)
	if !ok {
		return
	}

	if strings.TrimSpace(req.Tag) == "" {
		respondError(w, "validation_error", "Tag cannot be empty", http.StatusBadRequest)
		return
	}

	memory, err := h.memoryService.AddTag(r.Context(), memoryID, req.Tag)
	if err != nil {
		respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, memoryToResponse(memory), http.StatusOK)
}

// RemoveTag handles DELETE /api/v1/memories/{id}/tags/{tag}
func (h *MemoryHandler) RemoveTag(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	tag, ok := validateURLParam(r, w, "tag", "Tag")
	if !ok {
		return
	}

	memory, err := h.memoryService.RemoveTag(r.Context(), memoryID, tag)
	if err != nil {
		respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, memoryToResponse(memory), http.StatusOK)
}

// SearchMemories handles POST /api/v1/memories/search
func (h *MemoryHandler) SearchMemories(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	req, ok := decodeJSON[SearchMemoriesRequest](r, w)
	if !ok {
		return
	}

	if strings.TrimSpace(req.Query) == "" {
		respondError(w, "validation_error", "Search query is required", http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	threshold := float32(0.7)
	if req.Threshold != nil {
		threshold = *req.Threshold
	}

	results, err := h.memoryService.SearchWithScores(r.Context(), req.Query, threshold, limit)
	if err != nil {
		respondError(w, "search_error", err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]MemorySearchResultResponse, len(results))
	for i, r := range results {
		responses[i] = MemorySearchResultResponse{
			Memory:     memoryToResponse(r.Memory),
			Similarity: r.Similarity,
		}
	}

	respondJSON(w, &SearchResultsResponse{
		Results: responses,
		Total:   len(responses),
	}, http.StatusOK)
}

// SetImportance handles PUT /api/v1/memories/{id}/importance
func (h *MemoryHandler) SetImportance(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	type ImportanceRequest struct {
		Importance float32 `json:"importance"`
	}

	req, ok := decodeJSON[ImportanceRequest](r, w)
	if !ok {
		return
	}

	memory, err := h.memoryService.SetImportance(r.Context(), memoryID, req.Importance)
	if err != nil {
		respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, memoryToResponse(memory), http.StatusOK)
}

// GetByTags handles GET /api/v1/memories/by-tags (kept for backwards compatibility)
func (h *MemoryHandler) GetByTags(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	tags := r.URL.Query()["tags"]
	if len(tags) == 0 {
		respondError(w, "validation_error", "At least one tag is required", http.StatusBadRequest)
		return
	}

	limit := parseIntQuery(r, "limit", 50)
	if limit > 500 {
		limit = 500
	}

	memories, err := h.memoryService.GetByTags(r.Context(), tags, limit)
	if err != nil {
		respondError(w, "search_error", err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]MemoryResponse, len(memories))
	for i, m := range memories {
		responses[i] = memoryToResponse(m)
	}

	respondJSON(w, &MemoryListResponse{
		Memories: responses,
		Total:    len(responses),
	}, http.StatusOK)
}

// PinMemory handles POST /api/v1/memories/{id}/pin
func (h *MemoryHandler) PinMemory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	type PinRequest struct {
		Pinned bool `json:"pinned"`
	}

	req, ok := decodeJSON[PinRequest](r, w)
	if !ok {
		return
	}

	memory, err := h.memoryService.Pin(r.Context(), memoryID, req.Pinned)
	if err != nil {
		respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, memoryToResponse(memory), http.StatusOK)
}

// ArchiveMemory handles POST /api/v1/memories/{id}/archive
func (h *MemoryHandler) ArchiveMemory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	memory, err := h.memoryService.Archive(r.Context(), memoryID)
	if err != nil {
		respondError(w, "update_error", err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, memoryToResponse(memory), http.StatusOK)
}
