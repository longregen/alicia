package livekit

import (
	"testing"
	"time"

	"github.com/longregen/alicia/pkg/protocol"
)

func TestMessageBuffer_GetMessagesSince_ServerMessages(t *testing.T) {
	buffer := NewMessageBuffer(10)

	// Add some server messages (negative stanzaIDs)
	buffer.Add(&protocol.Envelope{StanzaID: -1, Type: protocol.TypeAssistantMessage})
	buffer.Add(&protocol.Envelope{StanzaID: -2, Type: protocol.TypeAssistantMessage})
	buffer.Add(&protocol.Envelope{StanzaID: -3, Type: protocol.TypeAssistantMessage})
	buffer.Add(&protocol.Envelope{StanzaID: -4, Type: protocol.TypeAssistantMessage})

	// Get messages after stanzaID -2 (should return -3 and -4)
	messages := buffer.GetMessagesSince(-2)

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].StanzaID != -3 || messages[1].StanzaID != -4 {
		t.Errorf("expected stanzaIDs -3 and -4, got %d and %d", messages[0].StanzaID, messages[1].StanzaID)
	}
}

func TestMessageBuffer_GetMessagesSince_ClientMessages(t *testing.T) {
	buffer := NewMessageBuffer(10)

	// Add some client messages (positive stanzaIDs)
	buffer.Add(&protocol.Envelope{StanzaID: 1, Type: protocol.TypeUserMessage})
	buffer.Add(&protocol.Envelope{StanzaID: 2, Type: protocol.TypeUserMessage})
	buffer.Add(&protocol.Envelope{StanzaID: 3, Type: protocol.TypeUserMessage})
	buffer.Add(&protocol.Envelope{StanzaID: 4, Type: protocol.TypeUserMessage})

	// Get messages after stanzaID 2 (should return 3 and 4)
	messages := buffer.GetMessagesSince(2)

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].StanzaID != 3 || messages[1].StanzaID != 4 {
		t.Errorf("expected stanzaIDs 3 and 4, got %d and %d", messages[0].StanzaID, messages[1].StanzaID)
	}
}

func TestMessageBuffer_GetMessagesSince_EmptyResult(t *testing.T) {
	buffer := NewMessageBuffer(10)

	buffer.Add(&protocol.Envelope{StanzaID: -1, Type: protocol.TypeAssistantMessage})
	buffer.Add(&protocol.Envelope{StanzaID: -2, Type: protocol.TypeAssistantMessage})

	// Request messages after the latest
	messages := buffer.GetMessagesSince(-2)

	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestMessageBuffer_GetLastStanzaID(t *testing.T) {
	buffer := NewMessageBuffer(10)

	// Empty buffer
	if buffer.GetLastStanzaID() != 0 {
		t.Errorf("expected 0 for empty buffer, got %d", buffer.GetLastStanzaID())
	}

	// Add messages
	buffer.Add(&protocol.Envelope{StanzaID: -1, Type: protocol.TypeAssistantMessage})
	buffer.Add(&protocol.Envelope{StanzaID: -2, Type: protocol.TypeAssistantMessage})
	buffer.Add(&protocol.Envelope{StanzaID: -3, Type: protocol.TypeAssistantMessage})

	if buffer.GetLastStanzaID() != -3 {
		t.Errorf("expected -3, got %d", buffer.GetLastStanzaID())
	}
}

func TestMessageBuffer_Clear(t *testing.T) {
	buffer := NewMessageBuffer(10)

	buffer.Add(&protocol.Envelope{StanzaID: -1, Type: protocol.TypeAssistantMessage})
	buffer.Add(&protocol.Envelope{StanzaID: -2, Type: protocol.TypeAssistantMessage})

	if buffer.Size() != 2 {
		t.Errorf("expected size 2, got %d", buffer.Size())
	}

	buffer.Clear()

	if buffer.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", buffer.Size())
	}

	if buffer.GetLastStanzaID() != 0 {
		t.Errorf("expected last stanza ID 0 after clear, got %d", buffer.GetLastStanzaID())
	}
}

