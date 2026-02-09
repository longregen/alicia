package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/longregen/alicia/pkg/langfuse"
	openai "github.com/sashabaranov/go-openai"
)

const (
	memorySimilarityThreshold      = 0.5
	memorySimilarityTopK           = 10
	maxstrings            = 3
	conversationMaxLength          = 4000
	defaultMemoryExtractionTimeout = 60 * time.Second
)

// DimensionResult holds the output of a single eval dimension LLM call.
type DimensionResult struct {
	Rating        int
	Thinking      string
	PromptName    string
	PromptVersion int
}

// MemoryEvalResult holds all dimension results for a memory candidate.
type MemoryEvalResult struct {
	Importance DimensionResult
	Historical DimensionResult
	Personal   DimensionResult
	Factual    DimensionResult
}

func getMemoryExtractionTimeout() time.Duration {
	if s := os.Getenv("MEMORY_EXTRACTION_TIMEOUT"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		} else {
			slog.Warn("invalid MEMORY_EXTRACTION_TIMEOUT, using default", "value", s, "error", err, "default", defaultMemoryExtractionTimeout)
		}
	}
	return defaultMemoryExtractionTimeout
}

func ExtractAndSaveMemories(ctx context.Context, convID, msgID string, deps AgentDeps) {
	if os.Getenv("MEMORY_EXTRACTION_ENABLED") == "false" {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, getMemoryExtractionTimeout())
	defer cancel()

	messages, err := LoadConversationFull(ctx, deps.DB, convID)
	if err != nil {
		slog.ErrorContext(ctx, "memory extraction: failed to load conversation", "conv_id", convID, "error", err)
		return
	}
	if len(messages) < 2 {
		slog.DebugContext(ctx, "memory extraction: skipping, too few messages", "conv_id", convID, "count", len(messages))
		return
	}

	extractPrompt := RetrievePromptTemplate("alicia/agent/memory-extract", fallbackMemoryExtract,
		map[string]string{"conversation": langfuse.TruncateString(buildConversationText(messages), conversationMaxLength, "...")})

	candidates := extractstrings(ctx, deps.LLM, extractPrompt)
	if len(candidates) == 0 {
		slog.InfoContext(ctx, "memory extraction: no candidates found", "conv_id", convID, "msg_id", msgID)
		return
	}

	slog.InfoContext(ctx, "memory extraction started", "candidates", len(candidates), "user_id", deps.UserID)

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		preview := langfuse.TruncateString(candidate, 50, "...")
		evalResult := evaluateMemory(ctx, deps.LLM, candidate)
		slog.InfoContext(ctx, "memory evaluated", "preview", preview,
			"importance", evalResult.Importance.Rating, "historical", evalResult.Historical.Rating,
			"personal", evalResult.Personal.Rating, "factual", evalResult.Factual.Rating)

		// Build generation record — persisted via defer at end of each candidate
		gen := MemoryGeneration{
			ID:                       NewMemoryGenerationID(),
			ConversationID:           convID,
			MessageID:                msgID,
			MemoryContent:            candidate,
			ExtractPromptName:        extractPrompt.Name,
			ExtractPromptVersion:     extractPrompt.Version,
			ImportanceRating:         evalResult.Importance.Rating,
			ImportanceThinking:       evalResult.Importance.Thinking,
			ImportancePromptName:     evalResult.Importance.PromptName,
			ImportancePromptVersion:  evalResult.Importance.PromptVersion,
			HistoricalRating:         evalResult.Historical.Rating,
			HistoricalThinking:       evalResult.Historical.Thinking,
			HistoricalPromptName:     evalResult.Historical.PromptName,
			HistoricalPromptVersion:  evalResult.Historical.PromptVersion,
			PersonalRating:           evalResult.Personal.Rating,
			PersonalThinking:         evalResult.Personal.Thinking,
			PersonalPromptName:       evalResult.Personal.PromptName,
			PersonalPromptVersion:    evalResult.Personal.PromptVersion,
			FactualRating:            evalResult.Factual.Rating,
			FactualThinking:          evalResult.Factual.Thinking,
			FactualPromptName:        evalResult.Factual.PromptName,
			FactualPromptVersion:     evalResult.Factual.PromptVersion,
		}

		func() {
			defer func() {
				if err := CreateMemoryGeneration(ctx, deps.DB, gen); err != nil {
					slog.ErrorContext(ctx, "memory generation record failed", "error", err)
				}
			}()

			embedding, err := deps.LLM.Embed(ctx, candidate)
			if err != nil {
				slog.ErrorContext(ctx, "memory embedding failed", "error", err)
				return
			}
			if len(embedding) == 0 {
				slog.WarnContext(ctx, "memory embedding returned empty vector, skipping search")
				return
			}

			existing, err := SearchMemories(ctx, deps.DB, embedding, memorySimilarityThreshold, memorySimilarityTopK)
			if err != nil {
				slog.ErrorContext(ctx, "memory search failed", "error", err)
				return
			}
			slog.InfoContext(ctx, "memory similarity search", "preview", preview, "similar_count", len(existing))

			decision, promptName, promptVersion := decideMemory(ctx, deps.LLM, candidate, evalResult, existing)
			gen.RerankDecision = decision
			gen.RerankPromptName = promptName
			gen.RerankPromptVersion = promptVersion
			gen.Accepted = decision == "KEEP"

			go sendMemoryScoresToLangfuse(evalResult, gen.Accepted, convID, msgID, deps.UserID)

			if decision != "KEEP" {
				slog.InfoContext(ctx, "memory discarded", "preview", preview, "decision", decision)
				return
			}

			importance := float32(evalResult.Importance.Rating) / 5.0
			memID := NewMemoryID()
			if err := CreateMemory(ctx, deps.DB, memID, candidate, embedding, importance); err != nil {
				slog.ErrorContext(ctx, "memory creation failed", "error", err)
			} else {
				slog.InfoContext(ctx, "memory created", "preview", preview)
				gen.MemoryID = &memID
			}
		}()
	}
}

