package prompt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/XiaoConstantine/dspy-go/pkg/core"
	"github.com/longregen/alicia/internal/domain/models"
)

// MemoryConverter handles conversion of memories to DSPy demonstrations
type MemoryConverter struct {
	// Pattern matchers for different formats
	qaPattern          *regexp.Regexp
	inputOutputPattern *regexp.Regexp
	keyValuePattern    *regexp.Regexp
}

// NewMemoryConverter creates a new memory converter
func NewMemoryConverter() *MemoryConverter {
	return &MemoryConverter{
		qaPattern:          regexp.MustCompile(`(?i)Q:\s*(.+?)\s*A:\s*(.+)`),
		inputOutputPattern: regexp.MustCompile(`(?i)Input:\s*(.+?)\s*Output:\s*(.+)`),
		keyValuePattern:    regexp.MustCompile(`(?i)(\w+):\s*(.+)`),
	}
}

// ConvertMemoriesToExamples converts a list of memories to Example format
func (c *MemoryConverter) ConvertMemoriesToExamples(memories []*models.Memory) []Example {
	examples := make([]Example, 0, len(memories))

	for _, memory := range memories {
		if example, ok := c.ConvertMemory(memory); ok {
			examples = append(examples, example)
		}
	}

	return examples
}

// ConvertMemory converts a single memory to an Example
func (c *MemoryConverter) ConvertMemory(memory *models.Memory) (Example, bool) {
	content := strings.TrimSpace(memory.Content)
	if content == "" {
		return Example{}, false
	}

	// Try different parsing strategies based on tags
	category := detectCategory(memory.Tags)

	switch category {
	case "qa":
		return c.parseQA(content)
	case "preference":
		return c.parsePreferenceMemory(content)
	case "fact":
		return c.parseFactMemory(content)
	case "instruction":
		return c.parseInstructionMemory(content)
	case "conversation":
		return c.parseConversationMemory(content, memory)
	default:
		// Try pattern matching in order of specificity
		if example, ok := c.parseQA(content); ok {
			return example, true
		}
		if example, ok := c.parseInputOutput(content); ok {
			return example, true
		}
		if example, ok := c.parseJSON(content); ok {
			return example, true
		}
		// Fallback: use as context
		return c.parseAsContext(content), true
	}
}

// parseQA parses Q&A format: "Q: ... A: ..."
func (c *MemoryConverter) parseQA(content string) (Example, bool) {
	matches := c.qaPattern.FindStringSubmatch(content)
	if len(matches) == 3 {
		return Example{
			Inputs: map[string]any{
				"question": strings.TrimSpace(matches[1]),
			},
			Outputs: map[string]any{
				"answer": strings.TrimSpace(matches[2]),
			},
		}, true
	}
	return Example{}, false
}

// parseInputOutput parses Input/Output format: "Input: ... Output: ..."
func (c *MemoryConverter) parseInputOutput(content string) (Example, bool) {
	matches := c.inputOutputPattern.FindStringSubmatch(content)
	if len(matches) == 3 {
		return Example{
			Inputs: map[string]any{
				"input": strings.TrimSpace(matches[1]),
			},
			Outputs: map[string]any{
				"output": strings.TrimSpace(matches[2]),
			},
		}, true
	}
	return Example{}, false
}

// parseJSON attempts to parse JSON-formatted memories
func (c *MemoryConverter) parseJSON(content string) (Example, bool) {
	// Try to parse as JSON object with inputs/outputs
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return Example{}, false
	}

	// Check if it has inputs/outputs structure
	inputs, hasInputs := data["inputs"].(map[string]interface{})
	outputs, hasOutputs := data["outputs"].(map[string]interface{})

	if hasInputs && hasOutputs {
		return Example{
			Inputs:  convertToAnyMap(inputs),
			Outputs: convertToAnyMap(outputs),
		}, true
	}

	// Otherwise, split the data into inputs and outputs heuristically
	inputMap := make(map[string]any)
	outputMap := make(map[string]any)

	for k, v := range data {
		if isLikelyOutput(k) {
			outputMap[k] = v
		} else {
			inputMap[k] = v
		}
	}

	if len(outputMap) > 0 {
		return Example{
			Inputs:  inputMap,
			Outputs: outputMap,
		}, true
	}

	return Example{}, false
}

// parseAsContext creates an example using memory as context
func (c *MemoryConverter) parseAsContext(content string) Example {
	return Example{
		Inputs: map[string]any{
			"context": content,
		},
		Outputs: map[string]any{},
	}
}

// parsePreferenceMemory parses preference memories
// Format: "User prefers X when Y" or "Preference: X over Y"
func (c *MemoryConverter) parsePreferenceMemory(content string) (Example, bool) {
	return Example{
		Inputs: map[string]any{
			"preference_rule": content,
		},
		Outputs: map[string]any{
			"should_apply": "true",
		},
	}, true
}

// parseFactMemory parses fact memories
// Format: "Entity X has property Y" or "Fact: X is Y"
func (c *MemoryConverter) parseFactMemory(content string) (Example, bool) {
	// Extract entity and property if possible
	return Example{
		Inputs: map[string]any{
			"fact": content,
		},
		Outputs: map[string]any{
			"verified": "true",
		},
	}, true
}

