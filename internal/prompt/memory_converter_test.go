package prompt

import (
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

func TestNewMemoryConverter(t *testing.T) {
	converter := NewMemoryConverter()
	if converter == nil {
		t.Fatal("expected converter to be created")
	}

	if converter.qaPattern == nil {
		t.Error("expected Q&A pattern to be initialized")
	}

	if converter.inputOutputPattern == nil {
		t.Error("expected Input/Output pattern to be initialized")
	}
}

func TestParseQA(t *testing.T) {
	converter := NewMemoryConverter()

	tests := []struct {
		name        string
		content     string
		expectOK    bool
		expectedQ   string
		expectedA   string
	}{
		{
			name:      "basic Q&A",
			content:   "Q: What is AI? A: Artificial Intelligence",
			expectOK:  true,
			expectedQ: "What is AI?",
			expectedA: "Artificial Intelligence",
		},
		{
			name:      "multiline Q&A",
			content:   "Q: What is machine learning? A: A subset of AI that learns from data",
			expectOK:  true,
			expectedQ: "What is machine learning?",
			expectedA: "A subset of AI that learns from data",
		},
		{
			name:      "case insensitive",
			content:   "q: test question a: test answer",
			expectOK:  true,
			expectedQ: "test question",
			expectedA: "test answer",
		},
		{
			name:     "invalid format",
			content:  "This is not a Q&A format",
			expectOK: false,
		},
		{
			name:     "missing answer",
			content:  "Q: What is AI?",
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			example, ok := converter.parseQA(tt.content)

			if ok != tt.expectOK {
				t.Errorf("expected ok=%v, got ok=%v", tt.expectOK, ok)
			}

			if tt.expectOK {
				question, hasQ := example.Inputs["question"].(string)
				answer, hasA := example.Outputs["answer"].(string)

				if !hasQ || !hasA {
					t.Error("expected question and answer fields")
				}

				if question != tt.expectedQ {
					t.Errorf("expected question '%s', got '%s'", tt.expectedQ, question)
				}

				if answer != tt.expectedA {
					t.Errorf("expected answer '%s', got '%s'", tt.expectedA, answer)
				}
			}
		})
	}
}

