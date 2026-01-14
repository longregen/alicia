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

// Feedback (Type 20) represents a vote message sent from client to server
type Feedback struct {
	ID             string `msgpack:"id" json:"id"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	MessageID      string `msgpack:"messageId" json:"messageId"`
	TargetType     string `msgpack:"targetType" json:"targetType"` // message, tool_use, memory, reasoning
	TargetID       string `msgpack:"targetId" json:"targetId"`
	Vote           string `msgpack:"vote" json:"vote"` // up, down, critical, remove
	QuickFeedback  string `msgpack:"quickFeedback,omitempty" json:"quickFeedback,omitempty"`
	Note           string `msgpack:"note,omitempty" json:"note,omitempty"`
	Timestamp      int64  `msgpack:"timestamp" json:"timestamp"`
}

// FeedbackAggregates contains aggregated vote counts
type FeedbackAggregates struct {
	Upvotes      int            `msgpack:"upvotes" json:"upvotes"`
	Downvotes    int            `msgpack:"downvotes" json:"downvotes"`
	SpecialVotes map[string]int `msgpack:"specialVotes,omitempty" json:"specialVotes,omitempty"`
}

// FeedbackConfirmation (Type 21) represents server confirmation with aggregates
type FeedbackConfirmation struct {
	FeedbackID string             `msgpack:"feedbackId" json:"feedbackId"`
	TargetType string             `msgpack:"targetType" json:"targetType"`
	TargetID   string             `msgpack:"targetId" json:"targetId"`
	Aggregates FeedbackAggregates `msgpack:"aggregates" json:"aggregates"`
	UserVote   string             `msgpack:"userVote" json:"userVote"` // up, down, critical, or empty
}

// UserNote (Type 22) represents a note message
type UserNote struct {
	ID        string `msgpack:"id" json:"id"`
	MessageID string `msgpack:"messageId" json:"messageId"`
	Content   string `msgpack:"content" json:"content"`
	Category  string `msgpack:"category" json:"category"` // improvement, correction, context, general
	Action    string `msgpack:"action" json:"action"`     // create, update, delete
	Timestamp int64  `msgpack:"timestamp" json:"timestamp"`
}

// NoteConfirmation (Type 23) represents note confirmation
type NoteConfirmation struct {
	NoteID    string `msgpack:"noteId" json:"noteId"`
	MessageID string `msgpack:"messageId" json:"messageId"`
	Success   bool   `msgpack:"success" json:"success"`
}

// MemoryData contains the memory content and metadata
type MemoryData struct {
	Content  string `msgpack:"content" json:"content"`
	Category string `msgpack:"category" json:"category"`
	Pinned   bool   `msgpack:"pinned" json:"pinned"`
}

// MemoryAction (Type 24) represents memory CRUD actions
type MemoryAction struct {
	ID        string      `msgpack:"id" json:"id"`
	Action    string      `msgpack:"action" json:"action"` // create, update, delete, pin, archive
	Memory    *MemoryData `msgpack:"memory,omitempty" json:"memory,omitempty"`
	Timestamp int64       `msgpack:"timestamp" json:"timestamp"`
}

// MemoryConfirmation (Type 25) represents memory confirmation
type MemoryConfirmation struct {
	MemoryID string `msgpack:"memoryId" json:"memoryId"`
	Action   string `msgpack:"action" json:"action"`
	Success  bool   `msgpack:"success" json:"success"`
}

// ConnectionInfo contains connection status information
type ConnectionInfo struct {
	Status  string `msgpack:"status" json:"status"` // connected, connecting, disconnected, reconnecting
	Latency int    `msgpack:"latency" json:"latency"`
}

// ModelInfo contains model configuration information
type ModelInfo struct {
	Name     string `msgpack:"name" json:"name"`
	Provider string `msgpack:"provider" json:"provider"`
}

// MCPServerInfo contains MCP server status information
type MCPServerInfo struct {
	Name   string `msgpack:"name" json:"name"`
	Status string `msgpack:"status" json:"status"` // connected, disconnected, error
}

// ServerInfo (Type 26) represents server info broadcast
type ServerInfo struct {
	Connection ConnectionInfo  `msgpack:"connection" json:"connection"`
	Model      ModelInfo       `msgpack:"model" json:"model"`
	MCPServers []MCPServerInfo `msgpack:"mcpServers" json:"mcpServers"`
}

// SessionStats (Type 27) represents session statistics
type SessionStats struct {
	MessageCount    int `msgpack:"messageCount" json:"messageCount"`
	ToolCallCount   int `msgpack:"toolCallCount" json:"toolCallCount"`
	MemoriesUsed    int `msgpack:"memoriesUsed" json:"memoriesUsed"`
	SessionDuration int `msgpack:"sessionDuration" json:"sessionDuration"`
}

// ConversationUpdate (Type 28) represents conversation metadata updates
type ConversationUpdate struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Title          string `msgpack:"title,omitempty" json:"title,omitempty"`
	Status         string `msgpack:"status,omitempty" json:"status,omitempty"`
	UpdatedAt      string `msgpack:"updatedAt" json:"updatedAt"`
}

// DimensionWeights defines weights for GEPA optimization dimensions
type DimensionWeights struct {
	SuccessRate    float64 `msgpack:"successRate" json:"successRate"`
	Quality        float64 `msgpack:"quality" json:"quality"`
	Efficiency     float64 `msgpack:"efficiency" json:"efficiency"`
	Robustness     float64 `msgpack:"robustness" json:"robustness"`
	Generalization float64 `msgpack:"generalization" json:"generalization"`
	Diversity      float64 `msgpack:"diversity" json:"diversity"`
	Innovation     float64 `msgpack:"innovation" json:"innovation"`
}

// DimensionScores holds per-dimension performance metrics
type DimensionScores struct {
	SuccessRate    float64 `msgpack:"successRate" json:"successRate"`
	Quality        float64 `msgpack:"quality" json:"quality"`
	Efficiency     float64 `msgpack:"efficiency" json:"efficiency"`
	Robustness     float64 `msgpack:"robustness" json:"robustness"`
	Generalization float64 `msgpack:"generalization" json:"generalization"`
	Diversity      float64 `msgpack:"diversity" json:"diversity"`
	Innovation     float64 `msgpack:"innovation" json:"innovation"`
}

// DimensionPreference (Type 29) represents user dimension weight preferences
type DimensionPreference struct {
	ConversationID string           `msgpack:"conversationId" json:"conversationId"`
	Weights        DimensionWeights `msgpack:"weights" json:"weights"`
	Preset         string           `msgpack:"preset,omitempty" json:"preset,omitempty"` // accuracy, speed, reliable, creative, balanced
	Timestamp      int64            `msgpack:"timestamp" json:"timestamp"`
}

// EliteSelect (Type 30) represents user selection of an elite solution
type EliteSelect struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	EliteID        string `msgpack:"eliteId" json:"eliteId"`
	Timestamp      int64  `msgpack:"timestamp" json:"timestamp"`
}

// EliteSummary contains summary information about an elite solution
type EliteSummary struct {
	ID          string          `msgpack:"id" json:"id"`
	Label       string          `msgpack:"label" json:"label"`             // "High Accuracy", "Fast", "Balanced"
	Scores      DimensionScores `msgpack:"scores" json:"scores"`           // Per-dimension performance
	Description string          `msgpack:"description" json:"description"` // Auto-generated summary
	BestFor     string          `msgpack:"bestFor" json:"bestFor"`         // "Complex questions", "Active coding"
}

// EliteOptions (Type 31) represents available elite solutions from server
type EliteOptions struct {
	ConversationID string         `msgpack:"conversationId" json:"conversationId"`
	Elites         []EliteSummary `msgpack:"elites" json:"elites"`
	CurrentEliteID string         `msgpack:"currentEliteId" json:"currentEliteId"`
	Timestamp      int64          `msgpack:"timestamp" json:"timestamp"`
}
