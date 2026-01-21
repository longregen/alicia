package livekit

import (
	"sync"
	"time"

	"github.com/longregen/alicia/pkg/protocol"
)

type BufferedMessage struct {
	StanzaID  int32
	Envelope  *protocol.Envelope
	Timestamp time.Time
}

type MessageBuffer struct {
	buffer  []BufferedMessage
	maxSize int
	mu      sync.RWMutex
}

func NewMessageBuffer(maxSize int) *MessageBuffer {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &MessageBuffer{
		buffer:  make([]BufferedMessage, 0, maxSize),
		maxSize: maxSize,
	}
}

func (b *MessageBuffer) Add(envelope *protocol.Envelope) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buffer = append(b.buffer, BufferedMessage{
		StanzaID:  envelope.StanzaID,
		Envelope:  envelope,
		Timestamp: time.Now(),
	})

	if len(b.buffer) > b.maxSize {
		b.buffer = b.buffer[len(b.buffer)-b.maxSize:]
	}
}

// Server messages use negative stanzaIds (more negative = newer), client messages use positive (larger = newer).
// Only returns messages of the same sign as lastSeen to maintain separation between server/client message streams.
func (b *MessageBuffer) GetMessagesSince(lastSeen int32) []*protocol.Envelope {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []*protocol.Envelope
	for _, msg := range b.buffer {
		if lastSeen < 0 {
			if msg.StanzaID < 0 && msg.StanzaID < lastSeen {
				result = append(result, msg.Envelope)
			}
		} else if lastSeen > 0 {
			if msg.StanzaID > 0 && msg.StanzaID > lastSeen {
				result = append(result, msg.Envelope)
			}
		}
	}
	return result
}

func (b *MessageBuffer) GetLastStanzaID() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.buffer) == 0 {
		return 0
	}

	return b.buffer[len(b.buffer)-1].StanzaID
}

func (b *MessageBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buffer = make([]BufferedMessage, 0, b.maxSize)
}

func (b *MessageBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.buffer)
}
