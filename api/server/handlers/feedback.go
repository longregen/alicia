package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/store"
	"github.com/longregen/alicia/pkg/langfuse"
)

type FeedbackHandler struct {
	store    *store.Store
	langfuse *langfuse.Client
}

func NewFeedbackHandlerWithLangfuse(s *store.Store, lf *langfuse.Client) *FeedbackHandler {
	return &FeedbackHandler{
		store:    s,
		langfuse: lf,
	}
}

// feedbackRequest is the common request body for all feedback types.
type feedbackRequest struct {
	Rating int16  `json:"rating"`
	Note   string `json:"note"`
}

// feedbackOps defines the type-specific operations for a feedback handler.
// Each feedback type (message, tool use, memory use) provides its own implementation.
type feedbackOps[T any] struct {
	// getExisting retrieves any existing feedback for the given resource ID.
	getExisting func(ctx context.Context, resourceID string) ([]*T, error)
	// updateExisting applies the new rating/note to an existing feedback entry and persists it.
	updateExisting func(ctx context.Context, fb *T, rating int16, note string) error
	// createNew builds and persists a new feedback entry for the given resource ID.
	createNew func(ctx context.Context, resourceID string, rating int16, note string) (*T, error)
	// sendToLangfuse sends the feedback score to Langfuse asynchronously.
	sendToLangfuse func(resourceID string, rating int16, note string)
	// afterSuccess is an optional callback invoked after successful upsert (before return).
	// It receives the rating and the timestamp of the feedback entry.
	afterSuccess func(rating int16, createdAt time.Time)
}

// handleCreateFeedback is the generic handler for all create-feedback endpoints.
// It decodes the request, validates, upserts feedback, and fires async Langfuse reporting.
func handleCreateFeedback[T any](h *FeedbackHandler, w http.ResponseWriter, r *http.Request, resourceID string, ops feedbackOps[T]) {
	var req feedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Rating < -1 || req.Rating > 1 {
		respondError(w, "rating must be -1, 0, or 1", http.StatusBadRequest)
		return
	}

	existing, err := ops.getExisting(r.Context(), resourceID)
	if err != nil {
		respondError(w, "failed to check existing feedback", http.StatusInternalServerError)
		return
	}

	createdAt := time.Now().UTC()
	if len(existing) > 0 {
		fb := existing[0]
		if err := ops.updateExisting(r.Context(), fb, req.Rating, req.Note); err != nil {
			respondError(w, "failed to update feedback", http.StatusInternalServerError)
			return
		}
		respondJSON(w, fb, http.StatusOK)
	} else {
		fb, err := ops.createNew(r.Context(), resourceID, req.Rating, req.Note)
		if err != nil {
			respondError(w, "failed to create feedback", http.StatusInternalServerError)
			return
		}
		respondJSON(w, fb, http.StatusCreated)
	}

	// Send feedback score to Langfuse (non-blocking)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("panic in langfuse feedback goroutine", "resource_id", resourceID, "error", p)
			}
		}()
		ops.sendToLangfuse(resourceID, req.Rating, req.Note)
	}()

	if ops.afterSuccess != nil {
		ops.afterSuccess(req.Rating, createdAt)
	}
}

// handleGetFeedback is the generic handler for all get-feedback endpoints.
func handleGetFeedback[T any](w http.ResponseWriter, r *http.Request, resourceID string, getFn func(ctx context.Context, id string) ([]*T, error)) {
	fbs, err := getFn(r.Context(), resourceID)
	if err != nil {
		respondError(w, "failed to get feedback", http.StatusInternalServerError)
		return
	}

	if len(fbs) == 0 {
		respondJSON(w, nil, http.StatusOK)
		return
	}

	respondJSON(w, fbs[0], http.StatusOK)
}

// CreateMessageFeedback handles POST /messages/{id}/feedback
func (h *FeedbackHandler) CreateMessageFeedback(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "id")
	handleCreateFeedback(h, w, r, messageID, feedbackOps[domain.MessageFeedback]{
		getExisting: h.store.GetMessageFeedbackByMessage,
		updateExisting: func(ctx context.Context, fb *domain.MessageFeedback, rating int16, note string) error {
			fb.Rating = rating
			fb.Note = note
			return h.store.UpdateMessageFeedback(ctx, fb)
		},
		createNew: func(ctx context.Context, resourceID string, rating int16, note string) (*domain.MessageFeedback, error) {
			fb := &domain.MessageFeedback{
				ID:        store.NewMessageFeedbackID(),
				MessageID: resourceID,
				Rating:    rating,
				Note:      note,
				CreatedAt: time.Now().UTC(),
			}
			return fb, h.store.CreateMessageFeedback(ctx, fb)
		},
		sendToLangfuse: h.sendMessageFeedbackToLangfuse,
		afterSuccess: func(rating int16, createdAt time.Time) {
			// If positive feedback, add the Q&A pair to the golden dataset (async)
			if rating == domain.RatingUp {
				go func() {
					defer func() {
						if p := recover(); p != nil {
							slog.Error("panic in golden example goroutine", "message_id", messageID, "error", p)
						}
					}()
					h.addGoldenExample(messageID, createdAt)
				}()
			}
		},
	})
}

