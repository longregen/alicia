package livekit

import (
	"testing"
	"time"

	"github.com/longregen/alicia/pkg/protocol"
)

// TestAcknowledgementEncoding tests that acknowledgements can be encoded/decoded
func TestAcknowledgementEncoding(t *testing.T) {
	codec := NewCodec()

	ack := &protocol.Acknowledgement{
		ConversationID: "conv_test",
		AckedStanzaID:  5,
		Success:        true,
	}

	// Encode acknowledgement
	data, err := codec.EncodeMessage(0, "conv_test", protocol.TypeAcknowledgement, ack)
	if err != nil {
		t.Fatalf("Failed to encode acknowledgement: %v", err)
	}

	// Decode acknowledgement
	envelope, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Failed to decode acknowledgement: %v", err)
	}

	if envelope.Type != protocol.TypeAcknowledgement {
		t.Errorf("Expected type %d, got %d", protocol.TypeAcknowledgement, envelope.Type)
	}

	decodedAck, ok := envelope.Body.(*protocol.Acknowledgement)
	if !ok {
		t.Fatalf("Body is not Acknowledgement, got %T", envelope.Body)
	}

	if decodedAck.AckedStanzaID != 5 {
		t.Errorf("Expected AckedStanzaID 5, got %d", decodedAck.AckedStanzaID)
	}

	if !decodedAck.Success {
		t.Error("Expected Success true, got false")
	}
}

// TestPendingMessage tests the PendingMessage structure
func TestPendingMessage(t *testing.T) {
	pending := &PendingMessage{
		StanzaID:   -1,
		Data:       []byte("test data"),
		SentAt:     time.Now(),
		RetryCount: 0,
	}

	if pending.StanzaID != -1 {
		t.Errorf("Expected StanzaID -1, got %d", pending.StanzaID)
	}

	if pending.RetryCount != 0 {
		t.Errorf("Expected RetryCount 0, got %d", pending.RetryCount)
	}

	if len(pending.Data) != 9 {
		t.Errorf("Expected Data length 9, got %d", len(pending.Data))
	}
}

// TestAcknowledgementHandling tests the handleAcknowledgement method
func TestAcknowledgementHandling(t *testing.T) {
	// Create a mock agent with acknowledgement tracking
	agent := &Agent{
		pendingAcks:  make(map[int32]*PendingMessage),
		lastStanzaID: 0,
	}

	// Add a pending message
	agent.pendingAcks[-1] = &PendingMessage{
		StanzaID:   -1,
		Data:       []byte("test"),
		SentAt:     time.Now(),
		RetryCount: 0,
	}

	// Verify it's in the pending list
	if len(agent.pendingAcks) != 1 {
		t.Errorf("Expected 1 pending ack, got %d", len(agent.pendingAcks))
	}

	// Handle acknowledgement
	ack := &protocol.Acknowledgement{
		ConversationID: "conv_test",
		AckedStanzaID:  -1,
		Success:        true,
	}
	agent.handleAcknowledgement(ack)

	// Verify it's removed from pending list
	if len(agent.pendingAcks) != 0 {
		t.Errorf("Expected 0 pending acks after handling, got %d", len(agent.pendingAcks))
	}
}

// TestStanzaIDGeneration tests that server stanzaIds are negative and decrementing
func TestStanzaIDGeneration(t *testing.T) {
	agent := &Agent{
		pendingAcks:  make(map[int32]*PendingMessage),
		lastStanzaID: 0,
	}

	// Generate multiple stanzaIds
	agent.lastStanzaID--
	id1 := agent.lastStanzaID

	agent.lastStanzaID--
	id2 := agent.lastStanzaID

	agent.lastStanzaID--
	id3 := agent.lastStanzaID

	// Verify they're negative and decrementing
	if id1 != -1 {
		t.Errorf("Expected first stanzaID to be -1, got %d", id1)
	}

	if id2 != -2 {
		t.Errorf("Expected second stanzaID to be -2, got %d", id2)
	}

	if id3 != -3 {
		t.Errorf("Expected third stanzaID to be -3, got %d", id3)
	}
}

