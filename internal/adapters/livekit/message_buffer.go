package livekit

import (
	"sync"
	"time"

	"github.com/longregen/alicia/pkg/protocol"
)

// BufferedMessage represents a message stored for potential replay
type BufferedMessage struct {
	StanzaID  int32
	Envelope  *protocol.Envelope
	Timestamp time.Time
}

// MessageBuffer stores recent messages for reconnection replay
type MessageBuffer struct {
	buffer  []BufferedMessage
	maxSize int
	mu      sync.RWMutex
}

// NewMessageBuffer creates a new message buffer with the specified max size
func NewMessageBuffer(maxSize int) *MessageBuffer {
	if maxSize <= 0 {
		maxSize = 100 // Default buffer size
	}
	return &MessageBuffer{
		buffer:  make([]BufferedMessage, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add stores a message in the buffer for potential replay
func (b *MessageBuffer) Add(envelope *protocol.Envelope) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Add message to buffer
	b.buffer = append(b.buffer, BufferedMessage{
		StanzaID:  envelope.StanzaID,
		Envelope:  envelope,
		Timestamp: time.Now(),
	})

	// Trim to max size (keep most recent messages)
	if len(b.buffer) > b.maxSize {
		b.buffer = b.buffer[len(b.buffer)-b.maxSize:]
	}
}

// GetMessagesSince retrieves all messages after the given lastSeen stanzaId
// For server messages (negative stanzaIds), it returns messages with stanzaId < lastSeen (more negative)
// For client messages (positive stanzaIds), it returns messages with stanzaId > lastSeen
// Only returns messages of the same sign as lastSeen (server or client messages, not both)
func (b *MessageBuffer) GetMessagesSince(lastSeen int32) []*protocol.Envelope {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []*protocol.Envelope
	for _, msg := range b.buffer {
		// Only return messages of the same type (sign) as lastSeen
		// Server messages are negative, client messages are positive
		if lastSeen < 0 {
			// Looking for server messages after lastSeen (more negative)
			if msg.StanzaID < 0 && msg.StanzaID < lastSeen {
				result = append(result, msg.Envelope)
			}
		} else if lastSeen > 0 {
			// Looking for client messages after lastSeen (larger positive)
			if msg.StanzaID > 0 && msg.StanzaID > lastSeen {
				result = append(result, msg.Envelope)
			}
		}
		// If lastSeen == 0, return nothing (no valid reference point)
	}
	return result
}

// GetLastStanzaID returns the last (most recent) stanzaID in the buffer
func (b *MessageBuffer) GetLastStanzaID() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.buffer) == 0 {
		return 0
	}

	// Return the last message's stanzaID
	return b.buffer[len(b.buffer)-1].StanzaID
}

// Clear removes all messages from the buffer
func (b *MessageBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buffer = make([]BufferedMessage, 0, b.maxSize)
}

// Size returns the current number of messages in the buffer
func (b *MessageBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.buffer)
}
