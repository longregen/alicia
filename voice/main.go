package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/config"
)

const (
	// DefaultTTSSampleRate is the sample rate for TTS output audio (Kokoro outputs at 24kHz).
	DefaultTTSSampleRate = 24000
	// DefaultASRSampleRate is the sample rate Whisper expects for optimal transcription.
	DefaultASRSampleRate = 16000
	// DefaultCaptureSampleRate is the sample rate for LiveKit audio capture (Opus standard).
	DefaultCaptureSampleRate = 48000
)

type Config struct {
	BackendWSURL string
	AgentSecret  string

	LiveKitURL       string
	LiveKitAPIKey    string
	LiveKitAPISecret string

	ASRURL   string
	ASRModel string

	TTSURL        string
	TTSVoice      string
	TTSSampleRate int

	SampleRate      int
	Channels        int
	VADThreshold    float64
	SilenceDuration time.Duration
}

func LoadConfig() *Config {
	return &Config{
		BackendWSURL: config.GetEnv("BACKEND_WS_URL", "ws://localhost:8080/ws"),
		AgentSecret:  config.GetEnv("AGENT_SECRET", ""),

		LiveKitURL:       config.GetEnv("LIVEKIT_URL", "ws://localhost:7880"),
		LiveKitAPIKey:    config.GetEnv("LIVEKIT_API_KEY", "devkey"),
		LiveKitAPISecret: config.GetEnv("LIVEKIT_API_SECRET", "secret"),

		ASRURL:   config.GetEnv("ASR_URL", "http://localhost:9000/asr"),
		ASRModel: config.GetEnv("ASR_MODEL", "whisper-1"),

		TTSURL:        config.GetEnv("TTS_URL", "http://localhost:8880/v1/audio/speech"),
		TTSVoice:      config.GetEnv("TTS_VOICE", "af_heart"),
		TTSSampleRate: config.GetEnvInt("TTS_SAMPLE_RATE", DefaultTTSSampleRate),

		SampleRate:      config.GetEnvInt("SAMPLE_RATE", DefaultCaptureSampleRate),
		Channels:        config.GetEnvInt("CHANNELS", 1),
		VADThreshold:    config.GetEnvFloat("VAD_THRESHOLD", 0.01),
		SilenceDuration: config.GetEnvDuration("SILENCE_DURATION", 800*time.Millisecond),
	}
}

func main() {
	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	result, err := otel.Init(otel.Config{
		ServiceName:  "alicia-voice",
		Environment:  config.GetEnv("ENVIRONMENT", "development"),
		OTLPEndpoint: config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://alicia-data.hjkl.lol/otlp"),
	})
	if err != nil {
		slog.SetDefault(slog.New(otel.NewPrettyHandler()))
		slog.Warn("otel init failed, using stderr-only logger", "error", err)
	} else {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			result.Shutdown(shutdownCtx)
		}()
		slog.SetDefault(result.Logger)
	}

	slog.Info("starting alicia voice helper")

	cfg := LoadConfig()
	logConfig(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewSessionManager(cfg)

	if err := manager.Start(ctx); err != nil {
		slog.Error("failed to start session manager", "error", err)
		os.Exit(1)
	}

	slog.Info("voice helper is running")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down")
	cancel()
	manager.Stop()
	slog.Info("voice helper stopped")
}

func printHelp() {
	fmt.Println(`Alicia Voice Helper

A service that provides voice transcription and synthesis for the Alicia system.

Environment Variables:
  Backend Connection:
    BACKEND_WS_URL      Backend WebSocket URL (default: ws://localhost:8080/ws)
    AGENT_SECRET        Secret for agent authentication (default: "")

  LiveKit Connection:
    LIVEKIT_URL         LiveKit server URL (default: ws://localhost:7880)
    LIVEKIT_API_KEY     LiveKit API key (default: devkey)
    LIVEKIT_API_SECRET  LiveKit API secret (default: secret)

  ASR (Automatic Speech Recognition):
    ASR_URL             ASR service URL (default: http://localhost:9000/asr)
    ASR_MODEL           ASR model to use (default: whisper-1)

  TTS (Text-to-Speech):
    TTS_URL             TTS service URL (default: http://localhost:8880/v1/audio/speech)
    TTS_VOICE           TTS voice to use (default: af_heart)
    TTS_SAMPLE_RATE     TTS output sample rate (default: 24000)

  Audio Settings:
    SAMPLE_RATE         Audio capture sample rate for LiveKit Opus decoding (default: 48000)
    CHANNELS            Audio channels (default: 1)
    VAD_THRESHOLD       Voice activity detection threshold (default: 0.01)
    SILENCE_DURATION    Silence duration to end utterance (default: 800ms)

Usage:
  voice-helper [flags]

Flags:
  -h, -help  Show this help message`)
}

func logConfig(cfg *Config) {
	slog.Info("configuration",
		"backend_ws_url", cfg.BackendWSURL,
		"agent_secret", maskSecret(cfg.AgentSecret),
		"livekit_url", cfg.LiveKitURL,
		"livekit_api_key", cfg.LiveKitAPIKey,
		"livekit_api_secret", maskSecret(cfg.LiveKitAPISecret),
		"asr_url", cfg.ASRURL,
		"asr_model", cfg.ASRModel,
		"tts_url", cfg.TTSURL,
		"tts_voice", cfg.TTSVoice,
		"tts_sample_rate", cfg.TTSSampleRate,
		"sample_rate", cfg.SampleRate,
		"channels", cfg.Channels,
		"vad_threshold", cfg.VADThreshold,
		"silence_duration", cfg.SilenceDuration,
	)
}

func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
