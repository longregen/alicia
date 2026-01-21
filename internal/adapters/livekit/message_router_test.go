package livekit

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

// TestNewMessageRouter tests message router creation
func TestNewMessageRouter(t *testing.T) {
	codec := NewCodec()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	messageRepo := newMockMessageRepo()
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	conversationID := "conv_test1"
	asrService := &mockASRService{}
	ttsService := &mockTTSService{}
	idGenerator := newMockIDGenerator()

	// Create a mock agent sender for protocol handler
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}

	realProtocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	// Create a mock agent (without LiveKit dependencies)
	agent := &Agent{
		ctx: context.Background(),
	}

	router := NewMessageRouter(
		codec,
		realProtocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		asrService,
		ttsService,
		idGenerator,
		agent,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	if router == nil {
		t.Fatal("expected router to be created, got nil")
	}

	if router.codec != codec {
		t.Error("expected codec to be set")
	}

	if router.dispatcher == nil {
		t.Error("expected dispatcher to be created")
	}

	if router.generationManager == nil {
		t.Error("expected generationManager to be created")
	}

	if router.conversationID != conversationID {
		t.Errorf("expected conversationID %s, got %s", conversationID, router.conversationID)
	}

	// Voice pipeline should be created if both ASR and TTS are available
	if router.voicePipeline == nil {
		t.Error("expected voicePipeline to be created when ASR and TTS are available")
	}
}

// TestNewMessageRouter_WithoutVoiceServices tests router creation without ASR/TTS
func TestNewMessageRouter_WithoutVoiceServices(t *testing.T) {
	codec := NewCodec()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	messageRepo := newMockMessageRepo()
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	conversationID := "conv_test1"
	idGenerator := newMockIDGenerator()

	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}

	protocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	agent := &Agent{
		ctx: context.Background(),
	}

	router := NewMessageRouter(
		codec,
		protocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		nil, // No ASR
		nil, // No TTS
		idGenerator,
		agent,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	if router == nil {
		t.Fatal("expected router to be created, got nil")
	}

	// Voice pipeline should NOT be created without ASR and TTS
	if router.voicePipeline != nil {
		t.Error("expected voicePipeline to be nil when ASR/TTS are not available")
	}
}

