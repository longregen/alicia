package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/longregen/alicia/pkg/langfuse"
	"github.com/longregen/alicia/pkg/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var errGenerationFailed = errors.New("I wasn't able to generate a response. Please try again.")

type ParetoConfig struct {
	MaxGenerations  int
	BranchesPerGen  int
	TargetScore     float32
	ArchiveSize     int
	EnableCrossover bool
}

func GetParetoConfig(prefs *PreferencesStore, userID string) ParetoConfig {
	p := prefs.Get(userID)
	return ParetoConfig{
		MaxGenerations:  p.ParetoMaxGenerations,
		BranchesPerGen:  p.ParetoBranchesPerGen,
		TargetScore:     p.ParetoTargetScore,
		ArchiveSize:     p.ParetoArchiveSize,
		EnableCrossover: p.ParetoEnableCrossover,
	}
}

type PathCandidate struct {
	ID                 string
	Generation         int
	ParentIDs          []string
	StrategyPrompt     string
	AccumulatedLessons []string
	Trace              *ExecutionTrace
	Scores             PathScores
	Feedback           string
	CreatedAt          time.Time
}

// All scores are 1-5 star ratings where higher is better
type PathScores struct {
	Effectiveness       float64 // Primary: did we actually answer the question? (1-5 stars)
	AnswerQuality float64 // Quality of the answer content (1-5 stars)
	Hallucination float64 // Factual accuracy - no made up facts (1-5 stars)
	Specificity   float64 // Appropriate level of detail (1-5 stars)
	TokenCost     float64 // Token efficiency - exponential decay (1-5 stars)
	Latency       float64 // Response speed - exponential decay (1-5 stars)
}

type PathScoreWeights struct {
	Effectiveness       float64
	AnswerQuality float64
	Hallucination float64
	Specificity   float64
	TokenCost     float64
	Latency       float64
}

func DefaultPathScoreWeights() PathScoreWeights {
	return PathScoreWeights{
		Effectiveness:       0.35, // Most important - did we succeed?
		AnswerQuality: 0.25,
		Hallucination: 0.20,
		Specificity:   0.10,
		TokenCost:     0.05,
		Latency:       0.05,
	}
}

func (s PathScores) WeightedSum(weights PathScoreWeights) float64 {
	return s.Effectiveness*weights.Effectiveness +
		s.AnswerQuality*weights.AnswerQuality +
		s.Hallucination*weights.Hallucination +
		s.Specificity*weights.Specificity +
		s.TokenCost*weights.TokenCost +
		s.Latency*weights.Latency
}

type ExecutionTrace struct {
	Query          string
	ToolCalls      []ToolCallRecord
	ReasoningSteps []string
	FinalAnswer    string
	TotalTokens    int
	DurationMs     int64
}

type ToolCallRecord struct {
	ToolName  string
	Arguments map[string]any
	Result    any
	Success   bool
	Error     string
}

var fallbackSeedStrategies = []string{
	`You are solving a query task.
1. First understand what information is needed
2. Identify relevant data sources and relationships
3. Construct appropriate queries or tool calls
4. Verify results make sense before concluding
5. Synthesize findings into a clear, accurate answer`,
	`Approach this query methodically:
1. Break down the question into sub-questions
2. Address each sub-question with targeted tool use
3. Combine partial answers into a coherent response
4. Double-check for consistency`,
	`Focus on efficiency:
1. Identify the most direct path to the answer
2. Use minimal tool calls - prefer broader queries over multiple narrow ones
3. Avoid redundant operations
4. Provide a concise, accurate answer`,
	`Prioritize accuracy:
1. Gather comprehensive information first
2. Cross-reference data from multiple sources when possible
3. Be explicit about uncertainty
4. Prefer verified facts over inferences`,
	`Think step by step:
1. What is the user really asking?
2. What information do I need?
3. What's the best way to get it?
4. How do I present the answer clearly?`,
}

var seedStrategyPromptNames = []string{
	"alicia/pareto/seed-default",
	"alicia/pareto/seed-methodical",
	"alicia/pareto/seed-efficiency",
	"alicia/pareto/seed-accuracy",
	"alicia/pareto/seed-stepbystep",
}

