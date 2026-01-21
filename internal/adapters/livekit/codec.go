package livekit

import (
	"fmt"

	"github.com/longregen/alicia/pkg/protocol"
	"github.com/vmihailenco/msgpack/v5"
)

type Codec struct{}

func NewCodec() *Codec {
	return &Codec{}
}

type messageFactory func() interface{}

var messageTypeRegistry = map[protocol.MessageType]messageFactory{
	protocol.TypeErrorMessage:      func() interface{} { return &protocol.ErrorMessage{} },
	protocol.TypeUserMessage:       func() interface{} { return &protocol.UserMessage{} },
	protocol.TypeAssistantMessage:  func() interface{} { return &protocol.AssistantMessage{} },
	protocol.TypeAudioChunk:        func() interface{} { return &protocol.AudioChunk{} },
	protocol.TypeReasoningStep:     func() interface{} { return &protocol.ReasoningStep{} },
	protocol.TypeToolUseRequest:    func() interface{} { return &protocol.ToolUseRequest{} },
	protocol.TypeToolUseResult:     func() interface{} { return &protocol.ToolUseResult{} },
	protocol.TypeAcknowledgement:   func() interface{} { return &protocol.Acknowledgement{} },
	protocol.TypeTranscription:     func() interface{} { return &protocol.Transcription{} },
	protocol.TypeControlStop:       func() interface{} { return &protocol.ControlStop{} },
	protocol.TypeControlVariation:  func() interface{} { return &protocol.ControlVariation{} },
	protocol.TypeConfiguration:     func() interface{} { return &protocol.Configuration{} },
	protocol.TypeStartAnswer:       func() interface{} { return &protocol.StartAnswer{} },
	protocol.TypeMemoryTrace:       func() interface{} { return &protocol.MemoryTrace{} },
	protocol.TypeCommentary:        func() interface{} { return &protocol.Commentary{} },
	protocol.TypeAssistantSentence: func() interface{} { return &protocol.AssistantSentence{} },
	protocol.TypeFeedback:             func() interface{} { return &protocol.Feedback{} },
	protocol.TypeFeedbackConfirmation: func() interface{} { return &protocol.FeedbackConfirmation{} },
	protocol.TypeUserNote:             func() interface{} { return &protocol.UserNote{} },
	protocol.TypeNoteConfirmation:     func() interface{} { return &protocol.NoteConfirmation{} },
	protocol.TypeMemoryAction:         func() interface{} { return &protocol.MemoryAction{} },
	protocol.TypeMemoryConfirmation:   func() interface{} { return &protocol.MemoryConfirmation{} },
	protocol.TypeServerInfo:   func() interface{} { return &protocol.ServerInfo{} },
	protocol.TypeSessionStats: func() interface{} { return &protocol.SessionStats{} },
}

func (c *Codec) Encode(envelope *protocol.Envelope) ([]byte, error) {
	if envelope == nil {
		return nil, fmt.Errorf("envelope is nil")
	}

	if _, ok := messageTypeRegistry[envelope.Type]; !ok {
		return nil, fmt.Errorf("invalid message type: %d", envelope.Type)
	}

	if envelope.Body == nil {
		return nil, fmt.Errorf("envelope body is nil")
	}

	data, err := msgpack.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal envelope: %w", err)
	}

	return data, nil
}

func (c *Codec) Decode(data []byte) (*protocol.Envelope, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	var tempEnv struct {
		StanzaID       int32                  `msgpack:"stanzaId"`
		ConversationID string                 `msgpack:"conversationId"`
		Type           protocol.MessageType   `msgpack:"type"`
		Meta           map[string]interface{} `msgpack:"meta,omitempty"`
		Body           msgpack.RawMessage     `msgpack:"body"`
	}

	if err := msgpack.Unmarshal(data, &tempEnv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal envelope: %w", err)
	}

	factory, ok := messageTypeRegistry[tempEnv.Type]
	if !ok {
		return nil, fmt.Errorf("unknown message type: %d", tempEnv.Type)
	}

	body := factory()
	if err := msgpack.Unmarshal(tempEnv.Body, body); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message body (type %s): %w", tempEnv.Type.String(), err)
	}

	envelope := &protocol.Envelope{
		StanzaID:       tempEnv.StanzaID,
		ConversationID: tempEnv.ConversationID,
		Type:           tempEnv.Type,
		Meta:           tempEnv.Meta,
		Body:           body,
	}

	return envelope, nil
}

func (c *Codec) EncodeMessage(stanzaID int32, conversationID string, msgType protocol.MessageType, body interface{}) ([]byte, error) {
	envelope := protocol.NewEnvelope(stanzaID, conversationID, msgType, body)
	return c.Encode(envelope)
}