// GetMessageFeedback handles GET /messages/{id}/feedback
func (h *FeedbackHandler) GetMessageFeedback(w http.ResponseWriter, r *http.Request) {
	handleGetFeedback(w, r, chi.URLParam(r, "id"), h.store.GetMessageFeedbackByMessage)
}

// CreateToolUseFeedback handles POST /tool-uses/{id}/feedback
func (h *FeedbackHandler) CreateToolUseFeedback(w http.ResponseWriter, r *http.Request) {
	handleCreateFeedback(h, w, r, chi.URLParam(r, "id"), feedbackOps[domain.ToolUseFeedback]{
		getExisting: h.store.GetToolUseFeedbackByToolUse,
		updateExisting: func(ctx context.Context, fb *domain.ToolUseFeedback, rating int16, note string) error {
			fb.Rating = rating
			fb.Note = note
			return h.store.UpdateToolUseFeedback(ctx, fb)
		},
		createNew: func(ctx context.Context, resourceID string, rating int16, note string) (*domain.ToolUseFeedback, error) {
			fb := &domain.ToolUseFeedback{
				ID:        store.NewToolUseFeedbackID(),
				ToolUseID: resourceID,
				Rating:    rating,
				Note:      note,
				CreatedAt: time.Now().UTC(),
			}
			return fb, h.store.CreateToolUseFeedback(ctx, fb)
		},
		sendToLangfuse: h.sendToolUseFeedbackToLangfuse,
	})
}

// GetToolUseFeedback handles GET /tool-uses/{id}/feedback
func (h *FeedbackHandler) GetToolUseFeedback(w http.ResponseWriter, r *http.Request) {
	handleGetFeedback(w, r, chi.URLParam(r, "id"), h.store.GetToolUseFeedbackByToolUse)
}

// CreateMemoryUseFeedback handles POST /memory-uses/{id}/feedback
func (h *FeedbackHandler) CreateMemoryUseFeedback(w http.ResponseWriter, r *http.Request) {
	handleCreateFeedback(h, w, r, chi.URLParam(r, "id"), feedbackOps[domain.MemoryUseFeedback]{
		getExisting: h.store.GetMemoryUseFeedbackByMemoryUse,
		updateExisting: func(ctx context.Context, fb *domain.MemoryUseFeedback, rating int16, note string) error {
			fb.Rating = rating
			fb.Note = note
			return h.store.UpdateMemoryUseFeedback(ctx, fb)
		},
		createNew: func(ctx context.Context, resourceID string, rating int16, note string) (*domain.MemoryUseFeedback, error) {
			fb := &domain.MemoryUseFeedback{
				ID:          store.NewMemoryUseFeedbackID(),
				MemoryUseID: resourceID,
				Rating:      rating,
				Note:        note,
				CreatedAt:   time.Now().UTC(),
			}
			return fb, h.store.CreateMemoryUseFeedback(ctx, fb)
		},
		sendToLangfuse: h.sendMemoryUseFeedbackToLangfuse,
	})
}

// GetMemoryUseFeedback handles GET /memory-uses/{id}/feedback
func (h *FeedbackHandler) GetMemoryUseFeedback(w http.ResponseWriter, r *http.Request) {
	handleGetFeedback(w, r, chi.URLParam(r, "id"), h.store.GetMemoryUseFeedbackByMemoryUse)
}

// sendMessageFeedbackToLangfuse looks up the message's trace_id and sends feedback to Langfuse.
// This provides proper trace correlation for user feedback on AI responses.
func (h *FeedbackHandler) sendMessageFeedbackToLangfuse(messageID string, rating int16, note string) {
	if h.langfuse == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Look up the trace_id for this message
	traceID, err := h.store.GetMessageTraceID(ctx, messageID)
	if err != nil {
		slog.Error("failed to get trace_id for message", "message_id", messageID, "error", err)
		return
	}
	if traceID == "" {
		slog.Warn("no trace_id found for message, skipping langfuse score", "message_id", messageID)
		return
	}

	params := langfuse.ScoreParams{
		TraceID:  traceID,
		Name:     "user/message_feedback",
		Value:    float64(rating),
		DataType: langfuse.ScoreDataTypeNumeric,
		Comment:  note,
	}

	if err := h.langfuse.CreateScore(ctx, params); err != nil {
		slog.Error("failed to send message feedback to langfuse", "trace_id", traceID, "error", err)
	} else {
		slog.Info("sent user feedback score to langfuse", "rating", rating, "trace_id", traceID, "message_id", messageID)
	}
}

