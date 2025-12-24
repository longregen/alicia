package livekit

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pion/webrtc/v4/pkg/media"
	"gopkg.in/hraban/opus.v2"
)

const (
	// MaxOpusFrameSize is the maximum size of an Opus frame in bytes.
	// Opus frames can be up to 1275 bytes for a single frame, but we use 4000
	// as a safe buffer size to accommodate any encoding scenario.
	MaxOpusFrameSize = 4000

	// OpusFrameDuration is the duration of an Opus frame in nanoseconds (20ms).
	// Opus typically uses 20ms frames for voice encoding.
	OpusFrameDuration = 20_000_000

	// BytesPerPCMSample is the number of bytes per PCM16 sample.
	// PCM16 uses 16-bit (2-byte) signed integers for each sample.
	BytesPerPCMSample = 2

	// OpusFramesPerSecond is the number of Opus frames per second.
	// With 20ms frames, there are 50 frames per second (1000ms / 20ms = 50).
	OpusFramesPerSecond = 50

	// StreamBufferSize is the buffer size for streaming channels.
	// This allows buffering of up to 10 audio samples before blocking.
	StreamBufferSize = 10

	// OpusEncoderComplexity is the computational complexity for Opus encoding (0-10).
	// Higher values produce better quality but use more CPU. 10 is maximum quality.
	OpusEncoderComplexity = 10
)

// AudioConverter handles conversion between different audio formats
type AudioConverter struct {
	// Opus encoder for PCM -> Opus conversion
	opusEncoder *opus.Encoder
	// Opus decoder for Opus -> PCM conversion
	opusDecoder *opus.Decoder
	sampleRate  int
	channels    int
	frameSize   int // Opus frame size in samples
}

// NewAudioConverter creates a new audio converter
func NewAudioConverter(sampleRate, channels int) (*AudioConverter, error) {
	// Create Opus encoder
	encoder, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus encoder: %w", err)
	}

	// Set encoder parameters for voice
	encoder.SetBitrateToMax()
	encoder.SetComplexity(OpusEncoderComplexity)

	// Create Opus decoder
	decoder, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus decoder: %w", err)
	}

	// Calculate frame size (20ms at the given sample rate)
	frameSize := sampleRate / OpusFramesPerSecond // 20ms frames

	return &AudioConverter{
		opusEncoder: encoder,
		opusDecoder: decoder,
		sampleRate:  sampleRate,
		channels:    channels,
		frameSize:   frameSize,
	}, nil
}

// ConvertPCMToOpus converts PCM16 audio data to Opus format
// PCM data should be 16-bit signed little-endian samples
func (ac *AudioConverter) ConvertPCMToOpus(pcmData []byte) ([]media.Sample, error) {
	if len(pcmData) == 0 {
		return nil, fmt.Errorf("empty PCM data")
	}

	// PCM16 is 2 bytes per sample
	bytesPerSample := BytesPerPCMSample
	samplesCount := len(pcmData) / bytesPerSample

	// Convert bytes to int16 samples
	pcmSamples := make([]int16, samplesCount)
	reader := bytes.NewReader(pcmData)
	if err := binary.Read(reader, binary.LittleEndian, &pcmSamples); err != nil {
		return nil, fmt.Errorf("failed to read PCM samples: %w", err)
	}

	// Split into frames and encode each
	var samples []media.Sample
	frameSize := ac.frameSize * ac.channels // Total samples per frame (all channels)

	for i := 0; i < len(pcmSamples); i += frameSize {
		end := i + frameSize
		if end > len(pcmSamples) {
			// Pad the last frame if needed
			end = len(pcmSamples)
			paddedFrame := make([]int16, frameSize)
			copy(paddedFrame, pcmSamples[i:end])
			pcmSamples = append(pcmSamples[:i], paddedFrame...)
		}

		frame := pcmSamples[i:end]

		// Encode frame to Opus
		opusData := make([]byte, MaxOpusFrameSize)
		n, err := ac.opusEncoder.Encode(frame, opusData)
		if err != nil {
			return nil, fmt.Errorf("failed to encode opus frame: %w", err)
		}

		// Create media sample
		sample := media.Sample{
			Data:     opusData[:n],
			Duration: OpusFrameDuration,
		}
		samples = append(samples, sample)
	}

	return samples, nil
}

