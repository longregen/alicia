package livekit

import (
	"context"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// TestNewAgentFactory tests agent factory creation
func TestNewAgentFactory(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	asrService := &mockASRService{}
	ttsService := &mockTTSService{}
	idGenerator := newMockIDGenerator()

	factory := NewAgentFactory(
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		asrService,
		ttsService,
		idGenerator,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	if factory == nil {
		t.Fatal("expected factory to be created, got nil")
	}

	if factory.conversationRepo != conversationRepo {
		t.Error("expected conversationRepo to be set")
	}

	if factory.messageRepo != messageRepo {
		t.Error("expected messageRepo to be set")
	}

	if factory.voteRepo != voteRepo {
		t.Error("expected voteRepo to be set")
	}

	if factory.noteRepo != noteRepo {
		t.Error("expected noteRepo to be set")
	}

	if factory.memoryService != memoryService {
		t.Error("expected memoryService to be set")
	}

	if factory.optimizationService != optimizationService {
		t.Error("expected optimizationService to be set")
	}

	if factory.asrService == nil {
		t.Error("expected asrService to be set")
	}

	if factory.ttsService == nil {
		t.Error("expected ttsService to be set")
	}

	if factory.idGenerator == nil {
		t.Error("expected idGenerator to be set")
	}
}

// TestCreateAgent tests agent creation with all dependencies wired
func TestCreateAgent(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	asrService := &mockASRService{}
	ttsService := &mockTTSService{}
	idGenerator := newMockIDGenerator()

	factory := NewAgentFactory(
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		asrService,
		ttsService,
		idGenerator,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	// Create a conversation for testing
	conversationID := "conv_test_factory"
	now := time.Now()
	conversation := &models.Conversation{
		ID:                 conversationID,
		Title:              "Test Conversation",
		Status:             models.ConversationStatusActive,
		LastClientStanzaID: 0,
		LastServerStanzaID: -1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	conversationRepo.Create(context.Background(), conversation)

	config := &AgentConfig{
		URL:           "ws://localhost:7880",
		APIKey:        "test-api-key",
		APISecret:     "test-api-secret",
		AgentIdentity: "agent",
		AgentName:     "test-agent",
	}

	agent, router, err := factory.CreateAgent(config, conversationID)
	if err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	if agent == nil {
		t.Fatal("expected agent to be created, got nil")
	}

	if router == nil {
		t.Fatal("expected router to be created, got nil")
	}

	// Verify router is wired correctly
	if router.conversationID != conversationID {
		t.Errorf("expected conversationID %s, got %s", conversationID, router.conversationID)
	}

	if router.codec == nil {
		t.Error("expected codec to be set in router")
	}

	if router.dispatcher == nil {
		t.Error("expected dispatcher to be set in router")
	}

	if router.generationManager == nil {
		t.Error("expected generationManager to be set in router")
	}

	// Voice pipeline should be created with both ASR and TTS
	if router.voicePipeline == nil {
		t.Error("expected voicePipeline to be created with ASR and TTS")
	}
}

// TestCreateAgent_WithoutVoiceServices tests agent creation without ASR/TTS
func TestCreateAgent_WithoutVoiceServices(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	idGenerator := newMockIDGenerator()

	factory := NewAgentFactory(
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		nil, // No ASR
		nil, // No TTS
		idGenerator,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	conversationID := "conv_test_no_voice"
	now := time.Now()
	conversation := &models.Conversation{
		ID:                 conversationID,
		Title:              "Test Conversation",
		Status:             models.ConversationStatusActive,
		LastClientStanzaID: 0,
		LastServerStanzaID: -1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	conversationRepo.Create(context.Background(), conversation)

	config := &AgentConfig{
		URL:           "ws://localhost:7880",
		APIKey:        "test-api-key",
		APISecret:     "test-api-secret",
		AgentIdentity: "agent",
		AgentName:     "test-agent",
	}

	agent, router, err := factory.CreateAgent(config, conversationID)
	if err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	if agent == nil {
		t.Fatal("expected agent to be created, got nil")
	}

	if router == nil {
		t.Fatal("expected router to be created, got nil")
	}

	// Voice pipeline should NOT be created without ASR and TTS
	if router.voicePipeline != nil {
		t.Error("expected voicePipeline to be nil without ASR and TTS")
	}
}

// TestCreateAgent_InvalidConfig tests agent creation with invalid config
func TestCreateAgent_InvalidConfig(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	asrService := &mockASRService{}
	ttsService := &mockTTSService{}
	idGenerator := newMockIDGenerator()

	factory := NewAgentFactory(
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		asrService,
		ttsService,
		idGenerator,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	conversationID := "conv_test_invalid"

	// Invalid config with missing required fields
	config := &AgentConfig{
		URL: "", // Empty URL should cause error
	}

	agent, router, err := factory.CreateAgent(config, conversationID)
	if err == nil {
		t.Error("expected error for invalid config, got nil")
	}

	if agent != nil {
		t.Error("expected agent to be nil on error")
	}

	if router != nil {
		t.Error("expected router to be nil on error")
	}
}

// TestCreateAgentWithCallbacks tests agent creation with custom callbacks wrapper
func TestCreateAgentWithCallbacks(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	asrService := &mockASRService{}
	ttsService := &mockTTSService{}
	idGenerator := newMockIDGenerator()

	factory := NewAgentFactory(
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		asrService,
		ttsService,
		idGenerator,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	conversationID := "conv_test_callbacks"
	now := time.Now()
	conversation := &models.Conversation{
		ID:                 conversationID,
		Title:              "Test Conversation",
		Status:             models.ConversationStatusActive,
		LastClientStanzaID: 0,
		LastServerStanzaID: -1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	conversationRepo.Create(context.Background(), conversation)

	config := &AgentConfig{
		URL:           "ws://localhost:7880",
		APIKey:        "test-api-key",
		APISecret:     "test-api-secret",
		AgentIdentity: "agent",
		AgentName:     "test-agent",
	}

	// Custom wrapper that wraps the router
	wrapperCalled := false
	callbacksWrapper := func(router *MessageRouter) ports.LiveKitAgentCallbacks {
		wrapperCalled = true
		return router
	}

	agent, router, err := factory.CreateAgentWithCallbacks(config, conversationID, callbacksWrapper)
	if err != nil {
		t.Fatalf("CreateAgentWithCallbacks failed: %v", err)
	}

	if agent == nil {
		t.Fatal("expected agent to be created, got nil")
	}

	if router == nil {
		t.Fatal("expected router to be created, got nil")
	}

	if !wrapperCalled {
		t.Error("expected callbacks wrapper to be called")
	}
}

// TestCreateAgentWithCallbacks_NilWrapper tests agent creation with nil wrapper
func TestCreateAgentWithCallbacks_NilWrapper(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	asrService := &mockASRService{}
	ttsService := &mockTTSService{}
	idGenerator := newMockIDGenerator()

	factory := NewAgentFactory(
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		asrService,
		ttsService,
		idGenerator,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	conversationID := "conv_test_nil_wrapper"
	now := time.Now()
	conversation := &models.Conversation{
		ID:                 conversationID,
		Title:              "Test Conversation",
		Status:             models.ConversationStatusActive,
		LastClientStanzaID: 0,
		LastServerStanzaID: -1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	conversationRepo.Create(context.Background(), conversation)

	config := &AgentConfig{
		URL:           "ws://localhost:7880",
		APIKey:        "test-api-key",
		APISecret:     "test-api-secret",
		AgentIdentity: "agent",
		AgentName:     "test-agent",
	}

	// Nil wrapper should use default router callbacks
	agent, router, err := factory.CreateAgentWithCallbacks(config, conversationID, nil)
	if err != nil {
		t.Fatalf("CreateAgentWithCallbacks failed: %v", err)
	}

	if agent == nil {
		t.Fatal("expected agent to be created, got nil")
	}

	if router == nil {
		t.Fatal("expected router to be created, got nil")
	}
}

// TestAgentFactory_IntegrationWithAllComponents tests complete integration
func TestAgentFactory_IntegrationWithAllComponents(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	sentenceRepo := &mockSentenceRepo{}
	reasoningStepRepo := &mockReasoningStepRepo{}
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
	commentaryRepo := &mockCommentaryRepo{}
	voteRepo := newMockVoteRepository()
	noteRepo := newMockNoteRepository()
	memoryService := newMockMemoryService()
	optimizationService := newMockOptimizationService()
	handleToolUseCase := &mockHandleToolUseCase{}
	generateResponseUseCase := &mockGenerateResponseUseCase{}
	processUserMessageUseCase := &mockProcessUserMessageUseCase{}
	asrService := &mockASRService{}
	ttsService := &mockTTSService{}
	idGenerator := newMockIDGenerator()

	factory := NewAgentFactory(
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		asrService,
		ttsService,
		idGenerator,
		0.5, // minConfidence
		nil, // sendMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		nil, // editUserMessageUseCase
		nil, // editAssistantMessageUseCase
		nil, // synthesizeSpeechUseCase
	)

	conversationID := "conv_test_integration"
	now := time.Now()
	conversation := &models.Conversation{
		ID:                 conversationID,
		Title:              "Integration Test",
		Status:             models.ConversationStatusActive,
		LastClientStanzaID: 0,
		LastServerStanzaID: -1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	conversationRepo.Create(context.Background(), conversation)

	config := &AgentConfig{
		URL:           "ws://localhost:7880",
		APIKey:        "test-api-key",
		APISecret:     "test-api-secret",
		AgentIdentity: "agent-integration",
		AgentName:     "integration-agent",
	}

	agent, router, err := factory.CreateAgent(config, conversationID)
	if err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	// Verify agent structure
	if agent == nil {
		t.Fatal("expected agent to be created")
	}

	// Verify router has all components wired
	if router == nil {
		t.Fatal("expected router to be created")
	}

	// Verify codec
	if router.codec == nil {
		t.Error("expected codec to be set")
	}

	// Verify dispatcher
	if router.dispatcher == nil {
		t.Error("expected dispatcher to be set")
	}

	// Verify generation manager
	if router.generationManager == nil {
		t.Error("expected generationManager to be set")
	}

	// Verify protocol handler
	if router.protocolHandler == nil {
		t.Error("expected protocolHandler to be set")
	}

	// Verify services
	if router.asrService == nil {
		t.Error("expected asrService to be set")
	}

	if router.ttsService == nil {
		t.Error("expected ttsService to be set")
	}

	if router.idGenerator == nil {
		t.Error("expected idGenerator to be set")
	}

	// Verify voice pipeline created
	if router.voicePipeline == nil {
		t.Error("expected voicePipeline to be created")
	}
}
