package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/longregen/alicia/pkg/langfuse"
)

type HealthHandler struct {
	langfuse *langfuse.Client
	dbPing   func(context.Context) error
}

type HealthHandlerConfig struct {
	Langfuse *langfuse.Client
	DBPing   func(context.Context) error
}

func NewHealthHandler(cfg HealthHandlerConfig) *HealthHandler {
	return &HealthHandler{
		langfuse: cfg.Langfuse,
		dbPing:   cfg.DBPing,
	}
}

// HealthStatus represents the overall health status response.
type HealthStatus struct {
	Status     string              `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp  time.Time           `json:"timestamp"`
	Components map[string]Component `json:"components"`
}

// Component represents a single component's health status.
type Component struct {
	Status  string `json:"status"` // "healthy", "unhealthy"
	Message string `json:"message,omitempty"`
	Latency int64  `json:"latency_ms,omitempty"`
}

// Health handles GET /health/full
// This endpoint checks all service dependencies.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	status := HealthStatus{
		Timestamp:  time.Now().UTC(),
		Status:     "healthy",
		Components: make(map[string]Component),
	}

	// Check database
	if h.dbPing != nil {
		start := time.Now()
		err := h.dbPing(ctx)
		latency := time.Since(start).Milliseconds()

		if err != nil {
			status.Components["database"] = Component{
				Status:  "unhealthy",
				Message: err.Error(),
				Latency: latency,
			}
			status.Status = "unhealthy"
		} else {
			status.Components["database"] = Component{
				Status:  "healthy",
				Latency: latency,
			}
		}
	}

	// Check Langfuse
	if h.langfuse != nil {
		start := time.Now()
		err := h.langfuse.Ping(ctx)
		latency := time.Since(start).Milliseconds()

		if err != nil {
			status.Components["langfuse"] = Component{
				Status:  "unhealthy",
				Message: err.Error(),
				Latency: latency,
			}
			// Langfuse is not critical, so we only degrade instead of unhealthy
			if status.Status == "healthy" {
				status.Status = "degraded"
			}
		} else {
			status.Components["langfuse"] = Component{
				Status:  "healthy",
				Latency: latency,
			}
		}
	}

	httpStatus := http.StatusOK
	if status.Status == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(status)
}

// Readiness handles GET /health/ready
// This is a lightweight check for load balancer health checks.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Only check database for readiness (critical dependency)
	if h.dbPing != nil {
		if err := h.dbPing(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("database unavailable"))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// Liveness handles GET /health/live
// This is a minimal check that the service is running.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("alive"))
}
