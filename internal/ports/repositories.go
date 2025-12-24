package ports

import (
	"context"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// ConversationRepository defines operations for conversation persistence
type ConversationRepository interface {
	Create(ctx context.Context, conversation *models.Conversation) error
	GetByID(ctx context.Context, id string) (*models.Conversation, error)
	GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error)
	GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error)
	Update(ctx context.Context, conversation *models.Conversation) error
	UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error
	Delete(ctx context.Context, id string) error
	DeleteByIDAndUserID(ctx context.Context, id, userID string) error
	List(ctx context.Context, limit, offset int) ([]*models.Conversation, error)
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error)
	ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error)
	ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error)
}

// MessageRepository defines operations for message persistence
type MessageRepository interface {
	Create(ctx context.Context, message *models.Message) error
	GetByID(ctx context.Context, id string) (*models.Message, error)
	Update(ctx context.Context, message *models.Message) error
	Delete(ctx context.Context, id string) error
	GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error)
	GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error)
	GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error)
	GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error)
	// Offline sync support
	GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error)
	GetByLocalID(ctx context.Context, localID string) (*models.Message, error)
	// Cleanup support
	GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error)
	GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error)
}

// SentenceRepository defines operations for sentence persistence
type SentenceRepository interface {
	Create(ctx context.Context, sentence *models.Sentence) error
	GetByID(ctx context.Context, id string) (*models.Sentence, error)
	Update(ctx context.Context, sentence *models.Sentence) error
	Delete(ctx context.Context, id string) error
	GetByMessage(ctx context.Context, messageID string) ([]*models.Sentence, error)
	GetNextSequenceNumber(ctx context.Context, messageID string) (int, error)
	// GetIncompleteOlderThan returns sentences with non-completed status older than the given time
	GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Sentence, error)
	// GetIncompleteByConversation returns incomplete sentences for a specific conversation
	GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Sentence, error)
}

// AudioRepository defines operations for audio persistence
type AudioRepository interface {
	Create(ctx context.Context, audio *models.Audio) error
	GetByID(ctx context.Context, id string) (*models.Audio, error)
	GetByMessage(ctx context.Context, messageID string) (*models.Audio, error)
	GetByLiveKitTrack(ctx context.Context, trackSID string) (*models.Audio, error)
	Update(ctx context.Context, audio *models.Audio) error
	Delete(ctx context.Context, id string) error
}

// MemoryWithScore holds a memory along with its similarity score
type MemoryWithScore struct {
	Memory          *models.Memory
	SimilarityScore float32
}

// MemorySearchOptions contains options for searching memories
type MemorySearchOptions struct {
	Embedding     []float32
	Limit         int
	Threshold     *float32 // Optional minimum similarity threshold
	IncludeScores bool     // Whether to return similarity scores
}

// MemorySearchResult contains a memory and its optional similarity score
type MemorySearchResult struct {
	Memory     *models.Memory
	Similarity float32 // Set to 0 if IncludeScores is false
}

// MemoryRepository defines operations for memory persistence
type MemoryRepository interface {
	Create(ctx context.Context, memory *models.Memory) error
	GetByID(ctx context.Context, id string) (*models.Memory, error)
	Update(ctx context.Context, memory *models.Memory) error
	Delete(ctx context.Context, id string) error

	// SearchMemories performs a unified search with configurable options
	SearchMemories(ctx context.Context, opts MemorySearchOptions) ([]*MemorySearchResult, error)

	GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error)
}

// MemoryUsageStats contains statistics about memory usage
type MemoryUsageStats struct {
	MemoryID          string
	TotalUsageCount   int
	AverageSimilarity float32
	LastUsedAt        *time.Time
}

// MemoryUsageRepository defines operations for memory usage tracking
type MemoryUsageRepository interface {
	Create(ctx context.Context, usage *models.MemoryUsage) error
	GetByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error)
	GetByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error)
	GetByMemory(ctx context.Context, memoryID string) ([]*models.MemoryUsage, error)
	GetUsageStats(ctx context.Context, memoryID string) (*MemoryUsageStats, error)
}

// ToolRepository defines operations for tool persistence
type ToolRepository interface {
	Create(ctx context.Context, tool *models.Tool) error
	GetByID(ctx context.Context, id string) (*models.Tool, error)
	GetByName(ctx context.Context, name string) (*models.Tool, error)
	Update(ctx context.Context, tool *models.Tool) error
	Delete(ctx context.Context, id string) error
	ListEnabled(ctx context.Context) ([]*models.Tool, error)
	ListAll(ctx context.Context) ([]*models.Tool, error)
}

// ToolUseRepository defines operations for tool use tracking
type ToolUseRepository interface {
	Create(ctx context.Context, toolUse *models.ToolUse) error
	GetByID(ctx context.Context, id string) (*models.ToolUse, error)
	Update(ctx context.Context, toolUse *models.ToolUse) error
	GetByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error)
	GetPending(ctx context.Context, limit int) ([]*models.ToolUse, error)
}

// ReasoningStepRepository defines operations for reasoning step persistence
type ReasoningStepRepository interface {
	Create(ctx context.Context, step *models.ReasoningStep) error
	GetByMessage(ctx context.Context, messageID string) ([]*models.ReasoningStep, error)
	GetNextSequenceNumber(ctx context.Context, messageID string) (int, error)
}

