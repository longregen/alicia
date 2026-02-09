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
	deps.Notifier.SendStartAnswer(ctx, msgID)
	deps.Notifier.SendThinking(ctx, msgID, "Thinking...")

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
	} else if len(embedding) > 0 {
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
	// Always add final_answer tool to force responses through function calling API
	tools = append(tools, FinalAnswerTool())

	setupSpan.SetAttributes(
		attribute.Int("memory_count", len(memories)),
		attribute.Int("tool_count", len(tools)),
	)
	setupSpan.End()

	seeds := createSeedCandidates(paretoCfg.BranchesPerGen)
	weights := DefaultPathScoreWeights()

	for gen := 0; gen < paretoCfg.MaxGenerations; gen++ {
		genCtx, genSpan := otel.Tracer("alicia-agent").Start(ctx, "pareto.generation",
			trace.WithAttributes(
				attribute.Int("generation", gen+1),
				attribute.Int("branches", len(seeds)),
			))

		// Use the OTel span ID as the Langfuse span ID so LiteLLM observations
		// (which inherit the OTel parent) nest correctly under our Langfuse spans.
		genOTelSpanID := genSpan.SpanContext().SpanID().String()
		genStart := time.Now()
		if lfClient != nil {
			traceID := genSpan.SpanContext().TraceID().String()
			_ = lfClient.CreateSpan(genCtx, langfuse.SpanParams{
				TraceID:   traceID,
				ID:        genOTelSpanID,
				Name:      fmt.Sprintf("pareto.generation-%d", gen+1),
				StartTime: genStart,
				Metadata:  map[string]any{"generation": gen + 1, "branches": len(seeds)},
			})
		}
		genParentSpanID := genOTelSpanID

		var wg sync.WaitGroup
		var candidatesDone atomic.Int32
		var bestScoreX100 atomic.Int64
		tracker := &toolTracker{}
		results := make(chan *PathCandidate, len(seeds))

		totalCandidates := len(seeds)
		stopProgress := startProgressTicker(genCtx, deps.Notifier, msgID, 5*time.Second, func() string {
			remaining := totalCandidates - int(candidatesDone.Load())
			if remaining == 0 {
				return ""
			}

			if toolDesc := tracker.Describe(genCtx, deps.LLM, userQuery, convID, deps.UserID, genOTelSpanID); toolDesc != "" {
				return toolDesc
			}
			return ""
		})

		for i, candidate := range seeds {
			wg.Add(1)
			go func(c *PathCandidate, idx int) {
				defer wg.Done()
				defer candidatesDone.Add(1)

				// Candidate execution span (OTel)
				execCtx, execSpan := otel.Tracer("alicia-agent").Start(genCtx, "pareto.candidate_execution",
					trace.WithAttributes(
						attribute.String("candidate_id", c.ID),
						attribute.Int("generation", c.Generation),
					))

				// Create a per-candidate Langfuse span using the OTel span ID
				candidateSpanID := execSpan.SpanContext().SpanID().String()
				candidateStart := time.Now()
				if lfClient != nil {
					traceID := execSpan.SpanContext().TraceID().String()
					_ = lfClient.CreateSpan(execCtx, langfuse.SpanParams{
						TraceID:             traceID,
						ID:                  candidateSpanID,
						ParentObservationID: genParentSpanID,
						Name:                fmt.Sprintf("pareto.candidate-%d", idx+1),
						StartTime:           candidateStart,
						Metadata:            map[string]any{"candidate_id": c.ID, "generation": c.Generation},
					})
				}

				execTrace, err := executeCandidateWithStrategy(execCtx, c, messages, memories, tools, userQuery, convID, cfg, deps, tracker, candidateSpanID)
				if err != nil {
					execSpan.RecordError(err)
					execSpan.End()
					// Close per-candidate Langfuse span on failure
					if lfClient != nil {
						traceID := execSpan.SpanContext().TraceID().String()
						_ = lfClient.UpdateSpan(execCtx, langfuse.SpanParams{
							TraceID: traceID,
							ID:      candidateSpanID,
							EndTime: time.Now(),
							Output:  map[string]any{"error": err.Error()},
						})
					}
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

				// Evaluation span (OTel)
				_, evalSpan := otel.Tracer("alicia-agent").Start(genCtx, "pareto.candidate_evaluation",
					trace.WithAttributes(
						attribute.String("candidate_id", c.ID),
						attribute.Int("generation", c.Generation),
					))

				scores, feedback, evalErr := evaluator.Evaluate(genCtx, userQuery, execTrace, candidateSpanID)
				if evalErr != nil {
					evalSpan.RecordError(evalErr)
					evalSpan.End()
					// Close per-candidate Langfuse span on eval failure
					if lfClient != nil {
						spanCtx := trace.SpanContextFromContext(genCtx)
						if spanCtx.IsValid() {
							_ = lfClient.UpdateSpan(genCtx, langfuse.SpanParams{
								TraceID: spanCtx.TraceID().String(),
								ID:      candidateSpanID,
								EndTime: time.Now(),
								Output:  map[string]any{"error": evalErr.Error()},
							})
						}
					}
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

				// Close per-candidate Langfuse span on success
				if lfClient != nil {
					spanCtx := trace.SpanContextFromContext(genCtx)
					if spanCtx.IsValid() {
						ws := scores.WeightedSum(weights)
						_ = lfClient.UpdateSpan(genCtx, langfuse.SpanParams{
							TraceID: spanCtx.TraceID().String(),
							ID:      candidateSpanID,
							EndTime: time.Now(),
							Input:   c.Trace.FinalAnswer,
							Output: map[string]any{
								"weighted_score": ws,
								"feedback":       feedback,
							},
							Metadata: map[string]any{
								"pareto.score.effectiveness":  scores.Effectiveness,
								"pareto.score.answer_quality": scores.AnswerQuality,
								"pareto.score.hallucination":  scores.Hallucination,
								"pareto.score.specificity":    scores.Specificity,
								"pareto.score.token_cost":     scores.TokenCost,
								"pareto.score.latency":        scores.Latency,
								"pareto.score.weighted":       ws,
								"gen_ai.usage.input_tokens":   c.Trace.TotalTokens,
								"gen_ai.tool.call_count":      len(c.Trace.ToolCalls),
							},
						})
					}
				}

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
			}(candidate, i)
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

		if bestScore >= float64(paretoCfg.TargetScore) {
			genSpan.End()
			if lfClient != nil {
				spanCtx := trace.SpanContextFromContext(ctx)
				if spanCtx.IsValid() {
					_ = lfClient.UpdateSpan(genCtx, langfuse.SpanParams{
						TraceID: spanCtx.TraceID().String(),
						ID:      genOTelSpanID,
						EndTime: time.Now(),
						Output:  map[string]any{"best_score": bestScore, "early_termination": true},
					})
				}
			}
			break
		}

		if gen < paretoCfg.MaxGenerations-1 {
			_, mutSpan := otel.Tracer("alicia-agent").Start(genCtx, "pareto.mutation",
				trace.WithAttributes(
					attribute.Int("archive_size", archive.Size()),
				))

			// Create a Langfuse span for the mutation round, using the OTel span ID
			// so LiteLLM observations nest correctly.
			mutOTelSpanID := mutSpan.SpanContext().SpanID().String()
			mutStart := time.Now()
			if lfClient != nil {
				traceID := mutSpan.SpanContext().TraceID().String()
				_ = lfClient.CreateSpan(genCtx, langfuse.SpanParams{
					TraceID:             traceID,
					ID:                  mutOTelSpanID,
					ParentObservationID: genOTelSpanID,
					Name:                fmt.Sprintf("pareto.mutation-%d", gen+1),
					StartTime:           mutStart,
					Metadata:            map[string]any{"generation": gen + 1},
				})
			}
			mutator.parentSpanID = mutOTelSpanID

			parents := archive.SelectForMutation(paretoCfg.BranchesPerGen)

			var mutWg sync.WaitGroup
			mutResults := make(chan *PathCandidate, len(parents)+1)


			for _, parent := range parents {
				mutWg.Add(1)
				go func(p *PathCandidate) {
					defer mutWg.Done()
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

			if len(seeds) == 0 {
				slog.WarnContext(genCtx, "no mutations produced, using parents for next generation")
				seeds = parents
			}

			mutSpan.SetAttributes(attribute.Int("mutated_count", len(seeds)))
			mutSpan.End()

			// Close the Langfuse mutation span
			if lfClient != nil {
				spanCtx := trace.SpanContextFromContext(ctx)
				if spanCtx.IsValid() {
					_ = lfClient.UpdateSpan(genCtx, langfuse.SpanParams{
						TraceID: spanCtx.TraceID().String(),
						ID:      mutOTelSpanID,
						EndTime: time.Now(),
					})
				}
			}
		}

		genSpan.End()

		// Close the Langfuse generation span
		if lfClient != nil {
			spanCtx := trace.SpanContextFromContext(ctx)
			if spanCtx.IsValid() {
				_ = lfClient.UpdateSpan(genCtx, langfuse.SpanParams{
					TraceID: spanCtx.TraceID().String(),
					ID:      genOTelSpanID,
					EndTime: time.Now(),
					Output: map[string]any{
						"best_score":   bestScore,
						"archive_size": archive.Size(),
					},
				})
			}
		}
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
	// Carry span context so LiteLLM associates the request with the right trace
	titleCtx, titleCancel := context.WithTimeout(
		trace.ContextWithSpanContext(context.Background(), trace.SpanFromContext(ctx).SpanContext()),
		45*time.Second,
	)
	go func() {
		defer titleCancel()
		maybeUpdateTitle(titleCtx, deps, convID, userQuery, finalContent)
	}()

	// Extract and save memories asynchronously (detached context to survive client disconnect)
	go ExtractAndSaveMemories(context.Background(), convID, msgID, deps)

	return nil
}

func executeCandidateWithStrategy(ctx context.Context, candidate *PathCandidate, history []Message, memories []Memory, tools []Tool, userQuery, convID string, cfg GenerateConfig, deps AgentDeps, tracker *toolTracker, parentSpanID string) (*ExecutionTrace, error) {
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

	userPrefs := deps.Prefs.Get(deps.UserID)
	temperature := float32Ptr(userPrefs.Temperature)

	for i := 0; i < cfg.MaxToolIterations; i++ {
		llmStart := time.Now()
		// Use NoTelemetry because we need deferred send with IterToolCalls after tool execution
		resp, err := MakeLLMCall(ctx, deps.LLM, llmMsgs, tools, LLMCallOptions{
			Temperature:    temperature,
			ToolChoice:     "auto",
			GenerationName: "pareto.candidate",
			Prompt:         systemPrompt,
			NoTelemetry:    true,
		})
		if err != nil {
			return nil, fmt.Errorf("LLM chat failed: %w", err)
		}
		llmEnd := time.Now()

		// Deferred: send generation to Langfuse after tool calls are collected
		iterGenID := fmt.Sprintf("%s-iter-%d", candidate.ID, i)
		iterResp := resp
		iterStart := llmStart
		iterEnd := llmEnd
		iterTemp := temperature
		sendIterGeneration := func(iterToolCalls []ToolCallRecord) {
			go sendGenerationToLangfuse(LangfuseGeneration{
				TraceID: traceID, ID: iterGenID, ParentObservationID: parentSpanID,
				ConvID: convID, UserID: deps.UserID, Model: deps.LLM.model,
				TraceName: "agent:pareto", GenerationName: "pareto.candidate",
				Prompt: systemPrompt, Input: llmMsgs, Output: formatIterOutput(iterResp, iterToolCalls),
				StartTime: iterStart, EndTime: iterEnd,
				PromptTokens: iterResp.PromptTokens, CompletionTokens: iterResp.CompletionTokens, TotalTokens: iterResp.TotalTokens,
				Temperature: iterTemp, MaxTokens: deps.LLM.maxTokens,
				Tools: toolNames(tools), ReasoningTokens: iterResp.ReasoningTokens,
				Reasoning: iterResp.Reasoning, FinishReason: iterResp.FinishReason, IterToolCalls: iterToolCalls,
			})
		}

		totalTokens += resp.TotalTokens

		// Handle empty response with no tool calls - retry with increasing temperature
		if len(resp.ToolCalls) == 0 && strings.TrimSpace(resp.Content) == "" {
			for retry, temp := range emptyRetryTemperatures {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				slog.WarnContext(ctx, "empty response, retrying", "temperature", temp, "attempt", retry+1, "max_attempts", len(emptyRetryTemperatures))
				retryTemp := float32Ptr(temp)
				resp, err = MakeLLMCall(ctx, deps.LLM, llmMsgs, tools, LLMCallOptions{
					Temperature:         retryTemp,
					ToolChoice:          "auto",
					GenerationName:      "pareto.candidate",
					Prompt:              systemPrompt,
					ConvID:              convID,
					UserID:              deps.UserID,
					TraceName:           "agent:pareto",
					ParentObservationID: parentSpanID,
				})
				if err != nil {
					slog.ErrorContext(ctx, "retry failed", "attempt", retry+1, "error", err)
					continue
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

		// Check for final_answer tool call first
		var foundFinalAnswer bool
		for _, tc := range resp.ToolCalls {
			if IsFinalAnswerCall(tc) {
				execTrace.FinalAnswer = ExtractFinalAnswer(tc)
				foundFinalAnswer = true
				slog.InfoContext(ctx, "final answer received via tool", "content_length", len(execTrace.FinalAnswer))
				break
			}
		}
		if foundFinalAnswer {
			sendIterGeneration(nil)
			break
		}

		// No tool calls — plain text response, use content directly
		if len(resp.ToolCalls) == 0 {
			execTrace.FinalAnswer = resp.Content
			sendIterGeneration(nil)
			break
		}

		llmMsgs = append(llmMsgs, LLMMessage{Role: "assistant", Content: resp.Content, ToolCalls: resp.ToolCalls})

		var iterToolCalls []ToolCallRecord
		for _, tc := range resp.ToolCalls {
			// Skip final_answer - it's not a real tool to execute
			if IsFinalAnswerCall(tc) {
				continue
			}
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

			iterToolCalls = append(iterToolCalls, record)
			execTrace.ToolCalls = append(execTrace.ToolCalls, record)
			llmMsgs = append(llmMsgs, toolMsg)
		}
		sendIterGeneration(iterToolCalls)

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
func (t *toolTracker) Describe(ctx context.Context, llm *LLMClient, userQuery, convID, userID, parentSpanID string) string {
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

	desc := generateToolDescription(ctx, llm, userQuery, convID, userID, calls, parentSpanID)

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

func generateToolDescription(ctx context.Context, llm *LLMClient, userQuery, convID, userID string, calls []toolCallEntry, parentSpanID string) string {
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
	prompt := RetrievePromptTemplate("alicia/pareto/tool-description", fallbackToolDescPrompt, vars)

	descCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	resp, err := MakeLLMCall(descCtx, llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, nil, LLMCallOptions{
		GenerationName:      "pareto.tool_description",
		Prompt:              prompt,
		ConvID:              convID,
		UserID:              userID,
		TraceName:           "agent:pareto",
		ParentObservationID: parentSpanID,
		NoRetry:             true,
	})
	if err != nil {
		// Fallback: simple list format
		return fallbackToolSummary(calls)
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

// formatIterOutput builds a structured output for a candidate iteration generation,
// including the LLM content, tool call requests, and their execution results.
func formatIterOutput(resp *LLMResponse, toolCalls []ToolCallRecord) any {
	if len(resp.ToolCalls) == 0 && len(toolCalls) == 0 {
		return resp.Content
	}
	out := map[string]any{
		"content": resp.Content,
	}
	if len(resp.ToolCalls) > 0 {
		requests := make([]map[string]any, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			requests[i] = map[string]any{
				"tool": tc.Name,
				"args": tc.Arguments,
			}
		}
		out["tool_requests"] = requests
	}
	if len(toolCalls) > 0 {
		results := make([]map[string]any, len(toolCalls))
		for i, tc := range toolCalls {
			r := map[string]any{
				"tool":    tc.ToolName,
				"success": tc.Success,
			}
			if tc.Success {
				r["result"] = tc.Result
			} else {
				r["error"] = tc.Error
			}
			results[i] = r
		}
		out["tool_results"] = results
	}
	return out
}

func truncateStr(s string, maxLen int) string {
	return langfuse.TruncateString(strings.TrimSpace(s), maxLen, "...")
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

		if m.Role == "assistant" && len(m.ToolUses) > 0 {
			toolCalls := make([]LLMToolCall, len(m.ToolUses))
			for i, tu := range m.ToolUses {
				toolCalls[i] = LLMToolCall{
					ID:        tu.ID,
					Name:      tu.ToolName,
					Arguments: tu.Arguments,
				}
			}
			msgs = append(msgs, LLMMessage{Role: "assistant", Content: m.Content, ToolCalls: toolCalls})
			for _, tu := range m.ToolUses {
				content := fmt.Sprintf("%v", tu.Result)
				if !tu.Success {
					content = "Error: " + tu.Error
				}
				msgs = append(msgs, LLMMessage{Role: "tool", Content: content, ToolCallID: tu.ID})
			}
		} else {
			msgs = append(msgs, LLMMessage{Role: m.Role, Content: m.Content})
		}
	}

	return msgs, systemPrompt
}
