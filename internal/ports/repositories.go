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
	UpdateTip(ctx context.Context, conversationID, messageID string) error
	UpdatePromptVersion(ctx context.Context, convID, versionID string) error
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
	GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error)
	GetChainFromTipWithSiblings(ctx context.Context, tipMessageID string) ([]*models.Message, error)
	GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error)
	// Offline sync support
	GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error)
	GetByLocalID(ctx context.Context, localID string) (*models.Message, error)
	// Cleanup support
	GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error)
	GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error)
	// DeleteAfterSequence soft-deletes all messages in a conversation after the given sequence number
	DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error
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

	// DeleteByConversationID soft-deletes all memories that were extracted from messages
	// belonging to the specified conversation (via source_message_id)
	DeleteByConversationID(ctx context.Context, conversationID string) error

	// SearchMemories performs a unified search with configurable options
	SearchMemories(ctx context.Context, opts MemorySearchOptions) ([]*MemorySearchResult, error)

	GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error)

	// Pin and Archive operations
	Pin(ctx context.Context, id string, pinned bool) error
	Archive(ctx context.Context, id string) error
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
	// WasDeleted checks if a server with the given name was soft-deleted
	// Returns true if the server exists with a non-null deleted_at
	WasDeleted(ctx context.Context, name string) (bool, error)
}

// VoteRepository handles vote persistence
type VoteRepository interface {
	Create(ctx context.Context, vote *models.Vote) error
	Delete(ctx context.Context, targetType string, targetID string) error
	GetByTarget(ctx context.Context, targetType string, targetID string) ([]*models.Vote, error)
	GetByMessage(ctx context.Context, messageID string) ([]*models.Vote, error)
	GetAggregates(ctx context.Context, targetType string, targetID string) (*models.VoteAggregates, error)

	// GetToolUseVotesWithContext returns tool_use votes with full context for training
	GetToolUseVotesWithContext(ctx context.Context, limit int) ([]*VoteWithToolContext, error)

	// GetMemoryVotesWithContext returns memory votes with full context for training
	GetMemoryVotesWithContext(ctx context.Context, limit int) ([]*VoteWithMemoryContext, error)

	// GetMemoryUsageVotesWithContext returns memory_usage votes with full context for training
	GetMemoryUsageVotesWithContext(ctx context.Context, limit int) ([]*VoteWithMemoryContext, error)

	// GetMemoryExtractionVotesWithContext returns memory_extraction votes with full context for training
	GetMemoryExtractionVotesWithContext(ctx context.Context, limit int) ([]*VoteWithExtractionContext, error)

	// CountByTargetType returns count of votes for a target type
	CountByTargetType(ctx context.Context, targetType string) (int, error)
}

// VoteWithToolContext holds a vote with tool use context for training set building
type VoteWithToolContext struct {
	Vote           *models.Vote
	ToolUse        *models.ToolUse
	UserMessage    string // contents of the user message that triggered tool use
	ConversationID string
	AvailableTools []*models.Tool // tools available at the time
}

// VoteWithMemoryContext holds a vote with memory context for training set building
type VoteWithMemoryContext struct {
	Vote              *models.Vote
	Memory            *models.Memory
	MemoryUsage       *models.MemoryUsage
	UserMessage       string
	ConversationID    string
	SimilarityScore   float32
	CandidateMemories []*models.Memory // other memories from same query
}

// VoteWithExtractionContext holds a vote with memory extraction context for training set building
type VoteWithExtractionContext struct {
	Vote           *models.Vote
	Memory         *models.Memory  // The extracted memory
	SourceMessage  *models.Message // Message it was extracted from
	ConversationID string
}

// NoteRepository handles note persistence
type NoteRepository interface {
	Create(ctx context.Context, note *models.Note) error
	Update(ctx context.Context, id string, content string) error
	Delete(ctx context.Context, id string) error
	GetByMessage(ctx context.Context, messageID string) ([]*models.Note, error)
	GetByID(ctx context.Context, id string) (*models.Note, error)
}