// CommentaryRepository defines operations for commentary persistence
type CommentaryRepository interface {
	Create(ctx context.Context, commentary *models.Commentary) error
	GetByID(ctx context.Context, id string) (*models.Commentary, error)
	GetByConversation(ctx context.Context, conversationID string) ([]*models.Commentary, error)
	GetByMessage(ctx context.Context, messageID string) ([]*models.Commentary, error)
}

// MetaRepository defines operations for metadata persistence
type MetaRepository interface {
	Set(ctx context.Context, ref, key, value string) error
	Get(ctx context.Context, ref, key string) (string, error)
	GetAll(ctx context.Context, ref string) (map[string]string, error)
	Delete(ctx context.Context, ref, key string) error
	DeleteAll(ctx context.Context, ref string) error
}

// MCPServerRepository defines operations for MCP server configuration persistence
type MCPServerRepository interface {
	Create(ctx context.Context, server *models.MCPServer) error
	GetByID(ctx context.Context, id string) (*models.MCPServer, error)
	GetByName(ctx context.Context, name string) (*models.MCPServer, error)
	Update(ctx context.Context, server *models.MCPServer) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*models.MCPServer, error)
}

// TransactionManager handles database transactions
type TransactionManager interface {
	// WithTransaction executes a function within a database transaction
	// If the function returns an error, the transaction is rolled back
	// Otherwise, the transaction is committed
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// IDGenerator generates unique IDs for entities
type IDGenerator interface {
	// GenerateConversationID generates a new conversation ID (ac_xxx)
	GenerateConversationID() string

	// GenerateMessageID generates a new message ID (am_xxx)
	GenerateMessageID() string

	// GenerateSentenceID generates a new sentence ID (ams_xxx)
	GenerateSentenceID() string

	// GenerateAudioID generates a new audio ID (aa_xxx)
	GenerateAudioID() string

	// GenerateMemoryID generates a new memory ID (amem_xxx)
	GenerateMemoryID() string

	// GenerateMemoryUsageID generates a new memory usage ID (amu_xxx)
	GenerateMemoryUsageID() string

	// GenerateToolID generates a new tool ID (at_xxx)
	GenerateToolID() string

	// GenerateToolUseID generates a new tool use ID (atu_xxx)
	GenerateToolUseID() string

	// GenerateReasoningStepID generates a new reasoning step ID (ar_xxx)
	GenerateReasoningStepID() string

	// GenerateCommentaryID generates a new commentary ID (aucc_xxx)
	GenerateCommentaryID() string

	// GenerateMetaID generates a new meta ID (amt_xxx)
	GenerateMetaID() string

	// GenerateMCPServerID generates a new MCP server ID (amcp_xxx)
	GenerateMCPServerID() string
}

// ToolExecutor executes tools with given arguments
type ToolExecutor interface {
	Execute(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error)
}

// HandleToolUseCase defines the interface for tool execution use case
type HandleToolUseCase interface {
	Execute(ctx context.Context, input *HandleToolInput) (*HandleToolOutput, error)
}

// HandleToolInput contains parameters for executing a tool
type HandleToolInput struct {
	ToolUseID      string
	ToolName       string
	Arguments      map[string]any
	TimeoutMs      int
	MessageID      string
	ConversationID string
}

// HandleToolOutput contains the result of tool execution
type HandleToolOutput struct {
	ToolUseID string
	Result    any
	Success   bool
	Error     string
}

// ProcessUserMessageUseCase defines the interface for processing user messages
type ProcessUserMessageUseCase interface {
	Execute(ctx context.Context, input *ProcessUserMessageInput) (*ProcessUserMessageOutput, error)
}

// ProcessUserMessageInput contains parameters for processing user messages
type ProcessUserMessageInput struct {
	ConversationID string
	TextContent    string
	AudioData      []byte
	AudioFormat    string
	PreviousID     string
}

// ProcessUserMessageOutput contains the result of processing a user message
type ProcessUserMessageOutput struct {
	Message          *models.Message
	Audio            *models.Audio
	RelevantMemories []*models.Memory
}

// GenerateResponseUseCase defines the interface for response generation use case
type GenerateResponseUseCase interface {
	Execute(ctx context.Context, input *GenerateResponseInput) (*GenerateResponseOutput, error)
}

// GenerateResponseInput contains parameters for generating a response
type GenerateResponseInput struct {
	ConversationID   string
	UserMessageID    string
	MessageID        string // Optional pre-generated message ID (if empty, one will be generated)
	RelevantMemories []*models.Memory
	EnableTools      bool
	EnableReasoning  bool
	EnableStreaming  bool
	PreviousID       string
}

// GenerateResponseOutput contains the result of response generation
type GenerateResponseOutput struct {
	Message        *models.Message
	Sentences      []*models.Sentence
	ToolUses       []*models.ToolUse
	ReasoningSteps []*models.ReasoningStep
	StreamChannel  <-chan *ResponseStreamChunk
}

// ResponseStreamChunk represents a chunk of streaming response
type ResponseStreamChunk struct {
	SentenceID            string
	Sequence              int
	Text                  string
	IsFinal               bool
	ToolCall              *LLMToolCall
	ToolUseID             string // ID of the ToolUse created for this tool call
	IsToolExecutionResult bool   // True if this chunk represents the result of tool execution
	Reasoning             string
	Error                 error
}
