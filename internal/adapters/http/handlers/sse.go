package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/ports"
)

type SSEHandler struct {
	conversationRepo ports.ConversationRepository
	broadcaster      *SSEBroadcaster
}

func NewSSEHandler(
	conversationRepo ports.ConversationRepository,
	broadcaster *SSEBroadcaster,
) *SSEHandler {
	return &SSEHandler{
		conversationRepo: conversationRepo,
		broadcaster:      broadcaster,
	}
}

// StreamEvents handles GET /api/v1/conversations/{id}/events
// Establishes SSE connection for real-time message updates
func (h *SSEHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	// Verify conversation exists and belongs to the user
	conversation, err := h.conversationRepo.GetByIDAndUserID(r.Context(), conversationID, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, "not_found", "Conversation not found or access denied", http.StatusNotFound)
		} else {
			respondError(w, "internal_error", "Failed to retrieve conversation", http.StatusInternalServerError)
		}
		return
	}

	if !requireActiveConversation(conversation, w) {
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable buffering for nginx

	// Get flusher for immediate writes
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, "internal_error", "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to events for this conversation
	eventChan := h.broadcaster.Subscribe(conversationID)
	defer h.broadcaster.Unsubscribe(conversationID, eventChan)

	// Send initial connection event
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"conversation_id\":\"%s\"}\n\n", conversationID)
	flusher.Flush()

	log.Printf("SSE: Connection established for conversation %s (user: %s)", conversationID, userID)

	// Create context with cancel for cleanup
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Send keepalive pings every 30 seconds to prevent timeout
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			log.Printf("SSE: Client disconnected from conversation %s", conversationID)
			return

		case event, ok := <-eventChan:
			if !ok {
				// Channel closed
				return
			}

			// Write event to stream
			_, err := fmt.Fprint(w, event)
			if err != nil {
				log.Printf("SSE: Error writing event: %v", err)
				return
			}
			flusher.Flush()

		case <-ticker.C:
			// Send keepalive ping
			_, err := fmt.Fprintf(w, ": keepalive\n\n")
			if err != nil {
				log.Printf("SSE: Error writing keepalive: %v", err)
				return
			}
			flusher.Flush()
		}
	}
}
