package ports

import (
	"context"
	"io"

	"github.com/longregen/alicia/internal/domain/models"
)

// LLMMessage represents a message in the LLM conversation context
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMToolCall represents a tool call from the LLM
type LLMToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// LLMResponse represents a response from the LLM
type LLMResponse struct {
	Content   string         `json:"content,omitempty"`
	ToolCalls []*LLMToolCall `json:"tool_calls,omitempty"`
	Reasoning string         `json:"reasoning,omitempty"`
}

// LLMStreamChunk represents a streaming chunk from the LLM
type LLMStreamChunk struct {
	Content   string       `json:"content,omitempty"`
	ToolCall  *LLMToolCall `json:"tool_call,omitempty"`
	Reasoning string       `json:"reasoning,omitempty"`
	Done      bool         `json:"done"`
	Error     error        `json:"error,omitempty"`
}

// LLMService defines the interface for LLM interactions
type LLMService interface {
	Chat(ctx context.Context, messages []LLMMessage) (*LLMResponse, error)
	ChatWithTools(ctx context.Context, messages []LLMMessage, tools []*models.Tool) (*LLMResponse, error)
	ChatStream(ctx context.Context, messages []LLMMessage) (<-chan LLMStreamChunk, error)
	ChatStreamWithTools(ctx context.Context, messages []LLMMessage, tools []*models.Tool) (<-chan LLMStreamChunk, error)
}

// ASRResult represents the result of speech recognition
type ASRResult struct {
	Text       string           `json:"text"`
	Confidence float32          `json:"confidence,omitempty"`
	Language   string           `json:"language,omitempty"`
	Segments   []models.Segment `json:"segments,omitempty"`
	Duration   float32          `json:"duration,omitempty"`
}

// ASRService defines the interface for Automatic Speech Recognition
type ASRService interface {
	Transcribe(ctx context.Context, audio []byte, format string) (*ASRResult, error)
	TranscribeStream(ctx context.Context, audioStream io.Reader, format string) (<-chan *ASRResult, error)
}

// TTSOptions configures text-to-speech generation
type TTSOptions struct {
	Voice        string  `json:"voice,omitempty"`
	Speed        float32 `json:"speed,omitempty"`
	Pitch        float32 `json:"pitch,omitempty"`
	OutputFormat string  `json:"output_format,omitempty"`
}

// TTSResult represents the result of text-to-speech
type TTSResult struct {
	Audio      []byte `json:"audio"`
	Format     string `json:"format"`
	DurationMs int    `json:"duration_ms"`
}

// TTSService defines the interface for Text-to-Speech
type TTSService interface {
	Synthesize(ctx context.Context, text string, options *TTSOptions) (*TTSResult, error)
	SynthesizeStream(ctx context.Context, text string, options *TTSOptions) (<-chan *TTSResult, error)
}

// EmbeddingResult represents the result of embedding generation
type EmbeddingResult struct {
	Embedding  []float32 `json:"embedding"`
	Model      string    `json:"model"`
	Dimensions int       `json:"dimensions"`
}

// EmbeddingService defines the interface for generating embeddings
type EmbeddingService interface {
	Embed(ctx context.Context, text string) (*EmbeddingResult, error)
	EmbedBatch(ctx context.Context, texts []string) ([]*EmbeddingResult, error)
	GetDimensions() int
}

// LiveKitParticipant represents a participant in a LiveKit room
type LiveKitParticipant struct {
	ID       string `json:"id"`
	Identity string `json:"identity"`
	Name     string `json:"name,omitempty"`
}

// LiveKitTrack represents an audio/video track
type LiveKitTrack struct {
	SID    string `json:"sid"`
	Name   string `json:"name"`
	Kind   string `json:"kind"` // "audio" or "video"
	Source string `json:"source"`
}

// LiveKitRoom represents a room in LiveKit
type LiveKitRoom struct {
	Name         string                `json:"name"`
	SID          string                `json:"sid"`
	Participants []*LiveKitParticipant `json:"participants,omitempty"`
}

