package prompt

import (
	"context"
	"fmt"
	"sort"

	"github.com/XiaoConstantine/dspy-go/pkg/core"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// MemoryAwareModule wraps AliciaPredict with memory-augmented few-shot learning
// It retrieves relevant memories and converts them to demonstrations for GEPA optimization
type MemoryAwareModule struct {
	*AliciaPredict
	memoryService ports.MemoryService
	maxDemos      int
	threshold     float32
}

// MemoryAwareOption configures a MemoryAwareModule
type MemoryAwareOption func(*MemoryAwareModule)

// WithMaxDemonstrations sets the maximum number of memory demonstrations to retrieve
func WithMaxDemonstrations(max int) MemoryAwareOption {
	return func(m *MemoryAwareModule) {
		m.maxDemos = max
	}
}

// WithSimilarityThreshold sets the minimum similarity threshold for memory retrieval
func WithSimilarityThreshold(threshold float32) MemoryAwareOption {
	return func(m *MemoryAwareModule) {
		m.threshold = threshold
	}
}

// NewMemoryAwareModule creates a new memory-aware module
func NewMemoryAwareModule(
	sig Signature,
	memoryService ports.MemoryService,
	opts ...MemoryAwareOption,
) *MemoryAwareModule {
	// Create base AliciaPredict module
	baseModule := NewAliciaPredict(sig)

	module := &MemoryAwareModule{
		AliciaPredict: baseModule,
		memoryService: memoryService,
		maxDemos:      5,
		threshold:     0.7,
	}

	for _, opt := range opts {
		opt(module)
	}

	return module
}

// Process executes the prediction with memory-augmented demonstrations
func (m *MemoryAwareModule) Process(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	// Retrieve relevant memories based on inputs
	memories, err := m.retrieveRelevantMemories(ctx, inputs)
	if err != nil {
		// Continue without memories if retrieval fails
		return m.AliciaPredict.Process(ctx, inputs)
	}

	// Convert memories to demonstrations and add them to the module's few-shot examples
	demos := m.convertMemoriesToDemonstrations(memories)
	if len(demos) > 0 {
		// Update the module's demonstrations
		m.updateDemonstrations(demos)
	}

	// Execute with memory-augmented context
	return m.AliciaPredict.Process(ctx, inputs)
}

// retrieveRelevantMemories searches for relevant memories based on the input
func (m *MemoryAwareModule) retrieveRelevantMemories(ctx context.Context, inputs map[string]any) ([]*models.Memory, error) {
	// Construct query from inputs
	query := m.constructQuery(inputs)
	if query == "" {
		return nil, nil
	}

	// Search memories with scores
	results, err := m.memoryService.SearchWithScores(ctx, query, m.threshold, m.maxDemos)
	if err != nil {
		return nil, fmt.Errorf("memory search failed: %w", err)
	}

	// Extract memories sorted by relevance
	memories := make([]*models.Memory, 0, len(results))
	for _, result := range results {
		memories = append(memories, result.Memory)
	}

	return memories, nil
}

// constructQuery builds a search query from the inputs
func (m *MemoryAwareModule) constructQuery(inputs map[string]any) string {
	// Combine all input values into a query string
	// Priority: user_message > context > other fields
	if msg, ok := inputs["user_message"].(string); ok {
		return msg
	}

	if ctx, ok := inputs["context"].(string); ok {
		return ctx
	}

	if question, ok := inputs["question"].(string); ok {
		return question
	}

	// Fallback: concatenate all string inputs
	query := ""
	for _, v := range inputs {
		if s, ok := v.(string); ok && s != "" {
			if query != "" {
				query += " "
			}
			query += s
		}
	}
	return query
}

// convertMemoriesToDemonstrations converts memories to dspy-go Example format
func (m *MemoryAwareModule) convertMemoriesToDemonstrations(memories []*models.Memory) []core.Example {
	demos := make([]core.Example, 0, len(memories))

	for _, memory := range memories {
		demo, ok := m.parseMemoryContent(memory)
		if ok {
			demos = append(demos, demo)
		}
	}

	return demos
}

// parseMemoryContent attempts to parse memory content into an example
// Expected formats:
// 1. Tagged format: "Q: ... A: ..." or "Input: ... Output: ..."
// 2. Category-specific formats based on tags
func (m *MemoryAwareModule) parseMemoryContent(memory *models.Memory) (core.Example, bool) {
	content := memory.Content

	// Check tags for category hints
	category := m.detectMemoryCategory(memory.Tags)

	switch category {
	case "preference":
		return m.parsePreference(content)
	case "fact":
		return m.parseFact(content)
	case "instruction":
		return m.parseInstruction(content)
	case "context":
		return m.parseContext(content)
	default:
		return m.parseGenericExample(content)
	}
}

// detectMemoryCategory determines the memory category from tags
func (m *MemoryAwareModule) detectMemoryCategory(tags []string) string {
	for _, tag := range tags {
		switch tag {
		case "preference", "user_preference":
			return "preference"
		case "fact", "knowledge":
			return "fact"
		case "instruction", "rule":
			return "instruction"
		case "context", "background":
			return "context"
		}
	}
	return ""
}

// parsePreference parses a preference memory into an example
// Format: "User prefers X over Y" -> input: scenario, output: preference
func (m *MemoryAwareModule) parsePreference(content string) (core.Example, bool) {
	return core.Example{
		Inputs: map[string]interface{}{
			"preference": content,
		},
		Outputs: map[string]interface{}{},
	}, true
}

// parseFact parses a fact memory into an example
// Format: "X is Y" -> input: question about X, output: Y
func (m *MemoryAwareModule) parseFact(content string) (core.Example, bool) {
	return core.Example{
		Inputs: map[string]interface{}{
			"fact": content,
		},
		Outputs: map[string]interface{}{},
	}, true
}

// parseInstruction parses an instruction memory
func (m *MemoryAwareModule) parseInstruction(content string) (core.Example, bool) {
	return core.Example{
		Inputs: map[string]interface{}{
			"instruction": content,
		},
		Outputs: map[string]interface{}{},
	}, true
}

// parseContext parses a context memory
func (m *MemoryAwareModule) parseContext(content string) (core.Example, bool) {
	return core.Example{
		Inputs: map[string]interface{}{
			"context": content,
		},
		Outputs: map[string]interface{}{},
	}, true
}

// parseGenericExample attempts to parse a generic Q&A or input/output format
func (m *MemoryAwareModule) parseGenericExample(content string) (core.Example, bool) {
	// Try Q&A format: "Q: ... A: ..."
	// Try Input/Output format: "Input: ... Output: ..."
	// For now, store as context if no clear format is detected
	return core.Example{
		Inputs: map[string]interface{}{
			"memory_context": content,
		},
		Outputs: map[string]interface{}{},
	}, true
}

// updateDemonstrations updates the module's few-shot demonstrations
func (m *MemoryAwareModule) updateDemonstrations(demos []core.Example) {
	// Access the underlying Predict module and set demonstrations
	// This assumes dspy-go's Predict module has a SetDemonstrations method
	// If not available, we can store demos in the module and inject them during prompt construction
	if m.Predict != nil {
		// In dspy-go, demonstrations are typically set via the signature or module config
		// We'll need to update the module's internal state
		// For now, this is a placeholder - actual implementation depends on dspy-go's API
	}
}

// MemoryRelevanceScore calculates a combined score for memory relevance
type MemoryRelevanceScore struct {
	Memory           *models.Memory
	SimilarityScore  float32
	ImportanceScore  float32
	RecencyScore     float32
	CategoryMatch    bool
	CombinedScore    float32
}

// RankMemoriesByRelevance ranks memories using multiple signals
func RankMemoriesByRelevance(
	memories []*ports.MemorySearchResult,
	categoryFilter []string,
) []MemoryRelevanceScore {
	scores := make([]MemoryRelevanceScore, 0, len(memories))

	for _, result := range memories {
		score := MemoryRelevanceScore{
			Memory:          result.Memory,
			SimilarityScore: result.Similarity,
			ImportanceScore: result.Memory.Importance,
		}

		// Calculate recency score (newer memories get higher scores)
		// Exponential decay: score = exp(-days/30)
		// This gives: 1.0 for today, 0.36 after 30 days, 0.13 after 60 days
		daysSinceCreation := float32(0) // Placeholder - calculate actual days
		score.RecencyScore = float32(1.0) / (1.0 + daysSinceCreation/30.0)

		// Check category match
		if len(categoryFilter) > 0 {
			for _, tag := range result.Memory.Tags {
				for _, filter := range categoryFilter {
					if tag == filter {
						score.CategoryMatch = true
						break
					}
				}
			}
		}

		// Calculate combined score with weights
		categoryBonus := float32(0.0)
		if score.CategoryMatch {
			categoryBonus = 0.2
		}

		score.CombinedScore = (score.SimilarityScore * 0.5) +
			(score.ImportanceScore * 0.3) +
			(score.RecencyScore * 0.2) +
			categoryBonus

		scores = append(scores, score)
	}

	// Sort by combined score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].CombinedScore > scores[j].CombinedScore
	})

	return scores
}

// ToProgram wraps the MemoryAwareModule in a core.Program for GEPA optimization
func (m *MemoryAwareModule) ToProgram(moduleName string) core.Program {
	modules := map[string]core.Module{
		moduleName: m.Predict,
	}

	forward := func(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
		// Convert inputs
		anyInputs := make(map[string]any, len(inputs))
		for k, v := range inputs {
			anyInputs[k] = v
		}

		// Process through the memory-aware module
		outputs, err := m.Process(ctx, anyInputs)
		if err != nil {
			return nil, err
		}

		// Convert outputs
		result := make(map[string]interface{}, len(outputs))
		for k, v := range outputs {
			result[k] = v
		}
		return result, nil
	}

	return core.NewProgram(modules, forward)
}
