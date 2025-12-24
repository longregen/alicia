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
	speechPath         = "/v1/audio/speech"
	voicesPath         = "/v1/audio/voices"
	// TTSTimeout is the maximum time to wait for TTS synthesis
	TTSTimeout = 30 * time.Second
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
		breaker:      circuitbreaker.New(5, 30*time.Second), // 5 failures, 30s timeout
	}
}

// NewTTSAdapterWithModel creates a new TTS adapter with a specific model and default voice
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

type voicesResponse struct {
	Voices []voiceInfo `json:"voices"`
}

type voiceInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language,omitempty"`
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

	result := &ports.TTSResult{
		Audio:      audioData,
		Format:     req.ResponseFormat,
		DurationMs: 0,
	}

	return result, nil
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

// kokoroVoices contains all 54 built-in Kokoro TTS voices
var kokoroVoices = []string{
	// American English (20 voices)
	"af_heart", "af_alloy", "af_aoede", "af_bella", "af_jessica", "af_kore", "af_nicole", "af_nova", "af_river", "af_sarah", "af_sky",
	"am_adam", "am_echo", "am_eric", "am_fenrir", "am_liam", "am_michael", "am_onyx", "am_puck", "am_santa",
	// British English (8 voices)
	"bf_alice", "bf_emma", "bf_isabella", "bf_lily",
	"bm_daniel", "bm_fable", "bm_george", "bm_lewis",
	// Japanese (5 voices)
	"jf_alpha", "jf_gongitsune", "jf_nezumi", "jf_tebukuro", "jm_kumo",
	// Mandarin Chinese (8 voices)
	"zf_xiaobei", "zf_xiaoni", "zf_xiaoxiao", "zf_xiaoyi",
	"zm_yunjian", "zm_yunxi", "zm_yunxia", "zm_yunyang",
	// Spanish (3 voices)
	"ef_dora", "em_alex", "em_santa",
	// French (1 voice)
	"ff_siwis",
	// Hindi (4 voices)
	"hf_alpha", "hf_beta", "hm_omega", "hm_psi",
	// Italian (2 voices)
	"if_sara", "im_nicola",
	// Brazilian Portuguese (3 voices)
	"pf_dora", "pm_alex", "pm_santa",
}

func (t *TTSAdapter) GetVoices(ctx context.Context) ([]string, error) {
	var response voicesResponse
	err := t.client.Get(ctx, voicesPath, &response)
	if err != nil {
		// Return all built-in Kokoro voices as fallback
		return kokoroVoices, nil
	}

	voices := make([]string, len(response.Voices))
	for i, voice := range response.Voices {
		voices[i] = voice.ID
	}

	return voices, nil
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
