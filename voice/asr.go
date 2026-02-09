package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/httpclient"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ASRClient struct {
	cfg    *Config
	client *http.Client
}

type ASRResponse struct {
	Text string `json:"text"`
}

func NewASRClient(cfg *Config) *ASRClient {
	return &ASRClient{
		cfg:    cfg,
		client: httpclient.New(),
	}
}

func (c *ASRClient) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return c.TranscribeWithOptions(ctx, audio, "", "")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (c *ASRClient) pcmToWav(pcm []byte) []byte {
	sampleRate := uint32(c.cfg.SampleRate)
	channels := uint16(c.cfg.Channels)
	bitsPerSample := uint16(16)
	byteRate := sampleRate * uint32(channels) * uint32(bitsPerSample) / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := uint32(len(pcm))
	fileSize := 36 + dataSize

	header := make([]byte, 44)

	// RIFF header
	copy(header[0:4], "RIFF")
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)
	copy(header[8:12], "WAVE")

	// fmt chunk
	copy(header[12:16], "fmt ")
	header[16] = 16 // chunk size
	header[17] = 0
	header[18] = 0
	header[19] = 0
	header[20] = 1 // PCM format
	header[21] = 0
	header[22] = byte(channels)
	header[23] = byte(channels >> 8)
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)
	header[32] = byte(blockAlign)
	header[33] = byte(blockAlign >> 8)
	header[34] = byte(bitsPerSample)
	header[35] = byte(bitsPerSample >> 8)

	// data chunk
	copy(header[36:40], "data")
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	wav := make([]byte, 44+len(pcm))
	copy(wav, header)
	copy(wav[44:], pcm)

	return wav
}

func (c *ASRClient) TranscribeWithOptions(ctx context.Context, audio []byte, language string, prompt string) (string, error) {
	if len(audio) == 0 {
		slog.Info("asr: empty audio, skipping transcription")
		return "", nil
	}

	bytesPerMs := c.cfg.SampleRate * c.cfg.Channels * 2 / 1000
	if bytesPerMs == 0 {
		bytesPerMs = 1
	}
	audioDurationMs := len(audio) / bytesPerMs

	ctx, span := otel.Tracer("alicia-voice").Start(ctx, "asr.transcribe",
		trace.WithAttributes(
			attribute.Int("audio.bytes", len(audio)),
			attribute.Int("audio.duration_ms", audioDurationMs),
			attribute.String("asr.model", c.cfg.ASRModel),
			attribute.String("asr.url", c.cfg.ASRURL),
			attribute.String("asr.language", language),
			attribute.Int("asr.sample_rate", c.cfg.SampleRate),
			attribute.Int("asr.channels", c.cfg.Channels),
		))
	defer span.End()

	slog.Debug("asr: sending audio for transcription", "bytes", len(audio), "duration_ms", audioDurationMs, "model", c.cfg.ASRModel)

	startTime := time.Now()

	wav := c.pcmToWav(audio)
	span.SetAttributes(attribute.Int("audio.wav_bytes", len(wav)))

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create form file failed")
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(wav); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "write audio failed")
		return "", fmt.Errorf("write audio: %w", err)
	}

	if err := writer.WriteField("model", c.cfg.ASRModel); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "write model field failed")
		return "", fmt.Errorf("write model field: %w", err)
	}
	if language != "" {
		if err := writer.WriteField("language", language); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "write language field failed")
			return "", fmt.Errorf("write language field: %w", err)
		}
	}
	if prompt != "" {
		if err := writer.WriteField("prompt", prompt); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "write prompt field failed")
			return "", fmt.Errorf("write prompt field: %w", err)
		}
	}
	if err := writer.WriteField("response_format", "json"); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "write format field failed")
		return "", fmt.Errorf("write format field: %w", err)
	}
	if err := writer.Close(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "close writer failed")
		return "", fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.ASRURL, &buf)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create request failed")
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		slog.Error("asr: request failed", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "send request failed")
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("asr: error response", "status", resp.StatusCode, "body", string(body))
		err := fmt.Errorf("ASR error (status %d): %s", resp.StatusCode, string(body))
		span.RecordError(err)
		span.SetStatus(codes.Error, "ASR service error")
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "read response failed")
		return "", fmt.Errorf("read response: %w", err)
	}

	elapsed := time.Since(startTime)
	span.SetAttributes(attribute.Int64("asr.latency_ms", elapsed.Milliseconds()))
	if audioDurationMs > 0 {
		span.SetAttributes(attribute.Float64("asr.realtime_factor", float64(elapsed.Milliseconds())/float64(audioDurationMs)))
	}

	var asrResp ASRResponse
	if err := json.Unmarshal(body, &asrResp); err != nil {
		slog.Error("asr: failed to parse response", "error", err, "body", string(body))
		span.RecordError(err)
		span.SetStatus(codes.Error, "parse response failed")
		return "", fmt.Errorf("parse response: %w", err)
	}

	slog.Info("asr: transcription received", "latency", elapsed, "chars", len(asrResp.Text), "preview", truncateString(asrResp.Text, 50))
	span.SetAttributes(
		attribute.Int("transcript.length", len(asrResp.Text)),
		attribute.String("transcript.preview", truncateString(asrResp.Text, 100)),
	)
	span.SetStatus(codes.Ok, "transcription successful")
	return asrResp.Text, nil
}
