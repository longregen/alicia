package dto

type SubscribeRequest struct {
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
	FromSequence   *int   `json:"from_sequence,omitempty" msgpack:"fromSequence,omitempty"`
	AgentMode      bool   `json:"agent_mode,omitempty" msgpack:"agentMode,omitempty"`
}

type UnsubscribeRequest struct {
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
}

type SubscribeAck struct {
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
	Success        bool   `json:"success" msgpack:"success"`
	Error          string `json:"error,omitempty" msgpack:"error,omitempty"`
	MissedMessages *int   `json:"missed_messages,omitempty" msgpack:"missedMessages,omitempty"`
}

type UnsubscribeAck struct {
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
	Success        bool   `json:"success" msgpack:"success"`
}
