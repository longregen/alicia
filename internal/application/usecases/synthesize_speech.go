package usecases

import (
	"context"
	"fmt"
	"log"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type SynthesizeSpeechInput struct {
	Text            string
	MessageID       string
	SentenceID      string
	Voice           string
	Speed           float32
	Pitch           float32
	OutputFormat    string
	EnableStreaming bool
}

type SynthesizeSpeechOutput struct {
	Audio         *models.Audio
	Sentence      *models.Sentence
	AudioData     []byte
	Format        string
	DurationMs    int
	StreamChannel <-chan *AudioStreamChunk
}

type AudioStreamChunk struct {
	Data       []byte
	Format     string
	DurationMs int
	Sequence   int
	IsFinal    bool
	Error      error
}

type SynthesizeSpeech struct {
	audioRepo    ports.AudioRepository
	sentenceRepo ports.SentenceRepository
	ttsService   ports.TTSService
	idGenerator  ports.IDGenerator
}

func NewSynthesizeSpeech(
	audioRepo ports.AudioRepository,
	sentenceRepo ports.SentenceRepository,
	ttsService ports.TTSService,
	idGenerator ports.IDGenerator,
) *SynthesizeSpeech {
	return &SynthesizeSpeech{
		audioRepo:    audioRepo,
		sentenceRepo: sentenceRepo,
		ttsService:   ttsService,
		idGenerator:  idGenerator,
	}
}

func (uc *SynthesizeSpeech) Execute(ctx context.Context, input *SynthesizeSpeechInput) (*SynthesizeSpeechOutput, error) {
	if input.Text == "" {
		return nil, fmt.Errorf("text is required for speech synthesis")
	}

	if uc.ttsService == nil {
		return nil, fmt.Errorf("TTS service is not available")
	}

	options := &ports.TTSOptions{
		Voice:        input.Voice,
		Speed:        input.Speed,
		Pitch:        input.Pitch,
		OutputFormat: input.OutputFormat,
	}

	if options.OutputFormat == "" {
		options.OutputFormat = "audio/opus"
	}
	if options.Speed == 0 {
		options.Speed = 1.0
	}
	if options.Pitch == 0 {
		options.Pitch = 1.0
	}

	if input.EnableStreaming {
		return uc.synthesizeStreaming(ctx, input, options)
	}

	return uc.synthesizeNonStreaming(ctx, input, options)
}

func (uc *SynthesizeSpeech) synthesizeNonStreaming(
	ctx context.Context,
	input *SynthesizeSpeechInput,
	options *ports.TTSOptions,
) (*SynthesizeSpeechOutput, error) {
	ttsResult, err := uc.ttsService.Synthesize(ctx, input.Text, options)
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize speech: %w", err)
	}

	audioID := uc.idGenerator.GenerateAudioID()
	audio := models.NewOutputAudio(audioID, ttsResult.Format)
	audio.SetData(ttsResult.Audio, ttsResult.DurationMs)

	if input.MessageID != "" {
		audio.MessageID = input.MessageID
	}

	if err := uc.audioRepo.Create(ctx, audio); err != nil {
		return nil, fmt.Errorf("failed to store audio: %w", err)
	}

	var sentence *models.Sentence
	if input.SentenceID != "" && uc.sentenceRepo != nil {
		sentence, err = uc.sentenceRepo.GetByID(ctx, input.SentenceID)
		if err != nil {
			// Log warning but don't fail
			log.Printf("warning: failed to get sentence: %v", err)
		} else {
			sentence.SetAudio(models.AudioTypeOutput, ttsResult.Format, ttsResult.Audio, ttsResult.DurationMs)
			if err := uc.sentenceRepo.Update(ctx, sentence); err != nil {
				// Log warning but don't fail
				log.Printf("warning: failed to update sentence with audio: %v", err)
			}
		}
	}

	return &SynthesizeSpeechOutput{
		Audio:      audio,
		Sentence:   sentence,
		AudioData:  ttsResult.Audio,
		Format:     ttsResult.Format,
		DurationMs: ttsResult.DurationMs,
	}, nil
}

