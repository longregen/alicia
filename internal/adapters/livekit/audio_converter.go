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
	MaxOpusFrameSize      = 4000
	OpusFrameDuration     = 20_000_000
	BytesPerPCMSample     = 2
	OpusFramesPerSecond   = 50
	StreamBufferSize      = 10
	OpusEncoderComplexity = 10
)

type AudioConverter struct {
	opusEncoder *opus.Encoder
	opusDecoder *opus.Decoder
	sampleRate  int
	channels    int
	frameSize   int
}

func NewAudioConverter(sampleRate, channels int) (*AudioConverter, error) {
	encoder, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus encoder: %w", err)
	}

	// 64kbps is excellent quality for voice while keeping frames
	// well under the 1200-byte RTP MTU limit for WebRTC
	encoder.SetBitrate(64000)
	encoder.SetComplexity(OpusEncoderComplexity)

	decoder, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus decoder: %w", err)
	}

	frameSize := sampleRate / OpusFramesPerSecond

	return &AudioConverter{
		opusEncoder: encoder,
		opusDecoder: decoder,
		sampleRate:  sampleRate,
		channels:    channels,
		frameSize:   frameSize,
	}, nil
}

func (ac *AudioConverter) ConvertPCMToOpus(pcmData []byte) ([]media.Sample, error) {
	if len(pcmData) == 0 {
		return nil, fmt.Errorf("empty PCM data")
	}

	bytesPerSample := BytesPerPCMSample
	samplesCount := len(pcmData) / bytesPerSample

	pcmSamples := make([]int16, samplesCount)
	reader := bytes.NewReader(pcmData)
	if err := binary.Read(reader, binary.LittleEndian, &pcmSamples); err != nil {
		return nil, fmt.Errorf("failed to read PCM samples: %w", err)
	}

	var samples []media.Sample
	frameSize := ac.frameSize * ac.channels

	for i := 0; i < len(pcmSamples); i += frameSize {
		end := i + frameSize
		if end > len(pcmSamples) {
			end = len(pcmSamples)
			paddedFrame := make([]int16, frameSize)
			copy(paddedFrame, pcmSamples[i:end])
			pcmSamples = append(pcmSamples[:i], paddedFrame...)
		}

		frame := pcmSamples[i:end]

		opusData := make([]byte, MaxOpusFrameSize)
		n, err := ac.opusEncoder.Encode(frame, opusData)
		if err != nil {
			return nil, fmt.Errorf("failed to encode opus frame: %w", err)
		}

		sample := media.Sample{
			Data:     opusData[:n],
			Duration: OpusFrameDuration,
		}
		samples = append(samples, sample)
	}

	return samples, nil
}

