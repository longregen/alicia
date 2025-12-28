package livekit

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

// TestNewDefaultMessageDispatcher tests dispatcher creation
func TestNewDefaultMessageDispatcher(t *testing.T) {
	protocolHandler := newMockProtocolHandler()
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	toolUseRepo := &mockToolUseRepo{}
	memoryUsageRepo := &mockMemoryUsageRepo{}
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
	generationManager := newMockResponseGenerationManager()
	conversationID := "conv_test1"

	dispatcher := NewDefaultMessageDispatcher(
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
		ttsService,
		idGenerator,
		generationManager,
	)

	if dispatcher == nil {
		t.Fatal("expected dispatcher to be created, got nil")
	}

	if dispatcher.conversationID != conversationID {
		t.Errorf("expected conversationID %s, got %s", conversationID, dispatcher.conversationID)
	}

	if dispatcher.protocolHandler != protocolHandler {
		t.Error("expected protocolHandler to be set")
	}

	if dispatcher.messageRepo != messageRepo {
		t.Error("expected messageRepo to be set")
	}

	if dispatcher.voteRepo != voteRepo {
		t.Error("expected voteRepo to be set")
	}

	if dispatcher.noteRepo != noteRepo {
		t.Error("expected noteRepo to be set")
	}

	if dispatcher.memoryService != memoryService {
		t.Error("expected memoryService to be set")
	}

	if dispatcher.optimizationService != optimizationService {
		t.Error("expected optimizationService to be set")
	}
}

// TestDispatchMessage_UnknownType tests handling of unknown message types
func TestDispatchMessage_UnknownType(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	envelope := &protocol.Envelope{
		Type: protocol.MessageType(9999), // Unknown type
		Body: &protocol.UserMessage{},
	}

	// Should not return an error, just logs and ignores
	err := dispatcher.DispatchMessage(context.Background(), envelope)
	if err != nil {
		t.Errorf("expected nil error for unknown type, got %v", err)
	}
}

// TestDispatchMessage_ServerOnlyMessages tests that server-only message types are ignored
func TestDispatchMessage_ServerOnlyMessages(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	serverOnlyTypes := []protocol.MessageType{
		protocol.TypeAssistantMessage,
		protocol.TypeReasoningStep,
		protocol.TypeStartAnswer,
		protocol.TypeMemoryTrace,
		protocol.TypeAssistantSentence,
		protocol.TypeFeedbackConfirmation,
		protocol.TypeNoteConfirmation,
		protocol.TypeMemoryConfirmation,
		protocol.TypeServerInfo,
		protocol.TypeSessionStats,
		protocol.TypeEliteOptions,
	}

	for _, msgType := range serverOnlyTypes {
		t.Run(msgType.String(), func(t *testing.T) {
			envelope := &protocol.Envelope{
				Type: msgType,
				Body: &protocol.UserMessage{}, // Placeholder body
			}

			// These should be silently ignored (no error)
			err := dispatcher.DispatchMessage(context.Background(), envelope)
			if err != nil {
				t.Errorf("expected nil error for server-only type %s, got %v", msgType, err)
			}
		})
	}
}

// TestDispatchMessage_ErrorMessage tests error message routing
func TestDispatchMessage_ErrorMessage(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	envelope := &protocol.Envelope{
		Type: protocol.TypeErrorMessage,
		Body: &protocol.ErrorMessage{
			ID:             "err_1",
			ConversationID: "conv_test1",
			Code:           500,
			Message:        "Test error",
			Severity:       protocol.SeverityError,
		},
	}

	err := dispatcher.DispatchMessage(context.Background(), envelope)
	if err != nil {
		t.Errorf("expected nil error for error message, got %v", err)
	}
}

// TestDispatchMessage_Acknowledgement tests acknowledgement message routing
func TestDispatchMessage_Acknowledgement(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	envelope := &protocol.Envelope{
		Type: protocol.TypeAcknowledgement,
		Body: &protocol.Acknowledgement{
			ConversationID: "conv_test1",
			AckedStanzaID:  5,
			Success:        true,
		},
	}

	err := dispatcher.DispatchMessage(context.Background(), envelope)
	if err != nil {
		t.Errorf("expected nil error for acknowledgement, got %v", err)
	}
}