// parseInstructionMemory parses instruction memories
// Format: "When X, do Y" or "Instruction: Always Y when X"
func (c *MemoryConverter) parseInstructionMemory(content string) (Example, bool) {
	return Example{
		Inputs: map[string]any{
			"instruction": content,
		},
		Outputs: map[string]any{
			"should_follow": "true",
		},
	}, true
}

// parseConversationMemory parses memories from conversations
// Uses source info to construct richer examples
func (c *MemoryConverter) parseConversationMemory(content string, memory *models.Memory) (Example, bool) {
	example := Example{
		Inputs: map[string]any{
			"memory": content,
		},
		Outputs: map[string]any{},
	}

	// Add metadata if available
	if memory.SourceInfo != nil {
		if memory.SourceInfo.ConversationID != "" {
			example.Inputs["conversation_id"] = memory.SourceInfo.ConversationID
		}
	}

	return example, true
}

// detectCategory determines the memory category from tags
func detectCategory(tags []string) string {
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[strings.ToLower(tag)] = true
	}

	// Check for specific categories
	if tagSet["qa"] || tagSet["question"] || tagSet["answer"] {
		return "qa"
	}
	if tagSet["preference"] || tagSet["user_preference"] {
		return "preference"
	}
	if tagSet["fact"] || tagSet["knowledge"] {
		return "fact"
	}
	if tagSet["instruction"] || tagSet["rule"] {
		return "instruction"
	}
	if tagSet["conversation"] || tagSet["chat"] {
		return "conversation"
	}

	return ""
}

// isLikelyOutput checks if a field name suggests it's an output
func isLikelyOutput(fieldName string) bool {
	outputIndicators := []string{
		"output", "answer", "response", "result", "prediction",
		"completion", "generated", "reply",
	}

	lower := strings.ToLower(fieldName)
	for _, indicator := range outputIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

// convertToAnyMap converts map[string]interface{} to map[string]any
func convertToAnyMap(m map[string]interface{}) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// MemoryDemonstrationBuilder builds demonstrations from memories for few-shot learning
type MemoryDemonstrationBuilder struct {
	converter *MemoryConverter
	maxDemos  int
}

// NewMemoryDemonstrationBuilder creates a new builder
func NewMemoryDemonstrationBuilder(maxDemos int) *MemoryDemonstrationBuilder {
	return &MemoryDemonstrationBuilder{
		converter: NewMemoryConverter(),
		maxDemos:  maxDemos,
	}
}

// BuildDemonstrations converts memories to few-shot demonstrations
func (b *MemoryDemonstrationBuilder) BuildDemonstrations(memories []*models.Memory) []core.Example {
	examples := b.converter.ConvertMemoriesToExamples(memories)

	// Limit to maxDemos
	if len(examples) > b.maxDemos {
		examples = examples[:b.maxDemos]
	}

	// Convert to core.Example format
	coreExamples := make([]core.Example, 0, len(examples))
	for _, ex := range examples {
		coreExamples = append(coreExamples, core.Example{
			Inputs:  ConvertToInterfaceMap(ex.Inputs),
			Outputs: ConvertToInterfaceMap(ex.Outputs),
		})
	}

	return coreExamples
}

// EnrichExampleWithMemory adds memory context to an existing example
func EnrichExampleWithMemory(example Example, memories []*models.Memory) Example {
	enriched := Example{
		Inputs:  make(map[string]any, len(example.Inputs)+1),
		Outputs: make(map[string]any, len(example.Outputs)),
	}

	// Copy original inputs and outputs
	for k, v := range example.Inputs {
		enriched.Inputs[k] = v
	}
	for k, v := range example.Outputs {
		enriched.Outputs[k] = v
	}

	// Add memory context
	if len(memories) > 0 {
		memoryContexts := make([]string, 0, len(memories))
		for _, memory := range memories {
			memoryContexts = append(memoryContexts, memory.Content)
		}
		enriched.Inputs["memories"] = strings.Join(memoryContexts, "\n---\n")
	}

	return enriched
}

// CreateMemoryFromExample creates a memory from a training example
// This is useful for storing successful predictions as memories
func CreateMemoryFromExample(example Example, tags []string) (*models.Memory, error) {
	// Serialize the example to JSON for structured storage
	content, err := serializeExample(example)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize example: %w", err)
	}

	memory := &models.Memory{
		Content:    content,
		Tags:       tags,
		Importance: 0.5, // Default importance
		Confidence: 1.0, // High confidence for training data
	}

	return memory, nil
}

// serializeExample converts an example to a storable string format
func serializeExample(example Example) (string, error) {
	// Try to create a readable Q&A or Input/Output format if possible
	if question, hasQ := example.Inputs["question"]; hasQ {
		if answer, hasA := example.Outputs["answer"]; hasA {
			return fmt.Sprintf("Q: %v\nA: %v", question, answer), nil
		}
	}

	if input, hasI := example.Inputs["input"]; hasI {
		if output, hasO := example.Outputs["output"]; hasO {
			return fmt.Sprintf("Input: %v\nOutput: %v", input, output), nil
		}
	}

	// Fallback to JSON serialization
	data := map[string]interface{}{
		"inputs":  example.Inputs,
		"outputs": example.Outputs,
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
