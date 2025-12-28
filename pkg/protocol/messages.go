package protocol

// ErrorMessage (Type 1) conveys errors and exceptional conditions
type ErrorMessage struct {
	ID             string   `msgpack:"id" json:"id"`
	ConversationID string   `msgpack:"conversation_id" json:"conversation_id"`
	Code           int32    `msgpack:"code" json:"code"`
	Message        string   `msgpack:"message" json:"message"`
	Severity       Severity `msgpack:"severity" json:"severity"`
	Recoverable    bool     `msgpack:"recoverable" json:"recoverable"`
	OriginatingID  string   `msgpack:"originating_id,omitempty" json:"originating_id,omitempty"`
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
	PreviousID     string `msgpack:"previous_id,omitempty" json:"previous_id,omitempty"`
	ConversationID string `msgpack:"conversation_id" json:"conversation_id"`
	Content        string `msgpack:"content" json:"content"`
	Timestamp      int64  `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
}

// AssistantMessage (Type 3) conveys a complete assistant response (non-streaming)
type AssistantMessage struct {
	ID             string `msgpack:"id" json:"id"`
	PreviousID     string `msgpack:"previous_id,omitempty" json:"previous_id,omitempty"`
	ConversationID string `msgpack:"conversation_id" json:"conversation_id"`
	Content        string `msgpack:"content" json:"content"`
	Timestamp      int64  `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
}

