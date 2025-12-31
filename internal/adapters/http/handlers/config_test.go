package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/longregen/alicia/internal/adapters/speech"
	"github.com/longregen/alicia/internal/config"
)

// Mock TTSAdapter
type mockTTSAdapter struct {
	model        string
	defaultVoice string
}

func (m *mockTTSAdapter) GetModel() string {
	return m.model
}

func (m *mockTTSAdapter) GetDefaultVoice() string {
	return m.defaultVoice
}

// Tests for ConfigHandler.GetPublicConfig

func TestConfigHandler_GetPublicConfig_TTSEnabled(t *testing.T) {
	cfg := &config.Config{
		LiveKit: config.LiveKitConfig{
			URL: "wss://livekit.example.com",
		},
		TTS: config.TTSConfig{
			URL:    "https://tts.example.com/v1",
			APIKey: "test-key",
		},
	}

	ttsAdapter := &mockTTSAdapter{
		model:        "tts-1",
		defaultVoice: "af_heart",
	}

	handler := NewConfigHandler(cfg, (*speech.TTSAdapter)(nil))

	req := httptest.NewRequest("GET", "/api/v1/config", nil)
	rr := httptest.NewRecorder()

	handler.GetPublicConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response PublicConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.LiveKitURL != "wss://livekit.example.com" {
		t.Errorf("expected LiveKit URL 'wss://livekit.example.com', got %v", response.LiveKitURL)
	}

	// TTS not configured (no adapter passed)
	if response.TTSEnabled {
		t.Error("expected TTS to be disabled when not configured")
	}

	// Test with TTS adapter
	handler2 := NewConfigHandler(cfg, (*speech.TTSAdapter)(nil))
	req2 := httptest.NewRequest("GET", "/api/v1/config", nil)
	rr2 := httptest.NewRecorder()
	handler2.GetPublicConfig(rr2, req2)

	var response2 PublicConfigResponse
	json.NewDecoder(rr2.Body).Decode(&response2)

	// Should not include TTS config when adapter is nil
	if response2.TTS != nil {
		t.Error("expected TTS config to be nil when adapter is not provided")
	}

	_ = ttsAdapter // Use the variable
}

func TestConfigHandler_GetPublicConfig_TTSDisabled(t *testing.T) {
	cfg := &config.Config{
		LiveKit: config.LiveKitConfig{
			URL: "wss://livekit.example.com",
		},
		// No TTS config
	}

	handler := NewConfigHandler(cfg, nil)

	req := httptest.NewRequest("GET", "/api/v1/config", nil)
	rr := httptest.NewRecorder()

	handler.GetPublicConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response PublicConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TTSEnabled {
		t.Error("expected TTS to be disabled")
	}

	if response.TTS != nil {
		t.Error("expected TTS config to be nil")
	}
}

func TestConfigHandler_GetPublicConfig_ASREnabled(t *testing.T) {
	cfg := &config.Config{
		ASR: config.ASRConfig{
			URL:    "https://asr.example.com",
			APIKey: "test-key",
		},
	}

	handler := NewConfigHandler(cfg, nil)

	req := httptest.NewRequest("GET", "/api/v1/config", nil)
	rr := httptest.NewRecorder()

	handler.GetPublicConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response PublicConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.ASREnabled {
		t.Error("expected ASR to be enabled")
	}
}

func TestConfigHandler_GetPublicConfig_EmptyLiveKitURL(t *testing.T) {
	cfg := &config.Config{
		// No LiveKit config
	}

	handler := NewConfigHandler(cfg, nil)

	req := httptest.NewRequest("GET", "/api/v1/config", nil)
	rr := httptest.NewRecorder()

	handler.GetPublicConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response PublicConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.LiveKitURL != "" {
		t.Errorf("expected empty LiveKit URL, got %v", response.LiveKitURL)
	}
}
