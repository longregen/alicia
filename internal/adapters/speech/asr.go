package speech

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/longregen/alicia/internal/adapters/circuitbreaker"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

const (
	defaultASREndpoint = "http://localhost:8000"
	transcriptionsPath = "/v1/audio/transcriptions"
	ASRTimeout         = 30 * time.Second
)

type ASRAdapter struct {
	client  *Client
	model   string
	breaker *circuitbreaker.CircuitBreaker
}

func NewASRAdapter(endpoint string) *ASRAdapter {
	if endpoint == "" {
		endpoint = defaultASREndpoint
	}

	return &ASRAdapter{
		client:  NewClient(endpoint),
		model:   "whisper-1",
		breaker: circuitbreaker.New(5, 30*time.Second),
	}
}

func NewASRAdapterWithModel(endpoint, model string) *ASRAdapter {
	adapter := NewASRAdapter(endpoint)
	adapter.model = model
	return adapter
}

type whisperResponse struct {
	Text     string           `json:"text"`
	Language string           `json:"language,omitempty"`
	Duration float32          `json:"duration,omitempty"`
	Segments []whisperSegment `json:"segments,omitempty"`
}

type whisperSegment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float32 `json:"start"`
	End              float32 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens,omitempty"`
	Temperature      float32 `json:"temperature,omitempty"`
	AvgLogprob       float32 `json:"avg_logprob,omitempty"`
	CompressionRatio float32 `json:"compression_ratio,omitempty"`
	NoSpeechProb     float32 `json:"no_speech_prob,omitempty"`
}

func (a *ASRAdapter) Transcribe(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
	var result *ports.ASRResult
	err := a.breaker.Execute(func() error {
		var err error
		result, err = a.doTranscribe(ctx, audio, format)
		return err
	})
	return result, err
}

func (a *ASRAdapter) doTranscribe(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
	if len(audio) == 0 {
		return nil, fmt.Errorf("audio data is empty")
	}

	if format == "" {
		format = "wav"
	}

	// Add timeout to prevent hanging on slow/failed ASR requests
	ctx, cancel := context.WithTimeout(ctx, ASRTimeout)
	defer cancel()

	fields := map[string]string{
		"model":           a.model,
		"response_format": "verbose_json",
	}

	fileName := fmt.Sprintf("audio.%s", format)

	var response whisperResponse
	if err := a.client.PostMultipart(ctx, transcriptionsPath, fields, "file", fileName, audio, &response); err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	segments := make([]models.Segment, len(response.Segments))
	for i, seg := range response.Segments {
		segments[i] = models.Segment{
			ID:         seg.ID,
			Start:      seg.Start,
			End:        seg.End,
			Text:       seg.Text,
			Confidence: 1.0 - seg.NoSpeechProb,
		}
	}

	result := &ports.ASRResult{
		Text:     response.Text,
		Language: response.Language,
		Duration: response.Duration,
		Segments: segments,
	}

	if len(segments) > 0 {
		var totalConfidence float32
		for _, seg := range segments {
			totalConfidence += seg.Confidence
		}
		result.Confidence = totalConfidence / float32(len(segments))
	}

	return result, nil
}

func (a *ASRAdapter) TranscribeStream(ctx context.Context, audioStream io.Reader, format string) (<-chan *ports.ASRResult, error) {
	resultChan := make(chan *ports.ASRResult)

	// Add timeout to prevent hanging on slow/failed ASR requests
	ctx, cancel := context.WithTimeout(ctx, ASRTimeout)

	go func() {
		defer close(resultChan)
		defer cancel()

		audioData, err := io.ReadAll(audioStream)
		if err != nil {
			log.Printf("ASR: error reading audio stream: %v", err)
			return
		}

		result, err := a.Transcribe(ctx, audioData, format)
		if err != nil {
			log.Printf("ASR: transcription error: %v", err)
			return
		}

		select {
		case resultChan <- result:
		case <-ctx.Done():
			log.Printf("ASR: context cancelled while sending result")
			return
		}
	}()

	return resultChan, nil
}

func (a *ASRAdapter) SetModel(model string) {
	a.model = model
}

func (a *ASRAdapter) GetModel() string {
	return a.model
}
