package services

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Mock audio repository
type mockAudioRepo struct {
	store map[string]*models.Audio
}

func newMockAudioRepo() *mockAudioRepo {
	return &mockAudioRepo{
		store: make(map[string]*models.Audio),
	}
}

func (m *mockAudioRepo) Create(ctx context.Context, audio *models.Audio) error {
	m.store[audio.ID] = audio
	return nil
}

func (m *mockAudioRepo) GetByID(ctx context.Context, id string) (*models.Audio, error) {
	if audio, ok := m.store[id]; ok {
		return audio, nil
	}
	return nil, errNotFound
}

func (m *mockAudioRepo) GetByMessage(ctx context.Context, messageID string) (*models.Audio, error) {
	for _, audio := range m.store {
		if audio.MessageID == messageID {
			return audio, nil
		}
	}
	return nil, errNotFound
}

func (m *mockAudioRepo) GetByLiveKitTrack(ctx context.Context, trackSID string) (*models.Audio, error) {
	for _, audio := range m.store {
		if audio.LiveKitTrackSID == trackSID {
			return audio, nil
		}
	}
	return nil, errNotFound
}

func (m *mockAudioRepo) Update(ctx context.Context, audio *models.Audio) error {
	if _, ok := m.store[audio.ID]; !ok {
		return errNotFound
	}
	m.store[audio.ID] = audio
	return nil
}

func (m *mockAudioRepo) Delete(ctx context.Context, id string) error {
	if audio, ok := m.store[id]; ok {
		now := time.Now()
		audio.DeletedAt = &now
		m.store[id] = audio
		return nil
	}
	return errNotFound
}

// Mock ASR service
type mockASRService struct {
	shouldFail bool
}

func (m *mockASRService) Transcribe(ctx context.Context, audioData []byte, format string) (*ports.ASRResult, error) {
	if m.shouldFail {
		return nil, errors.New("transcription failed")
	}

	return &ports.ASRResult{
		Text:       "transcribed text",
		Language:   "en",
		Confidence: 0.95,
		Duration:   1000,
		Segments:   []models.Segment{},
	}, nil
}

func (m *mockASRService) TranscribeStream(ctx context.Context, audioStream io.Reader, format string) (<-chan *ports.ASRResult, error) {
	ch := make(chan *ports.ASRResult)
	close(ch)
	return ch, nil
}

// Mock TTS service
type mockTTSService struct {
	shouldFail bool
}

func (m *mockTTSService) Synthesize(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error) {
	if m.shouldFail {
		return nil, errors.New("synthesis failed")
	}

	return &ports.TTSResult{
		Audio:      []byte("synthesized audio"),
		Format:     "wav",
		DurationMs: 1000,
	}, nil
}

func (m *mockTTSService) SynthesizeStream(ctx context.Context, text string, options *ports.TTSOptions) (<-chan *ports.TTSResult, error) {
	ch := make(chan *ports.TTSResult)
	close(ch)
	return ch, nil
}

// Mock sentence repository
type mockSentenceRepo struct {
	store map[string]*models.Sentence
}

func newMockSentenceRepo() *mockSentenceRepo {
	return &mockSentenceRepo{
		store: make(map[string]*models.Sentence),
	}
}

func (m *mockSentenceRepo) Create(ctx context.Context, sentence *models.Sentence) error {
	m.store[sentence.ID] = sentence
	return nil
}

func (m *mockSentenceRepo) GetByID(ctx context.Context, id string) (*models.Sentence, error) {
	if sentence, ok := m.store[id]; ok {
		return sentence, nil
	}
	return nil, errNotFound
}

func (m *mockSentenceRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.Sentence, error) {
	sentences := make([]*models.Sentence, 0)
	for _, sentence := range m.store {
		if sentence.MessageID == messageID {
			sentences = append(sentences, sentence)
		}
	}
	return sentences, nil
}

func (m *mockSentenceRepo) Update(ctx context.Context, sentence *models.Sentence) error {
	if _, ok := m.store[sentence.ID]; !ok {
		return errNotFound
	}
	m.store[sentence.ID] = sentence
	return nil
}

func (m *mockSentenceRepo) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

func (m *mockSentenceRepo) GetNextSequenceNumber(ctx context.Context, messageID string) (int, error) {
	count := 0
	for _, sentence := range m.store {
		if sentence.MessageID == messageID {
			count++
		}
	}
	return count, nil
}

func (m *mockSentenceRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Sentence, error) {
	return []*models.Sentence{}, nil
}

func (m *mockSentenceRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Sentence, error) {
	return []*models.Sentence{}, nil
}

// Tests

func TestAudioService_CreateInputAudio(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, err := svc.CreateInputAudio(context.Background(), "wav", []byte("audio data"), 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if audio.ID != "audio_test1" {
		t.Errorf("expected ID audio_test1, got %s", audio.ID)
	}

	if audio.AudioType != models.AudioTypeInput {
		t.Errorf("expected audio type input, got %s", audio.AudioType)
	}

	if audio.AudioFormat != "wav" {
		t.Errorf("expected format wav, got %s", audio.AudioFormat)
	}
}

func TestAudioService_CreateInputAudio_EmptyFormat(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	_, err := svc.CreateInputAudio(context.Background(), "", []byte("data"), 1000)
	if err == nil {
		t.Fatal("expected error for empty format, got nil")
	}
}

func TestAudioService_CreateInputAudio_EmptyData(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	_, err := svc.CreateInputAudio(context.Background(), "wav", []byte{}, 1000)
	if err == nil {
		t.Fatal("expected error for empty data, got nil")
	}
}

func TestAudioService_CreateOutputAudio(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, err := svc.CreateOutputAudio(context.Background(), "wav", []byte("audio data"), 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if audio.AudioType != models.AudioTypeOutput {
		t.Errorf("expected audio type output, got %s", audio.AudioType)
	}
}

func TestAudioService_GetByID(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	created, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("data"), 1000)

	retrieved, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}
}

