package speech

import (
	"context"
	"fmt"
	"time"

	"github.com/longregen/alicia/internal/adapters/circuitbreaker"
	"github.com/longregen/alicia/internal/ports"
)

const (
	defaultTTSEndpoint = "http://localhost:8000"
	speechPath         = "/audio/speech"
	TTSTimeout         = 30 * time.Second
)

type TTSAdapter struct {
	client       *Client
	model        string
	defaultVoice string
	breaker      *circuitbreaker.CircuitBreaker
}

func NewTTSAdapter(endpoint string) *TTSAdapter {
	if endpoint == "" {
		endpoint = defaultTTSEndpoint
	}

	return &TTSAdapter{
		client:       NewClient(endpoint),
		model:        "kokoro",
		defaultVoice: "af_sarah",
		breaker:      circuitbreaker.New(5, 30*time.Second),
	}
}

func NewTTSAdapterWithModel(endpoint, model, defaultVoice string) *TTSAdapter {
	adapter := NewTTSAdapter(endpoint)
	adapter.model = model
	adapter.defaultVoice = defaultVoice
	return adapter
}

type ttsRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float32 `json:"speed,omitempty"`
}

func (t *TTSAdapter) Synthesize(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error) {
	var result *ports.TTSResult
	err := t.breaker.Execute(func() error {
		var err error
		result, err = t.doSynthesize(ctx, text, options)
		return err
	})
	return result, err
}

func (t *TTSAdapter) doSynthesize(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error) {
	if text == "" {
		return nil, fmt.Errorf("text is empty")
	}

	// Add timeout to prevent hanging on slow/failed TTS requests
	ctx, cancel := context.WithTimeout(ctx, TTSTimeout)
	defer cancel()

	req := ttsRequest{
		Model: t.model,
		Input: text,
		Voice: t.defaultVoice,
	}

	if options != nil {
		if options.Voice != "" {
			req.Voice = options.Voice
		}
		if options.Speed > 0 {
			req.Speed = options.Speed
		}
		if options.OutputFormat != "" {
			req.ResponseFormat = options.OutputFormat
		}
	}

	if req.ResponseFormat == "" {
		req.ResponseFormat = "pcm"
	}

	audioData, err := t.client.PostJSONRaw(ctx, speechPath, req)
	if err != nil {
		return nil, fmt.Errorf("TTS synthesis failed: %w", err)
	}

	durationMs := estimateAudioDuration(audioData, req.ResponseFormat)

	result := &ports.TTSResult{
		Audio:      audioData,
		Format:     req.ResponseFormat,
		DurationMs: int(durationMs),
	}

	return result, nil
}

func estimateAudioDuration(data []byte, format string) int64 {
	if len(data) == 0 {
		return 0
	}

	switch format {
	case "pcm", "pcm_s16le":
		// PCM 16-bit mono at 24000 Hz: 2 bytes per sample
		// duration = samples / sampleRate * 1000
		// samples = bytes / 2
		// duration = bytes / 2 / 24000 * 1000 = bytes * 1000 / 48000
		return int64(len(data)) * 1000 / 48000
	case "opus":
		// Opus typically uses 20ms frames at ~32kbps for speech
		// Rough estimate: ~4KB per second of audio
		// duration = bytes / 4000 * 1000 = bytes / 4
		return int64(len(data)) / 4
	case "mp3":
		// MP3 at 128kbps: ~16KB per second
		// duration = bytes / 16000 * 1000 = bytes / 16
		return int64(len(data)) / 16
	default:
		// Unknown format, assume PCM-like
		return int64(len(data)) * 1000 / 48000
	}
}

func (t *TTSAdapter) SynthesizeStream(ctx context.Context, text string, options *ports.TTSOptions) (<-chan *ports.TTSResult, error) {
	resultChan := make(chan *ports.TTSResult)

	// Add timeout to prevent hanging on slow/failed TTS requests
	ctx, cancel := context.WithTimeout(ctx, TTSTimeout)

	go func() {
		defer close(resultChan)
		defer cancel()

		result, err := t.Synthesize(ctx, text, options)
		if err != nil {
			resultChan <- &ports.TTSResult{
				Audio:  []byte(fmt.Sprintf("TTS error: %v", err)),
				Format: "error",
			}
			return
		}

		select {
		case resultChan <- result:
		case <-ctx.Done():
			return
		}
	}()

	return resultChan, nil
}

func (t *TTSAdapter) SetModel(model string) {
	t.model = model
}

func (t *TTSAdapter) GetModel() string {
	return t.model
}

func (t *TTSAdapter) SetDefaultVoice(voice string) {
	t.defaultVoice = voice
}

func (t *TTSAdapter) GetDefaultVoice() string {
	return t.defaultVoice
}
