package protocol

// Envelope wraps all protocol messages with common metadata for routing and ordering.
// Envelopes are serialized using MessagePack and transmitted over LiveKit data channels.
type Envelope struct {
	// StanzaID identifies the message's position in the conversation sequence.
	// Client messages: positive, incrementing (1, 2, 3, ...)
	// Server messages: negative, decrementing (-1, -2, -3, ...)
	StanzaID int32 `msgpack:"stanza_id" json:"stanza_id"`

	// ConversationID is the unique identifier of the conversation.
	// Format: conv_{nanoid} - maps directly to LiveKit room name.
	ConversationID string `msgpack:"conversation_id" json:"conversation_id"`

	// Type is the numeric message type (1-16)
	Type MessageType `msgpack:"type" json:"type"`

	// Meta contains optional metadata including OpenTelemetry tracing fields
	Meta map[string]interface{} `msgpack:"meta,omitempty" json:"meta,omitempty"`

	// Body contains the message-specific payload
	Body interface{} `msgpack:"body" json:"body"`
}

// Common meta keys
const (
	MetaKeyTimestamp     = "timestamp"
	MetaKeyClientVersion = "client_version"
	MetaKeyTraceID       = "messaging.trace_id"
	MetaKeySpanID        = "messaging.span_id"
)

func NewEnvelope(stanzaID int32, conversationID string, msgType MessageType, body interface{}) *Envelope {
	return &Envelope{
		StanzaID:       stanzaID,
		ConversationID: conversationID,
		Type:           msgType,
		Body:           body,
	}
}

func (e *Envelope) WithMeta(key string, value interface{}) *Envelope {
	if e.Meta == nil {
		e.Meta = make(map[string]interface{})
	}
	e.Meta[key] = value
	return e
}

// WithTracing adds OpenTelemetry tracing fields
func (e *Envelope) WithTracing(traceID, spanID string) *Envelope {
	return e.WithMeta(MetaKeyTraceID, traceID).WithMeta(MetaKeySpanID, spanID)
}
