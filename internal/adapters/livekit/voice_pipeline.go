package livekit

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/longregen/alicia/internal/ports"
)

const (
	// Audio buffer configuration
	DefaultSampleRate = 48000 // LiveKit uses 48kHz Opus
	DefaultChannels   = 2     // Stereo
	MaxBufferDuration = 30 * time.Second
	SilenceThreshold  = 500                     // RMS threshold for silence detection
	SilenceTimeout    = 1500 * time.Millisecond // How long to wait after silence before processing
	MinSpeechDuration = 300 * time.Millisecond  // Minimum speech duration to process

	// Transcription confidence threshold - reject low-confidence transcriptions
	// to avoid hallucinations from background noise
	MinTranscriptionConfidence = 0.5
)

// VoicePipeline manages the voice processing pipeline for a conversation
type VoicePipeline struct {
	asrService ports.ASRService
	ttsService ports.TTSService
	agent      *Agent

	// Audio buffering
	audioBuffer     *AudioBuffer
	audioConverter  *AudioConverter
	lastAudioTime   time.Time
	silenceTimer    *time.Timer
	silenceTimerGen int64 // Generation counter to prevent race between timer callback and cancellation
	processingAudio bool

	// Transcription confidence threshold
	minConfidence float64

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	// Callbacks for pipeline events
	onTranscription func(ctx context.Context, text string, isFinal bool)
	onAudioOutput   func(ctx context.Context, audio []byte, format string)
}

// AudioBuffer stores audio samples for processing
type AudioBuffer struct {
	samples      []byte
	sampleRate   int
	channels     int
	startTime    time.Time
	lastActivity time.Time
	mu           sync.Mutex
}

// NewAudioBuffer creates a new audio buffer
func NewAudioBuffer(sampleRate, channels int) *AudioBuffer {
	return &AudioBuffer{
		samples:      make([]byte, 0, DefaultSampleRate*DefaultChannels*2*int(MaxBufferDuration.Seconds())),
		sampleRate:   sampleRate,
		channels:     channels,
		startTime:    time.Now(),
		lastActivity: time.Now(),
	}
}

// Append adds audio samples to the buffer
func (ab *AudioBuffer) Append(samples []byte) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if len(ab.samples) == 0 {
		ab.startTime = time.Now()
	}

	ab.samples = append(ab.samples, samples...)
	ab.lastActivity = time.Now()

	// Trim buffer if it exceeds max duration
	maxBytes := ab.sampleRate * ab.channels * 2 * int(MaxBufferDuration.Seconds())
	if len(ab.samples) > maxBytes {
		// Keep only the most recent audio
		ab.samples = ab.samples[len(ab.samples)-maxBytes:]
		ab.startTime = time.Now().Add(-MaxBufferDuration)
	}
}

// GetSamples returns a copy of the buffered samples
func (ab *AudioBuffer) GetSamples() []byte {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	result := make([]byte, len(ab.samples))
	copy(result, ab.samples)
	return result
}

// Clear clears the buffer
func (ab *AudioBuffer) Clear() {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.samples = ab.samples[:0]
	ab.startTime = time.Now()
	ab.lastActivity = time.Now()
}

// Duration returns the duration of buffered audio
func (ab *AudioBuffer) Duration() time.Duration {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if len(ab.samples) == 0 {
		return 0
	}

	// Calculate duration from sample count
	// bytes / (sample_rate * channels * bytes_per_sample)
	samples := len(ab.samples) / (ab.channels * 2) // 16-bit = 2 bytes per sample
	return time.Duration(float64(samples)/float64(ab.sampleRate)) * time.Second
}

// IsEmpty returns true if buffer is empty
func (ab *AudioBuffer) IsEmpty() bool {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	return len(ab.samples) == 0
}

// NewVoicePipeline creates a new voice processing pipeline
// The provided context is used as the parent context for the pipeline's lifecycle
// minConfidence sets the minimum ASR confidence threshold (0.0-1.0), use 0 to use default
func NewVoicePipeline(
	ctx context.Context,
	asrService ports.ASRService,
	ttsService ports.TTSService,
	agent *Agent,
	minConfidence float64,
) (*VoicePipeline, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	pipelineCtx, cancel := context.WithCancel(ctx)

	// Create audio converter for Opus decoding/encoding
	audioConverter, err := NewAudioConverter(DefaultSampleRate, DefaultChannels)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create audio converter: %w", err)
	}

	// Use default if not specified
	if minConfidence <= 0 {
		minConfidence = MinTranscriptionConfidence
	}

	return &VoicePipeline{
		asrService:     asrService,
		ttsService:     ttsService,
		agent:          agent,
		audioBuffer:    NewAudioBuffer(DefaultSampleRate, DefaultChannels),
		audioConverter: audioConverter,
		minConfidence:  minConfidence,
		ctx:            pipelineCtx,
		cancel:         cancel,
	}, nil
}