func TestMessageBuffer_Size(t *testing.T) {
	buffer := NewMessageBuffer(10)

	if buffer.Size() != 0 {
		t.Errorf("expected size 0 for new buffer, got %d", buffer.Size())
	}

	buffer.Add(&protocol.Envelope{StanzaID: -1, Type: protocol.TypeAssistantMessage})
	if buffer.Size() != 1 {
		t.Errorf("expected size 1, got %d", buffer.Size())
	}

	buffer.Add(&protocol.Envelope{StanzaID: -2, Type: protocol.TypeAssistantMessage})
	if buffer.Size() != 2 {
		t.Errorf("expected size 2, got %d", buffer.Size())
	}
}

func TestMessageBuffer_MaxSizeLimit(t *testing.T) {
	maxSize := 5
	buffer := NewMessageBuffer(maxSize)

	// Add more messages than max size
	for i := 0; i < 10; i++ {
		buffer.Add(&protocol.Envelope{StanzaID: int32(-i - 1), Type: protocol.TypeAssistantMessage})
	}

	// Should only keep the most recent maxSize messages
	if buffer.Size() != maxSize {
		t.Errorf("expected size %d, got %d", maxSize, buffer.Size())
	}

	// Last message should be -10 (most recent)
	if buffer.GetLastStanzaID() != -10 {
		t.Errorf("expected last stanza ID -10, got %d", buffer.GetLastStanzaID())
	}
}

func TestMessageBuffer_ZeroMaxSize(t *testing.T) {
	// Should use default size
	buffer := NewMessageBuffer(0)

	buffer.Add(&protocol.Envelope{StanzaID: -1, Type: protocol.TypeAssistantMessage})

	if buffer.Size() != 1 {
		t.Errorf("expected size 1, got %d", buffer.Size())
	}
}

func TestMessageBuffer_Timestamp(t *testing.T) {
	buffer := NewMessageBuffer(10)

	before := time.Now()
	buffer.Add(&protocol.Envelope{StanzaID: -1, Type: protocol.TypeAssistantMessage})
	after := time.Now()

	// Access internal buffer to check timestamp (we know the structure from reading the code)
	buffer.mu.RLock()
	if len(buffer.buffer) != 1 {
		t.Fatal("expected 1 message in buffer")
	}
	timestamp := buffer.buffer[0].Timestamp
	buffer.mu.RUnlock()

	if timestamp.Before(before) || timestamp.After(after) {
		t.Errorf("timestamp %v is outside expected range [%v, %v]", timestamp, before, after)
	}
}

func TestMessageBuffer_MixedStanzaIDs(t *testing.T) {
	buffer := NewMessageBuffer(10)

	// Add mixed server (-) and client (+) messages
	buffer.Add(&protocol.Envelope{StanzaID: 1, Type: protocol.TypeUserMessage})
	buffer.Add(&protocol.Envelope{StanzaID: -1, Type: protocol.TypeAssistantMessage})
	buffer.Add(&protocol.Envelope{StanzaID: 2, Type: protocol.TypeUserMessage})
	buffer.Add(&protocol.Envelope{StanzaID: -2, Type: protocol.TypeAssistantMessage})

	// Get server messages after -1 (should return -2)
	serverMsgs := buffer.GetMessagesSince(-1)
	if len(serverMsgs) != 1 || serverMsgs[0].StanzaID != -2 {
		t.Errorf("expected 1 server message with ID -2, got %d messages", len(serverMsgs))
	}

	// Get client messages after 1 (should return 2)
	clientMsgs := buffer.GetMessagesSince(1)
	if len(clientMsgs) != 1 || clientMsgs[0].StanzaID != 2 {
		t.Errorf("expected 1 client message with ID 2, got %d messages", len(clientMsgs))
	}
}
