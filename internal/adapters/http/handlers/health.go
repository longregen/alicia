package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/adapters/embedding"
	"github.com/longregen/alicia/internal/adapters/speech"
	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/llm"
	"github.com/longregen/alicia/internal/ports"
)

// HealthCheckConfig holds configuration for health checks
type HealthCheckConfig struct {
	Timeout time.Duration // Timeout for each individual health check
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Timeout: 5 * time.Second,
	}
}

type HealthHandler struct {
	config          HealthCheckConfig
	cfg             *config.Config
	db              *pgxpool.Pool
	llmClient       *llm.Client
	asrAdapter      *speech.ASRAdapter
	ttsAdapter      *speech.TTSAdapter
	embeddingClient *embedding.Client
	liveKitService  ports.LiveKitService
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		config: DefaultHealthCheckConfig(),
	}
}

func NewHealthHandlerWithDeps(
	cfg *config.Config,
	db *pgxpool.Pool,
	llmClient *llm.Client,
	asrAdapter *speech.ASRAdapter,
	ttsAdapter *speech.TTSAdapter,
	embeddingClient *embedding.Client,
	liveKitService ports.LiveKitService,
) *HealthHandler {
	return &HealthHandler{
		config:          DefaultHealthCheckConfig(),
		cfg:             cfg,
		db:              db,
		llmClient:       llmClient,
		asrAdapter:      asrAdapter,
		ttsAdapter:      ttsAdapter,
		embeddingClient: embeddingClient,
		liveKitService:  liveKitService,
	}
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

type DetailedHealthResponse struct {
	Status   string                   `json:"status"`
	Version  string                   `json:"version"`
	Services map[string]ServiceHealth `json:"services"`
}

type ServiceHealth struct {
	Status    string  `json:"status"`
	LatencyMs *int64  `json:"latency_ms,omitempty"`
	Error     *string `json:"error,omitempty"`
}

// Handle provides a basic health check endpoint
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleDetailed provides a detailed health check endpoint that checks all dependencies
func (h *HealthHandler) HandleDetailed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	response := DetailedHealthResponse{
		Version:  "1.0.0",
		Services: make(map[string]ServiceHealth),
	}

	// Check database
	if h.db != nil {
		response.Services["database"] = h.checkDatabase(ctx)
	}

	// Check LLM service
	if h.llmClient != nil {
		response.Services["llm"] = h.checkLLM(ctx)
	}

	// Check ASR service (if configured)
	if h.cfg != nil && h.cfg.IsASRConfigured() && h.asrAdapter != nil {
		response.Services["asr"] = h.checkASR(ctx)
	}

	// Check TTS service (if configured)
	if h.cfg != nil && h.cfg.IsTTSConfigured() && h.ttsAdapter != nil {
		response.Services["tts"] = h.checkTTS(ctx)
	}

	// Check Embedding service (if configured)
	if h.cfg != nil && h.cfg.IsEmbeddingConfigured() && h.embeddingClient != nil {
		response.Services["embedding"] = h.checkEmbedding(ctx)
	}

	// Check LiveKit service (if configured)
	if h.cfg != nil && h.cfg.IsLiveKitConfigured() && h.liveKitService != nil {
		response.Services["livekit"] = h.checkLiveKit(ctx)
	}

	// Determine overall status
	response.Status = h.calculateOverallStatus(response.Services)

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if response.Status == "degraded" {
		statusCode = http.StatusOK // 200 OK but degraded
	} else if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable // 503
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// checkDatabase checks database connectivity
func (h *HealthHandler) checkDatabase(ctx context.Context) ServiceHealth {
	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	err := h.db.Ping(checkCtx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errMsg := err.Error()
		return ServiceHealth{
			Status:    "unhealthy",
			LatencyMs: &latency,
			Error:     &errMsg,
		}
	}

	return ServiceHealth{
		Status:    "healthy",
		LatencyMs: &latency,
	}
}

// checkLLM checks LLM service availability
func (h *HealthHandler) checkLLM(ctx context.Context) ServiceHealth {
	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	// Simple ping by sending a minimal chat request
	messages := []llm.ChatMessage{
		{Role: "system", Content: "health check"},
		{Role: "user", Content: "ping"},
	}

	_, err := h.llmClient.Chat(checkCtx, messages)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errMsg := err.Error()
		return ServiceHealth{
			Status:    "unhealthy",
			LatencyMs: &latency,
			Error:     &errMsg,
		}
	}

	return ServiceHealth{
		Status:    "healthy",
		LatencyMs: &latency,
	}
}

