package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/application/services"
)

// OptimizationStreamHandler handles SSE streaming for optimization progress
type OptimizationStreamHandler struct {
	optService *services.OptimizationService
}

// NewOptimizationStreamHandler creates a new optimization stream handler
func NewOptimizationStreamHandler(
	optService *services.OptimizationService,
) *OptimizationStreamHandler {
	return &OptimizationStreamHandler{
		optService: optService,
	}
}

// StreamOptimizationProgress handles GET /api/v1/optimizations/{id}/stream
// Establishes SSE connection for real-time optimization progress
func (h *OptimizationStreamHandler) StreamOptimizationProgress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	runID, ok := validateURLParam(r, w, "id", "Optimization run ID")
	if !ok {
		return
	}

	// Verify run exists
	run, err := h.optService.GetOptimizationRun(r.Context(), runID)
	if err != nil {
		respondError(w, "not_found", "Optimization run not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, "internal_error", "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	h.sendEvent(w, flusher, services.OptimizationProgressEvent{
		Type:      "connected",
		RunID:     runID,
		Status:    string(run.Status),
		Timestamp: time.Now().Format(time.RFC3339),
	})

	log.Printf("SSE: Optimization progress stream established for run %s (user: %s)", runID, userID)

	// Subscribe to progress updates
	progressChan := h.optService.SubscribeProgress(runID)
	defer h.optService.UnsubscribeProgress(runID, progressChan)

	// Setup keepalive ticker
	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			log.Printf("SSE: Client disconnected from optimization run %s", runID)
			return

		case event, ok := <-progressChan:
			if !ok {
				// Channel closed, optimization complete
				log.Printf("SSE: Progress channel closed for run %s", runID)
				return
			}

			// Send the event to the client
			h.sendEvent(w, flusher, event)

			// If this is a terminal event (completed or failed), close the connection
			if event.Type == "completed" || event.Type == "failed" {
				log.Printf("SSE: Optimization run %s completed with status: %s", runID, event.Status)
				return
			}

		case <-keepaliveTicker.C:
			// Send keepalive comment to prevent connection timeout
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// sendEvent sends an SSE event to the client
func (h *OptimizationStreamHandler) sendEvent(w http.ResponseWriter, flusher http.Flusher, event services.OptimizationProgressEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("SSE: Error marshaling event: %v", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