// AudioChunk (Type 4) represents raw audio data segment
type AudioChunk struct {
	ConversationID string `msgpack:"conversation_id" json:"conversation_id"`
	Format         string `msgpack:"format" json:"format"` // e.g., "audio/opus"
	Sequence       int32  `msgpack:"sequence" json:"sequence"`
	DurationMs     int32  `msgpack:"duration_ms" json:"duration_ms"`
	TrackSID       string `msgpack:"track_sid,omitempty" json:"track_sid,omitempty"`
	Data           []byte `msgpack:"data,omitempty" json:"data,omitempty"`
	IsLast         bool   `msgpack:"is_last,omitempty" json:"is_last,omitempty"`
	Timestamp      uint64 `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
}

// ReasoningStep (Type 5) represents internal reasoning trace
type ReasoningStep struct {
	ID             string `msgpack:"id" json:"id"`
	MessageID      string `msgpack:"message_id" json:"message_id"`
	ConversationID string `msgpack:"conversation_id" json:"conversation_id"`
	Sequence       int32  `msgpack:"sequence" json:"sequence"`
	Content        string `msgpack:"content" json:"content"`
}

// ToolUseRequest (Type 6) represents a request to execute a tool
type ToolUseRequest struct {
	ID             string                 `msgpack:"id" json:"id"`
	MessageID      string                 `msgpack:"message_id" json:"message_id"` // ID of the message that triggered this tool call
	ConversationID string                 `msgpack:"conversation_id" json:"conversation_id"`
	ToolName       string                 `msgpack:"tool_name" json:"tool_name"`
	Parameters     map[string]interface{} `msgpack:"parameters" json:"parameters"`
	Execution      ToolExecution          `msgpack:"execution" json:"execution"`
	TimeoutMs      int32                  `msgpack:"timeout_ms,omitempty" json:"timeout_ms,omitempty"`
}

// DefaultToolTimeout is the default timeout for tool execution (30 seconds)
const DefaultToolTimeout = 30000

// ToolUseResult (Type 7) represents a tool execution result
type ToolUseResult struct {
	ID             string      `msgpack:"id" json:"id"`
	RequestID      string      `msgpack:"request_id" json:"request_id"`
	ConversationID string      `msgpack:"conversation_id" json:"conversation_id"`
	Success        bool        `msgpack:"success" json:"success"`
	Result         interface{} `msgpack:"result,omitempty" json:"result,omitempty"`
	ErrorCode      string      `msgpack:"error_code,omitempty" json:"error_code,omitempty"`
	ErrorMessage   string      `msgpack:"error_message,omitempty" json:"error_message,omitempty"`
}

// Acknowledgement (Type 8) confirms receipt of a message
type Acknowledgement struct {
	ConversationID    string `msgpack:"conversation_id" json:"conversation_id"`
	AcknowledgedStanzaID int32  `msgpack:"acknowledged_stanza_id" json:"acknowledged_stanza_id"`
	Success           bool   `msgpack:"success" json:"success"`
}

// Transcription (Type 9) represents speech-to-text output
type Transcription struct {
	ID             string  `msgpack:"id" json:"id"`
	PreviousID     string  `msgpack:"previous_id,omitempty" json:"previous_id,omitempty"`
	ConversationID string  `msgpack:"conversation_id" json:"conversation_id"`
	Text           string  `msgpack:"text" json:"text"`
	Final          bool    `msgpack:"final" json:"final"`
	Confidence     float32 `msgpack:"confidence,omitempty" json:"confidence,omitempty"`
	Language       string  `msgpack:"language,omitempty" json:"language,omitempty"`
}

// ControlStop (Type 10) halts the assistant's current action
type ControlStop struct {
	ConversationID string   `msgpack:"conversation_id" json:"conversation_id"`
	TargetID       string   `msgpack:"target_id,omitempty" json:"target_id,omitempty"`
	Reason         string   `msgpack:"reason,omitempty" json:"reason,omitempty"`
	StopType       StopType `msgpack:"stop_type,omitempty" json:"stop_type,omitempty"`
}

// ControlVariation (Type 11) requests a variation of a previous message
type ControlVariation struct {
	ConversationID string        `msgpack:"conversation_id" json:"conversation_id"`
	TargetID       string        `msgpack:"target_id" json:"target_id"`
	Mode           VariationType `msgpack:"mode" json:"mode"`
	NewContent     string        `msgpack:"new_content,omitempty" json:"new_content,omitempty"`
}

// Configuration (Type 12) initializes and configures the connection
type Configuration struct {
	ConversationID    string   `msgpack:"conversation_id,omitempty" json:"conversation_id,omitempty"`
	LastSequenceSeen  int32    `msgpack:"last_sequence_seen,omitempty" json:"last_sequence_seen,omitempty"`
	ClientVersion     string   `msgpack:"client_version,omitempty" json:"client_version,omitempty"`
	PreferredLanguage string   `msgpack:"preferred_language,omitempty" json:"preferred_language,omitempty"`
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
	PreviousID           string     `msgpack:"previous_id" json:"previous_id"`
	ConversationID       string     `msgpack:"conversation_id" json:"conversation_id"`
	AnswerType           AnswerType `msgpack:"answer_type,omitempty" json:"answer_type,omitempty"`
	PlannedSentenceCount int32      `msgpack:"planned_sentence_count,omitempty" json:"planned_sentence_count,omitempty"`
}

// MemoryTrace (Type 14) logs memory retrieval events
type MemoryTrace struct {
	ID             string  `msgpack:"id" json:"id"`
	MessageID      string  `msgpack:"message_id" json:"message_id"`
	ConversationID string  `msgpack:"conversation_id" json:"conversation_id"`
	MemoryID       string  `msgpack:"memory_id" json:"memory_id"`
	Content        string  `msgpack:"content" json:"content"`
	Relevance      float32 `msgpack:"relevance" json:"relevance"`
}

// Commentary (Type 15) represents assistant's internal commentary
type Commentary struct {
	ID             string `msgpack:"id" json:"id"`
	MessageID      string `msgpack:"message_id" json:"message_id"`
	ConversationID string `msgpack:"conversation_id" json:"conversation_id"`
	Content        string `msgpack:"content" json:"content"`
	CommentaryType string `msgpack:"commentary_type,omitempty" json:"commentary_type,omitempty"`
}

// AssistantSentence (Type 16) delivers a streaming response chunk
type AssistantSentence struct {
	ID             string `msgpack:"id,omitempty" json:"id,omitempty"`
	PreviousID     string `msgpack:"previous_id" json:"previous_id"` // References StartAnswer's ID
	ConversationID string `msgpack:"conversation_id" json:"conversation_id"`
	Sequence       int32  `msgpack:"sequence" json:"sequence"`
	Text           string `msgpack:"text" json:"text"`
	IsFinal        bool   `msgpack:"is_final,omitempty" json:"is_final,omitempty"`
	Audio          []byte `msgpack:"audio,omitempty" json:"audio,omitempty"`
}