// sendToolUseFeedbackToLangfuse looks up the tool use's message, then the message's trace_id,
// and sends feedback to Langfuse with proper trace correlation.
func (h *FeedbackHandler) sendToolUseFeedbackToLangfuse(toolUseID string, rating int16, note string) {
	if h.langfuse == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Look up the tool use to get the message_id
	toolUse, err := h.store.GetToolUse(ctx, toolUseID)
	if err != nil {
		slog.Error("failed to get tool use for feedback", "tool_use_id", toolUseID, "error", err)
		return
	}

	// Look up the trace_id for the associated message
	traceID, err := h.store.GetMessageTraceID(ctx, toolUse.MessageID)
	if err != nil {
		slog.Error("failed to get trace_id for message", "message_id", toolUse.MessageID, "tool_use_id", toolUseID, "error", err)
		return
	}
	if traceID == "" {
		slog.Warn("no trace_id found for message, skipping langfuse score", "message_id", toolUse.MessageID, "tool_use_id", toolUseID)
		return
	}

	params := langfuse.ScoreParams{
		TraceID:  traceID,
		Name:     "user/tool_use_feedback",
		Value:    float64(rating),
		DataType: langfuse.ScoreDataTypeNumeric,
		Comment:  note,
	}

	if err := h.langfuse.CreateScore(ctx, params); err != nil {
		slog.Error("failed to send tool use feedback to langfuse", "trace_id", traceID, "error", err)
	} else {
		slog.Info("sent tool use feedback to langfuse", "rating", rating, "trace_id", traceID, "tool_use_id", toolUseID)
	}
}

// sendMemoryUseFeedbackToLangfuse looks up the memory use's message, then the message's trace_id,
// and sends feedback to Langfuse with proper trace correlation.
func (h *FeedbackHandler) sendMemoryUseFeedbackToLangfuse(memoryUseID string, rating int16, note string) {
	if h.langfuse == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Look up the memory use to get the message_id
	memoryUse, err := h.store.GetMemoryUse(ctx, memoryUseID)
	if err != nil {
		slog.Error("failed to get memory use for feedback", "memory_use_id", memoryUseID, "error", err)
		return
	}

	// Look up the trace_id for the associated message
	traceID, err := h.store.GetMessageTraceID(ctx, memoryUse.MessageID)
	if err != nil {
		slog.Error("failed to get trace_id for message", "message_id", memoryUse.MessageID, "memory_use_id", memoryUseID, "error", err)
		return
	}
	if traceID == "" {
		slog.Warn("no trace_id found for message, skipping langfuse score", "message_id", memoryUse.MessageID, "memory_use_id", memoryUseID)
		return
	}

	params := langfuse.ScoreParams{
		TraceID:  traceID,
		Name:     "user/memory_use_feedback",
		Value:    float64(rating),
		DataType: langfuse.ScoreDataTypeNumeric,
		Comment:  note,
	}

	if err := h.langfuse.CreateScore(ctx, params); err != nil {
		slog.Error("failed to send memory use feedback to langfuse", "trace_id", traceID, "error", err)
	} else {
		slog.Info("sent memory use feedback to langfuse", "rating", rating, "trace_id", traceID, "memory_use_id", memoryUseID)
	}
}

// addGoldenExample adds a high-quality Q&A pair to the Langfuse golden dataset.
// This is called asynchronously when a message receives positive feedback.
// It fetches the assistant response and its preceding user query, then adds them to the dataset.
func (h *FeedbackHandler) addGoldenExample(messageID string, feedbackTime time.Time) {
	if h.langfuse == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the rated message (should be assistant response)
	msg, err := h.store.GetMessage(ctx, messageID)
	if err != nil {
		slog.Error("failed to get message for golden example", "message_id", messageID, "error", err)
		return
	}

	// Skip if it's not an assistant message (we only want assistant responses)
	if msg.Role != domain.RoleAssistant {
		slog.Info("skipping golden example: not an assistant response", "message_id", messageID, "role", msg.Role)
		return
	}

	// Get the previous message (should be user query)
	if msg.PreviousID == nil {
		slog.Info("skipping golden example: no previous message", "message_id", messageID)
		return
	}

	prevMsg, err := h.store.GetMessage(ctx, *msg.PreviousID)
	if err != nil {
		slog.Error("failed to get previous message for golden example", "previous_id", *msg.PreviousID, "error", err)
		return
	}

	// Verify the previous message is from a user
	if prevMsg.Role != domain.RoleUser {
		slog.Info("skipping golden example: previous message is not a user query", "previous_id", *msg.PreviousID, "role", prevMsg.Role)
		return
	}

	// Add to the golden dataset
	metadata := map[string]any{
		"conversation_id": msg.ConversationID,
		"message_id":      messageID,
		"user_message_id": *msg.PreviousID,
		"feedback_time":   feedbackTime.Format(time.RFC3339),
	}

	if err := langfuse.AddGoldenExample(ctx, h.langfuse, prevMsg.Content, msg.Content, metadata); err != nil {
		slog.Error("failed to add golden example to langfuse dataset", "error", err)
	} else {
		slog.Info("added golden example to langfuse dataset", "conversation_id", msg.ConversationID, "message_id", messageID)
	}
}