// checkASR checks ASR service availability
func (h *HealthHandler) checkASR(ctx context.Context) ServiceHealth {
	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	// Create a minimal silent WAV file (44 bytes header + 1 second of silence at 16kHz, 16-bit mono)
	silentWAV := createSilentWAV(16000, 1) // 1 second

	_, err := h.asrAdapter.Transcribe(checkCtx, silentWAV, "wav")
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errMsg := err.Error()
		return ServiceHealth{
			Status:    "unhealthy",
			LatencyMs: &latency,
			Error:     &errMsg,
		}
	}

	return ServiceHealth{
		Status:    "healthy",
		LatencyMs: &latency,
	}
}

// checkTTS checks TTS service availability
func (h *HealthHandler) checkTTS(ctx context.Context) ServiceHealth {
	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	// Simple synthesis with minimal text
	_, err := h.ttsAdapter.Synthesize(checkCtx, "health", nil)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errMsg := err.Error()
		return ServiceHealth{
			Status:    "unhealthy",
			LatencyMs: &latency,
			Error:     &errMsg,
		}
	}

	return ServiceHealth{
		Status:    "healthy",
		LatencyMs: &latency,
	}
}

// checkEmbedding checks embedding service availability
func (h *HealthHandler) checkEmbedding(ctx context.Context) ServiceHealth {
	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	// Simple embedding with minimal text
	_, err := h.embeddingClient.Embed(checkCtx, "health check")
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errMsg := err.Error()
		return ServiceHealth{
			Status:    "unhealthy",
			LatencyMs: &latency,
			Error:     &errMsg,
		}
	}

	return ServiceHealth{
		Status:    "healthy",
		LatencyMs: &latency,
	}
}

// checkLiveKit checks LiveKit service availability
func (h *HealthHandler) checkLiveKit(ctx context.Context) ServiceHealth {
	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	// Try to create a temporary room and delete it
	testRoomName := fmt.Sprintf("healthcheck_%d", time.Now().Unix())

	_, err := h.liveKitService.CreateRoom(checkCtx, testRoomName)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		errMsg := err.Error()
		return ServiceHealth{
			Status:    "unhealthy",
			LatencyMs: &latency,
			Error:     &errMsg,
		}
	}

	// Clean up the test room
	_ = h.liveKitService.DeleteRoom(checkCtx, testRoomName)

	latency := time.Since(start).Milliseconds()
	return ServiceHealth{
		Status:    "healthy",
		LatencyMs: &latency,
	}
}

// calculateOverallStatus determines the overall system status based on individual services
func (h *HealthHandler) calculateOverallStatus(services map[string]ServiceHealth) string {
	if len(services) == 0 {
		return "healthy" // No services to check
	}

	hasUnhealthy := false
	hasDegraded := false

	for name, service := range services {
		if service.Status == "unhealthy" {
			// Core services (database, llm) are critical
			if name == "database" || name == "llm" {
				return "unhealthy"
			}
			hasUnhealthy = true
		}
		if service.Status == "degraded" {
			hasDegraded = true
		}
	}

	// If optional services are down, system is degraded
	if hasUnhealthy || hasDegraded {
		return "degraded"
	}

	return "healthy"
}

// createSilentWAV creates a minimal silent WAV file for health checking
func createSilentWAV(sampleRate, durationSec int) []byte {
	numSamples := sampleRate * durationSec
	dataSize := numSamples * 2 // 16-bit = 2 bytes per sample
	fileSize := 36 + dataSize  // 44 byte header - 8 + data size

	buf := new(bytes.Buffer)

	// RIFF header
	buf.WriteString("RIFF")
	writeUint32(buf, uint32(fileSize))
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	writeUint32(buf, 16)                   // chunk size
	writeUint16(buf, 1)                    // audio format (PCM)
	writeUint16(buf, 1)                    // number of channels (mono)
	writeUint32(buf, uint32(sampleRate))   // sample rate
	writeUint32(buf, uint32(sampleRate*2)) // byte rate
	writeUint16(buf, 2)                    // block align
	writeUint16(buf, 16)                   // bits per sample

	// data chunk
	buf.WriteString("data")
	writeUint32(buf, uint32(dataSize))

	// Write silent samples (all zeros)
	silence := make([]byte, dataSize)
	buf.Write(silence)

	return buf.Bytes()
}

func writeUint32(buf *bytes.Buffer, val uint32) {
	buf.WriteByte(byte(val))
	buf.WriteByte(byte(val >> 8))
	buf.WriteByte(byte(val >> 16))
	buf.WriteByte(byte(val >> 24))
}

func writeUint16(buf *bytes.Buffer, val uint16) {
	buf.WriteByte(byte(val))
	buf.WriteByte(byte(val >> 8))
}