func (ac *AudioConverter) ConvertPCMStream(ctx context.Context, pcmReader io.Reader) (<-chan media.Sample, <-chan error) {
	sampleChan := make(chan media.Sample, StreamBufferSize)
	errorChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				select {
				case errorChan <- fmt.Errorf("panic in ConvertPCMStream: %v", r):
				default:
				}
			}
			close(errorChan)
			close(sampleChan)
		}()

		bytesPerSample := BytesPerPCMSample
		frameBytes := ac.frameSize * ac.channels * bytesPerSample
		buffer := make([]byte, frameBytes)
		pcmSamples := make([]int16, ac.frameSize*ac.channels)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, err := io.ReadFull(pcmReader, buffer)
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					if n > 0 {
						reader := bytes.NewReader(buffer[:n])
						if err := binary.Read(reader, binary.LittleEndian, pcmSamples[:n/bytesPerSample]); err != nil {
							select {
							case errorChan <- fmt.Errorf("failed to read final PCM samples: %w", err):
							case <-ctx.Done():
							}
							return
						}

						opusData := make([]byte, MaxOpusFrameSize)
						encoded, err := ac.opusEncoder.Encode(pcmSamples, opusData)
						if err != nil {
							select {
							case errorChan <- fmt.Errorf("failed to encode final opus frame: %w", err):
							case <-ctx.Done():
							}
							return
						}

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

			reader := bytes.NewReader(buffer)
			if err := binary.Read(reader, binary.LittleEndian, &pcmSamples); err != nil {
				select {
				case errorChan <- fmt.Errorf("failed to read PCM samples: %w", err):
				case <-ctx.Done():
				}
				return
			}

			opusData := make([]byte, MaxOpusFrameSize)
			encoded, err := ac.opusEncoder.Encode(pcmSamples, opusData)
			if err != nil {
				select {
				case errorChan <- fmt.Errorf("failed to encode opus frame: %w", err):
				case <-ctx.Done():
				}
				return
			}

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

func (ac *AudioConverter) ConvertOpusToPCM(opusData []byte) ([]byte, error) {
	if len(opusData) == 0 {
		return nil, fmt.Errorf("empty Opus data")
	}

	pcmSamples := make([]int16, ac.frameSize*ac.channels)

	n, err := ac.opusDecoder.Decode(opusData, pcmSamples)
	if err != nil {
		return nil, fmt.Errorf("failed to decode opus frame: %w", err)
	}

	pcmBytes := new(bytes.Buffer)
	if err := binary.Write(pcmBytes, binary.LittleEndian, pcmSamples[:n*ac.channels]); err != nil {
		return nil, fmt.Errorf("failed to write PCM samples: %w", err)
	}

	return pcmBytes.Bytes(), nil
}

func (ac *AudioConverter) ConvertOpusToPCMFloat(opusData []byte) ([]float32, error) {
	if len(opusData) == 0 {
		return nil, fmt.Errorf("empty Opus data")
	}

	pcmSamples := make([]float32, ac.frameSize*ac.channels)

	n, err := ac.opusDecoder.DecodeFloat32(opusData, pcmSamples)
	if err != nil {
		return nil, fmt.Errorf("failed to decode opus frame: %w", err)
	}

	return pcmSamples[:n*ac.channels], nil
}

func ResamplePCM(input []byte, inputRate, outputRate, inputChannels, outputChannels int) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("empty input data")
	}

	if inputRate == outputRate && inputChannels == outputChannels {
		return input, nil
	}

	inputSamplesPerChannel := len(input) / (BytesPerPCMSample * inputChannels)

	totalInputSamples := len(input) / BytesPerPCMSample
	inputInt16 := make([]int16, totalInputSamples)
	reader := bytes.NewReader(input)
	if err := binary.Read(reader, binary.LittleEndian, &inputInt16); err != nil {
		return nil, fmt.Errorf("failed to read input samples: %w", err)
	}

	var monoSamples []int16
	if inputChannels == 2 {
		monoSamples = make([]int16, inputSamplesPerChannel)
		for i := 0; i < inputSamplesPerChannel; i++ {
			left := int32(inputInt16[i*2])
			right := int32(inputInt16[i*2+1])
			monoSamples[i] = int16((left + right) / 2)
		}
	} else {
		monoSamples = inputInt16
	}

	var resampledMono []int16
	if inputRate != outputRate {
		ratio := float64(outputRate) / float64(inputRate)
		outputSamplesPerChannel := int(float64(len(monoSamples)) * ratio)
		resampledMono = make([]int16, outputSamplesPerChannel)

		for i := 0; i < outputSamplesPerChannel; i++ {
			srcPos := float64(i) / ratio
			srcIdx := int(srcPos)
			frac := srcPos - float64(srcIdx)

			if srcIdx >= len(monoSamples)-1 {
				resampledMono[i] = monoSamples[len(monoSamples)-1]
			} else {
				sample1 := int32(monoSamples[srcIdx])
				sample2 := int32(monoSamples[srcIdx+1])
				resampledMono[i] = int16(sample1 + int32(float64(sample2-sample1)*frac))
			}
		}
	} else {
		resampledMono = monoSamples
	}

	outputBytes := len(resampledMono) * outputChannels * BytesPerPCMSample
	output := make([]byte, 0, outputBytes)
	writer := bytes.NewBuffer(output)

	for _, sample := range resampledMono {
		binary.Write(writer, binary.LittleEndian, sample)
		if outputChannels == 2 {
			binary.Write(writer, binary.LittleEndian, sample)
		}
	}

	return writer.Bytes(), nil
}
