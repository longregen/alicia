package livekit

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streamer45/silero-vad-go/speech"
)

const (
	VADSampleRate           = 16000
	VADThreshold            = 0.5
	VADMinSilenceDurationMs = 1200
	VADSpeechPadMs          = 100

	LiveKitSampleRate = 48000
	LiveKitChannels   = 2
)

type TurnState int

const (
	TurnStateIdle     TurnState = iota
	TurnStateSpeaking
	TurnStateEnding
)

type VADProcessor struct {
	detector *speech.Detector

	state           TurnState
	speechStartTime time.Time
	silenceStart    time.Time
	sampleCount     int64

	onTurnStart func()
	onTurnEnd   func(durationMs int64)

	resampleBuffer []float32

	mu sync.Mutex
}

type VADConfig struct {
	ModelPath            string
	MinSilenceDurationMs int
	Threshold            float32
	OnTurnStart          func()
	OnTurnEnd            func(durationMs int64)
}

func NewVADProcessor(cfg VADConfig) (*VADProcessor, error) {
	if cfg.ModelPath == "" {
		return nil, fmt.Errorf("VAD model path is required")
	}

	if cfg.MinSilenceDurationMs <= 0 {
		cfg.MinSilenceDurationMs = VADMinSilenceDurationMs
	}

	if cfg.Threshold <= 0 {
		cfg.Threshold = VADThreshold
	}

	detector, err := speech.NewDetector(speech.DetectorConfig{
		ModelPath:            cfg.ModelPath,
		SampleRate:           VADSampleRate,
		Threshold:            cfg.Threshold,
		MinSilenceDurationMs: cfg.MinSilenceDurationMs,
		SpeechPadMs:          VADSpeechPadMs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create VAD detector: %w", err)
	}

	return &VADProcessor{
		detector:       detector,
		state:          TurnStateIdle,
		onTurnStart:    cfg.OnTurnStart,
		onTurnEnd:      cfg.OnTurnEnd,
		resampleBuffer: make([]float32, 0, 4096),
	}, nil
}

func (v *VADProcessor) ProcessAudio(samples []float32) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(samples) == 0 {
		return nil
	}

	monoSamples := v.convertToMono(samples)
	resampledSamples := v.resample48kTo16k(monoSamples)

	segments, err := v.detector.Detect(resampledSamples)
	if err != nil {
		return fmt.Errorf("VAD detection failed: %w", err)
	}

	v.sampleCount += int64(len(resampledSamples))

	v.processSegments(segments)

	return nil
}

func (v *VADProcessor) processSegments(segments []speech.Segment) {
	now := time.Now()

	if len(segments) > 0 {
		for _, seg := range segments {
			if seg.SpeechStartAt >= 0 && v.state == TurnStateIdle {
				v.state = TurnStateSpeaking
				v.speechStartTime = now
				v.silenceStart = time.Time{}
				log.Printf("VAD: Turn started at %.2fs", seg.SpeechStartAt)
				if v.onTurnStart != nil {
					go v.onTurnStart()
				}
			}

			if seg.SpeechEndAt > 0 {
				v.state = TurnStateEnding
				v.silenceStart = now
				log.Printf("VAD: Speech segment ended at %.2fs, waiting for silence threshold", seg.SpeechEndAt)
			}
		}
	}

	if v.state == TurnStateEnding && !v.silenceStart.IsZero() {
		silenceDuration := now.Sub(v.silenceStart)
		if silenceDuration >= time.Duration(VADMinSilenceDurationMs)*time.Millisecond {
			turnDuration := now.Sub(v.speechStartTime)
			log.Printf("VAD: Turn ended after %.2fs silence, turn duration: %.2fs",
				silenceDuration.Seconds(), turnDuration.Seconds())

			v.state = TurnStateIdle
			v.silenceStart = time.Time{}

			if v.onTurnEnd != nil {
				go v.onTurnEnd(turnDuration.Milliseconds())
			}
		}
	}
}

func (v *VADProcessor) convertToMono(stereoSamples []float32) []float32 {
	monoLen := len(stereoSamples) / LiveKitChannels
	mono := make([]float32, monoLen)

	for i := 0; i < monoLen; i++ {
		left := stereoSamples[i*2]
		right := stereoSamples[i*2+1]
		mono[i] = (left + right) / 2.0
	}

	return mono
}

// Simple decimation with averaging - for production, consider using a proper resampling library
func (v *VADProcessor) resample48kTo16k(samples []float32) []float32 {
	ratio := LiveKitSampleRate / VADSampleRate
	outputLen := len(samples) / ratio

	resampled := make([]float32, outputLen)
	for i := 0; i < outputLen; i++ {
		sum := float32(0)
		for j := 0; j < ratio; j++ {
			idx := i*ratio + j
			if idx < len(samples) {
				sum += samples[idx]
			}
		}
		resampled[i] = sum / float32(ratio)
	}

	return resampled
}

func (v *VADProcessor) GetState() TurnState {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.state
}

func (v *VADProcessor) Reset() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if err := v.detector.Reset(); err != nil {
		return fmt.Errorf("failed to reset VAD detector: %w", err)
	}

	v.state = TurnStateIdle
	v.speechStartTime = time.Time{}
	v.silenceStart = time.Time{}
	v.sampleCount = 0

	return nil
}

func (v *VADProcessor) Destroy() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.detector != nil {
		if err := v.detector.Destroy(); err != nil {
			return fmt.Errorf("failed to destroy VAD detector: %w", err)
		}
		v.detector = nil
	}

	return nil
}

func (v *VADProcessor) IsSpeaking() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.state == TurnStateSpeaking || v.state == TurnStateEnding
}
