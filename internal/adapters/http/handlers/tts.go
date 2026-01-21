package handlers

import (
	"net/http"

	"github.com/longregen/alicia/internal/adapters/speech"
	"github.com/longregen/alicia/internal/ports"
)

type TTSHandler struct {
	ttsAdapter *speech.TTSAdapter
}

func NewTTSHandler(ttsAdapter *speech.TTSAdapter) *TTSHandler {
	return &TTSHandler{
		ttsAdapter: ttsAdapter,
	}
}

type TTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float32 `json:"speed,omitempty"`
}

func (h *TTSHandler) Speech(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024)
	defer r.Body.Close()

	req, ok := decodeJSON[TTSRequest](r, w)
	if !ok {
		return
	}

	if req.Input == "" {
		respondError(w, "validation_error", "Input text is required", http.StatusBadRequest)
		return
	}

	options := &ports.TTSOptions{
		Voice:        req.Voice,
		Speed:        req.Speed,
		OutputFormat: req.ResponseFormat,
	}

	if options.OutputFormat == "" {
		options.OutputFormat = "mp3"
	}

	result, err := h.ttsAdapter.Synthesize(r.Context(), req.Input, options)
	if err != nil {
		respondError(w, "tts_error", "Failed to synthesize speech: "+err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := "audio/mpeg"
	switch options.OutputFormat {
	case "opus":
		contentType = "audio/opus"
	case "aac":
		contentType = "audio/aac"
	case "flac":
		contentType = "audio/flac"
	case "wav":
		contentType = "audio/wav"
	case "pcm":
		contentType = "audio/pcm"
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(result.Audio)
}
