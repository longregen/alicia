package handlers

import (
	"net/http"
	"strings"

	"github.com/longregen/alicia/internal/adapters/speech"
	"github.com/longregen/alicia/internal/config"
)

type ConfigHandler struct {
	cfg        *config.Config
	ttsAdapter *speech.TTSAdapter
}

func NewConfigHandler(cfg *config.Config, ttsAdapter *speech.TTSAdapter) *ConfigHandler {
	return &ConfigHandler{
		cfg:        cfg,
		ttsAdapter: ttsAdapter,
	}
}

// Voice represents a TTS voice option
type Voice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// voiceList contains the curated list of TTS voices
var voiceList = []Voice{
	// American Female
	{ID: "af_heart", Name: "Heart (Default)", Category: "American Female"},
	{ID: "af", Name: "Mixed (Bella + Sarah)", Category: "American Female"},
	{ID: "af_bella", Name: "Bella", Category: "American Female"},
	{ID: "af_sarah", Name: "Sarah", Category: "American Female"},
	{ID: "af_nicole", Name: "Nicole", Category: "American Female"},
	{ID: "af_sky", Name: "Sky", Category: "American Female"},
	// American Male
	{ID: "am_adam", Name: "Adam", Category: "American Male"},
	{ID: "am_michael", Name: "Michael", Category: "American Male"},
	// British Female
	{ID: "bf_emma", Name: "Emma", Category: "British Female"},
	{ID: "bf_isabella", Name: "Isabella", Category: "British Female"},
	// British Male
	{ID: "bm_george", Name: "George", Category: "British Male"},
	{ID: "bm_lewis", Name: "Lewis", Category: "British Male"},
}

// TTSConfig contains TTS configuration exposed to clients
type TTSConfig struct {
	Endpoint     string  `json:"endpoint"`
	Model        string  `json:"model"`
	DefaultVoice string  `json:"default_voice"`
	DefaultSpeed float32 `json:"default_speed"`
	SpeedMin     float32 `json:"speed_min"`
	SpeedMax     float32 `json:"speed_max"`
	SpeedStep    float32 `json:"speed_step"`
	Voices       []Voice `json:"voices"`
}

// PublicConfigResponse contains only the configuration safe to expose to clients
type PublicConfigResponse struct {
	LiveKitURL string     `json:"livekit_url,omitempty"`
	TTSEnabled bool       `json:"tts_enabled"`
	ASREnabled bool       `json:"asr_enabled"`
	TTS        *TTSConfig `json:"tts,omitempty"`
}

// GetPublicConfig handles GET /api/v1/config
// Returns public configuration that clients need
func (h *ConfigHandler) GetPublicConfig(w http.ResponseWriter, r *http.Request) {
	response := &PublicConfigResponse{
		LiveKitURL: h.cfg.LiveKit.URL,
		TTSEnabled: h.cfg.IsTTSConfigured(),
		ASREnabled: h.cfg.IsASRConfigured(),
	}

	// Add TTS configuration if TTS is enabled
	if h.cfg.IsTTSConfigured() && h.ttsAdapter != nil {
		baseURL := strings.TrimSuffix(h.cfg.TTS.URL, "/")
		baseURL = strings.TrimSuffix(baseURL, "/v1")
		ttsConfig := &TTSConfig{
			Endpoint:     baseURL + "/v1/audio/speech",
			Model:        h.ttsAdapter.GetModel(),
			DefaultVoice: h.ttsAdapter.GetDefaultVoice(),
			DefaultSpeed: 1.0,
			SpeedMin:     0.5,
			SpeedMax:     2.0,
			SpeedStep:    0.1,
		}

		ttsConfig.Voices = voiceList
		response.TTS = ttsConfig
	}

	respondJSON(w, response, http.StatusOK)
}