func getSeedStrategies() []string {
	client := getLangfuseClient()

	if client == nil {
		return fallbackSeedStrategies
	}

	strategies := make([]string, len(seedStrategyPromptNames))
	for i, promptName := range seedStrategyPromptNames {
		prompt, err := client.GetPrompt(promptName, langfuse.WithLabel("production"))
		if err != nil {
			slog.Warn("failed to fetch strategy from langfuse, using fallback", "prompt", promptName, "error", err)
			strategies[i] = fallbackSeedStrategies[i%len(fallbackSeedStrategies)]
		} else {
			strategies[i] = prompt.GetText()
		}
	}

	return strategies
}

func createSeedCandidates(count int) []*PathCandidate {
	strategies := getSeedStrategies()

	candidates := make([]*PathCandidate, count)
	for i := 0; i < count; i++ {
		strategyIdx := i % len(strategies)
		candidates[i] = &PathCandidate{
			ID:                 NewID("path_"),
			Generation:         0,
			ParentIDs:          []string{},
			StrategyPrompt:     strategies[strategyIdx],
			AccumulatedLessons: []string{},
			CreatedAt:          time.Now(),
		}
	}
	return candidates
}

func runParetoExploration(ctx context.Context, convID, msgID, previousID, userQuery string, cfg GenerateConfig, paretoCfg ParetoConfig, deps AgentDeps) error {
	deps.Notifier.SetMessageID(msgID)
	deps.Notifier.SetPreviousID(previousID)
	deps.Notifier.SendThinking(ctx, msgID, "Starting pareto exploration...")

	// Initialize components
	archive := NewPathParetoArchive(paretoCfg.ArchiveSize)
	lfClient := getLangfuseClient()
	evaluator := NewPathEvaluator(deps.LLM, lfClient, convID, deps.UserID)
	mutator := NewPathMutator(deps.LLM, lfClient, convID, deps.UserID)

	// Extract trace ID from OTel context and store it for Langfuse correlation
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		traceID := spanCtx.TraceID().String()
		if err := UpdateMessageTraceID(ctx, deps.DB, msgID, traceID); err != nil {
			slog.ErrorContext(ctx, "failed to store trace_id for message", "message_id", msgID, "error", err)
		}
	}

	// --- Setup: load history, memories, tools ---
	setupCtx, setupSpan := otel.Tracer("alicia-agent").Start(ctx, "pareto.setup")

	messages, err := LoadConversationFull(setupCtx, deps.DB, convID)
	if err != nil {
		setupSpan.RecordError(err)
		setupSpan.End()
		slog.ErrorContext(setupCtx, "failed to load conversation history", "conversation_id", convID, "error", err)
		deps.Notifier.SendError(ctx, msgID, errGenerationFailed)
		return fmt.Errorf("load conversation: %w", err)
	}

	var memories []Memory
	embedding, err := deps.LLM.Embed(setupCtx, userQuery)
	if err != nil {
		slog.ErrorContext(setupCtx, "failed to generate embedding for memory search", "error", err)
	} else if embedding != nil {
		userPrefs := deps.Prefs.Get(deps.UserID)
		memories, err = SearchMemories(setupCtx, deps.DB, embedding, 0.7, userPrefs.MemoryRetrievalCount)
		if err != nil {
			slog.ErrorContext(setupCtx, "failed to search memories", "error", err)
		} else {
			for _, m := range memories {
				RecordMemoryUse(setupCtx, deps.DB, NewMemoryUseID(), m.ID, msgID, convID, m.Similarity)
				deps.Notifier.SendMemoryTrace(setupCtx, msgID, m.ID, m.Content, m.Similarity)
			}
		}
	}

	var tools []Tool
	if cfg.EnableTools {
		tools, _ = LoadTools(setupCtx, deps.DB)
		if deps.MCP != nil {
			tools = append(tools, deps.MCP.Tools()...)
		}
	}

	setupSpan.SetAttributes(
		attribute.Int("memory_count", len(memories)),
		attribute.Int("tool_count", len(tools)),
	)
	setupSpan.End()

	seeds := createSeedCandidates(paretoCfg.BranchesPerGen)
	weights := DefaultPathScoreWeights()

	// Load conversation title for progress context (may be empty for new conversations)
	convTitle, _ := GetConversationTitle(ctx, deps.DB, convID)

	for gen := 0; gen < paretoCfg.MaxGenerations; gen++ {
		genCtx, genSpan := otel.Tracer("alicia-agent").Start(ctx, "pareto.generation",
			trace.WithAttributes(
				attribute.Int("generation", gen+1),
				attribute.Int("branches", len(seeds)),
			))

		deps.Notifier.SendThinking(genCtx, msgID, fmt.Sprintf("Exploring generation %d/%d (%d branches)...", gen+1, paretoCfg.MaxGenerations, len(seeds)))

		var wg sync.WaitGroup
		var candidatesDone atomic.Int32
		var bestScoreX100 atomic.Int64
		tracker := &toolTracker{}
		results := make(chan *PathCandidate, len(seeds))

		totalCandidates := len(seeds)
		stopProgress := startProgressTicker(genCtx, deps.Notifier, msgID, 5*time.Second, func() string {
			done := int(candidatesDone.Load())
			best := float64(bestScoreX100.Load()) / 100.0
			remaining := totalCandidates - done
			if remaining == 0 {
				return ""
			}

			var status string
			if done == 0 {
				if gen == 0 {
					status = fmt.Sprintf("Running %d candidates in parallel...", totalCandidates)
				} else {
					status = fmt.Sprintf("Testing %d refined strategies...", totalCandidates)
				}
			} else if best > 0 {
				status = fmt.Sprintf("%d/%d evaluated (best: %.1f/5), %d still running...", done, totalCandidates, best, remaining)
			} else {
				status = fmt.Sprintf("%d/%d candidates evaluated, %d still running...", done, totalCandidates, remaining)
			}

			// Add LLM-generated tool activity description
			if toolDesc := tracker.Describe(genCtx, deps.LLM, userQuery, convID, deps.UserID); toolDesc != "" {
				status += "\n" + toolDesc
			}

			// Add context from best frontier answer so far
			bestAnswer := ""
			if archiveBest := archive.GetBestByWeightedSum(weights); archiveBest != nil && archiveBest.Trace != nil {
				bestAnswer = archiveBest.Trace.FinalAnswer
			}
			desc := buildProgressContext(userQuery, convTitle, bestAnswer)
			return status + "\n" + desc
		})

		for _, candidate := range seeds {
			wg.Add(1)
			go func(c *PathCandidate) {
				defer wg.Done()
				defer candidatesDone.Add(1)

				// Candidate execution span
				execCtx, execSpan := otel.Tracer("alicia-agent").Start(genCtx, "pareto.candidate_execution",
					trace.WithAttributes(
						attribute.String("candidate_id", c.ID),
						attribute.Int("generation", c.Generation),
					))

				execTrace, err := executeCandidateWithStrategy(execCtx, c, messages, memories, tools, userQuery, convID, cfg, deps, tracker)
				if err != nil {
					execSpan.RecordError(err)
					execSpan.End()
					slog.ErrorContext(execCtx, "candidate failed", "candidate_id", c.ID, "error", err)
					return
				}
				c.Trace = execTrace
				execSpan.SetAttributes(
					attribute.Int("tool_calls", len(execTrace.ToolCalls)),
					attribute.Int("total_tokens", execTrace.TotalTokens),
					attribute.Int64("duration_ms", execTrace.DurationMs),
				)
				execSpan.End()

				// Evaluation span
				_, evalSpan := otel.Tracer("alicia-agent").Start(genCtx, "pareto.candidate_evaluation",
					trace.WithAttributes(
						attribute.String("candidate_id", c.ID),
						attribute.Int("generation", c.Generation),
					))

				scores, feedback, evalErr := evaluator.Evaluate(genCtx, userQuery, execTrace)
				if evalErr != nil {
					evalSpan.RecordError(evalErr)
					evalSpan.End()
					slog.ErrorContext(genCtx, "evaluation failed", "candidate_id", c.ID, "error", evalErr)
					return
				}
				c.Scores = scores
				c.Feedback = feedback

				evalSpan.SetAttributes(
					attribute.Float64("score.weighted", scores.WeightedSum(weights)),
					attribute.Float64("score.effectiveness", scores.Effectiveness),
					attribute.Float64("score.quality", scores.AnswerQuality),
					attribute.Float64("score.hallucination", scores.Hallucination),
					attribute.Float64("score.specificity", scores.Specificity),
				)
				evalSpan.End()

				// Track best score for progress reporting
				scoreX100 := int64(scores.WeightedSum(weights) * 100)
				for {
					old := bestScoreX100.Load()
					if old >= scoreX100 {
						break
					}
					if bestScoreX100.CompareAndSwap(old, scoreX100) {
						break
					}
				}

				results <- c
			}(candidate)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		for c := range results {
			archive.Add(c)
		}
		stopProgress()

		if archive.Size() == 0 {
			genSpan.End()
			slog.ErrorContext(genCtx, "all candidates failed in generation", "generation", gen)
			deps.Notifier.SendError(ctx, msgID, errGenerationFailed)
			return fmt.Errorf("no candidates succeeded in generation %d", gen)
		}

		// Check for early termination
		best := archive.GetBestByWeightedSum(weights)
		bestScore := best.Scores.WeightedSum(weights)
		genSpan.SetAttributes(
			attribute.Float64("best_score", bestScore),
			attribute.Int("archive_size", archive.Size()),
		)

		progress := float32(bestScore * 20)
		if progress > 100 {
			progress = 100
		}

		statusMsg := generateThinkingStatus(genCtx, deps.LLM, userQuery, best.StrategyPrompt, convID, deps.UserID, progress)
		deps.Notifier.SendThinkingWithProgress(genCtx, msgID, statusMsg, progress)

		if bestScore >= float64(paretoCfg.TargetScore) {
			genSpan.End()
			break
		}

		if gen < paretoCfg.MaxGenerations-1 {
			_, mutSpan := otel.Tracer("alicia-agent").Start(genCtx, "pareto.mutation",
				trace.WithAttributes(
					attribute.Int("archive_size", archive.Size()),
				))

			deps.Notifier.SendThinking(genCtx, msgID, "Evolving strategies for next generation...")
			parents := archive.SelectForMutation(paretoCfg.BranchesPerGen)

			var mutWg sync.WaitGroup
			var mutationsDone atomic.Int32
			totalMutations := len(parents)
			mutResults := make(chan *PathCandidate, len(parents)+1)

			stopMutProgress := startProgressTicker(genCtx, deps.Notifier, msgID, 5*time.Second, func() string {
				done := int(mutationsDone.Load())
				if done == 0 || done >= totalMutations {
					return ""
				}
				status := fmt.Sprintf("Evolved %d/%d strategies...", done, totalMutations)
				bestAnswer := ""
				if archiveBest := archive.GetBestByWeightedSum(weights); archiveBest != nil && archiveBest.Trace != nil {
					bestAnswer = archiveBest.Trace.FinalAnswer
				}
				context := buildProgressContext(userQuery, convTitle, bestAnswer)
				return status + "\n" + context
			})

			for _, parent := range parents {
				mutWg.Add(1)
				go func(p *PathCandidate) {
					defer mutWg.Done()
					defer mutationsDone.Add(1)
					mutated, mutErr := mutator.MutateStrategy(genCtx, p, p.Trace, p.Feedback)
					if mutErr != nil {
						slog.ErrorContext(genCtx, "mutation failed", "candidate_id", p.ID, "error", mutErr)
						return
					}
					if mutated != nil {
						mutResults <- mutated
					}
				}(parent)
			}

			if paretoCfg.EnableCrossover && len(parents) >= 2 {
				mutWg.Add(1)
				go func() {
					defer mutWg.Done()
					crossed, crossErr := mutator.Crossover(genCtx, parents[0], parents[1])
					if crossErr == nil && crossed != nil {
						mutResults <- crossed
					}
				}()
			}

			go func() {
				mutWg.Wait()
				close(mutResults)
			}()

			seeds = seeds[:0]
			for mutated := range mutResults {
				seeds = append(seeds, mutated)
			}
			stopMutProgress()

			if len(seeds) == 0 {
				slog.WarnContext(genCtx, "no mutations produced, using parents for next generation")
				seeds = parents
			}

			mutSpan.SetAttributes(attribute.Int("mutated_count", len(seeds)))
			mutSpan.End()
		}

		genSpan.End()
	}

	// Select best result
	best := archive.GetBestByWeightedSum(weights)
	if best == nil || best.Trace == nil {
		slog.ErrorContext(ctx, "pareto exploration produced no results, archive empty or best has nil trace")
		deps.Notifier.SendError(ctx, msgID, errGenerationFailed)
		return fmt.Errorf("no results")
	}

	slog.InfoContext(ctx, "pareto exploration complete", "candidate_id", best.ID, "weighted_score", best.Scores.WeightedSum(weights))

	// Save tool uses from the best trace
	for _, tc := range best.Trace.ToolCalls {
		tu := ToolUse{
			ID:        NewToolUseID(),
			ToolName:  tc.ToolName,
			Arguments: tc.Arguments,
			Result:    tc.Result,
			Success:   tc.Success,
			Error:     tc.Error,
		}
		SaveToolUse(ctx, deps.DB, msgID, tu)
	}

	// Save and notify
	finalContent := strings.TrimSpace(best.Trace.FinalAnswer)
	if finalContent == "" {
		slog.ErrorContext(ctx, "best candidate has empty final answer", "candidate_id", best.ID)
		deps.Notifier.SendError(ctx, msgID, errGenerationFailed)
		return fmt.Errorf("best candidate has empty content")
	}
	reasoning := strings.Join(best.Trace.ReasoningSteps, "\n\n")
	if err := UpdateMessage(ctx, deps.DB, msgID, finalContent, reasoning, "completed"); err != nil {
		deps.Notifier.SendError(ctx, msgID, errGenerationFailed)
		return err
	}

	deps.Notifier.SendComplete(ctx, msgID, finalContent)
	slog.InfoContext(ctx, "response complete", "message_id", msgID, "content_length", len(finalContent))

	// Update title asynchronously (detached context with timeout to survive client disconnect)
	titleCtx, titleCancel := context.WithTimeout(context.Background(), 45*time.Second)
	go func() {
		defer titleCancel()
		maybeUpdateTitle(titleCtx, deps, convID, userQuery, finalContent)
	}()

	// Extract and save memories asynchronously (detached context to survive client disconnect)
	// Carry span context for Langfuse score ingestion
	memCtx := trace.ContextWithSpanContext(context.Background(), trace.SpanFromContext(ctx).SpanContext())
	go ExtractAndSaveMemories(memCtx, convID, deps)

	return nil
}

