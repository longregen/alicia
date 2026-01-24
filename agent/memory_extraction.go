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

	"go.opentelemetry.io/otel/trace"
)

const (
	memorySimilarityThreshold      = 0.5
	memorySimilarityTopK           = 10
	maxMemoryCandidates            = 3
	conversationMaxLength          = 4000
	defaultMemoryExtractionTimeout = 60 * time.Second
)

type MemoryCandidate struct {
	Content string `json:"content"`
}

type MemoryScores struct {
	Importance int
	Historical int
	Personal   int
	Factual    int
}

type MemoryThresholds struct {
	Importance *int
	Historical *int
	Personal   *int
	Factual    *int
}

func (s MemoryScores) Passes(t MemoryThresholds) bool {
	if t.Importance != nil && s.Importance < *t.Importance {
		return false
	}
	if t.Historical != nil && s.Historical < *t.Historical {
		return false
	}
	if t.Personal != nil && s.Personal < *t.Personal {
		return false
	}
	if t.Factual != nil && s.Factual < *t.Factual {
		return false
	}
	return true
}

func (s MemoryScores) NormalizedImportance() float32 {
	return float32(s.Importance) / 5.0
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

func ExtractAndSaveMemories(ctx context.Context, convID string, deps AgentDeps) {
	if os.Getenv("MEMORY_EXTRACTION_ENABLED") == "false" {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, getMemoryExtractionTimeout())
	defer cancel()

	messages, err := LoadConversationFull(ctx, deps.DB, convID)
	if err != nil || len(messages) < 2 {
		return
	}

	candidates := extractMemoryCandidates(ctx, deps.LLM, buildConversationText(messages))
	if len(candidates) == 0 {
		return
	}

	thresholds := deps.Prefs.GetThresholds(deps.UserID)
	slog.InfoContext(ctx, "memory extraction started", "candidates", len(candidates), "user_id", deps.UserID)

	for _, candidate := range candidates {
		if candidate.Content == "" {
			continue
		}

		preview := langfuse.TruncateString(candidate.Content, 50, "...")
		scores := evaluateMemory(ctx, deps.LLM, candidate.Content)
		slog.InfoContext(ctx, "memory evaluated", "preview", preview, "importance", scores.Importance, "historical", scores.Historical, "personal", scores.Personal, "factual", scores.Factual)

		accepted := scores.Passes(thresholds)

		go sendMemoryScoresToLangfuse(ctx, scores, accepted, convID, deps.UserID)

		if !accepted {
			continue
		}

		embedding, err := deps.LLM.Embed(ctx, candidate.Content)
		if err != nil {
			slog.ErrorContext(ctx, "memory embedding failed", "error", err)
			continue
		}

		existing, err := SearchMemories(ctx, deps.DB, embedding, memorySimilarityThreshold, memorySimilarityTopK)
		if err != nil {
			slog.ErrorContext(ctx, "memory search failed", "error", err)
			continue
		}

		action := rerankMemory(ctx, deps.LLM, candidate.Content, existing)
		if action != "KEEP" {
			slog.InfoContext(ctx, "memory discarded", "preview", preview, "action", action)
			continue
		}

		if err := CreateMemory(ctx, deps.DB, NewMemoryID(), candidate.Content, embedding, scores.NormalizedImportance()); err != nil {
			slog.ErrorContext(ctx, "memory creation failed", "error", err)
		} else {
			slog.InfoContext(ctx, "memory created", "preview", preview)
		}
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
		Schema: json.RawMessage(`{"type":"object","properties":{"memories":{"type":"array","items":{"type":"object","properties":{"content":{"type":"string"}},"required":["content"],"additionalProperties":false}}},"required":["memories"],"additionalProperties":false}`),
	},
}

type memoryCandidatesWrapper struct {
	Memories []MemoryCandidate `json:"memories"`
}

func extractMemoryCandidates(ctx context.Context, llm *LLMClient, conversation string) []MemoryCandidate {
	prompt := getPrompt("alicia/agent/memory-extract", fallbackMemoryExtract,
		map[string]string{"conversation": langfuse.TruncateString(conversation, conversationMaxLength, "...")})

	resp, err := llm.ChatWithOptions(ctx, []LLMMessage{{Role: "user", Content: prompt.Text}}, nil, ChatOptions{
		MaxTokens:      1000,
		ResponseFormat: memoryExtractResponseFormat,
		GenerationName: "memory.extract",
		PromptName:     prompt.Name,
		PromptVersion:  prompt.Version,
	})
	if err != nil {
		slog.ErrorContext(ctx, "memory extraction failed", "error", err)
		return nil
	}

	content := strings.TrimSpace(resp.Content)

	var wrapper memoryCandidatesWrapper
	if err := json.Unmarshal([]byte(content), &wrapper); err == nil && len(wrapper.Memories) > 0 {
		if len(wrapper.Memories) > maxMemoryCandidates {
			return wrapper.Memories[:maxMemoryCandidates]
		}
		return wrapper.Memories
	}

	// Some models ignore json_schema and return a raw array.
	var candidates []MemoryCandidate
	if err := json.Unmarshal([]byte(content), &candidates); err != nil {
		start, end := strings.Index(content, "["), strings.LastIndex(content, "]")
		if start < 0 || end <= start {
			slog.WarnContext(ctx, "memory extraction: no valid JSON in response", "response", content)
			return nil
		}
		if err := json.Unmarshal([]byte(content[start:end+1]), &candidates); err != nil {
			slog.ErrorContext(ctx, "memory JSON parse failed", "error", err)
			return nil
		}
	}

	if len(candidates) > maxMemoryCandidates {
		candidates = candidates[:maxMemoryCandidates]
	}
	return candidates
}

func evaluateMemory(ctx context.Context, llm *LLMClient, content string) MemoryScores {
	var wg sync.WaitGroup
	var scores MemoryScores

	dims := []struct {
		prompt   string
		fallback string
		dest     *int
	}{
		{"alicia/agent/memory-eval-importance", fallbackEvalImportance, &scores.Importance},
		{"alicia/agent/memory-eval-historical", fallbackEvalHistorical, &scores.Historical},
		{"alicia/agent/memory-eval-personal", fallbackEvalPersonal, &scores.Personal},
		{"alicia/agent/memory-eval-factual", fallbackEvalFactual, &scores.Factual},
	}

	for _, dim := range dims {
		wg.Add(1)
		go func(prompt, fallback string, dest *int) {
			defer wg.Done()
			*dest = evalDimension(ctx, llm, prompt, fallback, content)
		}(dim.prompt, dim.fallback, dim.dest)
	}

	wg.Wait()
	return scores
}

func evalDimension(ctx context.Context, llm *LLMClient, promptName, fallback, content string) int {
	prompt := getPrompt(promptName, fallback, map[string]string{"memory": content})

	resp, err := llm.ChatWithOptions(ctx, []LLMMessage{{Role: "user", Content: prompt.Text}}, nil, ChatOptions{
		GenerationName: "memory.eval_dimension",
		PromptName:     prompt.Name,
		PromptVersion:  prompt.Version,
	})
	if err != nil {
		slog.ErrorContext(ctx, "memory eval failed", "prompt", promptName, "error", err)
		return 3
	}

	if text := strings.TrimSpace(resp.Content); len(text) > 0 {
		if rating, err := strconv.Atoi(text[:1]); err == nil && rating >= 1 && rating <= 5 {
			return rating
		}
	}
	return 3
}

func rerankMemory(ctx context.Context, llm *LLMClient, newMemory string, existing []Memory) string {
	if len(existing) == 0 {
		return "KEEP"
	}

	var existingStr strings.Builder
	for i, m := range existing {
		existingStr.WriteString(strconv.Itoa(i+1) + ". " + m.Content + "\n")
	}

	prompt := getPrompt("alicia/agent/memory-rerank", fallbackMemoryRerank, map[string]string{
		"new_memory":        newMemory,
		"existing_memories": existingStr.String(),
	})

	resp, err := llm.ChatWithOptions(ctx, []LLMMessage{{Role: "user", Content: prompt.Text}}, nil, ChatOptions{
		GenerationName: "memory.rerank",
		PromptName:     prompt.Name,
		PromptVersion:  prompt.Version,
	})
	if err != nil {
		return "KEEP"
	}

	decision := strings.ToUpper(strings.TrimSpace(resp.Content))
	if strings.HasPrefix(decision, "DROP") {
		return "DROP"
	}
	return "KEEP"
}

func getPrompt(name, fallback string, vars map[string]string) PromptResult {
	if client := getLangfuseClient(); client != nil {
		if prompt, err := client.GetPrompt(name, langfuse.WithLabel("production")); err == nil {
			return PromptResult{
				Text:    prompt.Compile(vars),
				Name:    prompt.Name,
				Version: prompt.Version,
			}
		}
	}
	return PromptResult{Text: langfuse.CompileTemplate(fallback, vars)}
}

func sendMemoryScoresToLangfuse(ctx context.Context, scores MemoryScores, accepted bool, convID, userID string) {
	client := getLangfuseClient()
	if client == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		slog.WarnContext(ctx, "memory evaluation: no valid trace context, skipping langfuse score ingestion")
		return
	}
	traceID := span.SpanContext().TraceID().String()

	// Detached context: must complete even if caller's context is cancelled
	sendCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Trace ensures scores inherit session/user context in Langfuse UI
	if err := client.CreateTrace(sendCtx, langfuse.TraceParams{
		ID:        traceID,
		Name:      "memory-evaluation",
		SessionID: convID,
		UserID:    userID,
		Tags:      []string{"memory"},
	}); err != nil {
		slog.ErrorContext(ctx, "failed to create langfuse trace for memory scores", "error", err)
	}

	acceptedValue := 0.0
	if accepted {
		acceptedValue = 1.0
	}

	lfScores := []langfuse.ScoreParams{
		{TraceID: traceID, Name: "memory/importance", Value: float64(scores.Importance), DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "memory/historical", Value: float64(scores.Historical), DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "memory/personal", Value: float64(scores.Personal), DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "memory/factual", Value: float64(scores.Factual), DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "memory/accepted", Value: acceptedValue, DataType: langfuse.ScoreDataTypeBoolean},
	}

	if err := client.CreateScoreBatch(sendCtx, lfScores); err != nil {
		slog.ErrorContext(ctx, "failed to send memory scores to langfuse", "error", err)
	} else {
		slog.InfoContext(ctx, "sent memory scores to langfuse", "trace_id", traceID, "session_id", convID, "user_id", userID, "accepted", accepted)
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

Return a JSON array. Each memory should be a concise, standalone statement.
Example: [{"content": "User prefers dark mode"}, {"content": "User is building a Go project called Alicia"}]

Return [] if nothing worth remembering. Only return the JSON array, no explanation.`

const fallbackEvalImportance = `Rate this memory's importance (1-5).

Memory: {{memory}}

1 = Trivial, forgettable
2 = Minor, nice to know
3 = Moderately important
4 = Important, should remember
5 = Critical, must remember

Respond with only the number.`

const fallbackEvalHistorical = `Rate this memory's future usefulness (1-5).

Memory: {{memory}}

1 = One-time use, unlikely to matter again
2 = Rarely useful
3 = Occasionally useful
4 = Frequently useful
5 = Always relevant, foundational

Respond with only the number.`

const fallbackEvalPersonal = `Rate this memory's personal relevance (1-5).

Memory: {{memory}}

1 = Generic, applies to anyone
2 = Slightly personal
3 = Moderately personal
4 = Quite personal
5 = Deeply personal, unique to this user

Respond with only the number.`

const fallbackEvalFactual = `Rate this memory's factfulness (1-5).

Memory: {{memory}}

1 = Speculation or uncertain
2 = Likely but unverified
3 = Reasonably confident
4 = High confidence
5 = Explicitly stated as fact

Respond with only the number.`

const fallbackMemoryRerank = `Decide whether to add this new memory to the memory bank.

New memory: {{new_memory}}

Existing similar memories:
{{existing_memories}}

Respond with exactly one of:
KEEP - if new memory adds unique value not in existing memories
DROP - if redundant, duplicate, or already covered

No explanation needed.`
