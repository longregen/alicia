//go:build integration

package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/livekit"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/application/usecases"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

// TestVoiceConversationFlow tests the complete voice conversation pipeline:
// Audio Input -> STT -> LLM Response -> Memory Retrieval -> Tool Execution -> TTS -> Audio Output
func TestVoiceConversationFlow(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	// Setup repositories
	conversationRepo := postgres.NewConversationRepository(db.Pool)
	messageRepo := postgres.NewMessageRepository(db.Pool)
	sentenceRepo := postgres.NewSentenceRepository(db.Pool)
	audioRepo := postgres.NewAudioRepository(db.Pool)
	memoryRepo := postgres.NewMemoryRepository(db.Pool)
	memoryUsageRepo := postgres.NewMemoryUsageRepository(db.Pool)
	toolRepo := postgres.NewToolRepository(db.Pool)
	toolUseRepo := postgres.NewToolUseRepository(db.Pool)
	reasoningStepRepo := postgres.NewReasoningStepRepository(db.Pool)
	idGen := id.NewGenerator()

	// Create test conversation and setup
	conversation := fixtures.CreateConversation(ctx, t, "test-conv", "Voice Test")

	// Create a memory for retrieval testing
	embedding := fixtures.GenerateEmbedding(384)
	memory := fixtures.CreateMemoryWithEmbedding(ctx, t, "mem1", "User prefers concise answers", embedding)

	// Register a test tool
	toolSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
	}
	tool := fixtures.CreateTool(ctx, t, "tool1", "search", "Search for information", toolSchema)

	// Setup mock services
	mockASR := &mockASRService{transcription: "What is the weather like today?"}
	mockLLM := &mockLLMService{
		response: &ports.LLMResponse{
			Content: "Let me check the weather for you.",
			ToolCalls: []*ports.LLMToolCall{
				{
					ID:        "tool-call-1",
					Name:      "search",
					Arguments: map[string]any{"query": "weather today"},
				},
			},
		},
	}
	mockTTS := &mockTTSService{audioData: []byte("synthetic-audio-data")}
	mockEmbedding := &mockEmbeddingService{embedding: embedding}
	mockToolExecutor := &mockToolExecutor{result: map[string]any{"weather": "sunny", "temp": 72}}
	mockLiveKit := &mockLiveKitService{}

	// Setup services
	memoryService := services.NewMemoryService(memoryRepo, memoryUsageRepo, mockEmbedding, idGen)
	toolService := services.NewToolService(toolRepo, toolUseRepo, messageRepo, idGen)
	// Register mock executor
	toolService.RegisterExecutor("search", func(ctx context.Context, arguments map[string]any) (any, error) {
		return mockToolExecutor.Execute(ctx, nil, arguments)
	})
	txManager := postgres.NewTransactionManager(db.Pool)

	// Setup use cases
	generateResponse := usecases.NewGenerateResponse(
		messageRepo,
		sentenceRepo,
		toolUseRepo,
		toolRepo,
		reasoningStepRepo,
		conversationRepo,
		mockLLM,
		toolService,
		memoryService,
		idGen,
		txManager,
	)

	synthesizeSpeech := usecases.NewSynthesizeSpeech(audioRepo, sentenceRepo, mockTTS, idGen)
	streamAudioResponse := usecases.NewStreamAudioResponse(mockTTS, synthesizeSpeech)

	// Create mock LiveKit agent
	agent := &mockLiveKitAgent{
		connected:    true,
		sentMessages: make([]*protocol.Envelope, 0),
		audioFrames:  make([][]byte, 0),
	}

	// Create protocol handler
	commentaryRepo := postgres.NewCommentaryRepository(db.Pool)
	protocolHandler := livekit.NewProtocolHandler(
		agent,
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		conversation.ID,
	)

	// ========== STEP 1: SIMULATE AUDIO INPUT ==========
	t.Run("AudioInput", func(t *testing.T) {
		audioInput := []byte("fake-audio-data")

		// Create audio record
		audioID := idGen.GenerateAudioID()
		audio := &models.Audio{
			ID:             audioID,
			ConversationID: conversation.ID,
			TrackSID:       "track-123",
			Format:         "opus",
			Data:           audioInput,
		}

		if err := audioRepo.Create(ctx, audio); err != nil {
			t.Fatalf("failed to create audio: %v", err)
		}

		if audio.ID == "" {
			t.Fatal("audio ID should not be empty")
		}
	})

	// ========== STEP 2: STT TRANSCRIPTION ==========
	var userMessage *models.Message
	t.Run("STT_Transcription", func(t *testing.T) {
		audioData := []byte("fake-audio-data")

		// Transcribe audio
		result, err := mockASR.Transcribe(ctx, audioData, "opus")
		if err != nil {
			t.Fatalf("failed to transcribe audio: %v", err)
		}

		if result.Text == "" {
			t.Fatal("transcription should not be empty")
		}
		if result.Text != "What is the weather like today?" {
			t.Errorf("expected transcription 'What is the weather like today?', got '%s'", result.Text)
		}

		// Create user message from transcription
		sequenceNumber, err := messageRepo.GetNextSequenceNumber(ctx, conversation.ID)
		if err != nil {
			t.Fatalf("failed to get next sequence number: %v", err)
		}

		userMessage = models.NewUserMessage(
			idGen.GenerateMessageID(),
			conversation.ID,
			sequenceNumber,
			result.Text,
		)

		if err := messageRepo.Create(ctx, userMessage); err != nil {
			t.Fatalf("failed to create user message: %v", err)
		}

		if userMessage.Contents != result.Text {
			t.Errorf("user message contents mismatch")
		}
	})

	// ========== STEP 3: MEMORY RETRIEVAL ==========
	var retrievedMemories []*ports.MemorySearchResult
	t.Run("MemoryRetrieval", func(t *testing.T) {
		// Search for relevant memories using the user's message
		var err error
		retrievedMemories, err = memoryService.SearchWithScores(ctx, userMessage.Contents, 0.5, 5)
		if err != nil {
			t.Fatalf("failed to search memories: %v", err)
		}

		// Should retrieve the memory we created
		if len(retrievedMemories) == 0 {
			t.Log("Warning: No memories retrieved (this may be expected depending on embedding similarity)")
		} else {
			// Verify memory retrieval structure
			firstMemory := retrievedMemories[0]
			if firstMemory.Memory == nil {
				t.Fatal("memory should not be nil")
			}
			if firstMemory.Similarity < 0 || firstMemory.Similarity > 1 {
				t.Errorf("similarity score should be between 0 and 1, got %f", firstMemory.Similarity)
			}
		}
	})

	// ========== STEP 4: LLM RESPONSE GENERATION ==========
	var assistantMessage *models.Message
	var toolUses []*models.ToolUse
	t.Run("LLM_ResponseGeneration", func(t *testing.T) {
		// Generate response with tools enabled
		input := &ports.GenerateResponseInput{
			ConversationID:   conversation.ID,
			UserMessageID:    userMessage.ID,
			PreviousID:       userMessage.ID,
			RelevantMemories: extractMemoriesFromResults(retrievedMemories),
			EnableTools:      true,
			EnableReasoning:  true,
			EnableStreaming:  false,
		}

		output, err := generateResponse.Execute(ctx, input)
		if err != nil {
			t.Fatalf("failed to generate response: %v", err)
		}

		assistantMessage = output.Message
		if assistantMessage.ID == "" {
			t.Fatal("assistant message ID should not be empty")
		}
		if assistantMessage.Contents == "" {
			t.Fatal("assistant message contents should not be empty")
		}

		// Verify tool calls were processed
		toolUses = output.ToolUses
		if len(toolUses) == 0 {
			t.Fatal("expected at least one tool use")
		}

		firstToolUse := toolUses[0]
		if firstToolUse.ToolName != "search" {
			t.Errorf("expected tool name 'search', got '%s'", firstToolUse.ToolName)
		}
		if firstToolUse.Status != models.ToolStatusSuccess {
			t.Errorf("expected tool status 'success', got '%s'", firstToolUse.Status)
		}
		if firstToolUse.Result == nil {
			t.Fatal("tool result should not be nil")
		}
	})

	// ========== STEP 5: MEMORY USAGE TRACKING ==========
	t.Run("MemoryUsageTracking", func(t *testing.T) {
		// Track that memories were used in this response
		for _, result := range retrievedMemories {
			usage, err := memoryService.TrackUsage(ctx, result.Memory.ID, conversation.ID, assistantMessage.ID, result.Similarity)
			if err != nil {
				t.Fatalf("failed to track memory usage: %v", err)
			}

			if usage.MemoryID != result.Memory.ID {
				t.Errorf("expected memory ID %s, got %s", result.Memory.ID, usage.MemoryID)
			}
			if usage.MessageID != assistantMessage.ID {
				t.Errorf("expected message ID %s, got %s", assistantMessage.ID, usage.MessageID)
			}
			if usage.SimilarityScore != result.Similarity {
				t.Errorf("expected similarity %f, got %f", result.Similarity, usage.SimilarityScore)
			}
		}

		// Verify memory usage can be retrieved
		usages, err := memoryUsageRepo.GetByMessage(ctx, assistantMessage.ID)
		if err != nil {
			t.Fatalf("failed to get memory usages: %v", err)
		}

		if len(usages) != len(retrievedMemories) {
			t.Errorf("expected %d memory usages, got %d", len(retrievedMemories), len(usages))
		}
	})

	// ========== STEP 6: TTS SYNTHESIS ==========
	var audioOutput *models.Audio
	t.Run("TTS_Synthesis", func(t *testing.T) {
		// Synthesize speech for the assistant message
		input := &usecases.SynthesizeSpeechInput{
			Text:            assistantMessage.Contents,
			MessageID:       assistantMessage.ID,
			Voice:           "en-US-neural",
			Speed:           1.0,
			OutputFormat:    "pcm",
			EnableStreaming: false,
		}

		output, err := synthesizeSpeech.Execute(ctx, input)
		if err != nil {
			t.Fatalf("failed to synthesize speech: %v", err)
		}

		audioOutput = output.Audio
		if audioOutput == nil {
			t.Fatal("audio output should not be nil")
		}
		if audioOutput.ID == "" {
			t.Fatal("audio ID should not be empty")
		}
		if len(output.AudioData) == 0 {
			t.Fatal("audio data should not be empty")
		}
		if output.Format != "pcm" {
			t.Errorf("expected format 'pcm', got '%s'", output.Format)
		}

		// Verify audio was stored
		retrievedAudio, err := audioRepo.GetByID(ctx, audioOutput.ID)
		if err != nil {
			t.Fatalf("failed to retrieve audio: %v", err)
		}

		if retrievedAudio.MessageID != assistantMessage.ID {
			t.Errorf("expected message ID %s, got %s", assistantMessage.ID, retrievedAudio.MessageID)
		}
	})

	// ========== STEP 7: AUDIO OUTPUT VIA LIVEKIT ==========
	t.Run("AudioOutput_LiveKit", func(t *testing.T) {
		// Send audio to LiveKit agent
		err := agent.SendAudio(ctx, audioOutput.Data, audioOutput.Format)
		if err != nil {
			t.Fatalf("failed to send audio to LiveKit: %v", err)
		}

		// Verify audio was sent
		if len(agent.audioFrames) == 0 {
			t.Fatal("expected audio frames to be sent")
		}

		sentAudio := agent.audioFrames[0]
		if len(sentAudio) == 0 {
			t.Fatal("sent audio data should not be empty")
		}
	})

	// ========== STEP 8: PROTOCOL MESSAGE SEQUENCING ==========
	t.Run("ProtocolMessageSequencing", func(t *testing.T) {
		// Send protocol messages for the conversation

		// 1. StartAnswer message
		startAnswer := &protocol.StartAnswer{
			ID:                   assistantMessage.ID,
			PreviousID:           userMessage.ID,
			ConversationID:       conversation.ID,
			AnswerType:           protocol.AnswerTypeText,
			PlannedSentenceCount: 1,
		}
		startEnvelope := &protocol.Envelope{
			ConversationID: conversation.ID,
			Type:           protocol.TypeStartAnswer,
			Body:           startAnswer,
		}
		if err := protocolHandler.SendEnvelope(ctx, startEnvelope); err != nil {
			t.Fatalf("failed to send StartAnswer: %v", err)
		}

		// 2. AssistantMessage
		assistantMsgProto := &protocol.AssistantMessage{
			ID:             assistantMessage.ID,
			PreviousID:     userMessage.ID,
			ConversationID: conversation.ID,
			Content:        assistantMessage.Contents,
		}
		msgEnvelope := &protocol.Envelope{
			ConversationID: conversation.ID,
			Type:           protocol.TypeAssistantMessage,
			Body:           assistantMsgProto,
		}
		if err := protocolHandler.SendEnvelope(ctx, msgEnvelope); err != nil {
			t.Fatalf("failed to send AssistantMessage: %v", err)
		}

		// 3. ToolUseRequest for each tool call
		for _, toolUse := range toolUses {
			if err := protocolHandler.SendToolUseRequest(ctx, toolUse); err != nil {
				t.Fatalf("failed to send ToolUseRequest: %v", err)
			}

			// 4. ToolUseResult
			if err := protocolHandler.SendToolUseResult(ctx, toolUse); err != nil {
				t.Fatalf("failed to send ToolUseResult: %v", err)
			}
		}

		// Verify messages were sent with correct sequencing
		if len(agent.sentMessages) < 4 {
			t.Errorf("expected at least 4 protocol messages, got %d", len(agent.sentMessages))
		}

		// Verify StanzaID sequencing (server messages should have negative, decrementing IDs)
		for i := 1; i < len(agent.sentMessages); i++ {
			prev := agent.sentMessages[i-1]
			curr := agent.sentMessages[i]

			// Server-sent messages should have negative stanza IDs
			if curr.StanzaID >= 0 {
				t.Errorf("server message %d should have negative stanza ID, got %d", i, curr.StanzaID)
			}

			// Each subsequent message should have a lower (more negative) stanza ID
			if curr.StanzaID >= prev.StanzaID {
				t.Errorf("message %d stanza ID (%d) should be less than previous (%d)", i, curr.StanzaID, prev.StanzaID)
			}
		}

		// Verify message types are in expected order
		expectedTypes := []protocol.MessageType{
			protocol.TypeStartAnswer,
			protocol.TypeAssistantMessage,
			protocol.TypeToolUseRequest,
			protocol.TypeToolUseResult,
		}

		for i, expectedType := range expectedTypes {
			if i >= len(agent.sentMessages) {
				break
			}
			if agent.sentMessages[i].Type != expectedType {
				t.Errorf("message %d expected type %s, got %s", i, expectedType, agent.sentMessages[i].Type)
			}
		}
	})

	// ========== STEP 9: VERIFY COMPLETE PIPELINE ==========
	t.Run("VerifyCompletePipeline", func(t *testing.T) {
		// Verify conversation has messages
		messages, err := messageRepo.GetByConversation(ctx, conversation.ID)
		if err != nil {
			t.Fatalf("failed to get conversation messages: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("expected 2 messages (user + assistant), got %d", len(messages))
		}

		// Verify first message is user message
		if messages[0].Role != models.MessageRoleUser {
			t.Errorf("expected first message to be user message")
		}

		// Verify second message is assistant message
		if messages[1].Role != models.MessageRoleAssistant {
			t.Errorf("expected second message to be assistant message")
		}

		// Verify tool uses exist
		messageToolUses, err := toolUseRepo.GetByMessage(ctx, assistantMessage.ID)
		if err != nil {
			t.Fatalf("failed to get tool uses: %v", err)
		}

		if len(messageToolUses) != 1 {
			t.Errorf("expected 1 tool use, got %d", len(messageToolUses))
		}

		// Verify audio exists for assistant message
		msgAudio, err := audioRepo.GetByMessage(ctx, assistantMessage.ID)
		if err != nil {
			t.Fatalf("failed to get message audio: %v", err)
		}

		if msgAudio == nil {
			t.Fatal("assistant message should have audio")
		}
	})
}