func executeCandidateWithStrategy(ctx context.Context, candidate *PathCandidate, history []Message, memories []Memory, tools []Tool, userQuery, convID string, cfg GenerateConfig, deps AgentDeps, tracker *toolTracker) (*ExecutionTrace, error) {
	startTime := time.Now()

	execTrace := &ExecutionTrace{
		Query:     userQuery,
		ToolCalls: []ToolCallRecord{},
	}

	// Build messages with strategy injected
	llmMsgs, systemPrompt := buildMessagesWithStrategy(history, memories, tools, candidate.StrategyPrompt, candidate.AccumulatedLessons)

	traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
	totalTokens := 0
	emptyRetryTemperatures := []float32{0.3, 0.7, 1.0}

	for i := 0; i < cfg.MaxToolIterations; i++ {
		llmStart := time.Now()
		resp, err := deps.LLM.ChatWithOptions(ctx, llmMsgs, tools, ChatOptions{
			GenerationName: "pareto.candidate",
			PromptName:     systemPrompt.Name,
			PromptVersion:  systemPrompt.Version,
		})
		if err != nil {
			return nil, fmt.Errorf("LLM chat failed: %w", err)
		}
		llmEnd := time.Now()

		if systemPrompt.Name != "" {
			genID := fmt.Sprintf("%s-iter-%d", candidate.ID, i)
			go sendGenerationToLangfuse(traceID, genID, convID, deps.UserID, deps.LLM.model, "agent:pareto", "pareto.candidate", systemPrompt, resp.Content, llmStart, llmEnd)
		}

		totalTokens += len(resp.Content) / 4

		// Handle empty response with no tool calls - retry with increasing temperature
		if len(resp.ToolCalls) == 0 && strings.TrimSpace(resp.Content) == "" {
			for retry, temp := range emptyRetryTemperatures {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				slog.WarnContext(ctx, "empty response, retrying", "temperature", temp, "attempt", retry+1, "max_attempts", len(emptyRetryTemperatures))
				retryStart := time.Now()
				resp, err = deps.LLM.ChatWithOptions(ctx, llmMsgs, tools, ChatOptions{
					Temperature:    float32Ptr(temp),
					GenerationName: "pareto.candidate",
					PromptName:     systemPrompt.Name,
					PromptVersion:  systemPrompt.Version,
				})
				if err != nil {
					slog.ErrorContext(ctx, "retry failed", "attempt", retry+1, "error", err)
					continue
				}
				retryEnd := time.Now()
				if systemPrompt.Name != "" {
					genID := fmt.Sprintf("%s-iter-%d-retry-%d", candidate.ID, i, retry)
					go sendGenerationToLangfuse(traceID, genID, convID, deps.UserID, deps.LLM.model, "agent:pareto", "pareto.candidate", systemPrompt, resp.Content, retryStart, retryEnd)
				}
				if strings.TrimSpace(resp.Content) != "" || len(resp.ToolCalls) > 0 {
					break
				}
			}
			if strings.TrimSpace(resp.Content) == "" && len(resp.ToolCalls) == 0 {
				return nil, fmt.Errorf("LLM returned empty response after %d retries", len(emptyRetryTemperatures))
			}
		}

		if resp.Reasoning != "" {
			execTrace.ReasoningSteps = append(execTrace.ReasoningSteps, resp.Reasoning)
		}

		// No tool calls means we have our final answer
		if len(resp.ToolCalls) == 0 {
			execTrace.FinalAnswer = resp.Content
			slog.InfoContext(ctx, "final answer received", "content_length", len(execTrace.FinalAnswer))
			break
		}

		llmMsgs = append(llmMsgs, LLMMessage{Role: "assistant", Content: resp.Content, ToolCalls: resp.ToolCalls})

		for _, tc := range resp.ToolCalls {
			mcpName := tc.Name
			if strings.HasPrefix(mcpName, "mcp_garden_") {
				mcpName = strings.TrimPrefix(mcpName, "mcp_garden_")
			}

			if tracker != nil {
				tracker.Record(mcpName, tc.Arguments)
			}

			if deps.MCP == nil {
				return nil, fmt.Errorf("MCP not available for tool call: %s", tc.Name)
			}

			result, execErr := deps.MCP.Call(ctx, mcpName, tc.Arguments)

			record := ToolCallRecord{
				ToolName:  tc.Name,
				Arguments: tc.Arguments,
			}

			var toolMsg LLMMessage
			if execErr != nil {
				record.Success = false
				record.Error = execErr.Error()
				toolMsg = LLMMessage{Role: "tool", Content: "Error: " + execErr.Error(), ToolCallID: tc.ID}
			} else {
				record.Success = true
				record.Result = result
				toolMsg = LLMMessage{Role: "tool", Content: fmt.Sprintf("%v", result), ToolCallID: tc.ID}
			}

			execTrace.ToolCalls = append(execTrace.ToolCalls, record)
			llmMsgs = append(llmMsgs, toolMsg)
		}

		if i == cfg.MaxToolIterations-1 {
			execTrace.FinalAnswer = resp.Content
			slog.WarnContext(ctx, "max iterations reached", "content_length", len(execTrace.FinalAnswer))
			if execTrace.FinalAnswer == "" {
				execTrace.FinalAnswer = "Max tool iterations reached."
			}
		}
	}

	if strings.TrimSpace(execTrace.FinalAnswer) == "" {
		return nil, fmt.Errorf("empty response after all iterations")
	}

	execTrace.TotalTokens = totalTokens
	execTrace.DurationMs = time.Since(startTime).Milliseconds()

	return execTrace, nil
}

