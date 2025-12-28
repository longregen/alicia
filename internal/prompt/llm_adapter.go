package prompt

import (
	"context"
	"fmt"

	"github.com/XiaoConstantine/dspy-go/pkg/core"
	"github.com/longregen/alicia/internal/ports"
)

// LLMServiceAdapter adapts Alicia's LLMService to dspy-go's LLM interface
type LLMServiceAdapter struct {
	service ports.LLMService
}

// NewLLMServiceAdapter creates a new LLM service adapter
func NewLLMServiceAdapter(service ports.LLMService) *LLMServiceAdapter {
	return &LLMServiceAdapter{service: service}
}

// Generate implements the dspy-go LLM interface
func (a *LLMServiceAdapter) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	messages := []ports.LLMMessage{
		{Role: "user", Content: prompt},
	}

	resp, err := a.service.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("llm service chat failed: %w", err)
	}

	return &core.LLMResponse{
		Content: resp.Content,
	}, nil
}

// GenerateWithJSON implements structured JSON output
// NOT NEEDED for GEPA: GEPA only uses Generate() for basic prompt optimization.
// This would be needed for:
// - Structured output modules requiring JSON schema validation
// - Chain-of-thought with guaranteed JSON response format
func (a *LLMServiceAdapter) GenerateWithJSON(ctx context.Context, prompt string, opts ...core.GenerateOption) (map[string]interface{}, error) {
	return nil, fmt.Errorf("GenerateWithJSON not implemented: not required for GEPA optimization")
}

// GenerateWithFunctions implements function calling
// NOT NEEDED for GEPA: GEPA only uses Generate() for basic prompt optimization.
// This would be needed for:
// - ReAct modules that use tool calling
// - Function-based agents
// - Advanced Chain-of-Thought with tool integration
func (a *LLMServiceAdapter) GenerateWithFunctions(ctx context.Context, prompt string, functions []map[string]interface{}, opts ...core.GenerateOption) (map[string]interface{}, error) {
	return nil, fmt.Errorf("GenerateWithFunctions not implemented: not required for GEPA optimization")
}

// CreateEmbedding creates an embedding for the input
// NOT NEEDED for GEPA: GEPA uses the LLM's Generate() for semantic similarity via text comparison.
// This would be needed for:
// - RAG (Retrieval-Augmented Generation) modules
// - Semantic similarity metrics using embeddings
// - Vector-based memory retrieval in optimization
// NOTE: Alicia has a separate EmbeddingService interface (ports.EmbeddingService)
func (a *LLMServiceAdapter) CreateEmbedding(ctx context.Context, input string, opts ...core.EmbeddingOption) (*core.EmbeddingResult, error) {
	return nil, fmt.Errorf("CreateEmbedding not implemented: not required for GEPA optimization, use ports.EmbeddingService for embeddings")
}

// CreateEmbeddings creates embeddings for multiple inputs
// NOT NEEDED for GEPA: GEPA uses the LLM's Generate() for semantic similarity via text comparison.
// This would be needed for:
// - Batch embedding generation for large datasets
// - RAG indexing pipelines
// NOTE: Alicia has a separate EmbeddingService interface (ports.EmbeddingService)
func (a *LLMServiceAdapter) CreateEmbeddings(ctx context.Context, inputs []string, opts ...core.EmbeddingOption) (*core.BatchEmbeddingResult, error) {
	return nil, fmt.Errorf("CreateEmbeddings not implemented: not required for GEPA optimization, use ports.EmbeddingService for embeddings")
}

// StreamGenerate implements streaming generation
// NOT NEEDED for GEPA: GEPA optimization runs in batch mode, not streaming.
// This would be needed for:
// - Real-time streaming responses in production
// - Interactive Chain-of-Thought visualization
// NOTE: To implement, use ports.LLMService.ChatStream() and convert the channel
func (a *LLMServiceAdapter) StreamGenerate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.StreamResponse, error) {
	return nil, fmt.Errorf("StreamGenerate not implemented: not required for GEPA optimization, use ports.LLMService.ChatStream() for streaming")
}