func TestAudioService_GetByID_EmptyID(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	_, err := svc.GetByID(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
}

func TestAudioService_GetByID_Deleted(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("data"), 1000)
	svc.Delete(context.Background(), audio.ID)

	_, err := svc.GetByID(context.Background(), audio.ID)
	if err == nil {
		t.Fatal("expected error for deleted audio, got nil")
	}
}

func TestAudioService_Update(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("data"), 1000)
	audio.Transcription = "new transcription"

	err := svc.Update(context.Background(), audio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := svc.GetByID(context.Background(), audio.ID)
	if retrieved.Transcription != "new transcription" {
		t.Errorf("expected transcription to be updated")
	}
}

func TestAudioService_Delete(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("data"), 1000)

	err := svc.Delete(context.Background(), audio.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GetByID(context.Background(), audio.ID)
	if err == nil {
		t.Error("expected error when getting deleted audio")
	}
}

func TestAudioService_Transcribe(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("audio data"), 1000)

	transcribed, err := svc.Transcribe(context.Background(), audio.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transcribed.Transcription != "transcribed text" {
		t.Errorf("expected transcription 'transcribed text', got %s", transcribed.Transcription)
	}

	if transcribed.TranscriptionMeta == nil {
		t.Error("expected transcription meta to be set")
	}
}

func TestAudioService_Transcribe_NoASRService(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, nil, nil, idGen, txManager)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("audio data"), 1000)

	_, err := svc.Transcribe(context.Background(), audio.ID)
	if err == nil {
		t.Fatal("expected error when ASR service not available, got nil")
	}
}

func TestAudioService_Transcribe_EmptyData(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio := models.NewInputAudio("audio_test", "wav")
	audioRepo.Create(context.Background(), audio)

	_, err := svc.Transcribe(context.Background(), audio.ID)
	if err == nil {
		t.Fatal("expected error for empty audio data, got nil")
	}
}

func TestAudioService_Synthesize(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, err := svc.Synthesize(context.Background(), "hello world", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if audio.AudioType != models.AudioTypeOutput {
		t.Errorf("expected audio type output, got %s", audio.AudioType)
	}

	if len(audio.AudioData) == 0 {
		t.Error("expected audio data to be set")
	}
}

func TestAudioService_Synthesize_EmptyText(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	_, err := svc.Synthesize(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty text, got nil")
	}
}

func TestAudioService_Synthesize_NoTTSService(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, nil, nil, idGen, txManager)

	_, err := svc.Synthesize(context.Background(), "hello world", nil)
	if err == nil {
		t.Fatal("expected error when TTS service not available, got nil")
	}
}

func TestAudioService_SynthesizeForMessage(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	msg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleAssistant, "hello world")
	msgRepo.Create(context.Background(), msg)

	audio, err := svc.SynthesizeForMessage(context.Background(), msg.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if audio.MessageID != msg.ID {
		t.Errorf("expected message ID %s, got %s", msg.ID, audio.MessageID)
	}
}

func TestAudioService_AssociateLiveKitTrack(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("data"), 1000)

	updated, err := svc.AssociateLiveKitTrack(context.Background(), audio.ID, "track_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.LiveKitTrackSID != "track_123" {
		t.Error("expected LiveKit track SID to be set")
	}
}

func TestAudioService_AssociateWithMessage(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	msg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleUser, "hello")
	msgRepo.Create(context.Background(), msg)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("data"), 1000)

	updated, err := svc.AssociateWithMessage(context.Background(), audio.ID, msg.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.MessageID != msg.ID {
		t.Errorf("expected message ID %s, got %s", msg.ID, updated.MessageID)
	}
}

func TestAudioService_SetTranscription(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("data"), 1000)

	updated, err := svc.SetTranscription(context.Background(), audio.ID, "manual transcription")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Transcription != "manual transcription" {
		t.Errorf("expected transcription 'manual transcription', got %s", updated.Transcription)
	}
}

func TestAudioService_GetByMessage(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	msg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleUser, "hello")
	msgRepo.Create(context.Background(), msg)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("data"), 1000)
	svc.AssociateWithMessage(context.Background(), audio.ID, msg.ID)

	retrieved, err := svc.GetByMessage(context.Background(), msg.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != audio.ID {
		t.Errorf("expected audio ID %s, got %s", audio.ID, retrieved.ID)
	}
}

func TestAudioService_GetByLiveKitTrack(t *testing.T) {
	audioRepo := newMockAudioRepo()
	msgRepo := newMockMessageRepo()
	asr := &mockASRService{}
	tts := &mockTTSService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewAudioService(audioRepo, msgRepo, asr, tts, idGen, txManager)

	audio, _ := svc.CreateInputAudio(context.Background(), "wav", []byte("data"), 1000)
	svc.AssociateLiveKitTrack(context.Background(), audio.ID, "track_123")

	retrieved, err := svc.GetByLiveKitTrack(context.Background(), "track_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != audio.ID {
		t.Errorf("expected audio ID %s, got %s", audio.ID, retrieved.ID)
	}
}
