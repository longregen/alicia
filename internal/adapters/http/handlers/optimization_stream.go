package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// OptimizationProgressEvent represents a progress update during optimization
type OptimizationProgressEvent struct {
	Type           string                 `json:"type"`
	RunID          string                 `json:"run_id"`
	Iteration      int                    `json:"iteration"`
	MaxIterations  int                    `json:"max_iterations"`
	CurrentScore   float64                `json:"current_score"`
	BestScore      float64                `json:"best_score"`
	DimensionScores map[string]float64     `json:"dimension_scores,omitempty"`
	Status         string                 `json:"status"`
	Message        string                 `json:"message,omitempty"`
	Timestamp      string                 `json:"timestamp"`
}

// OptimizationStreamHandler handles SSE streaming for optimization progress
type OptimizationStreamHandler struct {
	optService ports.OptimizationService
	repo       ports.PromptOptimizationRepository
}

// NewOptimizationStreamHandler creates a new optimization stream handler
func NewOptimizationStreamHandler(
	optService ports.OptimizationService,
	repo ports.PromptOptimizationRepository,
) *OptimizationStreamHandler {
	return &OptimizationStreamHandler{
		optService: optService,
		repo:       repo,
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
	h.sendEvent(w, flusher, OptimizationProgressEvent{
		Type:      "connected",
		RunID:     runID,
		Status:    string(run.Status),
		Timestamp: time.Now().Format(time.RFC3339),
	})

	log.Printf("SSE: Optimization progress stream established for run %s (user: %s)", runID, userID)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Poll for updates
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastIteration := 0
	lastScore := 0.0

	for {
		select {
		case <-ctx.Done():
			log.Printf("SSE: Client disconnected from optimization run %s", runID)
			return

		case <-ticker.C:
			// Fetch current run state
			currentRun, err := h.repo.GetRun(ctx, runID)
			if err != nil {
				log.Printf("SSE: Error fetching run %s: %v", runID, err)
				continue
			}

			// Check if there's progress to report
			if currentRun.Iterations > lastIteration || currentRun.BestScore != lastScore {
				event := OptimizationProgressEvent{
					Type:            "progress",
					RunID:           runID,
					Iteration:       currentRun.Iterations,
					MaxIterations:   currentRun.MaxIterations,
					CurrentScore:    currentRun.BestScore,
					BestScore:       currentRun.BestScore,
					DimensionScores: currentRun.BestDimScores,
					Status:          string(currentRun.Status),
					Timestamp:       time.Now().Format(time.RFC3339),
				}

				h.sendEvent(w, flusher, event)

				lastIteration = currentRun.Iterations
				lastScore = currentRun.BestScore
			}

			// Check if run is complete or failed
			if currentRun.Status == models.OptimizationStatusCompleted ||
				currentRun.Status == models.OptimizationStatusFailed {

				finalEvent := OptimizationProgressEvent{
					Type:            "completed",
					RunID:           runID,
					Iteration:       currentRun.Iterations,
					MaxIterations:   currentRun.MaxIterations,
					CurrentScore:    currentRun.BestScore,
					BestScore:       currentRun.BestScore,
					DimensionScores: currentRun.BestDimScores,
					Status:          string(currentRun.Status),
					Timestamp:       time.Now().Format(time.RFC3339),
				}

				if currentRun.Status == models.OptimizationStatusFailed {
					finalEvent.Type = "failed"
					if reason, ok := currentRun.Config["failure_reason"].(string); ok {
						finalEvent.Message = reason
					}
				}

				h.sendEvent(w, flusher, finalEvent)

				log.Printf("SSE: Optimization run %s completed with status: %s", runID, currentRun.Status)
				return
			}

			// Send keepalive every 30 seconds
			if time.Now().Unix()%30 == 0 {
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	}
}

// sendEvent sends an SSE event to the client
func (h *OptimizationStreamHandler) sendEvent(w http.ResponseWriter, flusher http.Flusher, event OptimizationProgressEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("SSE: Error marshaling event: %v", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