// TestOnDataReceived tests data message handling
func TestOnDataReceived(t *testing.T) {
	codec := NewCodec()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	messageRepo := newMockMessageRepo()
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	conversationID := "conv_test1"
	idGenerator := newMockIDGenerator()

	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}

	protocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	agent := &Agent{
		ctx: context.Background(),
	}

	router := NewMessageRouter(
		codec,
		protocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		nil, // No ASR for this test
		nil, // No TTS for this test
		idGenerator,
		agent,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	// Create a valid configuration message
	configMsg := &protocol.Configuration{
		ConversationID:   conversationID,
		LastSequenceSeen: 0,
	}
	envelope := protocol.NewEnvelope(1, conversationID, protocol.TypeConfiguration, configMsg)

	// Encode it
	data, err := codec.Encode(envelope)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	// Send it through the router
	dataMsg := &ports.DataChannelMessage{
		Data: data,
	}

	err = router.OnDataReceived(context.Background(), dataMsg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestOnDataReceived_InvalidData tests handling of malformed data
func TestOnDataReceived_InvalidData(t *testing.T) {
	codec := NewCodec()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	messageRepo := newMockMessageRepo()
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	conversationID := "conv_test1"
	idGenerator := newMockIDGenerator()

	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}

	protocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	agent := &Agent{
		ctx: context.Background(),
	}

	router := NewMessageRouter(
		codec,
		protocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		nil,
		nil,
		idGenerator,
		agent,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	// Send invalid data
	dataMsg := &ports.DataChannelMessage{
		Data: []byte("invalid msgpack data"),
	}

	err := router.OnDataReceived(context.Background(), dataMsg)
	if err == nil {
		t.Error("expected error for invalid data, got nil")
	}
}

// TestOnDataReceived_WithStanzaID tests stanza ID tracking
func TestOnDataReceived_WithStanzaID(t *testing.T) {
	codec := NewCodec()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	messageRepo := newMockMessageRepo()
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	conversationID := "conv_test1"
	idGenerator := newMockIDGenerator()

	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}

	protocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	agent := &Agent{
		ctx: context.Background(),
	}

	router := NewMessageRouter(
		codec,
		protocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		nil,
		nil,
		idGenerator,
		agent,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	// Create message with stanza ID
	configMsg := &protocol.Configuration{
		ConversationID:   conversationID,
		LastSequenceSeen: 0,
	}
	envelope := protocol.NewEnvelope(42, conversationID, protocol.TypeConfiguration, configMsg)

	data, err := codec.Encode(envelope)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	dataMsg := &ports.DataChannelMessage{
		Data: data,
	}

	err = router.OnDataReceived(context.Background(), dataMsg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Stanza ID should have been tracked (verified through protocol handler)
}

// TestOnAudioReceived_WithoutVoicePipeline tests fallback audio handling
func TestOnAudioReceived_WithoutVoicePipeline(t *testing.T) {
	codec := NewCodec()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	messageRepo := newMockMessageRepo()
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	conversationID := "conv_test1"
	idGenerator := newMockIDGenerator()

	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}

	protocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	transcribeCalled := false
	asrService := &mockASRService{
		transcribeFunc: func(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
			transcribeCalled = true
			return &ports.ASRResult{
				Text:       "test transcription",
				Confidence: 0.9,
				Language:   "en-US",
			}, nil
		},
	}

	agent := &Agent{
		ctx: context.Background(),
	}

	router := NewMessageRouter(
		codec,
		protocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		asrService,
		nil, // No TTS, so no voice pipeline
		idGenerator,
		agent,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	// Should not have voice pipeline
	if router.voicePipeline != nil {
		t.Error("expected no voice pipeline without TTS")
	}

	audioFrame := &ports.AudioFrame{
		Data:       []byte{0x01, 0x02, 0x03, 0x04},
		SampleRate: 48000,
		Channels:   1,
		TrackSID:   "track_123",
	}

	err := router.OnAudioReceived(context.Background(), audioFrame)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !transcribeCalled {
		t.Error("expected ASR transcribe to be called in fallback mode")
	}
}

// TestOnParticipantConnected tests participant connection handling
func TestOnParticipantConnected(t *testing.T) {
	codec := NewCodec()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	messageRepo := newMockMessageRepo()
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	conversationID := "conv_test1"
	idGenerator := newMockIDGenerator()

	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}

	protocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	agent := &Agent{
		ctx: context.Background(),
	}

	router := NewMessageRouter(
		codec,
		protocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		nil,
		nil,
		idGenerator,
		agent,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	participant := &ports.LiveKitParticipant{
		Identity: "user_123",
		Name:     "Test User",
	}

	err := router.OnParticipantConnected(context.Background(), participant)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestOnParticipantDisconnected tests participant disconnection handling
func TestOnParticipantDisconnected(t *testing.T) {
	codec := NewCodec()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	messageRepo := newMockMessageRepo()
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	conversationID := "conv_test1"
	idGenerator := newMockIDGenerator()

	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}

	protocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	agent := &Agent{
		ctx: context.Background(),
	}

	router := NewMessageRouter(
		codec,
		protocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		nil,
		nil,
		idGenerator,
		agent,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	participant := &ports.LiveKitParticipant{
		Identity: "user_123",
		Name:     "Test User",
	}

	err := router.OnParticipantDisconnected(context.Background(), participant)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestCleanup tests router cleanup
func TestCleanup(t *testing.T) {
	codec := NewCodec()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	messageRepo := newMockMessageRepo()
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	conversationID := "conv_test1"
	idGenerator := newMockIDGenerator()

	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}

	protocolHandler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversationID,
	)

	agent := &Agent{
		ctx: context.Background(),
	}

	router := NewMessageRouter(
		codec,
		protocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		&mockASRService{},
		&mockTTSService{},
		idGenerator,
		agent,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	// Voice pipeline should exist
	if router.voicePipeline == nil {
		t.Fatal("expected voice pipeline to exist")
	}

	// Cleanup
	router.Cleanup()

	// Voice pipeline should be nil after cleanup
	if router.voicePipeline != nil {
		t.Error("expected voice pipeline to be nil after cleanup")
	}
}
