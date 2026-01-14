package dto

// SubscribeRequest represents a request to subscribe to a conversation's messages
type SubscribeRequest struct {
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
	FromSequence   *int   `json:"from_sequence,omitempty" msgpack:"fromSequence,omitempty"`
}

// UnsubscribeRequest represents a request to unsubscribe from a conversation
type UnsubscribeRequest struct {
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
}

// SubscribeAck represents the server's response to a subscription request
type SubscribeAck struct {
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
	Success        bool   `json:"success" msgpack:"success"`
	Error          string `json:"error,omitempty" msgpack:"error,omitempty"`
	MissedMessages *int   `json:"missed_messages,omitempty" msgpack:"missedMessages,omitempty"`
}

// UnsubscribeAck represents the server's response to an unsubscription request
type UnsubscribeAck struct {
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
	Success        bool   `json:"success" msgpack:"success"`
}