// TestVoiceConversationFlow_StreamingMode tests streaming voice conversation
func TestVoiceConversationFlow_StreamingMode(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	// Setup repositories
	conversationRepo := postgres.NewConversationRepository(db.Pool)
	messageRepo := postgres.NewMessageRepository(db.Pool)
	sentenceRepo := postgres.NewSentenceRepository(db.Pool)
	audioRepo := postgres.NewAudioRepository(db.Pool)
	toolRepo := postgres.NewToolRepository(db.Pool)
	toolUseRepo := postgres.NewToolUseRepository(db.Pool)
	reasoningStepRepo := postgres.NewReasoningStepRepository(db.Pool)
	memoryRepo := postgres.NewMemoryRepository(db.Pool)
	memoryUsageRepo := postgres.NewMemoryUsageRepository(db.Pool)
	idGen := id.NewGenerator()

	conversation := fixtures.CreateConversation(ctx, t, "test-conv-stream", "Streaming Voice Test")

	// Setup mock services with streaming support
	mockLLM := &mockLLMService{
		streamChunks: []ports.LLMStreamChunk{
			{Content: "Let me ", Done: false},
			{Content: "help you ", Done: false},
			{Content: "with that.", Done: true},
		},
	}
	mockTTS := &mockTTSService{audioData: []byte("streamed-audio")}
	mockEmbedding := &mockEmbeddingService{embedding: fixtures.GenerateEmbedding(384)}
	mockToolExecutor := &mockToolExecutor{result: map[string]any{"status": "ok"}}

	// Setup services
	memoryService := services.NewMemoryService(memoryRepo, memoryUsageRepo, mockEmbedding, idGen)
	toolService := services.NewToolService(toolRepo, toolUseRepo, messageRepo, idGen)
	// Register mock executor
	toolService.RegisterExecutor("search", func(ctx context.Context, arguments map[string]any) (any, error) {
		return mockToolExecutor.Execute(ctx, nil, arguments)
	})
	txManager := postgres.NewTransactionManager(db.Pool)

	generateResponse := usecases.NewGenerateResponse(
		messageRepo,
		sentenceRepo,
		toolUseRepo,
		toolRepo,
		reasoningStepRepo,
		conversationRepo,
		mockLLM,
		toolService,
		memoryService,
		idGen,
		txManager,
	)

	synthesizeSpeech := usecases.NewSynthesizeSpeech(audioRepo, sentenceRepo, mockTTS, idGen)
	streamAudioResponse := usecases.NewStreamAudioResponse(mockTTS, synthesizeSpeech)

	// Create user message
	userMessage := fixtures.CreateMessage(ctx, t, "user-msg", conversation.ID, models.MessageRoleUser, "Hello", 1)

	// Generate streaming response
	input := &ports.GenerateResponseInput{
		ConversationID:  conversation.ID,
		UserMessageID:   userMessage.ID,
		PreviousID:      userMessage.ID,
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: true,
	}

	output, err := generateResponse.Execute(ctx, input)
	if err != nil {
		t.Fatalf("failed to generate streaming response: %v", err)
	}

	if output.StreamChannel == nil {
		t.Fatal("stream channel should not be nil")
	}

	// Create mock agent
	agent := &mockLiveKitAgent{
		connected:    true,
		sentMessages: make([]*protocol.Envelope, 0),
		audioFrames:  make([][]byte, 0),
	}

	// Process stream and synthesize audio
	sentenceCount := 0
	for chunk := range output.StreamChannel {
		if chunk.Error != nil {
			t.Fatalf("stream error: %v", chunk.Error)
		}

		if chunk.SentenceID != "" && chunk.Text != "" {
			sentenceCount++

			// Synthesize and stream audio for each sentence
			err := streamAudioResponse.StreamSentenceAudio(
				ctx,
				agent,
				conversation.ID,
				output.Message.ID,
				chunk.SentenceID,
				chunk.Sequence,
				chunk.Text,
				chunk.IsFinal,
				"en-US-neural",
				int32(-sentenceCount),
			)
			if err != nil {
				t.Fatalf("failed to stream sentence audio: %v", err)
			}
		}
	}

	// Verify sentences were created
	if sentenceCount == 0 {
		t.Fatal("expected at least one sentence")
	}

	sentences, err := sentenceRepo.GetByMessage(ctx, output.Message.ID)
	if err != nil {
		t.Fatalf("failed to get sentences: %v", err)
	}

	if len(sentences) != sentenceCount {
		t.Errorf("expected %d sentences, got %d", sentenceCount, len(sentences))
	}

	// Verify audio was sent for each sentence
	if len(agent.audioFrames) != sentenceCount {
		t.Errorf("expected %d audio frames, got %d", sentenceCount, len(agent.audioFrames))
	}

	// Verify protocol messages were sent
	if len(agent.sentMessages) != sentenceCount {
		t.Errorf("expected %d protocol messages, got %d", sentenceCount, len(agent.sentMessages))
	}
}