// GenerateWithContent implements multimodal generation
// NOT NEEDED for GEPA: GEPA optimizes text-based prompts only.
// This would be needed for:
// - Vision-based Chain-of-Thought modules
// - Multimodal RAG with images/audio
// - Document understanding with images
func (a *LLMServiceAdapter) GenerateWithContent(ctx context.Context, content []core.ContentBlock, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	return nil, fmt.Errorf("GenerateWithContent not implemented: not required for GEPA optimization")
}

// StreamGenerateWithContent implements streaming multimodal generation
// NOT NEEDED for GEPA: GEPA optimizes text-based prompts only and runs in batch mode.
// This would be needed for:
// - Real-time multimodal streaming responses
// - Interactive vision-based applications
func (a *LLMServiceAdapter) StreamGenerateWithContent(ctx context.Context, content []core.ContentBlock, opts ...core.GenerateOption) (*core.StreamResponse, error) {
	return nil, fmt.Errorf("StreamGenerateWithContent not implemented: not required for GEPA optimization")
}

// ProviderName returns the provider name
func (a *LLMServiceAdapter) ProviderName() string {
	return "alicia"
}

// ModelID returns the model identifier
func (a *LLMServiceAdapter) ModelID() string {
	return "alicia-llm-service"
}

// Capabilities returns the capabilities of this LLM
func (a *LLMServiceAdapter) Capabilities() []core.Capability {
	return []core.Capability{core.CapabilityChat, core.CapabilityCompletion}
}

// DatasetAdapter adapts Alicia's []Example to dspy-go's core.Dataset interface
type DatasetAdapter struct {
	examples []Example
	index    int
}

// NewDatasetAdapter creates a new dataset adapter
func NewDatasetAdapter(examples []Example) *DatasetAdapter {
	return &DatasetAdapter{
		examples: examples,
		index:    0,
	}
}

// Next returns the next example in the dataset
func (d *DatasetAdapter) Next() (core.Example, bool) {
	if d.index >= len(d.examples) {
		return core.Example{}, false
	}
	ex := d.examples[d.index]
	d.index++

	// Convert Alicia Example to dspy-go core.Example
	return core.Example{
		Inputs:  ConvertToInterfaceMap(ex.Inputs),
		Outputs: ConvertToInterfaceMap(ex.Outputs),
	}, true
}

// Reset resets the dataset iterator
func (d *DatasetAdapter) Reset() {
	d.index = 0
}

// ConvertToInterfaceMap converts map[string]any to map[string]interface{}
func ConvertToInterfaceMap(m map[string]any) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// MetricAdapter adapts Alicia's Metric to dspy-go's core.Metric function type
type MetricAdapter struct {
	metric Metric
}

// NewMetricAdapter creates a new metric adapter
func NewMetricAdapter(metric Metric) *MetricAdapter {
	return &MetricAdapter{metric: metric}
}

// ToCoreMetric converts to the dspy-go core.Metric function type
func (m *MetricAdapter) ToCoreMetric() core.Metric {
	return func(expected, actual map[string]interface{}) float64 {
		// Convert to Alicia Example types
		goldExample := Example{
			Inputs:  ConvertFromInterfaceMap(expected),
			Outputs: ConvertFromInterfaceMap(expected),
		}
		predExample := Example{
			Inputs:  ConvertFromInterfaceMap(actual),
			Outputs: ConvertFromInterfaceMap(actual),
		}

		// Score using the Alicia metric
		result, err := m.metric.Score(context.Background(), goldExample, predExample, nil)
		if err != nil {
			return 0.0
		}
		return result.Score
	}
}

// ConvertFromInterfaceMap converts map[string]interface{} to map[string]any
func ConvertFromInterfaceMap(m map[string]interface{}) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