// SetTranscriptionCallback sets the callback for transcription events
func (vp *VoicePipeline) SetTranscriptionCallback(cb func(ctx context.Context, text string, isFinal bool)) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	vp.onTranscription = cb
}

// SetAudioOutputCallback sets the callback for audio output events
func (vp *VoicePipeline) SetAudioOutputCallback(cb func(ctx context.Context, audio []byte, format string)) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	vp.onAudioOutput = cb
}

// ProcessAudioFrame processes an incoming audio frame from the user
// The frame contains Opus-encoded audio data from LiveKit which must be decoded to PCM
func (vp *VoicePipeline) ProcessAudioFrame(ctx context.Context, frame *ports.AudioFrame) error {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	// Validate audio frame parameters
	if frame == nil {
		return fmt.Errorf("audio frame is nil")
	}

	if len(frame.Data) == 0 {
		return fmt.Errorf("audio frame has empty data")
	}

	// Validate sample rate and channels match expected values
	if frame.SampleRate != DefaultSampleRate {
		log.Printf("WARNING: Audio frame sample rate %d does not match expected %d", frame.SampleRate, DefaultSampleRate)
	}

	if frame.Channels != DefaultChannels {
		log.Printf("WARNING: Audio frame channels %d does not match expected %d", frame.Channels, DefaultChannels)
	}

	// Check if pipeline has been stopped
	select {
	case <-vp.ctx.Done():
		return fmt.Errorf("pipeline stopped")
	default:
	}

	// Decode Opus audio to PCM
	// LiveKit sends Opus-encoded audio frames via RTP. We must decode them to PCM
	// for silence detection and ASR processing.
	pcmData, err := vp.audioConverter.ConvertOpusToPCM(frame.Data)
	if err != nil {
		return fmt.Errorf("failed to decode Opus to PCM: %w", err)
	}

	// Add to buffer
	vp.audioBuffer.Append(pcmData)
	vp.lastAudioTime = time.Now()

	// Calculate RMS to detect speech
	rms := calculateRMS(pcmData)
	isSpeech := rms > SilenceThreshold

	if isSpeech {
		// Reset silence timer if we detect speech
		if vp.silenceTimer != nil {
			vp.silenceTimer.Stop()
			vp.silenceTimer = nil
			// Increment generation to invalidate any pending timer callback
			vp.silenceTimerGen++
		}
	} else {
		// Silence detected - start timer if not already running
		if vp.silenceTimer == nil && !vp.audioBuffer.IsEmpty() {
			// Capture the current generation before creating the timer
			// This prevents race where old timer callback could clear a new timer
			gen := vp.silenceTimerGen

			// Start timer to process audio after silence timeout
			// Synchronization strategy: The callback will call handleSilenceTimeout()
			// which atomically checks the generation and clears the timer while holding the lock
			vp.silenceTimer = time.AfterFunc(SilenceTimeout, func() {
				vp.handleSilenceTimeout(gen)
			})
		}
	}

	return nil
}

// handleSilenceTimeout is called by the silence timer callback
// It atomically clears the timer reference and initiates audio processing
// Synchronization strategy: All state checks and updates happen atomically under the lock
// The generation parameter prevents race where old timer clears a new timer
func (vp *VoicePipeline) handleSilenceTimeout(gen int64) {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	// Check if this timer is still current by comparing generations
	// If generations don't match, this timer was cancelled and a new one may have been created
	if vp.silenceTimerGen != gen {
		// Timer was cancelled, don't process
		return
	}

	// Clear timer reference - this timer has fired
	vp.silenceTimer = nil
	// Increment generation to prevent this callback from running again if called multiple times
	vp.silenceTimerGen++

	// Check if context has been cancelled
	select {
	case <-vp.ctx.Done():
		log.Println("Pipeline context cancelled, skipping audio processing")
		return
	default:
	}

	// Check if we should process (same checks as processBufferedAudio but atomic)
	if vp.processingAudio {
		return
	}

	if vp.audioBuffer.IsEmpty() {
		return
	}

	// Check minimum duration
	duration := vp.audioBuffer.Duration()
	if duration < MinSpeechDuration {
		log.Printf("Audio too short (%v), discarding", duration)
		vp.audioBuffer.Clear()
		return
	}

	// Get buffered audio and mark as processing
	// All state modifications happen atomically under the lock
	audioData := vp.audioBuffer.GetSamples()
	vp.audioBuffer.Clear()
	vp.processingAudio = true

	log.Printf("Processing buffered audio: %v duration, %d bytes", duration, len(audioData))

	// Process audio through ASR in background
	// Lock is released before spawning goroutine to avoid holding lock during I/O
	go func() {
		defer func() {
			vp.mu.Lock()
			vp.processingAudio = false
			vp.mu.Unlock()
		}()

		// Convert PCM to WAV format for ASR
		wavData, err := pcmToWav(audioData, DefaultSampleRate, DefaultChannels)
		if err != nil {
			log.Printf("Failed to convert PCM to WAV: %v", err)
			return
		}

		// Create timeout context for ASR transcription (30 seconds should be plenty)
		asrCtx, asrCancel := context.WithTimeout(vp.ctx, 30*time.Second)
		defer asrCancel()

		// Transcribe audio
		result, err := vp.asrService.Transcribe(asrCtx, wavData, "wav")
		if err != nil {
			log.Printf("ASR transcription failed: %v", err)
			return
		}

		if result.Text == "" {
			log.Println("Empty transcription result")
			return
		}

		log.Printf("Transcription: %s (confidence: %.2f)", result.Text, result.Confidence)

		// Reject low-confidence transcriptions to avoid processing hallucinations
		// from background noise
		if result.Confidence < vp.minConfidence {
			log.Printf("Rejecting low-confidence transcription (%.2f < %.2f): %s",
				result.Confidence, vp.minConfidence, result.Text)
			return
		}

		// Call transcription callback
		vp.mu.Lock()
		callback := vp.onTranscription
		vp.mu.Unlock()

		if callback != nil {
			callback(vp.ctx, result.Text, true)
		}
	}()
}

