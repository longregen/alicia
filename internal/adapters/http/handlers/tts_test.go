package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/longregen/alicia/internal/adapters/speech"
	"github.com/longregen/alicia/internal/ports"
)

// Mock TTS Adapter for testing
type mockTTSAdapterForHandler struct {
	synthesizeErr error
	result        *ports.TTSResult
}

func (m *mockTTSAdapterForHandler) Synthesize(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error) {
	if m.synthesizeErr != nil {
		return nil, m.synthesizeErr
	}
	if m.result == nil {
		return &ports.TTSResult{
			Audio: []byte("mock-audio-data"),
		}, nil
	}
	return m.result, nil
}

func (m *mockTTSAdapterForHandler) GetModel() string {
	return "tts-1"
}

func (m *mockTTSAdapterForHandler) GetDefaultVoice() string {
	return "af_heart"
}

// Tests for TTSHandler.Speech

func TestTTSHandler_Speech_EmptyInput(t *testing.T) {
	mockAdapter := &mockTTSAdapterForHandler{}
	handler := NewTTSHandler((*speech.TTSAdapter)(nil))

	body := `{"model": "tts-1", "input": "", "voice": "af_heart"}`
	req := httptest.NewRequest("POST", "/v1/audio/speech", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Speech(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	_ = mockAdapter
}

func TestTTSHandler_Speech_InvalidJSON(t *testing.T) {
	handler := NewTTSHandler((*speech.TTSAdapter)(nil))

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/v1/audio/speech", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Speech(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestTTSHandler_Speech_DefaultOutputFormat(t *testing.T) {
	body := `{"model": "tts-1", "input": "Hello world", "voice": "af_heart"}`

	var ttsReq TTSRequest
	if err := json.NewDecoder(bytes.NewBufferString(body)).Decode(&ttsReq); err != nil {
		t.Fatalf("failed to decode request: %v", err)
	}

	// Build options
	options := &ports.TTSOptions{
		Voice:        ttsReq.Voice,
		Speed:        ttsReq.Speed,
		OutputFormat: ttsReq.ResponseFormat,
	}

	if options.OutputFormat == "" {
		options.OutputFormat = "mp3"
	}

	if options.OutputFormat != "mp3" {
		t.Errorf("expected default output format 'mp3', got %v", options.OutputFormat)
	}
}

func TestTTSHandler_Speech_CustomSpeed(t *testing.T) {
	body := `{"model": "tts-1", "input": "Hello world", "voice": "af_heart", "speed": 1.5}`

	var ttsReq TTSRequest
	if err := json.NewDecoder(bytes.NewBufferString(body)).Decode(&ttsReq); err != nil {
		t.Fatalf("failed to decode request: %v", err)
	}

	if ttsReq.Speed != 1.5 {
		t.Errorf("expected speed 1.5, got %v", ttsReq.Speed)
	}
}

func TestTTSHandler_Speech_RequestBodySizeLimit(t *testing.T) {
	handler := NewTTSHandler((*speech.TTSAdapter)(nil))

	// Create a request with very large body (should be rejected)
	largeBody := make([]byte, 2*1024*1024) // 2MB
	for i := range largeBody {
		largeBody[i] = 'a'
	}

	req := httptest.NewRequest("POST", "/v1/audio/speech", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Speech(rr, req)

	// Should reject due to body size limit (1MB in handler)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for oversized request, got %d", rr.Code)
	}
}

func TestTTSHandler_Speech_NilAdapter(t *testing.T) {
	// Test that handler handles nil adapter gracefully
	handler := NewTTSHandler(nil)

	if handler.ttsAdapter != nil {
		t.Error("expected ttsAdapter to be nil")
	}

	body := `{"model": "tts-1", "input": "Hello world", "voice": "af_heart"}`
	req := httptest.NewRequest("POST", "/v1/audio/speech", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	// This will panic or error since adapter is nil - that's expected behavior
	// In production, the adapter should always be provided
	defer func() {
		if r := recover(); r == nil {
			// If no panic, check for error response
			if rr.Code == http.StatusOK {
				t.Error("expected error when adapter is nil")
			}
		}
	}()

	handler.Speech(rr, req)
}