// ConvertPCMStream converts a stream of PCM data to Opus samples
// This is useful for streaming scenarios
// The context parameter allows cancellation to prevent goroutine leaks if the caller stops reading
func (ac *AudioConverter) ConvertPCMStream(ctx context.Context, pcmReader io.Reader) (<-chan media.Sample, <-chan error) {
	sampleChan := make(chan media.Sample, StreamBufferSize)
	errorChan := make(chan error, 1)

	go func() {
		// Ensure channels are always closed, even on panic
		defer func() {
			if r := recover(); r != nil {
				// Send panic as error if possible
				select {
				case errorChan <- fmt.Errorf("panic in ConvertPCMStream: %v", r):
				default:
				}
			}
			// Close channels in reverse order of creation to avoid races
			close(errorChan)
			close(sampleChan)
		}()

		// Read PCM data in chunks
		bytesPerSample := BytesPerPCMSample
		frameBytes := ac.frameSize * ac.channels * bytesPerSample
		buffer := make([]byte, frameBytes)
		pcmSamples := make([]int16, ac.frameSize*ac.channels)

		for {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				// Context cancelled, exit gracefully
				return
			default:
			}

			// Read a full frame
			n, err := io.ReadFull(pcmReader, buffer)
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// End of stream
					if n > 0 {
						// Process remaining data
						reader := bytes.NewReader(buffer[:n])
						if err := binary.Read(reader, binary.LittleEndian, pcmSamples[:n/bytesPerSample]); err != nil {
							select {
							case errorChan <- fmt.Errorf("failed to read final PCM samples: %w", err):
							case <-ctx.Done():
							}
							return
						}

						// Encode final frame
						opusData := make([]byte, MaxOpusFrameSize)
						encoded, err := ac.opusEncoder.Encode(pcmSamples, opusData)
						if err != nil {
							select {
							case errorChan <- fmt.Errorf("failed to encode final opus frame: %w", err):
							case <-ctx.Done():
							}
							return
						}

						// Send sample with context cancellation support
						select {
						case sampleChan <- media.Sample{
							Data:     opusData[:encoded],
							Duration: OpusFrameDuration,
						}:
						case <-ctx.Done():
							return
						}
					}
					return
				}
				select {
				case errorChan <- fmt.Errorf("failed to read PCM data: %w", err):
				case <-ctx.Done():
				}
				return
			}

			// Convert to int16 samples
			reader := bytes.NewReader(buffer)
			if err := binary.Read(reader, binary.LittleEndian, &pcmSamples); err != nil {
				select {
				case errorChan <- fmt.Errorf("failed to read PCM samples: %w", err):
				case <-ctx.Done():
				}
				return
			}

			// Encode to Opus
			opusData := make([]byte, MaxOpusFrameSize)
			encoded, err := ac.opusEncoder.Encode(pcmSamples, opusData)
			if err != nil {
				select {
				case errorChan <- fmt.Errorf("failed to encode opus frame: %w", err):
				case <-ctx.Done():
				}
				return
			}

			// Send sample with context cancellation support to prevent goroutine leak
			select {
			case sampleChan <- media.Sample{
				Data:     opusData[:encoded],
				Duration: OpusFrameDuration,
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return sampleChan, errorChan
}

// ConvertOpusToPCM decodes Opus audio data to PCM16 format
// Returns PCM data as 16-bit signed little-endian samples
func (ac *AudioConverter) ConvertOpusToPCM(opusData []byte) ([]byte, error) {
	if len(opusData) == 0 {
		return nil, fmt.Errorf("empty Opus data")
	}

	// Allocate buffer for PCM samples
	// Opus frames are typically 20ms, which at 48kHz stereo is 960 samples/channel * 2 channels = 1920 samples
	pcmSamples := make([]int16, ac.frameSize*ac.channels)

	// Decode Opus frame to PCM
	n, err := ac.opusDecoder.Decode(opusData, pcmSamples)
	if err != nil {
		return nil, fmt.Errorf("failed to decode opus frame: %w", err)
	}

	// Convert int16 samples to bytes
	pcmBytes := new(bytes.Buffer)
	if err := binary.Write(pcmBytes, binary.LittleEndian, pcmSamples[:n*ac.channels]); err != nil {
		return nil, fmt.Errorf("failed to write PCM samples: %w", err)
	}

	return pcmBytes.Bytes(), nil
}

// ConvertOpusToPCMFloat decodes Opus audio data to float32 PCM format
// This is useful when you need floating-point samples for audio processing
func (ac *AudioConverter) ConvertOpusToPCMFloat(opusData []byte) ([]float32, error) {
	if len(opusData) == 0 {
		return nil, fmt.Errorf("empty Opus data")
	}

	// Allocate buffer for PCM samples
	pcmSamples := make([]float32, ac.frameSize*ac.channels)

	// Decode Opus frame to PCM (float32)
	n, err := ac.opusDecoder.DecodeFloat32(opusData, pcmSamples)
	if err != nil {
		return nil, fmt.Errorf("failed to decode opus frame: %w", err)
	}

	return pcmSamples[:n*ac.channels], nil
}