// SessionStatsRepository handles session statistics
type SessionStatsRepository interface {
	Create(ctx context.Context, stats *models.SessionStats) error
	Update(ctx context.Context, stats *models.SessionStats) error
	GetByConversation(ctx context.Context, conversationID string) (*models.SessionStats, error)
}

// SystemPromptVersionRepository manages system prompt versions
type SystemPromptVersionRepository interface {
	Create(ctx context.Context, version *models.SystemPromptVersion) error
	GetByID(ctx context.Context, id string) (*models.SystemPromptVersion, error)
	GetActiveByType(ctx context.Context, promptType string) (*models.SystemPromptVersion, error)
	GetByHash(ctx context.Context, promptType, hash string) (*models.SystemPromptVersion, error)
	SetActive(ctx context.Context, id string) error
	List(ctx context.Context, promptType string, limit int) ([]*models.SystemPromptVersion, error)
	GetLatestByType(ctx context.Context, promptType string) (*models.SystemPromptVersion, error)
}

// OptimizedTool represents a GEPA-optimized tool with enhanced schema and examples
type OptimizedTool struct {
	ID                   string
	ToolID               string
	OptimizedDescription string
	OptimizedSchema      map[string]any
	ResultTemplate       string
	Examples             []map[string]any
	Version              int
	Score                *float64
	OptimizedAt          time.Time
	Active               bool
	DeletedAt            *time.Time
}

