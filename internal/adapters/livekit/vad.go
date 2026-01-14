package livekit

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streamer45/silero-vad-go/speech"
)

const (
	// VAD configuration
	VADSampleRate           = 16000 // Silero VAD requires 16kHz
	VADThreshold            = 0.5   // Speech detection threshold
	VADMinSilenceDurationMs = 1200  // 1.2 seconds of silence to mark turn end
	VADSpeechPadMs          = 100   // Padding to avoid cutting speech

	// Audio conversion
	LiveKitSampleRate = 48000 // LiveKit uses 48kHz
	LiveKitChannels   = 2     // Stereo
)

// TurnState represents the current state of turn detection
type TurnState int

const (
	TurnStateIdle     TurnState = iota // No speech detected
	TurnStateSpeaking                  // User is speaking
	TurnStateEnding                    // Silence detected, waiting for turn end
)

// VADProcessor handles voice activity detection using Silero VAD
type VADProcessor struct {
	detector *speech.Detector

	// State tracking
	state           TurnState
	speechStartTime time.Time
	silenceStart    time.Time
	sampleCount     int64 // Total samples processed for timing

	// Callbacks
	onTurnStart func()
	onTurnEnd   func(durationMs int64)

	// Resampling buffer
	resampleBuffer []float32

	mu sync.Mutex
}

// VADConfig contains configuration for the VAD processor
type VADConfig struct {
	// ModelPath is the path to the Silero VAD ONNX model
	ModelPath string
	// MinSilenceDurationMs is the silence duration to mark end of turn (default: 1200ms)
	MinSilenceDurationMs int
	// Threshold is the speech detection threshold (default: 0.5)
	Threshold float32
	// OnTurnStart is called when user starts speaking
	OnTurnStart func()
	// OnTurnEnd is called when turn ends (after silence threshold)
	OnTurnEnd func(durationMs int64)
}

// NewVADProcessor creates a new VAD processor with Silero VAD
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

	// Create Silero VAD detector
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

// ProcessAudio processes audio frames and detects speech/silence
// Input: 48kHz stereo float32 samples from LiveKit
func (v *VADProcessor) ProcessAudio(samples []float32) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(samples) == 0 {
		return nil
	}

	// Convert 48kHz stereo to 16kHz mono
	monoSamples := v.convertToMono(samples)
	resampledSamples := v.resample48kTo16k(monoSamples)

	// Process through VAD
	segments, err := v.detector.Detect(resampledSamples)
	if err != nil {
		return fmt.Errorf("VAD detection failed: %w", err)
	}

	// Update sample count for timing
	v.sampleCount += int64(len(resampledSamples))

	// Process detected segments
	v.processSegments(segments)

	return nil
}

// processSegments handles the VAD output and manages turn state
func (v *VADProcessor) processSegments(segments []speech.Segment) {
	now := time.Now()

	if len(segments) > 0 {
		// Speech detected
		for _, seg := range segments {
			if seg.SpeechStartAt >= 0 && v.state == TurnStateIdle {
				// Turn started
				v.state = TurnStateSpeaking
				v.speechStartTime = now
				v.silenceStart = time.Time{}
				log.Printf("VAD: Turn started at %.2fs", seg.SpeechStartAt)
				if v.onTurnStart != nil {
					go v.onTurnStart()
				}
			}

			if seg.SpeechEndAt > 0 {
				// Speech segment ended, start silence timer
				v.state = TurnStateEnding
				v.silenceStart = now
				log.Printf("VAD: Speech segment ended at %.2fs, waiting for silence threshold", seg.SpeechEndAt)
			}
		}
	}

	// Check if silence duration exceeds threshold
	if v.state == TurnStateEnding && !v.silenceStart.IsZero() {
		silenceDuration := now.Sub(v.silenceStart)
		if silenceDuration >= time.Duration(VADMinSilenceDurationMs)*time.Millisecond {
			// Turn ended
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

// convertToMono converts stereo audio to mono by averaging channels
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

// resample48kTo16k downsamples from 48kHz to 16kHz (3:1 ratio)
// Uses simple decimation - for production, consider using a proper resampling library
func (v *VADProcessor) resample48kTo16k(samples []float32) []float32 {
	// 48kHz / 16kHz = 3, so we take every 3rd sample
	ratio := LiveKitSampleRate / VADSampleRate
	outputLen := len(samples) / ratio

	resampled := make([]float32, outputLen)
	for i := 0; i < outputLen; i++ {
		// Simple decimation with averaging for anti-aliasing
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

// GetState returns the current turn state
func (v *VADProcessor) GetState() TurnState {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.state
}

// Reset resets the VAD processor state
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

// Destroy cleans up VAD resources
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

// IsSpeaking returns true if the user is currently speaking
func (v *VADProcessor) IsSpeaking() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.state == TurnStateSpeaking || v.state == TurnStateEnding
}