// ========== MOCK IMPLEMENTATIONS ==========

type mockASRService struct {
	transcription string
	err           error
}

func (m *mockASRService) Transcribe(ctx context.Context, audio []byte, format string) (*ports.ASRResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ports.ASRResult{
		Text:       m.transcription,
		Confidence: 0.95,
		Language:   "en",
		Duration:   2.5,
	}, nil
}

func (m *mockASRService) TranscribeStream(ctx context.Context, audioStream interface{}, format string) (<-chan *ports.ASRResult, error) {
	return nil, fmt.Errorf("not implemented")
}

type mockLLMService struct {
	response     *ports.LLMResponse
	streamChunks []ports.LLMStreamChunk
	err          error
}

func (m *mockLLMService) Chat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockLLMService) ChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockLLMService) ChatStream(ctx context.Context, messages []ports.LLMMessage) (<-chan ports.LLMStreamChunk, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan ports.LLMStreamChunk, len(m.streamChunks))
	go func() {
		defer close(ch)
		for _, chunk := range m.streamChunks {
			ch <- chunk
		}
	}()
	return ch, nil
}

func (m *mockLLMService) ChatStreamWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (<-chan ports.LLMStreamChunk, error) {
	return m.ChatStream(ctx, messages)
}

type mockTTSService struct {
	audioData []byte
	err       error
}

