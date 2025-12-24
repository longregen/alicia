package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestHealthHandler_ServiceHealth_JSONMarshaling(t *testing.T) {
	latency := int64(100)
	errMsg := "connection refused"

	tests := []struct {
		name    string
		health  ServiceHealth
		wantErr bool
	}{
		{
			name: "healthy service",
			health: ServiceHealth{
				Status:    "healthy",
				LatencyMs: &latency,
			},
			wantErr: false,
		},
		{
			name: "unhealthy service",
			health: ServiceHealth{
				Status:    "unhealthy",
				LatencyMs: &latency,
				Error:     &errMsg,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.health)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var unmarshaled ServiceHealth
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Errorf("json.Unmarshal() error = %v", err)
				return
			}

			if unmarshaled.Status != tt.health.Status {
				t.Errorf("Status = %v, want %v", unmarshaled.Status, tt.health.Status)
			}
		})
	}
}

func TestHealthHandler_CreateSilentWAV(t *testing.T) {
	// Test creating a silent WAV file
	wav := createSilentWAV(16000, 1)

	// Check minimum size (44 byte header + 1 second at 16kHz 16-bit mono = 32000 bytes)
	expectedSize := 44 + 32000
	if len(wav) != expectedSize {
		t.Errorf("expected WAV size %d, got %d", expectedSize, len(wav))
	}

	// Check RIFF header
	if string(wav[0:4]) != "RIFF" {
		t.Errorf("expected RIFF header, got %s", string(wav[0:4]))
	}

	// Check WAVE format
	if string(wav[8:12]) != "WAVE" {
		t.Errorf("expected WAVE format, got %s", string(wav[8:12]))
	}

	// Check fmt chunk
	if string(wav[12:16]) != "fmt " {
		t.Errorf("expected fmt chunk, got %s", string(wav[12:16]))
	}

	// Check data chunk
	if string(wav[36:40]) != "data" {
		t.Errorf("expected data chunk, got %s", string(wav[36:40]))
	}
}

func TestHealthHandler_HealthCheckConfig(t *testing.T) {
	cfg := DefaultHealthCheckConfig()

	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", cfg.Timeout)
	}
}
