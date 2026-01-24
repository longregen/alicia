package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/httpclient"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type TTSClient struct {
	cfg    *Config
	client *http.Client
}

type TTSRequest struct {
	Model          string  `json:"model,omitempty"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
}

func NewTTSClient(cfg *Config) *TTSClient {
	return &TTSClient{
		cfg:    cfg,
		client: httpclient.NewLong(),
	}
}

func (c *TTSClient) doTTSRequest(ctx context.Context, text, format string, speed float64) (*http.Response, error) {
	if speed <= 0 {
		speed = 1.0
	}

	reqBody := TTSRequest{
		Model:          "kokoro",
		Input:          text,
		Voice:          c.cfg.TTSVoice,
		ResponseFormat: format,
		Speed:          speed,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.TTSURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("TTS error (status %d): %s", resp.StatusCode, string(errBody))
	}

	return resp, nil
}

func (c *TTSClient) Synthesize(ctx context.Context, text string, speed float64) ([]byte, error) {
	if text == "" {
		return nil, nil
	}

	ctx, span := otel.Tracer("alicia-voice").Start(ctx, "tts.synthesize",
		trace.WithAttributes(
			attribute.Int("text.length", len(text)),
			attribute.String("text.preview", truncateString(text, 100)),
			attribute.String("tts.model", "kokoro"),
			attribute.String("tts.voice", c.cfg.TTSVoice),
			attribute.String("tts.url", c.cfg.TTSURL),
			attribute.Int("tts.sample_rate", c.cfg.TTSSampleRate),
			attribute.Float64("tts.speed", speed),
		))
	defer span.End()

	startTime := time.Now()

	resp, err := c.doTTSRequest(ctx, text, "pcm", speed)
	if err != nil {
		slog.Error("tts: request failed", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "TTS request failed")
		return nil, err
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "read response failed")
		return nil, fmt.Errorf("read response: %w", err)
	}

	elapsed := time.Since(startTime)
	bytesPerMs := c.cfg.TTSSampleRate * 2 / 1000 // 16-bit mono
	if bytesPerMs == 0 {
		bytesPerMs = 1
	}
	audioDurationMs := len(audio) / bytesPerMs
	slog.Info("tts: synthesis complete", "audio_bytes", len(audio), "audio_duration_ms", audioDurationMs, "latency", elapsed, "preview", truncateString(text, 50))

	attrs := []attribute.KeyValue{
		attribute.Int("audio.bytes", len(audio)),
		attribute.Int("audio.duration_ms", audioDurationMs),
		attribute.Int64("tts.latency_ms", elapsed.Milliseconds()),
	}
	if audioDurationMs > 0 {
		attrs = append(attrs, attribute.Float64("tts.realtime_factor", float64(elapsed.Milliseconds())/float64(audioDurationMs)))
	}
	span.SetAttributes(attrs...)
	span.SetStatus(codes.Ok, "synthesis successful")

	return audio, nil
}