const fallbackThinkingStatusPrompt = `Generate a short, fun status message (1-10 words) about working on this question.

Question: {{question}}
Current approach: {{strategy}}
Progress: {{progress}}%

Be witty, playful, or encouraging. Output ONLY the message, nothing else.`

func generateThinkingStatus(ctx context.Context, llm *LLMClient, question, strategy, convID, userID string, progress float32) string {
	vars := map[string]string{
		"question": question,
		"strategy": strategy,
		"progress": fmt.Sprintf("%.0f", progress),
	}
	prompt := getPrompt("alicia/pareto/thinking-status", fallbackThinkingStatusPrompt, vars)

	llmStart := time.Now()
	resp, err := llm.ChatWithOptions(ctx, []LLMMessage{{Role: "user", Content: prompt.Text}}, nil, ChatOptions{
		GenerationName: "pareto.thinking_status",
		PromptName:     prompt.Name,
		PromptVersion:  prompt.Version,
	})
	if err != nil {
		return fmt.Sprintf("Exploring... %.0f%%", progress)
	}
	llmEnd := time.Now()

	if prompt.Name != "" {
		traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
		genID := fmt.Sprintf("thinking-status-%d", llmStart.UnixNano())
		go sendGenerationToLangfuse(traceID, genID, convID, userID, llm.model, "agent:pareto", "pareto.thinking_status", prompt, resp.Content, llmStart, llmEnd)
	}

	status := strings.TrimSpace(resp.Content)
	status = strings.Trim(status, "\"'")
	if len(status) > 100 {
		status = status[:100]
	}
	return status
}