func buildConversationText(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		if m.Role == "" || m.Role == "system" {
			continue
		}
		sb.WriteString(strings.ToUpper(m.Role[:1]) + m.Role[1:] + ": " + m.Content + "\n\n")
	}
	return strings.TrimSpace(sb.String())
}

var memoryExtractResponseFormat = &openai.ChatCompletionResponseFormat{
	Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
	JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
		Name:   "memory_candidates",
		Strict: true,
		Schema: json.RawMessage(`{"type":"object","properties":{"memories":{"type":"array","items":{"type":"string"}}},"required":["memories"],"additionalProperties":false}`),
	},
}

var evalDimensionResponseFormat = &openai.ChatCompletionResponseFormat{
	Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
	JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
		Name:   "dimension_rating",
		Strict: true,
		Schema: json.RawMessage(`{"type":"object","properties":{"rating":{"type":"integer","enum":[1,2,3,4,5]}},"required":["rating"],"additionalProperties":false}`),
	},
}

func extractstrings(ctx context.Context, llm *LLMClient, prompt PromptResult) []string {
	slog.InfoContext(ctx, "memory extraction: calling LLM", "prompt_name", prompt.Name, "prompt_version", prompt.Version)
	resp, err := MakeLLMCall(ctx, llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, nil, LLMCallOptions{
		MaxTokens:      1000,
		ResponseFormat: memoryExtractResponseFormat,
		GenerationName: "memory.extract",
		Prompt:         prompt,
		TraceName:      "agent:memory",
		NoRetry:        true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "memory extraction LLM call failed", "error", err)
		return nil
	}
	slog.InfoContext(ctx, "memory extraction: LLM responded", "content_length", len(resp.Content), "tokens", resp.TotalTokens)

	content := strings.TrimSpace(resp.Content)

	var wrapper struct {
		Memories []string `json:"memories"`
	}
	if err := json.Unmarshal([]byte(content), &wrapper); err != nil {
		slog.WarnContext(ctx, "memory extraction: failed to parse response", "error", err, "response", content)
		return nil
	}

	if len(wrapper.Memories) > maxstrings {
		return wrapper.Memories[:maxstrings]
	}
	return wrapper.Memories
}

func evaluateMemory(ctx context.Context, llm *LLMClient, content string) MemoryEvalResult {
	var wg sync.WaitGroup
	var result MemoryEvalResult

	dims := []struct {
		prompt   string
		fallback string
		dest     *DimensionResult
	}{
		{"alicia/agent/memory-eval-importance", fallbackEvalImportance, &result.Importance},
		{"alicia/agent/memory-eval-historical", fallbackEvalHistorical, &result.Historical},
		{"alicia/agent/memory-eval-personal", fallbackEvalPersonal, &result.Personal},
		{"alicia/agent/memory-eval-factual", fallbackEvalFactual, &result.Factual},
	}

	for _, dim := range dims {
		wg.Add(1)
		go func(promptName, fallback string, dest *DimensionResult) {
			defer wg.Done()
			*dest = evalDimension(ctx, llm, promptName, fallback, content)
		}(dim.prompt, dim.fallback, dim.dest)
	}

	wg.Wait()
	return result
}

