package usecases

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

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
	return nil, errors.New("not found")
}

func (m *mockAudioRepo) GetByMessage(ctx context.Context, messageID string) (*models.Audio, error) {
	for _, audio := range m.store {
		if audio.MessageID == messageID {
			return audio, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockAudioRepo) GetByLiveKitTrack(ctx context.Context, trackSID string) (*models.Audio, error) {
	return nil, errors.New("not found")
}

func (m *mockAudioRepo) Update(ctx context.Context, audio *models.Audio) error {
	if _, ok := m.store[audio.ID]; !ok {
		return errors.New("not found")
	}
	m.store[audio.ID] = audio
	return nil
}

func (m *mockAudioRepo) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

type mockASRService struct {
	transcribeFunc func(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error)
}

func newMockASRService() *mockASRService {
	return &mockASRService{}
}

func (m *mockASRService) Transcribe(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
	if m.transcribeFunc != nil {
		return m.transcribeFunc(ctx, audio, format)
	}
	return &ports.ASRResult{
		Text:       "Transcribed text",
		Language:   "en",
		Confidence: 0.95,
		Duration:   2.5,
	}, nil
}

func (m *mockASRService) TranscribeStream(ctx context.Context, audioStream io.Reader, format string) (<-chan *ports.ASRResult, error) {
	return nil, errors.New("not implemented")
}

func TestProcessUserMessage_WithTextOnly(t *testing.T) {
	msgRepo := newMockMessageRepo()
	audioRepo := newMockAudioRepo()
	convRepo := newMockConversationRepo()
	asrService := newMockASRService()
	memoryService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation with a tip message
	conv := models.NewConversation("conv_123", "Test Conversation")
	tipID := "msg_previous_tip"
	conv.TipMessageID = &tipID
	convRepo.Create(context.Background(), conv)

	uc := NewProcessUserMessage(
		msgRepo,
		audioRepo,
		convRepo,
		asrService,
		memoryService,
		idGen,
		txManager,
	)

	input := &ports.ProcessUserMessageInput{
		ConversationID: "conv_123",
		TextContent:    "Hello there",
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Message == nil {
		t.Fatal("expected message to be created")
	}

	if output.Message.Contents != "Hello there" {
		t.Errorf("expected content 'Hello there', got %s", output.Message.Contents)
	}

	if output.Message.Role != models.MessageRoleUser {
		t.Errorf("expected role user, got %s", output.Message.Role)
	}

	if output.Audio != nil {
		t.Error("expected no audio for text-only message")
	}

	stored, _ := msgRepo.GetByID(context.Background(), output.Message.ID)
	if stored == nil {
		t.Error("message not stored in repository")
	}

	// Verify previous_id was set from conversation tip
	if output.Message.PreviousID != "msg_previous_tip" {
		t.Errorf("expected previous ID 'msg_previous_tip', got %s", output.Message.PreviousID)
	}

	// Verify conversation tip was updated
	updatedConv, _ := convRepo.GetByID(context.Background(), "conv_123")
	if updatedConv.TipMessageID == nil || *updatedConv.TipMessageID != output.Message.ID {
		t.Error("expected conversation tip to be updated to new message ID")
	}
}

func TestProcessUserMessage_WithAudio(t *testing.T) {
	msgRepo := newMockMessageRepo()
	audioRepo := newMockAudioRepo()
	convRepo := newMockConversationRepo()
	asrService := newMockASRService()
	memoryService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation
	conv := models.NewConversation("conv_123", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	asrService.transcribeFunc = func(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
		return &ports.ASRResult{
			Text:       "Audio transcription",
			Language:   "en",
			Confidence: 0.92,
			Duration:   3.0,
		}, nil
	}

	uc := NewProcessUserMessage(
		msgRepo,
		audioRepo,
		convRepo,
		asrService,
		memoryService,
		idGen,
		txManager,
	)

	audioData := []byte("fake audio data")
	input := &ports.ProcessUserMessageInput{
		ConversationID: "conv_123",
		AudioData:      audioData,
		AudioFormat:    "audio/opus",
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Message == nil {
		t.Fatal("expected message to be created")
	}

	if output.Message.Contents != "Audio transcription" {
		t.Errorf("expected transcribed content 'Audio transcription', got %s", output.Message.Contents)
	}

	if output.Audio == nil {
		t.Fatal("expected audio to be created")
	}

	if output.Audio.AudioFormat != "audio/opus" {
		t.Errorf("expected format 'audio/opus', got %s", output.Audio.AudioFormat)
	}

	storedAudio, _ := audioRepo.GetByID(context.Background(), output.Audio.ID)
	if storedAudio == nil {
		t.Error("audio not stored in repository")
	} else if storedAudio.MessageID != output.Message.ID {
		t.Errorf("expected audio message ID %s, got %s", output.Message.ID, storedAudio.MessageID)
	}
}

func TestProcessUserMessage_WithMemoryRetrieval(t *testing.T) {
	msgRepo := newMockMessageRepo()
	audioRepo := newMockAudioRepo()
	convRepo := newMockConversationRepo()
	asrService := newMockASRService()
	memoryService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation
	conv := models.NewConversation("conv_123", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	memoryService.searchWithScoresFunc = func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
		mem1 := models.NewMemory("mem_1", "User likes coffee")
		mem2 := models.NewMemory("mem_2", "User prefers mornings")
		return []*ports.MemorySearchResult{
			{Memory: mem1, Similarity: 0.88},
			{Memory: mem2, Similarity: 0.75},
		}, nil
	}

	memoryUsageTracked := false
	memoryService.trackUsageFunc = func(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error) {
		memoryUsageTracked = true
		mu := models.NewMemoryUsage("mu_1", conversationID, messageID, memoryID)
		mu.SimilarityScore = similarityScore
		return mu, nil
	}

	uc := NewProcessUserMessage(
		msgRepo,
		audioRepo,
		convRepo,
		asrService,
		memoryService,
		idGen,
		txManager,
	)

	input := &ports.ProcessUserMessageInput{
		ConversationID: "conv_123",
		TextContent:    "I want some coffee",
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.RelevantMemories) != 2 {
		t.Errorf("expected 2 relevant memories, got %d", len(output.RelevantMemories))
	}

	if !memoryUsageTracked {
		t.Error("expected memory usage to be tracked")
	}
}

func TestProcessUserMessage_EmptyContent(t *testing.T) {
	msgRepo := newMockMessageRepo()
	audioRepo := newMockAudioRepo()
	convRepo := newMockConversationRepo()
	asrService := newMockASRService()
	memoryService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation
	conv := models.NewConversation("conv_123", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	uc := NewProcessUserMessage(
		msgRepo,
		audioRepo,
		convRepo,
		asrService,
		memoryService,
		idGen,
		txManager,
	)

	input := &ports.ProcessUserMessageInput{
		ConversationID: "conv_123",
		TextContent:    "",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
}

func TestProcessUserMessage_ASRServiceUnavailable(t *testing.T) {
	msgRepo := newMockMessageRepo()
	audioRepo := newMockAudioRepo()
	convRepo := newMockConversationRepo()
	memoryService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation
	conv := models.NewConversation("conv_123", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	uc := NewProcessUserMessage(
		msgRepo,
		audioRepo,
		convRepo,
		nil, // ASR service unavailable
		memoryService,
		idGen,
		txManager,
	)

	audioData := []byte("fake audio data")
	input := &ports.ProcessUserMessageInput{
		ConversationID: "conv_123",
		AudioData:      audioData,
		AudioFormat:    "audio/opus",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when ASR service unavailable, got nil")
	}
}

func TestProcessUserMessage_ASRTranscriptionFails(t *testing.T) {
	msgRepo := newMockMessageRepo()
	audioRepo := newMockAudioRepo()
	convRepo := newMockConversationRepo()
	asrService := newMockASRService()
	memoryService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation
	conv := models.NewConversation("conv_123", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	asrService.transcribeFunc = func(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
		return nil, errors.New("transcription failed")
	}

	uc := NewProcessUserMessage(
		msgRepo,
		audioRepo,
		convRepo,
		asrService,
		memoryService,
		idGen,
		txManager,
	)

	audioData := []byte("fake audio data")
	input := &ports.ProcessUserMessageInput{
		ConversationID: "conv_123",
		AudioData:      audioData,
		AudioFormat:    "audio/opus",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when transcription fails, got nil")
	}
}

func TestProcessUserMessage_WithConversationTip(t *testing.T) {
	msgRepo := newMockMessageRepo()
	audioRepo := newMockAudioRepo()
	convRepo := newMockConversationRepo()
	asrService := newMockASRService()
	memoryService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation with a tip message
	conv := models.NewConversation("conv_123", "Test Conversation")
	tipID := "msg_previous_tip"
	conv.TipMessageID = &tipID
	convRepo.Create(context.Background(), conv)

	uc := NewProcessUserMessage(
		msgRepo,
		audioRepo,
		convRepo,
		asrService,
		memoryService,
		idGen,
		txManager,
	)

	input := &ports.ProcessUserMessageInput{
		ConversationID: "conv_123",
		TextContent:    "Follow-up message",
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Previous ID should be set from conversation tip, not from input
	if output.Message.PreviousID != "msg_previous_tip" {
		t.Errorf("expected previous ID 'msg_previous_tip', got %s", output.Message.PreviousID)
	}

	// Verify conversation tip was updated to new message
	updatedConv, _ := convRepo.GetByID(context.Background(), "conv_123")
	if updatedConv.TipMessageID == nil || *updatedConv.TipMessageID != output.Message.ID {
		t.Error("expected conversation tip to be updated to new message ID")
	}
}

func TestProcessUserMessage_MemoryServiceFailure(t *testing.T) {
	msgRepo := newMockMessageRepo()
	audioRepo := newMockAudioRepo()
	convRepo := newMockConversationRepo()
	asrService := newMockASRService()
	memoryService := newMockMemoryService()
	idGen := newMockIDGenerator()
	txManager := &mockTransactionManager{}

	// Create a conversation
	conv := models.NewConversation("conv_123", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	memoryService.searchWithScoresFunc = func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
		return nil, errors.New("memory service error")
	}

	uc := NewProcessUserMessage(
		msgRepo,
		audioRepo,
		convRepo,
		asrService,
		memoryService,
		idGen,
		txManager,
	)

	input := &ports.ProcessUserMessageInput{
		ConversationID: "conv_123",
		TextContent:    "Test message",
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Message == nil {
		t.Fatal("expected message to be created even when memory service fails")
	}

	if len(output.RelevantMemories) != 0 {
		t.Errorf("expected 0 memories when service fails, got %d", len(output.RelevantMemories))
	}
}
