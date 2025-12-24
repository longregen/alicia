package protocol

// ErrorMessage (Type 1) conveys errors and exceptional conditions
type ErrorMessage struct {
	ID             string   `msgpack:"id" json:"id"`
	ConversationID string   `msgpack:"conversationId" json:"conversationId"`
	Code           int32    `msgpack:"code" json:"code"`
	Message        string   `msgpack:"message" json:"message"`
	Severity       Severity `msgpack:"severity" json:"severity"`
	Recoverable    bool     `msgpack:"recoverable" json:"recoverable"`
	OriginatingID  string   `msgpack:"originatingId,omitempty" json:"originatingId,omitempty"`
}

// Error codes
const (
	// Format and protocol errors (100-199)
	ErrCodeMalformedData = 101
	ErrCodeUnknownType   = 102

	// Conversation errors (200-299)
	ErrCodeConversationNotFound = 201
	ErrCodeInvalidState         = 202

	// Tool errors (300-399)
	ErrCodeToolNotFound = 301
	ErrCodeToolTimeout  = 304

	// Server errors (500-599)
	ErrCodeInternalError      = 501
	ErrCodeServiceUnavailable = 503
	ErrCodeQueueOverflow      = 504
)

// UserMessage (Type 2) carries a user's input message
type UserMessage struct {
	ID             string `msgpack:"id" json:"id"`
	PreviousID     string `msgpack:"previousId,omitempty" json:"previousId,omitempty"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Content        string `msgpack:"content" json:"content"`
	Timestamp      int64  `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
}

// AssistantMessage (Type 3) conveys a complete assistant response (non-streaming)
type AssistantMessage struct {
	ID             string `msgpack:"id" json:"id"`
	PreviousID     string `msgpack:"previousId,omitempty" json:"previousId,omitempty"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Content        string `msgpack:"content" json:"content"`
	Timestamp      int64  `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
}

// AudioChunk (Type 4) represents raw audio data segment
type AudioChunk struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Format         string `msgpack:"format" json:"format"` // e.g., "audio/opus"
	Sequence       int32  `msgpack:"sequence" json:"sequence"`
	DurationMs     int32  `msgpack:"durationMs" json:"durationMs"`
	TrackSID       string `msgpack:"trackSid,omitempty" json:"trackSid,omitempty"`
	Data           []byte `msgpack:"data,omitempty" json:"data,omitempty"`
	IsLast         bool   `msgpack:"isLast,omitempty" json:"isLast,omitempty"`
	Timestamp      uint64 `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
}

// ReasoningStep (Type 5) represents internal reasoning trace
type ReasoningStep struct {
	ID             string `msgpack:"id" json:"id"`
	MessageID      string `msgpack:"messageId" json:"messageId"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Sequence       int32  `msgpack:"sequence" json:"sequence"`
	Content        string `msgpack:"content" json:"content"`
}

// ToolUseRequest (Type 6) represents a request to execute a tool
type ToolUseRequest struct {
	ID             string                 `msgpack:"id" json:"id"`
	MessageID      string                 `msgpack:"messageId" json:"messageId"` // ID of the message that triggered this tool call
	ConversationID string                 `msgpack:"conversationId" json:"conversationId"`
	ToolName       string                 `msgpack:"toolName" json:"toolName"`
	Parameters     map[string]interface{} `msgpack:"parameters" json:"parameters"`
	Execution      ToolExecution          `msgpack:"execution" json:"execution"`
	TimeoutMs      int32                  `msgpack:"timeoutMs,omitempty" json:"timeoutMs,omitempty"`
}

// DefaultToolTimeout is the default timeout for tool execution (30 seconds)
const DefaultToolTimeout = 30000

// ToolUseResult (Type 7) represents a tool execution result
type ToolUseResult struct {
	ID             string      `msgpack:"id" json:"id"`
	RequestID      string      `msgpack:"requestId" json:"requestId"`
	ConversationID string      `msgpack:"conversationId" json:"conversationId"`
	Success        bool        `msgpack:"success" json:"success"`
	Result         interface{} `msgpack:"result,omitempty" json:"result,omitempty"`
	ErrorCode      string      `msgpack:"errorCode,omitempty" json:"errorCode,omitempty"`
	ErrorMessage   string      `msgpack:"errorMessage,omitempty" json:"errorMessage,omitempty"`
}

// Acknowledgement (Type 8) confirms receipt of a message
type Acknowledgement struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	AckedStanzaID  int32  `msgpack:"acknowledgedStanzaId" json:"acknowledgedStanzaId"`
	Success        bool   `msgpack:"success" json:"success"`
}