// toolTracker records tool calls across concurrent candidates for progress reporting
type toolTracker struct {
	mu          sync.Mutex
	calls       []toolCallEntry
	cachedDesc  string
	cachedCount int
}

type toolCallEntry struct {
	Name string
	Args string
}

func (t *toolTracker) Record(name string, args map[string]any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	argStr := formatToolArgs(args)
	t.calls = append(t.calls, toolCallEntry{Name: name, Args: argStr})
}

// Describe generates a natural language one-liner describing current tool activity.
// Uses an LLM with caching — only regenerates when new calls have been recorded.
func (t *toolTracker) Describe(ctx context.Context, llm *LLMClient, userQuery, convID, userID string) string {
	t.mu.Lock()
	count := len(t.calls)
	if count == 0 {
		t.mu.Unlock()
		return ""
	}
	if count == t.cachedCount && t.cachedDesc != "" {
		desc := t.cachedDesc
		t.mu.Unlock()
		return desc
	}
	calls := make([]toolCallEntry, count)
	copy(calls, t.calls)
	t.mu.Unlock()

	desc := generateToolDescription(ctx, llm, userQuery, convID, userID, calls)

	t.mu.Lock()
	if len(t.calls) == count {
		t.cachedDesc = desc
		t.cachedCount = count
	}
	t.mu.Unlock()

	return desc
}

func formatToolArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	for _, key := range []string{"query", "sql", "table", "url", "search_query"} {
		if v, ok := args[key]; ok {
			return fmt.Sprintf("%s=%s", key, truncateStr(fmt.Sprintf("%v", v), 120))
		}
	}
	return truncateStr(fmt.Sprintf("%v", args), 120)
}

const fallbackToolDescPrompt = `Summarize in 5-15 words what these tool operations are doing to answer the user's question. Be specific about data being accessed.

User question: {{question}}

Operations:
{{operations}}

Write ONLY a short natural description. Examples:
- "Exploring database schema to find message tables"
- "Querying recent WhatsApp messages from the database"
- "Searching the web for concert tour dates"
- "Running SQL to count orders by customer"
No quotes, no prefix.`

func generateToolDescription(ctx context.Context, llm *LLMClient, userQuery, convID, userID string, calls []toolCallEntry) string {
	// Build operations list
	var ops []string
	for _, c := range calls {
		if c.Args != "" {
			ops = append(ops, fmt.Sprintf("- %s(%s)", c.Name, c.Args))
		} else {
			ops = append(ops, fmt.Sprintf("- %s()", c.Name))
		}
	}

	vars := map[string]string{
		"question":   truncateStr(userQuery, 200),
		"operations": strings.Join(ops, "\n"),
	}
	prompt := getPrompt("alicia/pareto/tool-description", fallbackToolDescPrompt, vars)

	descCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	llmStart := time.Now()
	resp, err := llm.ChatWithOptions(descCtx, []LLMMessage{{Role: "user", Content: prompt.Text}}, nil, ChatOptions{
		GenerationName: "pareto.tool_description",
		PromptName:     prompt.Name,
		PromptVersion:  prompt.Version,
	})
	if err != nil {
		// Fallback: simple list format
		return fallbackToolSummary(calls)
	}
	llmEnd := time.Now()

	if prompt.Name != "" {
		traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
		genID := fmt.Sprintf("tool-desc-%d", llmStart.UnixNano())
		go sendGenerationToLangfuse(traceID, genID, convID, userID, llm.model, "agent:pareto", "pareto.tool_description", prompt, resp.Content, llmStart, llmEnd)
	}

	desc := strings.TrimSpace(resp.Content)
	desc = strings.Trim(desc, "\"'")
	if len(desc) > 120 {
		desc = desc[:120]
	}
	return desc
}

