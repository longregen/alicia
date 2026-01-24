package main

import "context"

type Message struct {
	ID             string
	ConversationID string
	PreviousID     string
	BranchIndex    int16
	Role           string
	Content        string
	Reasoning      string
	Status         string // pending, streaming, completed, error
	ToolUses       []ToolUse
	Memories       []Memory // memories retrieved for this message
}

type Memory struct {
	ID            string
	Content       string
	Importance    float32
	Pinned        bool
	Archived      bool
	SourceMsgID   *string
	Tags          []string
	DeletedReason *string
	Similarity    float32 // computed during search
}

type Tool struct {
	ID          string
	Name        string
	Description string
	Schema      map[string]any
}

type ToolUse struct {
	ID        string
	ToolName  string
	Arguments map[string]any
	Result    any
	Success   bool
	Error     string
}

type LLMMessage struct {
	Role       string
	Content    string
	ToolCalls  []LLMToolCall
	ToolCallID string
}

type LLMToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

type LLMResponse struct {
	Content   string
	Reasoning string
	ToolCalls []LLMToolCall
}

type ResponseGenerationRequest struct {
	ID             string  `msgpack:"id"`
	MessageID      string  `msgpack:"messageId"`
	ConversationID string  `msgpack:"conversationId"`
	RequestType    string  `msgpack:"requestType"`
	NewContent     string  `msgpack:"newContent,omitempty"`
	EnableTools    bool    `msgpack:"enableTools"`
	UsePareto      bool    `msgpack:"usePareto"`
	PreviousID     string  `msgpack:"previousId,omitempty"`
	Timestamp      float64 `msgpack:"timestamp,omitempty"` // float64 for JS compatibility
}

type GenerateConfig struct {
	MaxToolIterations int
	EnableTools       bool
	ParetoMode        bool
}

type Notifier interface {
	SetMessageID(id string)
	SetPreviousID(id string)
	SendThinking(ctx context.Context, messageID, text string)
	SendThinkingWithProgress(ctx context.Context, messageID, text string, progress float32)
	SendToolStart(ctx context.Context, id, name string, args map[string]any)
	SendToolComplete(ctx context.Context, id string, success bool, result any, errMsg string)
	SendComplete(ctx context.Context, messageID, content string)
	SendError(ctx context.Context, messageID string, err error)
	SendTitleUpdate(ctx context.Context, title string)
	SendMemoryTrace(ctx context.Context, messageID, memoryID, content string, relevance float32)
}