func evalDimension(ctx context.Context, llm *LLMClient, promptName, fallback, content string) DimensionResult {
	prompt := RetrievePromptTemplate(promptName, fallback, map[string]string{"memory": content})

	result := DimensionResult{
		Rating:        3,
		PromptName:    prompt.Name,
		PromptVersion: prompt.Version,
	}

	resp, err := MakeLLMCall(ctx, llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, nil, LLMCallOptions{
		ResponseFormat: evalDimensionResponseFormat,
		GenerationName: "memory.eval_dimension",
		Prompt:         prompt,
		TraceName:      "agent:memory",
		NoRetry:        true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "memory eval failed", "prompt", promptName, "error", err)
		return result
	}

	result.Thinking = resp.Reasoning

	text := strings.TrimSpace(resp.Content)

	var parsed struct {
		Rating int `json:"rating"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err == nil && parsed.Rating >= 1 && parsed.Rating <= 5 {
		result.Rating = parsed.Rating
	}
	return result
}

func decideMemory(ctx context.Context, llm *LLMClient, newMemory string, eval MemoryEvalResult, existing []Memory) (decision, promptName string, promptVersion int) {
	var existingStr string
	if len(existing) == 0 {
		existingStr = "(none)"
	} else {
		var sb strings.Builder
		for i, m := range existing {
			sb.WriteString(strconv.Itoa(i+1) + ". " + m.Content + "\n")
		}
		existingStr = sb.String()
	}

	prompt := RetrievePromptTemplate("alicia/agent/memory-decide", fallbackMemoryDecide, map[string]string{
		"new_memory":        newMemory,
		"importance":        strconv.Itoa(eval.Importance.Rating),
		"historical":        strconv.Itoa(eval.Historical.Rating),
		"personal":          strconv.Itoa(eval.Personal.Rating),
		"factual":           strconv.Itoa(eval.Factual.Rating),
		"existing_memories": existingStr,
	})

	promptName = prompt.Name
	promptVersion = prompt.Version
	decision = "KEEP"

	resp, err := MakeLLMCall(ctx, llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, nil, LLMCallOptions{
		GenerationName: "memory.decide",
		Prompt:         prompt,
		TraceName:      "agent:memory",
		NoRetry:        true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "memory decide LLM call failed", "error", err)
		return
	}

	d := strings.ToUpper(strings.TrimSpace(resp.Content))
	slog.InfoContext(ctx, "memory decide result", "decision", d, "existing_count", len(existing))
	if strings.HasPrefix(d, "DROP") {
		decision = "DROP"
	}
	return
}

func sendMemoryScoresToLangfuse(eval MemoryEvalResult, accepted bool, convID, msgID, userID string) {
	client := getLangfuseClient()
	if client == nil {
		return
	}

	traceID := "memeval-" + convID + "-" + msgID

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.CreateTrace(ctx, langfuse.TraceParams{
		ID:        traceID,
		Name:      "memory-evaluation",
		SessionID: convID,
		UserID:    userID,
		Tags:      []string{"memory"},
	}); err != nil {
		slog.Error("failed to create langfuse trace for memory scores", "error", err)
	}

	acceptedValue := 0.0
	if accepted {
		acceptedValue = 1.0
	}

	lfScores := []langfuse.ScoreParams{
		{TraceID: traceID, Name: "memory/importance", Value: float64(eval.Importance.Rating), DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "memory/historical", Value: float64(eval.Historical.Rating), DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "memory/personal", Value: float64(eval.Personal.Rating), DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "memory/factual", Value: float64(eval.Factual.Rating), DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "memory/accepted", Value: acceptedValue, DataType: langfuse.ScoreDataTypeBoolean},
	}

	if err := client.CreateScoreBatch(ctx, lfScores); err != nil {
		slog.Error("failed to send memory scores to langfuse", "error", err)
	} else {
		slog.Info("sent memory scores to langfuse", "trace_id", traceID, "session_id", convID, "user_id", userID, "accepted", accepted)
	}
}

const fallbackMemoryExtract = `Extract up to 3 memorable facts from this conversation worth remembering for future interactions.

Focus on:
- User preferences, habits, or characteristics
- Important facts or information shared
- Decisions made or conclusions reached
- Details that personalize future interactions

Conversation:
{{conversation}}

Each memory should be a concise, standalone statement.
Example: {"memories": ["User prefers dark mode", "User is building a Go project called Alicia"]}

Return {"memories": []} if nothing worth remembering.`

const fallbackEvalImportance = `Rate this memory's importance (1-5).

Memory: {{memory}}

1 = Trivial, forgettable
2 = Minor, nice to know
3 = Moderately important
4 = Important, should remember
5 = Critical, must remember

Respond with only the number, in format: {"rating": 1-5}`

const fallbackEvalHistorical = `Rate this memory's future usefulness (1-5).

Memory: {{memory}}

1 = One-time use, unlikely to matter again
2 = Rarely useful
3 = Occasionally useful
4 = Frequently useful
5 = Always relevant, foundational

Respond with only the number, in format: {"rating": 1-5}`

const fallbackEvalPersonal = `Rate this memory's personal relevance (1-5).

Memory: {{memory}}

1 = Generic, applies to anyone
2 = Slightly personal
3 = Moderately personal
4 = Quite personal
5 = Deeply personal, unique to this user

Respond with only the number, in format: {"rating": 1-5}`

const fallbackEvalFactual = `Rate this memory's factfulness (1-5).

Memory: {{memory}}

1 = Speculation or uncertain
2 = Likely but unverified
3 = Reasonably confident
4 = High confidence
5 = Explicitly stated as fact

Respond with only the number, in format: {"rating": 1-5}`

const fallbackMemoryDecide = `Decide whether to store this new memory.

New memory: {{new_memory}}

Evaluation scores (1-5):
- Importance: {{importance}}
- Historical usefulness: {{historical}}
- Personal relevance: {{personal}}
- Factual confidence: {{factual}}

Existing similar memories:
{{existing_memories}}

Respond KEEP if this memory is worth storing (high quality and not redundant).
Respond DROP if low quality, redundant, or already covered.

No explanation needed.`