func fallbackToolSummary(calls []toolCallEntry) string {
	counts := make(map[string]int)
	var order []string
	for _, c := range calls {
		if counts[c.Name] == 0 {
			order = append(order, c.Name)
		}
		counts[c.Name]++
	}
	var parts []string
	for _, name := range order {
		if counts[name] > 1 {
			parts = append(parts, fmt.Sprintf("%s x%d", name, counts[name]))
		} else {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, ", ")
}

func truncateStr(s string, maxLen int) string {
	return langfuse.TruncateString(strings.TrimSpace(s), maxLen, "...")
}

func snippetFirstLast(s string, first, last int) string {
	s = strings.TrimSpace(s)
	if len(s) <= first+last+3 {
		return s
	}
	return s[:first] + "..." + s[len(s)-last:]
}

func buildProgressContext(userQuery, title, bestAnswer string) string {
	var parts []string
	q := truncateStr(userQuery, 200)
	if title != "" {
		parts = append(parts, fmt.Sprintf("[%s] %s", title, q))
	} else {
		parts = append(parts, q)
	}
	if bestAnswer != "" {
		parts = append(parts, "Best: "+snippetFirstLast(bestAnswer, 100, 100))
	}
	return strings.Join(parts, "\n")
}

func startProgressTicker(ctx context.Context, notifier Notifier, msgID string, interval time.Duration, statusFn func() string) func() {
	tickCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-tickCtx.Done():
				return
			case <-ticker.C:
				if msg := statusFn(); msg != "" {
					notifier.SendThinking(ctx, msgID, msg)
				}
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

func buildMessagesWithStrategy(history []Message, memories []Memory, tools []Tool, strategy string, lessons []string) ([]LLMMessage, PromptResult) {
	var msgs []LLMMessage

	instructions := "## Approach Strategy\n" + strategy
	if len(lessons) > 0 {
		instructions += "\n\n## Lessons from previous attempts\n"
		for _, lesson := range lessons {
			instructions += "- " + lesson + "\n"
		}
	}

	systemPrompt := getSystemPrompt(memories, nil, tools, instructions)

	msgs = append(msgs, LLMMessage{Role: "system", Content: systemPrompt.Text})

	for _, m := range history {
		if m.Role == "system" {
			continue
		}
		msgs = append(msgs, LLMMessage{Role: m.Role, Content: m.Content})
		for _, tu := range m.ToolUses {
			content := fmt.Sprintf("[%s] %v", tu.ToolName, tu.Result)
			if !tu.Success {
				content = fmt.Sprintf("[%s] Error: %s", tu.ToolName, tu.Error)
			}
			msgs = append(msgs, LLMMessage{Role: "tool", Content: content})
		}
	}

	return msgs, systemPrompt
}