// LiveKitToken contains an access token for LiveKit
type LiveKitToken struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// LiveKitService defines the interface for LiveKit operations
type LiveKitService interface {
	CreateRoom(ctx context.Context, name string) (*LiveKitRoom, error)
	GetRoom(ctx context.Context, name string) (*LiveKitRoom, error)
	DeleteRoom(ctx context.Context, name string) error
	GenerateToken(ctx context.Context, roomName, participantID, participantName string) (*LiveKitToken, error)
	ListParticipants(ctx context.Context, roomName string) ([]*LiveKitParticipant, error)
	SendData(ctx context.Context, roomName string, data []byte, participantIDs []string) error
}

// DataChannelMessage represents a message received over LiveKit data channel
type DataChannelMessage struct {
	Data     []byte `json:"data"`
	SenderID string `json:"sender_id"`
	Topic    string `json:"topic,omitempty"`
}

// AudioFrame represents a frame of audio data from LiveKit
type AudioFrame struct {
	Data       []byte `json:"data"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	TrackSID   string `json:"track_sid"`
}

// LiveKitAgentCallbacks defines callbacks for a LiveKit agent
type LiveKitAgentCallbacks interface {
	OnDataReceived(ctx context.Context, msg *DataChannelMessage) error
	OnAudioReceived(ctx context.Context, frame *AudioFrame) error
	OnParticipantConnected(ctx context.Context, participant *LiveKitParticipant) error
	OnParticipantDisconnected(ctx context.Context, participant *LiveKitParticipant) error
}

// LiveKitAgent defines the interface for a LiveKit agent
type LiveKitAgent interface {
	Connect(ctx context.Context, roomName string) error
	Disconnect(ctx context.Context) error
	SendData(ctx context.Context, data []byte) error
	SendAudio(ctx context.Context, audio []byte, format string) error
	IsConnected() bool
	GetRoom() *LiveKitRoom
}

// ToolService defines the interface for tool management and execution
type ToolService interface {
	// Tool registration and management
	RegisterTool(ctx context.Context, name, description string, schema map[string]any) (*models.Tool, error)
	EnsureTool(ctx context.Context, name, description string, schema map[string]any) (*models.Tool, error) // Idempotent: returns existing or creates new
	RegisterExecutor(name string, executor func(ctx context.Context, arguments map[string]any) (any, error)) error
	GetByID(ctx context.Context, id string) (*models.Tool, error)
	GetByName(ctx context.Context, name string) (*models.Tool, error)
	Update(ctx context.Context, tool *models.Tool) error
	Enable(ctx context.Context, id string) (*models.Tool, error)
	Disable(ctx context.Context, id string) (*models.Tool, error)
	Delete(ctx context.Context, id string) error
	ListEnabled(ctx context.Context) ([]*models.Tool, error)
	ListAll(ctx context.Context) ([]*models.Tool, error)

	// Tool execution
	ExecuteTool(ctx context.Context, toolName string, arguments map[string]any) (any, error)
	CreateToolUse(ctx context.Context, messageID, toolName string, arguments map[string]any) (*models.ToolUse, error)
	ExecuteToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error)
	GetToolUseByID(ctx context.Context, id string) (*models.ToolUse, error)
	GetToolUsesByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error)
	GetPendingToolUses(ctx context.Context, limit int) ([]*models.ToolUse, error)
	CancelToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error)
}

// MemoryService defines the interface for memory management and retrieval
type MemoryService interface {
	// Memory creation
	Create(ctx context.Context, content string) (*models.Memory, error)
	CreateWithEmbeddings(ctx context.Context, content string) (*models.Memory, error)
	CreateFromConversation(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error)

	// Memory retrieval
	GetByID(ctx context.Context, id string) (*models.Memory, error)
	GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error)

	// Memory search - returns memories with similarity scores
	Search(ctx context.Context, query string, limit int) ([]*models.Memory, error)
	SearchWithThreshold(ctx context.Context, query string, threshold float32, limit int) ([]*models.Memory, error)
	SearchWithDynamicImportance(ctx context.Context, query string, limit int) ([]*models.Memory, error)

	// SearchWithScores returns memories along with their similarity scores for tracking
	SearchWithScores(ctx context.Context, query string, threshold float32, limit int) ([]*MemorySearchResult, error)

	// Memory usage tracking
	TrackUsage(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error)
	GetUsageByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error)
	GetUsageByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error)

	// Memory management
	Update(ctx context.Context, memory *models.Memory) error
	Delete(ctx context.Context, id string) error
	SetImportance(ctx context.Context, id string, importance float32) (*models.Memory, error)
	SetConfidence(ctx context.Context, id string, confidence float32) (*models.Memory, error)
	SetUserRating(ctx context.Context, id string, rating int) (*models.Memory, error)
	AddTag(ctx context.Context, id, tag string) (*models.Memory, error)
	RemoveTag(ctx context.Context, id, tag string) (*models.Memory, error)
	RegenerateEmbeddings(ctx context.Context, id string) (*models.Memory, error)
	Pin(ctx context.Context, id string, pinned bool) (*models.Memory, error)
	Archive(ctx context.Context, id string) (*models.Memory, error)
}

// PromptVersionService defines the interface for managing system prompt versions
type PromptVersionService interface {
	// EnsureVersion creates a version if it doesn't exist, or returns existing
	EnsureVersion(ctx context.Context, promptType, content, description string) (*models.SystemPromptVersion, error)
	// GetActiveVersion returns the currently active version for a prompt type
	GetActiveVersion(ctx context.Context, promptType string) (*models.SystemPromptVersion, error)
	// ActivateVersion sets a version as active
	ActivateVersion(ctx context.Context, versionID string) error
	// GetOrCreateForConversation ensures prompt version exists and returns ID for conversation
	GetOrCreateForConversation(ctx context.Context, systemPrompt string) (string, error)
	// ListVersions returns versions for a prompt type
	ListVersions(ctx context.Context, promptType string, limit int) ([]*models.SystemPromptVersion, error)
}

// OptimizationService defines the interface for prompt optimization
type OptimizationService interface {
	// Run management
	StartOptimizationRun(ctx context.Context, name, promptType, baselinePrompt string) (*models.OptimizationRun, error)
	GetOptimizationRun(ctx context.Context, id string) (*models.OptimizationRun, error)
	ListOptimizationRuns(ctx context.Context, opts ListOptimizationRunsOptions) ([]*models.OptimizationRun, error)
	CompleteRun(ctx context.Context, runID string, bestScore float64) error
	FailRun(ctx context.Context, runID string, reason string) error
	UpdateProgress(ctx context.Context, runID string, iteration int, currentScore float64) error

	// Candidate management
	AddCandidate(ctx context.Context, runID, promptText string, iteration int) (*models.PromptCandidate, error)
	GetCandidates(ctx context.Context, runID string) ([]*models.PromptCandidate, error)
	GetBestCandidate(ctx context.Context, runID string) (*models.PromptCandidate, error)

	// Evaluation management
	RecordEvaluation(ctx context.Context, candidateID, runID, input, output string, score float64, success bool, latencyMs int64) (*models.PromptEvaluation, error)
	GetEvaluations(ctx context.Context, candidateID string) ([]*models.PromptEvaluation, error)

	// Optimized program retrieval
	GetOptimizedProgram(ctx context.Context, runID string) (*OptimizedProgram, error)

	// Dimension weight management - uses map for decoupling from prompt package
	SetDimensionWeights(weights map[string]float64)
	GetDimensionWeights() map[string]float64
}

// OptimizedProgram represents the result of an optimization run
type OptimizedProgram struct {
	RunID       string
	BestPrompt  string
	BestScore   float64
	Iterations  int
	CompletedAt string
	Elites      []EliteSolution // Pareto-optimal solutions
}

// EliteSolution represents an elite solution from the Pareto archive
type EliteSolution struct {
	ID          string
	Label       string
	Description string
	BestFor     string
	Scores      EliteDimensionScores
}

// EliteDimensionScores holds per-dimension performance metrics for an elite
type EliteDimensionScores struct {
	SuccessRate    float64
	Quality        float64
	Efficiency     float64
	Robustness     float64
	Generalization float64
	Diversity      float64
	Innovation     float64
}

// ConversationBroadcaster defines the interface for broadcasting conversation updates
type ConversationBroadcaster interface {
	BroadcastConversationUpdate(conversation *models.Conversation)
}
