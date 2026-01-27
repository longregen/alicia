package domain

import "time"

type Conversation struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Title        string     `json:"title"`
	Status       string     `json:"status"` // active, archived
	TipMessageID *string    `json:"tip_message_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"-"`
}

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	PreviousID     *string    `json:"previous_id,omitempty"`
	BranchIndex    int16      `json:"branch_index"`
	Role           string     `json:"role"` // user, assistant
	Content        string     `json:"content"`
	Reasoning      string     `json:"reasoning,omitempty"`
	Status         string     `json:"status"`             // pending, streaming, completed, error
	TraceID        *string    `json:"trace_id,omitempty"` // OTel trace ID for Langfuse correlation
	CreatedAt      time.Time  `json:"created_at"`
	DeletedAt      *time.Time `json:"-"`
}

type Memory struct {
	ID            string     `json:"id"`
	Content       string     `json:"content"`
	Embedding     []float32  `json:"-"` // pgvector, not exposed via API
	Importance    float32    `json:"importance"`
	Pinned        bool       `json:"pinned"`
	Archived      bool       `json:"archived"`
	SourceMsgID   *string    `json:"source_message_id,omitempty"`
	Tags          []string   `json:"tags"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"-"`
	DeletedReason *string    `json:"deleted_reason,omitempty"`
}

type MemoryUse struct {
	ID             string    `json:"id"`
	MemoryID       string    `json:"memory_id"`
	MessageID      string    `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	Similarity     float32   `json:"similarity"`
	CreatedAt      time.Time `json:"created_at"`
}

type MessageFeedback struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	Rating    int16     `json:"rating"` // -1 = down, 0 = neutral, 1 = up
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type ToolUseFeedback struct {
	ID        string    `json:"id"`
	ToolUseID string    `json:"tool_use_id"`
	Rating    int16     `json:"rating"` // -1 = harmful, 0 = neutral, 1 = helpful
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type MemoryUseFeedback struct {
	ID          string    `json:"id"`
	MemoryUseID string    `json:"memory_use_id"`
	Rating      int16     `json:"rating"` // -1 = irrelevant, 0 = neutral, 1 = relevant
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

type Tool struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"` // JSON Schema
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ToolUse struct {
	ID        string         `json:"id"`
	MessageID string         `json:"message_id"`
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments"`
	Result    any            `json:"result,omitempty"`
	Status    string         `json:"status"` // pending, success, error
	Error     string         `json:"error,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type Note struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Embedding []float32  `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"`
}

type MCPServer struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	TransportType string     `json:"transport_type"` // stdio, sse
	Command       string     `json:"command,omitempty"`
	Args          []string   `json:"args,omitempty"`
	URL           string     `json:"url,omitempty"`
	Enabled       bool       `json:"enabled"`
	CreatedAt     time.Time  `json:"created_at"`
	DeletedAt     *time.Time `json:"-"`
}

const (
	ConversationStatusActive   = "active"
	ConversationStatusArchived = "archived"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

const (
	MessageStatusPending   = "pending"
	MessageStatusStreaming = "streaming"
	MessageStatusCompleted = "completed"
	MessageStatusError     = "error"
)

const (
	ToolUseStatusPending = "pending"
	ToolUseStatusSuccess = "success"
	ToolUseStatusError   = "error"
)

const (
	MCPTransportStdio = "stdio"
	MCPTransportSSE   = "sse"
)

const (
	RatingDown    int16 = -1
	RatingNeutral int16 = 0
	RatingUp      int16 = 1
)

type UserPreferences struct {
	UserID string `json:"user_id"`

	// Appearance
	Theme string `json:"theme"`

	// Voice
	AudioOutputEnabled bool    `json:"audio_output_enabled"`
	VoiceSpeed         float32 `json:"voice_speed"`

	// Memory thresholds (1-5, nil = don't filter on this dimension)
	MemoryMinImportance *int `json:"memory_min_importance"`
	MemoryMinHistorical *int `json:"memory_min_historical"`
	MemoryMinPersonal   *int `json:"memory_min_personal"`
	MemoryMinFactual    *int `json:"memory_min_factual"`

	// Memory retrieval
	MemoryRetrievalCount int `json:"memory_retrieval_count"`

	// Agent
	MaxTokens         int     `json:"max_tokens"`
	MaxToolIterations int     `json:"max_tool_iterations"`
	Temperature       float32 `json:"temperature"`

	// Pareto exploration
	ParetoTargetScore     float32 `json:"pareto_target_score"`
	ParetoMaxGenerations  int     `json:"pareto_max_generations"`
	ParetoBranchesPerGen  int     `json:"pareto_branches_per_gen"`
	ParetoArchiveSize     int     `json:"pareto_archive_size"`
	ParetoEnableCrossover bool    `json:"pareto_enable_crossover"`

	// Notes
	NotesSimilarityThreshold float32 `json:"notes_similarity_threshold"`
	NotesMaxCount            int     `json:"notes_max_count"`

	// UI behavior
	ConfirmDeleteMemory bool `json:"confirm_delete_memory"`
	ShowRelevanceScores bool `json:"show_relevance_scores"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const (
	ThemeLight  = "light"
	ThemeDark   = "dark"
	ThemeSystem = "system"
)