// TestDispatchMessage_AllClientMessages tests all valid client→server message types
func TestDispatchMessage_AllClientMessages(t *testing.T) {
	tests := []struct {
		name        string
		messageType protocol.MessageType
		body        any
		expectError bool
	}{
		{
			name:        "TypeErrorMessage",
			messageType: protocol.TypeErrorMessage,
			body: &protocol.ErrorMessage{
				ConversationID: "conv_test1",
				Code:           100,
				Message:        "test",
			},
			expectError: false,
		},
		{
			name:        "TypeUserMessage",
			messageType: protocol.TypeUserMessage,
			body: &protocol.UserMessage{
				ID:             "user_1",
				ConversationID: "conv_test1",
				Content:        "hello",
			},
			expectError: false,
		},
		{
			name:        "TypeAudioChunk",
			messageType: protocol.TypeAudioChunk,
			body: &protocol.AudioChunk{
				ConversationID: "conv_test1",
				Data:           []byte{0x01, 0x02},
				Format:         "pcm",
			},
			expectError: false,
		},
		{
			name:        "TypeToolUseRequest",
			messageType: protocol.TypeToolUseRequest,
			body: &protocol.ToolUseRequest{
				ID:             "tu_1",
				MessageID:      "msg_1",
				ConversationID: "conv_test1",
				ToolName:       "test",
				Parameters:     map[string]any{},
			},
			expectError: false,
		},
		{
			name:        "TypeToolUseResult",
			messageType: protocol.TypeToolUseResult,
			body: &protocol.ToolUseResult{
				ID:             "tur_1",
				RequestID:      "tu_1",
				ConversationID: "conv_test1",
				Success:        true,
			},
			expectError: false,
		},
		{
			name:        "TypeAcknowledgement",
			messageType: protocol.TypeAcknowledgement,
			body: &protocol.Acknowledgement{
				ConversationID: "conv_test1",
				AckedStanzaID:  1,
				Success:        true,
			},
			expectError: false,
		},
		{
			name:        "TypeTranscription",
			messageType: protocol.TypeTranscription,
			body: &protocol.Transcription{
				ID:             "trans_1",
				ConversationID: "conv_test1",
				Text:           "test",
				Final:          true,
			},
			expectError: false,
		},
		{
			name:        "TypeControlStop",
			messageType: protocol.TypeControlStop,
			body: &protocol.ControlStop{
				ConversationID: "conv_test1",
				StopType:       protocol.StopTypeGeneration,
				TargetID:       "msg_1",
			},
			expectError: false,
		},
		{
			name:        "TypeControlVariation",
			messageType: protocol.TypeControlVariation,
			body: &protocol.ControlVariation{
				ConversationID: "conv_test1",
				TargetID:       "msg_1",
				Mode:           protocol.VariationTypeRegenerate,
			},
			expectError: false,
		},
		{
			name:        "TypeConfiguration",
			messageType: protocol.TypeConfiguration,
			body: &protocol.Configuration{
				ConversationID:   "conv_test1",
				LastSequenceSeen: 0,
			},
			expectError: false,
		},
		{
			name:        "TypeCommentary",
			messageType: protocol.TypeCommentary,
			body: &protocol.Commentary{
				ID:             "comm_1",
				MessageID:      "msg_1",
				ConversationID: "conv_test1",
				Content:        "test",
			},
			expectError: false,
		},
		{
			name:        "TypeFeedback",
			messageType: protocol.TypeFeedback,
			body: &protocol.Feedback{
				ID:         "fb_1",
				TargetType: "message",
				TargetID:   "msg_1",
				Vote:       "up",
			},
			expectError: false,
		},
		{
			name:        "TypeUserNote",
			messageType: protocol.TypeUserNote,
			body: &protocol.UserNote{
				MessageID: "msg_1",
				Content:   "note",
				Action:    "create",
			},
			expectError: false,
		},
		{
			name:        "TypeMemoryAction",
			messageType: protocol.TypeMemoryAction,
			body: &protocol.MemoryAction{
				Action: "create",
				Memory: &protocol.MemoryData{
					Content: "test",
				},
			},
			expectError: false,
		},
		{
			name:        "TypeDimensionPreference",
			messageType: protocol.TypeDimensionPreference,
			body: &protocol.DimensionPreference{
				ConversationID: "conv_test1",
				Weights:        protocol.DimensionWeights{},
			},
			expectError: false,
		},
		{
			name:        "TypeEliteSelect",
			messageType: protocol.TypeEliteSelect,
			body: &protocol.EliteSelect{
				ConversationID: "conv_test1",
				EliteID:        "elite_1",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher, _, _ := createTestDispatcher()

			envelope := &protocol.Envelope{
				Type: tt.messageType,
				Body: tt.body,
			}

			err := dispatcher.DispatchMessage(context.Background(), envelope)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

// TestSendError tests the sendError helper method
func TestSendError(t *testing.T) {
	dispatcher, _, _ := createTestDispatcher()

	err := dispatcher.sendError(context.Background(), protocol.ErrCodeMalformedData, "test error", true)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Note: sendError calls protocolHandler.SendError, not SendEnvelope directly
	// So we need to verify through the protocol handler's implementation
}

// Mock TTS service for testing
type mockTTSService struct {
	synthesizeFunc func(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error)
}

func (m *mockTTSService) Synthesize(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, text, options)
	}
	return &ports.TTSResult{
		Audio:      []byte("mock audio data"),
		Format:     "audio/pcm",
		DurationMs: 1000,
	}, nil
}

func (m *mockTTSService) SynthesizeStream(ctx context.Context, text string, options *ports.TTSOptions) (<-chan *ports.TTSResult, error) {
	ch := make(chan *ports.TTSResult)
	close(ch)
	return ch, nil
}

func (m *mockProtocolHandler) UpdateClientStanzaID(ctx context.Context, stanzaID int32) error {
	return nil
}
