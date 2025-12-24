package livekit

import (
	"context"
	"errors"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/pkg/protocol"
	"github.com/vmihailenco/msgpack/v5"
)

func TestProtocolHandler_SendEnvelope(t *testing.T) {
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()

	conversationID := "conv_test1"
	conversation := &models.Conversation{
		ID:                 conversationID,
		LastServerStanzaID: -1,
	}
	conversationRepo.Create(context.Background(), conversation)

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		messageRepo,
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		conversationID,
	)

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeAssistantMessage,
		Body: &protocol.AssistantMessage{
			ID:             "msg_1",
			ConversationID: conversationID,
			Content:        "test",
		},
	}

	err := handler.SendEnvelope(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that stanza ID was assigned
	if envelope.StanzaID != -2 {
		t.Errorf("expected stanza ID -2, got %d", envelope.StanzaID)
	}

	// Check that message was sent
	if len(mockSender.sentData) != 1 {
		t.Errorf("expected 1 message sent, got %d", len(mockSender.sentData))
	}

	// Check that message was buffered
	if handler.messageBuffer.Size() != 1 {
		t.Errorf("expected 1 message buffered, got %d", handler.messageBuffer.Size())
	}
}

func TestProtocolHandler_SendEnvelopeWithPresetStanzaID(t *testing.T) {
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()

	conversationID := "conv_test1"
	conversation := &models.Conversation{
		ID:                 conversationID,
		LastServerStanzaID: -5,
	}
	conversationRepo.Create(context.Background(), conversation)

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		conversationID,
	)

	envelope := &protocol.Envelope{
		StanzaID:       -10, // Pre-set stanza ID
		ConversationID: conversationID,
		Type:           protocol.TypeAssistantMessage,
	}

	err := handler.SendEnvelope(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stanza ID should not be changed
	if envelope.StanzaID != -10 {
		t.Errorf("expected stanza ID -10, got %d", envelope.StanzaID)
	}
}

func TestProtocolHandler_HandleConfiguration_FirstConnection(t *testing.T) {
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()

	conversationID := "conv_test1"
	conversation := &models.Conversation{
		ID:                 conversationID,
		LastServerStanzaID: -1,
	}
	conversationRepo.Create(context.Background(), conversation)

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		conversationID,
	)

	config := &protocol.Configuration{
		ConversationID:   conversationID,
		LastSequenceSeen: 0, // First connection
	}

	err := handler.HandleConfiguration(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should send acknowledgement
	if len(mockSender.sentData) != 1 {
		t.Errorf("expected 1 message (acknowledgement), got %d", len(mockSender.sentData))
	}
}

func TestProtocolHandler_HandleConfiguration_ConversationNotFound(t *testing.T) {
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		"conv_nonexistent",
	)

	config := &protocol.Configuration{
		ConversationID:   "conv_nonexistent",
		LastSequenceSeen: 0,
	}

	err := handler.HandleConfiguration(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that an error envelope was sent
	if len(mockSender.sentData) != 1 {
		t.Fatalf("expected 1 message sent, got %d", len(mockSender.sentData))
	}

	var envelope protocol.Envelope
	if err := msgpack.Unmarshal(mockSender.sentData[0], &envelope); err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}

	if envelope.Type != protocol.TypeErrorMessage {
		t.Errorf("expected error message type, got %v", envelope.Type)
	}
}

func TestProtocolHandler_HandleReconnection_GapTooLarge(t *testing.T) {
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()

	conversationID := "conv_test1"
	conversation := &models.Conversation{
		ID:                 conversationID,
		LastServerStanzaID: -2000, // Very far ahead
	}
	conversationRepo.Create(context.Background(), conversation)

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		conversationID,
	)

	config := &protocol.Configuration{
		ConversationID:   conversationID,
		LastSequenceSeen: -1, // Client is far behind
	}

	err := handler.HandleConfiguration(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that an error envelope was sent
	if len(mockSender.sentData) != 1 {
		t.Fatalf("expected 1 message sent, got %d", len(mockSender.sentData))
	}

	var envelope protocol.Envelope
	if err := msgpack.Unmarshal(mockSender.sentData[0], &envelope); err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}

	if envelope.Type != protocol.TypeErrorMessage {
		t.Errorf("expected error message type, got %v", envelope.Type)
	}
}

func TestProtocolHandler_UpdateClientStanzaID(t *testing.T) {
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()

	conversationID := "conv_test1"
	conversation := &models.Conversation{
		ID:                 conversationID,
		LastClientStanzaID: 0,
	}
	conversationRepo.Create(context.Background(), conversation)

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		conversationID,
	)

	handler.UpdateClientStanzaID(context.Background(), 5)

	// Check that stanza ID was updated
	conv, _ := conversationRepo.GetByID(context.Background(), conversationID)
	if conv.LastClientStanzaID != 5 {
		t.Errorf("expected client stanza ID 5, got %d", conv.LastClientStanzaID)
	}
}

