package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type mockTTSService struct {
	synthesizeFunc       func(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error)
	synthesizeStreamFunc func(ctx context.Context, text string, options *ports.TTSOptions) (<-chan *ports.TTSResult, error)
}

func newMockTTSService() *mockTTSService {
	return &mockTTSService{}
}

func (m *mockTTSService) Synthesize(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, text, options)
	}
	return &ports.TTSResult{
		Audio:      []byte("fake audio data"),
		Format:     options.OutputFormat,
		DurationMs: 1500,
	}, nil
}

func (m *mockTTSService) SynthesizeStream(ctx context.Context, text string, options *ports.TTSOptions) (<-chan *ports.TTSResult, error) {
	if m.synthesizeStreamFunc != nil {
		return m.synthesizeStreamFunc(ctx, text, options)
	}
	ch := make(chan *ports.TTSResult, 2)
	go func() {
		defer close(ch)
		ch <- &ports.TTSResult{
			Audio:      []byte("chunk1"),
			Format:     options.OutputFormat,
			DurationMs: 500,
		}
		ch <- &ports.TTSResult{
			Audio:      []byte("chunk2"),
			Format:     options.OutputFormat,
			DurationMs: 500,
		}
	}()
	return ch, nil
}


func TestSynthesizeSpeech_NonStreaming(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)

	input := &SynthesizeSpeechInput{
		Text:            "Hello world",
		MessageID:       "msg_123",
		Voice:           "test_voice",
		Speed:           1.0,
		Pitch:           1.0,
		OutputFormat:    "audio/opus",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Audio == nil {
		t.Fatal("expected audio to be created")
	}

	if output.Audio.AudioFormat != "audio/opus" {
		t.Errorf("expected format 'audio/opus', got %s", output.Audio.AudioFormat)
	}

	if output.DurationMs != 1500 {
		t.Errorf("expected duration 1500ms, got %d", output.DurationMs)
	}

	if len(output.AudioData) == 0 {
		t.Error("expected audio data to be populated")
	}

	stored, _ := audioRepo.GetByID(context.Background(), output.Audio.ID)
	if stored == nil {
		t.Error("audio not stored in repository")
	}
}

func TestSynthesizeSpeech_WithDefaults(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)

	input := &SynthesizeSpeechInput{
		Text:            "Test text",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Audio.AudioFormat != "audio/opus" {
		t.Errorf("expected default format 'audio/opus', got %s", output.Audio.AudioFormat)
	}
}

func TestSynthesizeSpeech_WithSentence(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	sentence := models.NewSentence("sent_1", "msg_123", 0, "Test sentence")
	sentenceRepo.Create(context.Background(), sentence)

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)

	input := &SynthesizeSpeechInput{
		Text:            "Test sentence",
		MessageID:       "msg_123",
		SentenceID:      "sent_1",
		OutputFormat:    "audio/opus",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Sentence == nil {
		t.Error("expected sentence to be returned")
	}

	updatedSentence, _ := sentenceRepo.GetByID(context.Background(), "sent_1")
	if updatedSentence.AudioData == nil {
		t.Error("expected sentence to have audio data")
	}
}

func TestSynthesizeSpeech_Streaming(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)

	input := &SynthesizeSpeechInput{
		Text:            "Streaming test",
		MessageID:       "msg_123",
		OutputFormat:    "audio/opus",
		EnableStreaming: true,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.StreamChannel == nil {
		t.Fatal("expected stream channel to be provided")
	}

	chunks := 0
	var finalChunk *AudioStreamChunk
	for chunk := range output.StreamChannel {
		if chunk.Error != nil {
			t.Fatalf("unexpected stream error: %v", chunk.Error)
		}
		chunks++
		if chunk.IsFinal {
			finalChunk = chunk
		}
	}

	if chunks == 0 {
		t.Error("expected to receive stream chunks")
	}

	if finalChunk == nil {
		t.Error("expected final chunk")
	}
}

func TestSynthesizeSpeech_EmptyText(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)

	input := &SynthesizeSpeechInput{
		Text:            "",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty text, got nil")
	}
}

func TestSynthesizeSpeech_TTSServiceUnavailable(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, nil, idGen)

	input := &SynthesizeSpeechInput{
		Text:            "Test text",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when TTS service unavailable, got nil")
	}
}

func TestSynthesizeSpeech_TTSServiceFailure(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	ttsService.synthesizeFunc = func(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error) {
		return nil, errors.New("TTS service error")
	}

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)

	input := &SynthesizeSpeechInput{
		Text:            "Test text",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when TTS fails, got nil")
	}
}

func TestSynthesizeSpeech_StreamingFailure(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	ttsService.synthesizeStreamFunc = func(ctx context.Context, text string, options *ports.TTSOptions) (<-chan *ports.TTSResult, error) {
		return nil, errors.New("streaming failed")
	}

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)

	input := &SynthesizeSpeechInput{
		Text:            "Test text",
		EnableStreaming: true,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when streaming fails, got nil")
	}
}

func TestSynthesizeSpeech_SynthesizeForMessage(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	sent1 := models.NewSentence("sent_1", "msg_123", 0, "First sentence.")
	sent2 := models.NewSentence("sent_2", "msg_123", 1, "Second sentence.")
	sentenceRepo.Create(context.Background(), sent1)
	sentenceRepo.Create(context.Background(), sent2)

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)

	audioRecords, err := uc.SynthesizeForMessage(context.Background(), "msg_123", "test_voice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(audioRecords) != 2 {
		t.Errorf("expected 2 audio records, got %d", len(audioRecords))
	}
}

func TestSynthesizeSpeech_SynthesizeForMessageNoSentences(t *testing.T) {
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	uc := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)

	_, err := uc.SynthesizeForMessage(context.Background(), "msg_nonexistent", "test_voice")
	if err == nil {
		t.Fatal("expected error for message with no sentences, got nil")
	}
}

func TestSynthesizeSpeech_SynthesizeForMessageWithoutSentenceRepo(t *testing.T) {
	audioRepo := newMockAudioRepo()
	ttsService := newMockTTSService()
	idGen := newMockIDGenerator()

	uc := NewSynthesizeSpeech(audioRepo, nil, ttsService, idGen)

	_, err := uc.SynthesizeForMessage(context.Background(), "msg_123", "test_voice")
	if err == nil {
		t.Fatal("expected error when sentence repository unavailable, got nil")
	}
}