// SynthesizeSpeech converts text to speech and sends it to the agent's audio track
func (vp *VoicePipeline) SynthesizeSpeech(ctx context.Context, text string) error {
	if vp.ttsService == nil {
		return fmt.Errorf("TTS service not configured")
	}

	log.Printf("Synthesizing speech: %s", text)

	// Synthesize speech
	result, err := vp.ttsService.Synthesize(ctx, text, &ports.TTSOptions{
		OutputFormat: "pcm",
	})
	if err != nil {
		return fmt.Errorf("TTS synthesis failed: %w", err)
	}

	if result == nil {
		return fmt.Errorf("TTS synthesis returned nil result")
	}

	log.Printf("Synthesized %d bytes of audio", len(result.Audio))

	// Send audio to agent's audio track
	if err := vp.agent.SendAudio(ctx, result.Audio, result.Format); err != nil {
		return fmt.Errorf("failed to send audio: %w", err)
	}

	// Call audio output callback
	vp.mu.Lock()
	callback := vp.onAudioOutput
	vp.mu.Unlock()

	if callback != nil {
		callback(ctx, result.Audio, result.Format)
	}

	return nil
}

// Stop stops the voice pipeline and ensures all resources are cleaned up
// Synchronization strategy: Cancel context first to prevent new timers, then clean up existing timer
func (vp *VoicePipeline) Stop() {
	// Cancel the context first to prevent new operations from starting
	// This ensures ProcessAudioFrame will return early if called after this point
	vp.cancel()

	// Now safely clean up the silence timer
	vp.mu.Lock()
	defer vp.mu.Unlock()

	// Stop and clear the silence timer
	// The timer callback checks both ctx.Done() and generation, so it will exit early if cancelled
	if vp.silenceTimer != nil {
		// Stop returns false if timer already fired, which is okay
		// The callback will check ctx.Done() and generation and exit early
		vp.silenceTimer.Stop()
		vp.silenceTimer = nil
		// Increment generation to invalidate any pending callback
		vp.silenceTimerGen++
	}
}

// calculateRMS calculates the root mean square of audio samples
func calculateRMS(samples []byte) float64 {
	if len(samples) == 0 {
		return 0
	}

	var sum float64
	// Assume 16-bit PCM samples
	for i := 0; i < len(samples)-1; i += 2 {
		sample := int16(binary.LittleEndian.Uint16(samples[i : i+2]))
		sum += float64(sample) * float64(sample)
	}

	count := len(samples) / 2
	if count == 0 {
		return 0
	}

	mean := sum / float64(count)
	return mean // Return mean square (not taking square root for threshold comparison)
}

// pcmToWav converts raw PCM data to WAV format
func pcmToWav(pcm []byte, sampleRate, channels int) ([]byte, error) {
	buf := new(bytes.Buffer)

	// WAV header
	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36+len(pcm))) // File size - 8
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))                    // fmt chunk size
	binary.Write(buf, binary.LittleEndian, uint16(1))                     // Audio format (1 = PCM)
	binary.Write(buf, binary.LittleEndian, uint16(channels))              // Number of channels
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))            // Sample rate
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate*channels*2)) // Byte rate
	binary.Write(buf, binary.LittleEndian, uint16(channels*2))            // Block align
	binary.Write(buf, binary.LittleEndian, uint16(16))                    // Bits per sample

	// data chunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(len(pcm))) // Data size
	buf.Write(pcm)

	return buf.Bytes(), nil
}