func TestParseInputOutput(t *testing.T) {
	converter := NewMemoryConverter()

	tests := []struct {
		name           string
		content        string
		expectOK       bool
		expectedInput  string
		expectedOutput string
	}{
		{
			name:           "basic input/output",
			content:        "Input: hello world Output: processed text",
			expectOK:       true,
			expectedInput:  "hello world",
			expectedOutput: "processed text",
		},
		{
			name:           "case insensitive",
			content:        "input: data output: result",
			expectOK:       true,
			expectedInput:  "data",
			expectedOutput: "result",
		},
		{
			name:     "invalid format",
			content:  "This is not input/output format",
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			example, ok := converter.parseInputOutput(tt.content)

			if ok != tt.expectOK {
				t.Errorf("expected ok=%v, got ok=%v", tt.expectOK, ok)
			}

			if tt.expectOK {
				input, hasI := example.Inputs["input"].(string)
				output, hasO := example.Outputs["output"].(string)

				if !hasI || !hasO {
					t.Error("expected input and output fields")
				}

				if input != tt.expectedInput {
					t.Errorf("expected input '%s', got '%s'", tt.expectedInput, input)
				}

				if output != tt.expectedOutput {
					t.Errorf("expected output '%s', got '%s'", tt.expectedOutput, output)
				}
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	converter := NewMemoryConverter()

	tests := []struct {
		name     string
		content  string
		expectOK bool
	}{
		{
			name:     "structured JSON",
			content:  `{"inputs": {"question": "test"}, "outputs": {"answer": "result"}}`,
			expectOK: true,
		},
		{
			name:     "flat JSON with outputs",
			content:  `{"question": "test", "answer": "result"}`,
			expectOK: true,
		},
		{
			name:     "invalid JSON",
			content:  `{invalid json}`,
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			example, ok := converter.parseJSON(tt.content)

			if ok != tt.expectOK {
				t.Errorf("expected ok=%v, got ok=%v", tt.expectOK, ok)
			}

			if tt.expectOK && len(example.Inputs) == 0 && len(example.Outputs) == 0 {
				t.Error("expected non-empty inputs or outputs")
			}
		})
	}
}

func TestDetectCategory(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "QA category",
			tags:     []string{"qa", "question"},
			expected: "qa",
		},
		{
			name:     "preference category",
			tags:     []string{"preference"},
			expected: "preference",
		},
		{
			name:     "fact category",
			tags:     []string{"fact", "knowledge"},
			expected: "fact",
		},
		{
			name:     "instruction category",
			tags:     []string{"instruction", "rule"},
			expected: "instruction",
		},
		{
			name:     "conversation category",
			tags:     []string{"conversation", "chat"},
			expected: "conversation",
		},
		{
			name:     "no category",
			tags:     []string{"random", "unrelated"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectCategory(tt.tags)
			if result != tt.expected {
				t.Errorf("expected category '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestConvertMemoriesToExamples(t *testing.T) {
	converter := NewMemoryConverter()

	now := time.Now()
	memories := []*models.Memory{
		{
			ID:        "mem1",
			Content:   "Q: What is Go? A: A programming language",
			Tags:      []string{"qa"},
			CreatedAt: now,
		},
		{
			ID:        "mem2",
			Content:   "Input: test data Output: processed result",
			Tags:      []string{},
			CreatedAt: now,
		},
		{
			ID:        "mem3",
			Content:   "", // Empty content should be skipped
			Tags:      []string{},
			CreatedAt: now,
		},
	}

	examples := converter.ConvertMemoriesToExamples(memories)

	// Should have 2 examples (mem3 has empty content)
	if len(examples) != 2 {
		t.Errorf("expected 2 examples, got %d", len(examples))
	}

	// Check first example (Q&A format)
	if question, ok := examples[0].Inputs["question"].(string); ok {
		if question != "What is Go?" {
			t.Errorf("expected 'What is Go?', got '%s'", question)
		}
	} else {
		t.Error("expected question field in first example")
	}

	// Check second example (Input/Output format)
	if input, ok := examples[1].Inputs["input"].(string); ok {
		if input != "test data" {
			t.Errorf("expected 'test data', got '%s'", input)
		}
	} else {
		t.Error("expected input field in second example")
	}
}

func TestIsLikelyOutput(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected bool
	}{
		{
			name:     "answer field",
			field:    "answer",
			expected: true,
		},
		{
			name:     "response field",
			field:    "response",
			expected: true,
		},
		{
			name:     "output field",
			field:    "output",
			expected: true,
		},
		{
			name:     "result field",
			field:    "result",
			expected: true,
		},
		{
			name:     "generated field",
			field:    "generated_text",
			expected: true,
		},
		{
			name:     "input field",
			field:    "input",
			expected: false,
		},
		{
			name:     "question field",
			field:    "question",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLikelyOutput(tt.field)
			if result != tt.expected {
				t.Errorf("expected %v for field '%s', got %v", tt.expected, tt.field, result)
			}
		})
	}
}

func TestMemoryDemonstrationBuilder(t *testing.T) {
	builder := NewMemoryDemonstrationBuilder(3)

	if builder.maxDemos != 3 {
		t.Errorf("expected maxDemos to be 3, got %d", builder.maxDemos)
	}

	now := time.Now()
	memories := []*models.Memory{
		{
			ID:        "mem1",
			Content:   "Q: Question 1? A: Answer 1",
			Tags:      []string{"qa"},
			CreatedAt: now,
		},
		{
			ID:        "mem2",
			Content:   "Q: Question 2? A: Answer 2",
			Tags:      []string{"qa"},
			CreatedAt: now,
		},
		{
			ID:        "mem3",
			Content:   "Q: Question 3? A: Answer 3",
			Tags:      []string{"qa"},
			CreatedAt: now,
		},
		{
			ID:        "mem4",
			Content:   "Q: Question 4? A: Answer 4",
			Tags:      []string{"qa"},
			CreatedAt: now,
		},
	}

	demos := builder.BuildDemonstrations(memories)

	// Should limit to maxDemos (3)
	if len(demos) != 3 {
		t.Errorf("expected 3 demonstrations, got %d", len(demos))
	}

	// Verify structure
	for i, demo := range demos {
		if len(demo.Inputs) == 0 {
			t.Errorf("demo %d has no inputs", i)
		}
		if len(demo.Outputs) == 0 {
			t.Errorf("demo %d has no outputs", i)
		}
	}
}

func TestEnrichExampleWithMemory(t *testing.T) {
	example := Example{
		Inputs: map[string]any{
			"question": "What is AI?",
		},
		Outputs: map[string]any{
			"answer": "Artificial Intelligence",
		},
	}

	now := time.Now()
	memories := []*models.Memory{
		{
			ID:        "mem1",
			Content:   "AI is a branch of computer science",
			CreatedAt: now,
		},
		{
			ID:        "mem2",
			Content:   "Machine learning is part of AI",
			CreatedAt: now,
		},
	}

	enriched := EnrichExampleWithMemory(example, memories)

	// Check original fields are preserved
	if enriched.Inputs["question"] != "What is AI?" {
		t.Error("original question not preserved")
	}

	if enriched.Outputs["answer"] != "Artificial Intelligence" {
		t.Error("original answer not preserved")
	}

	// Check memory context was added
	memoryContext, ok := enriched.Inputs["memories"].(string)
	if !ok {
		t.Fatal("expected memories field in enriched inputs")
	}

	if memoryContext == "" {
		t.Error("expected non-empty memory context")
	}

	// Should contain both memory contents
	if !containsString(memoryContext, "AI is a branch of computer science") {
		t.Error("memory context missing first memory")
	}

	if !containsString(memoryContext, "Machine learning is part of AI") {
		t.Error("memory context missing second memory")
	}
}

func TestCreateMemoryFromExample(t *testing.T) {
	example := Example{
		Inputs: map[string]any{
			"question": "What is Go?",
		},
		Outputs: map[string]any{
			"answer": "A programming language",
		},
	}

	tags := []string{"qa", "programming"}
	memory, err := CreateMemoryFromExample(example, tags)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if memory == nil {
		t.Fatal("expected memory to be created")
	}

	// Check Q&A format was used
	if !containsString(memory.Content, "Q:") || !containsString(memory.Content, "A:") {
		t.Error("expected Q&A format in memory content")
	}

	if !containsString(memory.Content, "What is Go?") {
		t.Error("expected question in memory content")
	}

	if !containsString(memory.Content, "A programming language") {
		t.Error("expected answer in memory content")
	}

	// Check tags
	if len(memory.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(memory.Tags))
	}

	// Check default values
	if memory.Importance != 0.5 {
		t.Errorf("expected default importance 0.5, got %f", memory.Importance)
	}

	if memory.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", memory.Confidence)
	}
}

func TestSerializeExample(t *testing.T) {
	tests := []struct {
		name        string
		example     Example
		shouldMatch string
	}{
		{
			name: "Q&A format",
			example: Example{
				Inputs: map[string]any{
					"question": "Test question",
				},
				Outputs: map[string]any{
					"answer": "Test answer",
				},
			},
			shouldMatch: "Q:",
		},
		{
			name: "Input/Output format",
			example: Example{
				Inputs: map[string]any{
					"input": "Test input",
				},
				Outputs: map[string]any{
					"output": "Test output",
				},
			},
			shouldMatch: "Input:",
		},
		{
			name: "JSON fallback",
			example: Example{
				Inputs: map[string]any{
					"custom_field": "value",
				},
				Outputs: map[string]any{
					"custom_output": "result",
				},
			},
			shouldMatch: "inputs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := serializeExample(tt.example)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == "" {
				t.Error("expected non-empty result")
			}

			if !containsString(result, tt.shouldMatch) {
				t.Errorf("expected result to contain '%s', got: %s", tt.shouldMatch, result)
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr || containsString(s[1:], substr))))
}
