package protocol

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

type Envelope struct {
	ConversationID string      `msgpack:"conversationId,omitempty" json:"conversationId,omitempty"`
	Type           MessageType `msgpack:"type" json:"type"`
	Body           any         `msgpack:"body" json:"body"`

	// W3C Trace Context
	TraceID    string `msgpack:"trace_id,omitempty" json:"traceId,omitempty"`    // 32 hex chars
	SpanID     string `msgpack:"span_id,omitempty" json:"spanId,omitempty"`      // 16 hex chars
	TraceFlags byte   `msgpack:"trace_flags,omitempty" json:"traceFlags,omitempty"` // 0x01 = sampled

	// Langfuse session context
	SessionID string `msgpack:"session_id,omitempty" json:"sessionId,omitempty"`
	UserID    string `msgpack:"user_id,omitempty" json:"userId,omitempty"`
}

func (e *Envelope) HasTraceContext() bool {
	return e.TraceID != "" && e.SpanID != ""
}

// Returns W3C traceparent format: 00-{trace_id}-{span_id}-{flags}
func (e *Envelope) TraceParent() string {
	if !e.HasTraceContext() {
		return ""
	}
	return fmt.Sprintf("00-%s-%s-%02x", e.TraceID, e.SpanID, e.TraceFlags)
}

func NewEnvelope(conversationID string, msgType MessageType, body any) *Envelope {
	return &Envelope{
		ConversationID: conversationID,
		Type:           msgType,
		Body:           body,
	}
}

func (e *Envelope) Encode() ([]byte, error) {
	data, err := msgpack.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("encode envelope: %w", err)
	}
	return data, nil
}

func DecodeEnvelope(data []byte) (*Envelope, error) {
	var e Envelope
	if err := msgpack.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	return &e, nil
}

func DecodeBody[T any](e *Envelope) (*T, error) {
	if typed, ok := e.Body.(T); ok {
		return &typed, nil
	}

	// Re-encode and decode to convert map[string]any to struct
	data, err := msgpack.Marshal(e.Body)
	if err != nil {
		return nil, fmt.Errorf("re-encode body: %w", err)
	}

	var result T
	if err := msgpack.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode body to %T: %w", result, err)
	}
	return &result, nil
}
