package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_Handle_Success(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response.Status)
	}

	if response.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", response.Version)
	}
}

func TestHealthHandler_Handle_ContentType(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestHealthHandler_HandleDetailed_NoDependencies(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest("GET", "/health/detailed", nil)
	rr := httptest.NewRecorder()

	handler.HandleDetailed(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response DetailedHealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", response.Status)
	}

	if response.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", response.Version)
	}

	// Should have no services since no dependencies were provided
	if len(response.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(response.Services))
	}
}

func TestHealthHandler_HandleDetailed_WithMockDatabase(t *testing.T) {
	// Create a test database connection (requires real database or mock)
	// For unit tests, we'll test the structure without a real DB
	handler := NewHealthHandler()

	req := httptest.NewRequest("GET", "/health/detailed", nil)
	rr := httptest.NewRecorder()

	handler.HandleDetailed(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestHealthHandler_CalculateOverallStatus(t *testing.T) {
	handler := NewHealthHandler()

	tests := []struct {
		name     string
		services map[string]ServiceHealth
		want     string
	}{
		{
			name:     "no services",
			services: map[string]ServiceHealth{},
			want:     "healthy",
		},
		{
			name: "all healthy",
			services: map[string]ServiceHealth{
				"database": {Status: "healthy"},
				"llm":      {Status: "healthy"},
				"asr":      {Status: "healthy"},
			},
			want: "healthy",
		},
		{
			name: "database unhealthy",
			services: map[string]ServiceHealth{
				"database": {Status: "unhealthy"},
				"llm":      {Status: "healthy"},
			},
			want: "unhealthy",
		},
		{
			name: "llm unhealthy",
			services: map[string]ServiceHealth{
				"database": {Status: "healthy"},
				"llm":      {Status: "unhealthy"},
			},
			want: "unhealthy",
		},
		{
			name: "optional service unhealthy",
			services: map[string]ServiceHealth{
				"database": {Status: "healthy"},
				"llm":      {Status: "healthy"},
				"asr":      {Status: "unhealthy"},
			},
			want: "degraded",
		},
		{
			name: "service degraded",
			services: map[string]ServiceHealth{
				"database": {Status: "healthy"},
				"llm":      {Status: "healthy"},
				"tts":      {Status: "degraded"},
			},
			want: "degraded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.calculateOverallStatus(tt.services)
			if got != tt.want {
				t.Errorf("calculateOverallStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