func (uc *SynthesizeSpeech) synthesizeStreaming(
	ctx context.Context,
	input *SynthesizeSpeechInput,
	options *ports.TTSOptions,
) (*SynthesizeSpeechOutput, error) {
	streamChan, err := uc.ttsService.SynthesizeStream(ctx, input.Text, options)
	if err != nil {
		return nil, fmt.Errorf("failed to start streaming synthesis: %w", err)
	}

	audioID := uc.idGenerator.GenerateAudioID()
	audio := models.NewOutputAudio(audioID, options.OutputFormat)

	if input.MessageID != "" {
		audio.MessageID = input.MessageID
	}

	if err := uc.audioRepo.Create(ctx, audio); err != nil {
		return nil, fmt.Errorf("failed to store audio: %w", err)
	}

	outputChan := make(chan *AudioStreamChunk, 10)

	go uc.processAudioStream(ctx, audio, streamChan, outputChan, input.SentenceID)

	return &SynthesizeSpeechOutput{
		Audio:         audio,
		Format:        options.OutputFormat,
		StreamChannel: outputChan,
	}, nil
}

func (uc *SynthesizeSpeech) processAudioStream(
	ctx context.Context,
	audio *models.Audio,
	inputChan <-chan *ports.TTSResult,
	outputChan chan<- *AudioStreamChunk,
	sentenceID string,
) {
	defer close(outputChan)

	var allAudioData []byte
	totalDuration := 0
	sequence := 0

	for chunk := range inputChan {
		allAudioData = append(allAudioData, chunk.Audio...)
		totalDuration += chunk.DurationMs

		outputChan <- &AudioStreamChunk{
			Data:       chunk.Audio,
			Format:     chunk.Format,
			DurationMs: chunk.DurationMs,
			Sequence:   sequence,
			IsFinal:    false,
		}

		sequence++
	}

	audio.SetData(allAudioData, totalDuration)
	if err := uc.audioRepo.Update(ctx, audio); err != nil {
		outputChan <- &AudioStreamChunk{
			Error: fmt.Errorf("failed to update audio: %w", err),
		}
		return
	}

	if sentenceID != "" && uc.sentenceRepo != nil {
		sentence, err := uc.sentenceRepo.GetByID(ctx, sentenceID)
		if err != nil {
			// Log warning but don't fail
			log.Printf("warning: failed to get sentence: %v", err)
		} else {
			sentence.SetAudio(models.AudioTypeOutput, audio.AudioFormat, allAudioData, totalDuration)
			if err := uc.sentenceRepo.Update(ctx, sentence); err != nil {
				// Log warning but don't fail
				log.Printf("warning: failed to update sentence with audio: %v", err)
			}
		}
	}

	outputChan <- &AudioStreamChunk{
		Data:       nil,
		Format:     audio.AudioFormat,
		DurationMs: totalDuration,
		Sequence:   sequence,
		IsFinal:    true,
	}
}

func (uc *SynthesizeSpeech) SynthesizeForMessage(ctx context.Context, messageID, voice string) ([]*models.Audio, error) {
	if uc.sentenceRepo == nil {
		return nil, fmt.Errorf("sentence repository is not available")
	}

	sentences, err := uc.sentenceRepo.GetByMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sentences for message: %w", err)
	}

	if len(sentences) == 0 {
		return nil, fmt.Errorf("no sentences found for message")
	}

	audioRecords := make([]*models.Audio, 0, len(sentences))

	for _, sentence := range sentences {
		input := &SynthesizeSpeechInput{
			Text:            sentence.Text,
			MessageID:       messageID,
			SentenceID:      sentence.ID,
			Voice:           voice,
			Speed:           1.0,
			Pitch:           1.0,
			OutputFormat:    "audio/opus",
			EnableStreaming: false,
		}

		output, err := uc.Execute(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to synthesize sentence %s: %w", sentence.ID, err)
		}

		audioRecords = append(audioRecords, output.Audio)
	}

	return audioRecords, nil
}