// Transcription (Type 9) represents speech-to-text output
type Transcription struct {
	ID             string  `msgpack:"id" json:"id"`
	PreviousID     string  `msgpack:"previousId,omitempty" json:"previousId,omitempty"`
	ConversationID string  `msgpack:"conversationId" json:"conversationId"`
	Text           string  `msgpack:"text" json:"text"`
	Final          bool    `msgpack:"final" json:"final"`
	Confidence     float32 `msgpack:"confidence,omitempty" json:"confidence,omitempty"`
	Language       string  `msgpack:"language,omitempty" json:"language,omitempty"`
}

// ControlStop (Type 10) halts the assistant's current action
type ControlStop struct {
	ConversationID string   `msgpack:"conversationId" json:"conversationId"`
	TargetID       string   `msgpack:"targetId,omitempty" json:"targetId,omitempty"`
	Reason         string   `msgpack:"reason,omitempty" json:"reason,omitempty"`
	StopType       StopType `msgpack:"stopType,omitempty" json:"stopType,omitempty"`
}

// ControlVariation (Type 11) requests a variation of a previous message
type ControlVariation struct {
	ConversationID string        `msgpack:"conversationId" json:"conversationId"`
	TargetID       string        `msgpack:"targetId" json:"targetId"`
	Mode           VariationType `msgpack:"mode" json:"mode"`
	NewContent     string        `msgpack:"newContent,omitempty" json:"newContent,omitempty"`
}

// Configuration (Type 12) initializes and configures the connection
type Configuration struct {
	ConversationID    string   `msgpack:"conversationId,omitempty" json:"conversationId,omitempty"`
	LastSequenceSeen  int32    `msgpack:"lastSequenceSeen,omitempty" json:"lastSequenceSeen,omitempty"`
	ClientVersion     string   `msgpack:"clientVersion,omitempty" json:"clientVersion,omitempty"`
	PreferredLanguage string   `msgpack:"preferredLanguage,omitempty" json:"preferredLanguage,omitempty"`
	Device            string   `msgpack:"device,omitempty" json:"device,omitempty"`
	Features          []string `msgpack:"features,omitempty" json:"features,omitempty"`
}

// Common features for Configuration
const (
	FeatureStreaming        = "streaming"
	FeaturePartialResponses = "partial_responses"
	FeatureAudioOutput      = "audio_output"
	FeatureReasoningSteps   = "reasoning_steps"
	FeatureToolUse          = "tool_use"
)

// StartAnswer (Type 13) initiates a streaming assistant response
type StartAnswer struct {
	ID                   string     `msgpack:"id" json:"id"`
	PreviousID           string     `msgpack:"previousId" json:"previousId"`
	ConversationID       string     `msgpack:"conversationId" json:"conversationId"`
	AnswerType           AnswerType `msgpack:"answerType,omitempty" json:"answerType,omitempty"`
	PlannedSentenceCount int32      `msgpack:"plannedSentenceCount,omitempty" json:"plannedSentenceCount,omitempty"`
}

// MemoryTrace (Type 14) logs memory retrieval events
type MemoryTrace struct {
	ID             string  `msgpack:"id" json:"id"`
	MessageID      string  `msgpack:"messageId" json:"messageId"`
	ConversationID string  `msgpack:"conversationId" json:"conversationId"`
	MemoryID       string  `msgpack:"memoryId" json:"memoryId"`
	Content        string  `msgpack:"content" json:"content"`
	Relevance      float32 `msgpack:"relevance" json:"relevance"`
}

// Commentary (Type 15) represents assistant's internal commentary
type Commentary struct {
	ID             string `msgpack:"id" json:"id"`
	MessageID      string `msgpack:"messageId" json:"messageId"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Content        string `msgpack:"content" json:"content"`
	CommentaryType string `msgpack:"commentaryType,omitempty" json:"commentaryType,omitempty"`
}

// AssistantSentence (Type 16) delivers a streaming response chunk
type AssistantSentence struct {
	ID             string `msgpack:"id,omitempty" json:"id,omitempty"`
	PreviousID     string `msgpack:"previousId" json:"previousId"` // References StartAnswer's ID
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Sequence       int32  `msgpack:"sequence" json:"sequence"`
	Text           string `msgpack:"text" json:"text"`
	IsFinal        bool   `msgpack:"isFinal,omitempty" json:"isFinal,omitempty"`
	Audio          []byte `msgpack:"audio,omitempty" json:"audio,omitempty"`
}