func TestProtocolHandler_UpdateClientStanzaID_NegativeIgnored(t *testing.T) {
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()

	conversationID := "conv_test1"
	conversation := &models.Conversation{
		ID:                 conversationID,
		LastClientStanzaID: 5,
	}
	conversationRepo.Create(context.Background(), conversation)

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		conversationID,
	)

	// Negative stanza IDs should be ignored
	handler.UpdateClientStanzaID(context.Background(), -10)

	// Should remain unchanged
	conv, _ := conversationRepo.GetByID(context.Background(), conversationID)
	if conv.LastClientStanzaID != 5 {
		t.Errorf("expected client stanza ID 5, got %d", conv.LastClientStanzaID)
	}
}

func TestProtocolHandler_SendAudio(t *testing.T) {
	mockSender := newMockAgentSender()

	handler := NewProtocolHandler(
		mockSender,
		newMockConversationRepo(),
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		"conv_test1",
	)

	audio := []byte{0x01, 0x02, 0x03}
	format := "pcm"

	err := handler.SendAudio(context.Background(), audio, format)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockSender.sentAudio) != 1 {
		t.Errorf("expected 1 audio message, got %d", len(mockSender.sentAudio))
	}

	if string(mockSender.sentAudio[0].data) != string(audio) {
		t.Error("audio data mismatch")
	}

	if mockSender.sentAudio[0].format != format {
		t.Errorf("expected format %s, got %s", format, mockSender.sentAudio[0].format)
	}
}

func TestProtocolHandler_SendAudio_NilAgent(t *testing.T) {
	handler := &ProtocolHandler{
		agent: nil, // No agent
	}

	err := handler.SendAudio(context.Background(), []byte{}, "pcm")
	if err == nil {
		t.Fatal("expected error when agent is nil")
	}
}

func TestProtocolHandler_SendToolUseRequest(t *testing.T) {
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()

	conversationID := "conv_test1"
	conversation := &models.Conversation{
		ID:                 conversationID,
		LastServerStanzaID: -1,
	}
	conversationRepo.Create(context.Background(), conversation)

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		conversationID,
	)

	toolUse := &models.ToolUse{
		ID:             "tu_1",
		MessageID:      "msg_1",
		ToolName:       "calculator",
		Arguments:      map[string]any{"expr": "2+2"},
		SequenceNumber: 1,
	}

	err := handler.SendToolUseRequest(context.Background(), toolUse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockSender.sentData) != 1 {
		t.Errorf("expected 1 message, got %d", len(mockSender.sentData))
	}
}

func TestProtocolHandler_SendToolUseResult(t *testing.T) {
	mockSender := newMockAgentSender()
	conversationRepo := newMockConversationRepo()

	conversationID := "conv_test1"
	conversation := &models.Conversation{
		ID:                 conversationID,
		LastServerStanzaID: -1,
	}
	conversationRepo.Create(context.Background(), conversation)

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		conversationID,
	)

	toolUse := &models.ToolUse{
		ID:             "tu_1",
		MessageID:      "msg_1",
		Status:         models.ToolStatusSuccess,
		Result:         map[string]any{"answer": 4},
		SequenceNumber: 1,
	}

	err := handler.SendToolUseResult(context.Background(), toolUse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockSender.sentData) != 1 {
		t.Errorf("expected 1 message, got %d", len(mockSender.sentData))
	}
}

func TestAbsInt32(t *testing.T) {
	tests := []struct {
		input    int32
		expected int32
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-1, 1},
		{1, 1},
	}

	for _, tt := range tests {
		result := absInt32(tt.input)
		if result != tt.expected {
			t.Errorf("absInt32(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

// Test error handling for SendEnvelope with failing sender
func TestProtocolHandler_SendEnvelope_SendError(t *testing.T) {
	mockSender := &mockAgentSender{}
	conversationRepo := newMockConversationRepo()

	conversationID := "conv_test1"
	conversation := &models.Conversation{
		ID:                 conversationID,
		LastServerStanzaID: -1,
	}
	conversationRepo.Create(context.Background(), conversation)

	// Override SendData to return error
	mockSender.sendDataFunc = func(ctx context.Context, data []byte) error {
		return errors.New("send failed")
	}

	handler := NewProtocolHandler(
		mockSender,
		conversationRepo,
		newMockMessageRepo(),
		&mockSentenceRepo{},
		&mockReasoningStepRepo{},
		&mockToolUseRepo{},
		&mockMemoryUsageRepo{},
		&mockCommentaryRepo{},
		conversationID,
	)

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeAssistantMessage,
	}

	err := handler.SendEnvelope(context.Background(), envelope)
	if err == nil {
		t.Fatal("expected error from SendData failure")
	}
}