// ToolResultFormatter represents formatting rules for tool results
type ToolResultFormatter struct {
	ID            string
	ToolName      string
	Template      string
	MaxLength     int
	SummarizeAt   int
	SummaryPrompt string
	KeyFields     []string
	CreatedAt     time.Time
	DeletedAt     *time.Time
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

	// GenerateVoteID generates a new vote ID (av_xxx)
	GenerateVoteID() string

	// GenerateNoteID generates a new note ID (an_xxx)
	GenerateNoteID() string

	// GenerateSessionStatsID generates a new session stats ID (ass_xxx)
	GenerateSessionStatsID() string

	// GenerateOptimizationRunID generates a new optimization run ID (aor_xxx)
	GenerateOptimizationRunID() string

	// GeneratePromptCandidateID generates a new prompt candidate ID (apc_xxx)
	GeneratePromptCandidateID() string

	// GeneratePromptEvaluationID generates a new prompt evaluation ID (ape_xxx)
	GeneratePromptEvaluationID() string

	// GenerateTrainingExampleID generates a new training example ID (gte_xxx)
	GenerateTrainingExampleID() string

	// GenerateSystemPromptVersionID generates a new system prompt version ID (spv_xxx)
	GenerateSystemPromptVersionID() string

	// GenerateRequestID generates a new request ID (areq_xxx)
	GenerateRequestID() string
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
	MessageID      string // Optional: if provided, use this ID instead of generating a new one
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
// DEPRECATED: Use ParetoResponseGenerator instead for all new code.
// This interface is kept for backwards compatibility during migration.
type GenerateResponseUseCase interface {
	Execute(ctx context.Context, input *GenerateResponseInput) (*GenerateResponseOutput, error)
}

// ParetoResponseGenerator is the SINGLE unified way to generate responses in Alicia.
// It uses GEPA (Genetic-Pareto) path search to find optimal responses:
//   - Explores multiple execution paths (branching attempts)
//   - Uses Pareto selection across 5 dimensions (quality, efficiency, cost, robustness, latency)
//   - Genetically mutates strategy/reflection TEXT via LLM
//   - Accumulates lessons to guide future attempts
//   - Actually executes tools and persists results
//
// This replaces the old GenerateResponse use case and AgentService.
type ParetoResponseGenerator interface {
	Execute(ctx context.Context, input *ParetoResponseInput) (*ParetoResponseOutput, error)
}

// ParetoResponseInput contains the input parameters for Pareto-based response generation.
type ParetoResponseInput struct {
	// ConversationID is the ID of the conversation
	ConversationID string

	// UserMessageID is the ID of the user message to respond to
	UserMessageID string

	// MessageID is an optional pre-generated message ID for the response
	MessageID string

	// PreviousID is the ID of the previous message for branching
	PreviousID string

	// EnableTools enables tool execution during generation
	EnableTools bool

	// EnableReasoning enables reasoning/thinking steps
	EnableReasoning bool

	// EnableStreaming enables streaming of results (via notifier)
	EnableStreaming bool

	// Notifier receives real-time updates during generation
	Notifier GenerationNotifier

	// Config contains optional Pareto search configuration overrides
	Config *ParetoResponseConfig

	// SeedStrategy is an optional custom seed strategy (if empty, uses default)
	SeedStrategy string
}

// ParetoResponseConfig configures the Pareto-based response generation.
type ParetoResponseConfig struct {
	// MaxGenerations is the maximum number of evolutionary generations
	MaxGenerations int

	// BranchesPerGen is the number of parallel paths to explore per generation
	BranchesPerGen int

	// TargetScore is the early exit threshold (0-1) for answer quality
	TargetScore float64

	// MaxToolCalls limits the total number of tool calls across all paths (budget)
	MaxToolCalls int

	// MaxLLMCalls limits the total number of LLM calls across all paths (budget)
	MaxLLMCalls int

	// ParetoArchiveSize is the maximum number of candidates in the Pareto archive
	ParetoArchiveSize int

	// EnableCrossover enables crossover between Pareto-optimal paths
	EnableCrossover bool

	// ExecutionTimeoutMs is the timeout for each path execution in milliseconds
	ExecutionTimeoutMs int64

	// EnableParallelBranches enables parallel processing of branches within a generation
	EnableParallelBranches bool

	// MaxParallelBranches limits the number of concurrent branch executions
	MaxParallelBranches int

	// MaxToolLoopIterations limits the number of LLM-tool loop iterations per path
	MaxToolLoopIterations int
}

// ParetoResponseOutput contains the result of Pareto-based response generation.
type ParetoResponseOutput struct {
	// Message is the created assistant message
	Message *models.Message

	// Sentences contains the individual sentences (for streaming mode)
	Sentences []*models.Sentence

	// ToolUses contains the tool executions performed
	ToolUses []*models.ToolUse

	// ReasoningSteps contains any reasoning/thinking steps
	ReasoningSteps []*models.ReasoningStep

	// StreamChannel is the channel for streaming response chunks
	StreamChannel <-chan *ResponseStreamChunk

	// ParetoFront contains all Pareto-optimal candidates (for analysis)
	ParetoFront []*models.PathCandidate

	// Score is the quality score of the best response
	Score float64

	// Iterations is the number of evolutionary generations completed
	Iterations int
}

// GenerateResponseInput contains parameters for generating a response
type GenerateResponseInput struct {
	ConversationID      string
	UserMessageID       string
	MessageID           string // Optional pre-generated message ID (if empty, one will be generated)
	RelevantMemories    []*models.Memory
	EnableTools         bool
	EnableReasoning     bool
	EnableStreaming     bool
	PreviousID          string
	ContinueFromContent string             // If set, this is the existing assistant content to continue from
	Notifier            GenerationNotifier // Optional notifier for real-time generation progress
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

// RegenerateResponseUseCase defines the interface for regenerating assistant responses
type RegenerateResponseUseCase interface {
	Execute(ctx context.Context, input *RegenerateResponseInput) (*RegenerateResponseOutput, error)
}

// RegenerateResponseInput contains parameters for regenerating a response
type RegenerateResponseInput struct {
	MessageID       string // The assistant message to regenerate
	EnableTools     bool
	EnableReasoning bool
	EnableStreaming bool
	Notifier        GenerationNotifier // Optional notifier for real-time updates
}

// RegenerateResponseOutput contains the result of response regeneration
type RegenerateResponseOutput struct {
	DeletedMessageID string                      // ID of the deleted assistant message
	NewMessage       *models.Message             // The newly generated message
	StreamChannel    <-chan *ResponseStreamChunk // Channel for streaming response (if EnableStreaming)
}

// ContinueResponseUseCase defines the interface for continuing an existing assistant response
type ContinueResponseUseCase interface {
	Execute(ctx context.Context, input *ContinueResponseInput) (*ContinueResponseOutput, error)
}

// ContinueResponseInput contains parameters for continuing an existing assistant response
type ContinueResponseInput struct {
	TargetMessageID string // The assistant message to continue from
	EnableTools     bool
	EnableReasoning bool
	EnableStreaming bool
	Notifier        GenerationNotifier // Optional notifier for real-time updates
}

// ContinueResponseOutput contains the result of continuing a response
type ContinueResponseOutput struct {
	TargetMessage   *models.Message             // The original message that was extended
	AppendedContent string                      // The content that was appended
	StreamChannel   <-chan *ResponseStreamChunk // For streaming mode
	GeneratedOutput *GenerateResponseOutput     // The full output from GenerateResponse (for non-streaming)
}

// SendMessageUseCase orchestrates user message creation and response generation
type SendMessageUseCase interface {
	Execute(ctx context.Context, input *SendMessageInput) (*SendMessageOutput, error)
}

type SendMessageInput struct {
	ConversationID  string
	TextContent     string
	AudioData       []byte
	AudioFormat     string
	PreviousID      string
	LocalID         string
	EnableTools     bool
	EnableReasoning bool
	EnableStreaming bool
}

type SendMessageOutput struct {
	UserMessage      *models.Message
	Audio            *models.Audio
	RelevantMemories []*models.Memory
	AssistantMessage *models.Message
	StreamChannel    <-chan *ResponseStreamChunk
}

// SyncMessagesUseCase handles offline sync
type SyncMessagesUseCase interface {
	Execute(ctx context.Context, input *SyncMessagesInput) (*SyncMessagesOutput, error)
}

type SyncMessagesInput struct {
	ConversationID string
	Messages       []SyncMessageItem
}

type SyncMessageItem struct {
	LocalID        string
	SequenceNumber int
	PreviousID     string
	Role           string
	Contents       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SyncMessagesOutput struct {
	Results  []SyncedMessageResult
	SyncedAt time.Time
}

type SyncedMessageResult struct {
	LocalID  string
	ServerID string
	Status   string // "synced" | "conflict"
	Message  *models.Message
}

// EditUserMessageUseCase updates user message and regenerates
type EditUserMessageUseCase interface {
	Execute(ctx context.Context, input *EditUserMessageInput) (*EditUserMessageOutput, error)
}

type EditUserMessageInput struct {
	ConversationID  string
	TargetMessageID string
	NewContent      string
	EnableTools     bool
	EnableReasoning bool
	EnableStreaming bool
	SkipGeneration  bool // If true, skip response generation (for agent-based architecture)
}

type EditUserMessageOutput struct {
	UpdatedMessage   *models.Message
	DeletedCount     int
	RelevantMemories []*models.Memory
	AssistantMessage *models.Message
	StreamChannel    <-chan *ResponseStreamChunk
}

// EditAssistantMessageUseCase updates assistant message in place
type EditAssistantMessageUseCase interface {
	Execute(ctx context.Context, input *EditAssistantMessageInput) (*EditAssistantMessageOutput, error)
}

type EditAssistantMessageInput struct {
	ConversationID  string
	TargetMessageID string
	NewContent      string
}

type EditAssistantMessageOutput struct {
	UpdatedMessage *models.Message
}

// SynthesizeSpeechUseCase handles text-to-speech synthesis with proper audio storage
type SynthesizeSpeechUseCase interface {
	Execute(ctx context.Context, input *SynthesizeSpeechInput) (*SynthesizeSpeechOutput, error)
}

// SynthesizeSpeechInput contains parameters for speech synthesis
type SynthesizeSpeechInput struct {
	Text            string
	MessageID       string
	SentenceID      string
	Voice           string
	Speed           float32
	Pitch           float32
	OutputFormat    string
	EnableStreaming bool
}

// SynthesizeSpeechOutput contains the result of speech synthesis
type SynthesizeSpeechOutput struct {
	Audio         *models.Audio
	Sentence      *models.Sentence
	AudioData     []byte
	Format        string
	DurationMs    int
	StreamChannel <-chan *AudioStreamChunk
}

// AudioStreamChunk represents a chunk of streaming audio
type AudioStreamChunk struct {
	Data       []byte
	Format     string
	DurationMs int
	Sequence   int
	IsFinal    bool
	Error      error
}