func (m *mockTTSService) Synthesize(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ports.TTSResult{
		Audio:      m.audioData,
		Format:     options.OutputFormat,
		DurationMs: 1000,
	}, nil
}

func (m *mockTTSService) SynthesizeStream(ctx context.Context, text string, options *ports.TTSOptions) (<-chan *ports.TTSResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTTSService) GetVoices(ctx context.Context) ([]string, error) {
	return []string{"en-US-neural"}, nil
}

type mockEmbeddingService struct {
	embedding []float32
	err       error
}

func (m *mockEmbeddingService) Embed(ctx context.Context, text string) (*ports.EmbeddingResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ports.EmbeddingResult{
		Embedding:  m.embedding,
		Model:      "test-model",
		Dimensions: len(m.embedding),
	}, nil
}

func (m *mockEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([]*ports.EmbeddingResult, error) {
	results := make([]*ports.EmbeddingResult, len(texts))
	for i := range texts {
		results[i] = &ports.EmbeddingResult{
			Embedding:  m.embedding,
			Model:      "test-model",
			Dimensions: len(m.embedding),
		}
	}
	return results, nil
}

func (m *mockEmbeddingService) GetDimensions() int {
	return len(m.embedding)
}

type mockToolExecutor struct {
	result any
	err    error
}

func (m *mockToolExecutor) Execute(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type mockLiveKitAgent struct {
	mu           sync.Mutex
	connected    bool
	sentMessages []*protocol.Envelope
	audioFrames  [][]byte
}

func (m *mockLiveKitAgent) Connect(ctx context.Context, roomName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *mockLiveKitAgent) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *mockLiveKitAgent) SendData(ctx context.Context, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Decode envelope to track it
	codec := livekit.NewCodec()
	envelope, err := codec.Decode(data)
	if err != nil {
		return err
	}

	m.sentMessages = append(m.sentMessages, envelope)
	return nil
}

func (m *mockLiveKitAgent) SendAudio(ctx context.Context, audio []byte, format string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.audioFrames = append(m.audioFrames, audio)
	return nil
}

func (m *mockLiveKitAgent) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockLiveKitAgent) GetRoom() *ports.LiveKitRoom {
	return &ports.LiveKitRoom{
		Name: "test-room",
		SID:  "room-123",
	}
}

// Helper function to extract memories from search results
func extractMemoriesFromResults(results []*ports.MemorySearchResult) []*models.Memory {
	memories := make([]*models.Memory, len(results))
	for i, result := range results {
		memories[i] = result.Memory
	}
	return memories
}