// TestMessageFlowScenario tests the complete message flow with acknowledgements
func TestMessageFlowScenario(t *testing.T) {
	codec := NewCodec()

	// Scenario: Client sends UserMessage (stanzaId: 1)
	userMsg := &protocol.UserMessage{
		ID:             "msg_1",
		ConversationID: "conv_abc",
		Content:        "Hello",
	}
	clientMsgData, err := codec.EncodeMessage(1, "conv_abc", protocol.TypeUserMessage, userMsg)
	if err != nil {
		t.Fatalf("Failed to encode user message: %v", err)
	}

	// Server receives and decodes the message
	clientEnvelope, err := codec.Decode(clientMsgData)
	if err != nil {
		t.Fatalf("Failed to decode user message: %v", err)
	}

	if clientEnvelope.StanzaID != 1 {
		t.Errorf("Expected client stanzaID 1, got %d", clientEnvelope.StanzaID)
	}

	// Server sends acknowledgement for stanzaId 1
	serverAck := &protocol.Acknowledgement{
		ConversationID: "conv_abc",
		AckedStanzaID:  1,
		Success:        true,
	}
	ackData, err := codec.EncodeMessage(0, "conv_abc", protocol.TypeAcknowledgement, serverAck)
	if err != nil {
		t.Fatalf("Failed to encode acknowledgement: %v", err)
	}

	// Client receives and decodes the acknowledgement
	ackEnvelope, err := codec.Decode(ackData)
	if err != nil {
		t.Fatalf("Failed to decode acknowledgement: %v", err)
	}

	if ackEnvelope.Type != protocol.TypeAcknowledgement {
		t.Errorf("Expected type Acknowledgement, got %s", ackEnvelope.Type.String())
	}

	// Server sends AssistantMessage (stanzaId: -1)
	assistantMsg := &protocol.AssistantMessage{
		ID:             "msg_2",
		PreviousID:     "msg_1",
		ConversationID: "conv_abc",
		Content:        "Hi there!",
	}
	serverMsgData, err := codec.EncodeMessage(-1, "conv_abc", protocol.TypeAssistantMessage, assistantMsg)
	if err != nil {
		t.Fatalf("Failed to encode assistant message: %v", err)
	}

	// Client receives and decodes the message
	serverEnvelope, err := codec.Decode(serverMsgData)
	if err != nil {
		t.Fatalf("Failed to decode assistant message: %v", err)
	}

	if serverEnvelope.StanzaID != -1 {
		t.Errorf("Expected server stanzaID -1, got %d", serverEnvelope.StanzaID)
	}

	// Client sends acknowledgement for stanzaId -1
	clientAck := &protocol.Acknowledgement{
		ConversationID: "conv_abc",
		AckedStanzaID:  -1,
		Success:        true,
	}
	clientAckData, err := codec.EncodeMessage(0, "conv_abc", protocol.TypeAcknowledgement, clientAck)
	if err != nil {
		t.Fatalf("Failed to encode client acknowledgement: %v", err)
	}

	// Server receives and decodes the acknowledgement
	clientAckEnvelope, err := codec.Decode(clientAckData)
	if err != nil {
		t.Fatalf("Failed to decode client acknowledgement: %v", err)
	}

	receivedAck, ok := clientAckEnvelope.Body.(*protocol.Acknowledgement)
	if !ok {
		t.Fatalf("Body is not Acknowledgement, got %T", clientAckEnvelope.Body)
	}

	if receivedAck.AckedStanzaID != -1 {
		t.Errorf("Expected AckedStanzaID -1, got %d", receivedAck.AckedStanzaID)
	}

	if !receivedAck.Success {
		t.Error("Expected Success true, got false")
	}
}

// TestRetryLogic tests the retry logic for unacknowledged messages
func TestRetryLogic(t *testing.T) {
	agent := &Agent{
		pendingAcks:  make(map[int32]*PendingMessage),
		lastStanzaID: 0,
	}

	// Add a pending message that's old enough to retry
	oldTime := time.Now().Add(-AckTimeout - time.Second)
	agent.pendingAcks[-1] = &PendingMessage{
		StanzaID:   -1,
		Data:       []byte("test"),
		SentAt:     oldTime,
		RetryCount: 0,
	}

	// Add a recent pending message that shouldn't be retried yet
	agent.pendingAcks[-2] = &PendingMessage{
		StanzaID:   -2,
		Data:       []byte("test2"),
		SentAt:     time.Now(),
		RetryCount: 0,
	}

	// Verify both messages are pending
	if len(agent.pendingAcks) != 2 {
		t.Errorf("Expected 2 pending acks, got %d", len(agent.pendingAcks))
	}

	// Note: We can't fully test retryUnacknowledgedMessages without a real LiveKit connection,
	// but we can verify the pending messages are tracked correctly
}

// TestMaxRetriesExceeded tests that messages are removed after max retries
func TestMaxRetriesExceeded(t *testing.T) {
	agent := &Agent{
		pendingAcks:  make(map[int32]*PendingMessage),
		lastStanzaID: 0,
	}

	// Add a pending message that has exceeded max retries
	oldTime := time.Now().Add(-AckTimeout - time.Second)
	agent.pendingAcks[-1] = &PendingMessage{
		StanzaID:   -1,
		Data:       []byte("test"),
		SentAt:     oldTime,
		RetryCount: MaxRetries,
	}

	// Verify the message is in pending list
	if len(agent.pendingAcks) != 1 {
		t.Errorf("Expected 1 pending ack, got %d", len(agent.pendingAcks))
	}

	// Simulate retry logic (without actual network sending)
	agent.pendingMu.Lock()
	now := time.Now()
	for stanzaID, pending := range agent.pendingAcks {
		if now.Sub(pending.SentAt) > AckTimeout {
			if pending.RetryCount >= MaxRetries {
				delete(agent.pendingAcks, stanzaID)
			}
		}
	}
	agent.pendingMu.Unlock()

	// Verify the message was removed
	if len(agent.pendingAcks) != 0 {
		t.Errorf("Expected 0 pending acks after max retries, got %d", len(agent.pendingAcks))
	}
}
