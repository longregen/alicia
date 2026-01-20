package usecases

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

// Type aliases for backwards compatibility and convenience.
// These types are defined in ports and re-exported here for ease of use.
type (
	// ParetoResponseConfig configures the Pareto-based response generation.
	// See ports.ParetoResponseConfig for full documentation.
	ParetoResponseConfig = ports.ParetoResponseConfig

	// ParetoResponseInput contains the input parameters for Pareto-based response generation.
	// See ports.ParetoResponseInput for full documentation.
	ParetoResponseInput = ports.ParetoResponseInput

	// ParetoResponseOutput contains the result of Pareto-based response generation.
	// See ports.ParetoResponseOutput for full documentation.
	ParetoResponseOutput = ports.ParetoResponseOutput
)

// DefaultParetoResponseConfig returns sensible defaults for Pareto response generation.
func DefaultParetoResponseConfig() *ParetoResponseConfig {
	return &ParetoResponseConfig{
		MaxGenerations:         3,   // Fewer generations for response generation (faster)
		BranchesPerGen:         2,   // Explore 2 paths per generation
		TargetScore:            0.8, // Target 80% quality score
		MaxToolCalls:           50,  // Budget: total tool calls across all paths
		MaxLLMCalls:            30,  // Budget: total LLM calls across all paths
		ParetoArchiveSize:      20,  // Keep up to 20 Pareto-optimal candidates
		EnableCrossover:        true,
		ExecutionTimeoutMs:     60000, // 60 seconds per path execution
		EnableParallelBranches: true,
		MaxParallelBranches:    3,
		MaxToolLoopIterations:  5,
	}
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
type ParetoResponseGenerator struct {
	// Core LLM services
	llmService    ports.LLMService
	reflectionLLM ports.LLMService // Optional stronger model for mutation/reflection

	// Repositories
	messageRepo      ports.MessageRepository
	conversationRepo ports.ConversationRepository
	toolRepo         ports.ToolRepository
	sentenceRepo     ports.SentenceRepository
	toolUseRepo      ports.ToolUseRepository
	reasoningRepo    ports.ReasoningStepRepository
	memoryUsageRepo  ports.MemoryUsageRepository

	// Services
	toolService   ports.ToolService
	memoryService ports.MemoryService

	// ID generator
	idGenerator ports.IDGenerator

	// Transaction manager
	txManager ports.TransactionManager

	// GEPA components (shared, stateless)
	mutator   *prompt.PathMutator
	evaluator *baselines.PathEvaluator

	// Default configuration
	config *ParetoResponseConfig

	// Title generator for new conversations
	titleGenerator *GenerateConversationTitle

	// Memory extraction
	extractMemories     *ExtractMemories
	memorizeFromToolUse *MemorizeFromToolUse
}

// NewParetoResponseGenerator creates a new unified Pareto-based response generator.
// This is the ONLY way to generate responses in Alicia - all other methods should use this.
func NewParetoResponseGenerator(
	llmService ports.LLMService,
	reflectionLLM ports.LLMService,
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	toolRepo ports.ToolRepository,
	sentenceRepo ports.SentenceRepository,
	toolUseRepo ports.ToolUseRepository,
	reasoningRepo ports.ReasoningStepRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	toolService ports.ToolService,
	memoryService ports.MemoryService,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
	broadcaster ports.ConversationBroadcaster,
	config *ParetoResponseConfig,
) *ParetoResponseGenerator {
	if config == nil {
		config = DefaultParetoResponseConfig()
	}

	// Use main LLM for reflection if none specified
	if reflectionLLM == nil {
		reflectionLLM = llmService
	}

	// Create GEPA components
	mutator := prompt.NewPathMutator(llmService, reflectionLLM)
	evaluator := baselines.NewPathEvaluator(llmService)

	gen := &ParetoResponseGenerator{
		llmService:       llmService,
		reflectionLLM:    reflectionLLM,
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		toolRepo:         toolRepo,
		sentenceRepo:     sentenceRepo,
		toolUseRepo:      toolUseRepo,
		reasoningRepo:    reasoningRepo,
		memoryUsageRepo:  memoryUsageRepo,
		toolService:      toolService,
		memoryService:    memoryService,
		idGenerator:      idGenerator,
		txManager:        txManager,
		mutator:          mutator,
		evaluator:        evaluator,
		config:           config,
	}

	// Initialize title generator
	gen.titleGenerator = NewGenerateConversationTitle(conversationRepo, messageRepo, llmService, broadcaster)

	// Initialize memory extraction
	if memoryService != nil {
		gen.extractMemories = NewExtractMemories(memoryService, llmService, idGenerator)
		gen.memorizeFromToolUse = NewMemorizeFromToolUse(llmService, memoryService, gen.extractMemories)
	}

	return gen
}

// executionContext holds per-execution state to ensure thread-safety.
type paretoExecutionContext struct {
	config        *ParetoResponseConfig
	paretoArchive *prompt.PathParetoArchive
	tools         []*models.Tool
	toolRunner    ports.ToolRunner
}

// Execute generates a response using the Pareto search evolutionary approach.
// This is the main entry point for response generation.
func (g *ParetoResponseGenerator) Execute(ctx context.Context, input *ParetoResponseInput) (*ParetoResponseOutput, error) {
	log.Printf("[ParetoResponseGenerator] Execute called for conversation=%s, userMessage=%s", input.ConversationID, input.UserMessageID)

	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if input.ConversationID == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}

	if input.UserMessageID == "" {
		return nil, fmt.Errorf("user message ID is required")
	}

	// Get conversation for context
	conversation, err := g.conversationRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	log.Printf("[ParetoResponseGenerator] Loaded conversation: %s (title=%q)", conversation.ID, conversation.Title)

	// Get user message to respond to
	userMessage, err := g.messageRepo.GetByID(ctx, input.UserMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user message: %w", err)
	}

	// Get conversation history for context
	messages, err := g.messageRepo.GetLatestByConversation(ctx, input.ConversationID, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	// Retrieve relevant memories
	log.Printf("[ParetoResponseGenerator] Retrieving relevant memories...")
	var relevantMemories []*models.Memory
	if g.memoryService != nil {
		searchResults, err := g.memoryService.SearchWithScores(ctx, userMessage.Contents, 0.7, 5)
		if err != nil {
			log.Printf("[ParetoResponseGenerator] WARNING: failed to retrieve memories: %v", err)
		} else {
			log.Printf("[ParetoResponseGenerator] Found %d relevant memories", len(searchResults))
			relevantMemories = make([]*models.Memory, len(searchResults))
			for i, result := range searchResults {
				relevantMemories[i] = result.Memory
				log.Printf("[ParetoResponseGenerator]   Memory[%d]: id=%s similarity=%.3f content=%q",
					i, result.Memory.ID, result.Similarity, truncateForLog(result.Memory.Content, 80))
				// Notify about retrieved memory
				if input.Notifier != nil {
					input.Notifier.NotifyMemoryRetrieved(
						input.UserMessageID,
						input.ConversationID,
						result.Memory.ID,
						result.Memory.Content,
						result.Similarity,
					)
				}
				// Track memory usage
				_, _ = g.memoryService.TrackUsage(ctx, result.Memory.ID, input.ConversationID, input.UserMessageID, result.Similarity)
			}
		}
	} else {
		log.Printf("[ParetoResponseGenerator] Memory service not available, skipping memory retrieval")
	}

	// Get available tools (only those with registered executors)
	log.Printf("[ParetoResponseGenerator] Loading tools (enableTools=%v)...", input.EnableTools)
	var tools []*models.Tool
	if input.EnableTools && g.toolService != nil {
		tools, err = g.toolService.ListAvailable(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get available tools: %w", err)
		}
		log.Printf("[ParetoResponseGenerator] Loaded %d available tools (with executors)", len(tools))
		for i, tool := range tools {
			log.Printf("[ParetoResponseGenerator]   Tool[%d]: %s", i, tool.Name)
		}
	} else {
		log.Printf("[ParetoResponseGenerator] Tools disabled or toolService not available")
	}

	// Create per-execution context
	execCtx := g.newExecutionContext(input.Config, tools)
	log.Printf("[ParetoResponseGenerator] Execution context created with config: maxGen=%d, branchesPerGen=%d, targetScore=%.2f, maxToolCalls=%d, maxLLMCalls=%d, parallelBranches=%v",
		execCtx.config.MaxGenerations, execCtx.config.BranchesPerGen, execCtx.config.TargetScore,
		execCtx.config.MaxToolCalls, execCtx.config.MaxLLMCalls, execCtx.config.EnableParallelBranches)

	// Create assistant message
	sequenceNumber, err := g.messageRepo.GetNextSequenceNumber(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next sequence number: %w", err)
	}

	messageID := input.MessageID
	if messageID == "" {
		messageID = g.idGenerator.GenerateMessageID()
	}
	message := models.NewAssistantMessage(messageID, input.ConversationID, sequenceNumber, "")
	if input.PreviousID != "" {
		message.SetPreviousMessage(input.PreviousID)
	}

	if err := g.messageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Update conversation tip
	if err := g.conversationRepo.UpdateTip(ctx, input.ConversationID, message.ID); err != nil {
		return nil, fmt.Errorf("failed to update conversation tip: %w", err)
	}

	// Notify generation started
	if input.Notifier != nil {
		input.Notifier.NotifyGenerationStarted(message.ID, input.PreviousID, input.ConversationID)
	}

	// Build the query from user message and context
	query := g.buildQueryWithContext(userMessage.Contents, messages, relevantMemories, conversation)
	log.Printf("[ParetoResponseGenerator] Built query context (%d chars) for Pareto search", len(query))

	// Run Pareto search
	log.Printf("[ParetoResponseGenerator] Starting Pareto search...")
	searchStartTime := time.Now()
	result, err := g.runParetoSearch(ctx, query, input, message, execCtx)
	searchDuration := time.Since(searchStartTime)
	if err != nil {
		log.Printf("[ParetoResponseGenerator] Pareto search FAILED after %v: %v", searchDuration, err)
		// Mark message as failed
		message.MarkAsFailed()
		_ = g.messageRepo.Update(ctx, message)
		if input.Notifier != nil {
			input.Notifier.NotifyGenerationFailed(message.ID, input.ConversationID, err)
		}
		return nil, fmt.Errorf("pareto search failed: %w", err)
	}
	log.Printf("[ParetoResponseGenerator] Pareto search completed in %v: score=%.3f iterations=%d answer=%d chars",
		searchDuration, result.Score, result.Iterations, len(result.Answer))

	// Update message with best response
	message.Contents = strings.TrimSpace(result.Answer)
	message.MarkAsCompleted()
	if err := g.messageRepo.Update(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	// Create tool uses from the best path's trace
	var toolUses []*models.ToolUse
	if result.BestPath != nil && result.BestPath.Trace != nil {
		log.Printf("[ParetoResponseGenerator] Best path has %d tool calls", len(result.BestPath.Trace.ToolCalls))
		for i, tc := range result.BestPath.Trace.ToolCalls {
			log.Printf("[ParetoResponseGenerator]   ToolCall[%d]: %s success=%v", i, tc.ToolName, tc.Success)
			toolUse, err := g.createToolUseFromTrace(ctx, message.ID, &tc, input)
			if err != nil {
				log.Printf("[ParetoResponseGenerator] WARNING: failed to create tool use record: %v", err)
				continue
			}
			toolUses = append(toolUses, toolUse)
		}
	} else {
		log.Printf("[ParetoResponseGenerator] No tool calls in best path")
	}

	// Notify completion
	if input.Notifier != nil {
		input.Notifier.NotifyGenerationComplete(message.ID, input.ConversationID, message.Contents)
	}

	// Extract memories asynchronously
	go func() {
		memCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		g.extractAndStoreMemories(memCtx, message, input.ConversationID)
	}()

	// Generate title if needed
	g.titleGenerator.ExecuteAsync(ctx, input.ConversationID)

	paretoFront := execCtx.paretoArchive.GetParetoFront()
	log.Printf("[ParetoResponseGenerator] Execute complete: messageID=%s paretoFrontSize=%d toolUses=%d finalScore=%.3f",
		message.ID, len(paretoFront), len(toolUses), result.Score)

	return &ParetoResponseOutput{
		Message:     message,
		ToolUses:    toolUses,
		ParetoFront: paretoFront,
		Score:       result.Score,
		Iterations:  result.Iterations,
	}, nil
}

// newExecutionContext creates a per-execution context with local config copy and fresh Pareto archive.
func (g *ParetoResponseGenerator) newExecutionContext(inputConfig *ParetoResponseConfig, tools []*models.Tool) *paretoExecutionContext {
	// Copy base config
	cfg := &ParetoResponseConfig{
		MaxGenerations:         g.config.MaxGenerations,
		BranchesPerGen:         g.config.BranchesPerGen,
		TargetScore:            g.config.TargetScore,
		MaxToolCalls:           g.config.MaxToolCalls,
		MaxLLMCalls:            g.config.MaxLLMCalls,
		ParetoArchiveSize:      g.config.ParetoArchiveSize,
		EnableCrossover:        g.config.EnableCrossover,
		ExecutionTimeoutMs:     g.config.ExecutionTimeoutMs,
		EnableParallelBranches: g.config.EnableParallelBranches,
		MaxParallelBranches:    g.config.MaxParallelBranches,
		MaxToolLoopIterations:  g.config.MaxToolLoopIterations,
	}

	// Apply overrides
	if inputConfig != nil {
		if inputConfig.MaxGenerations > 0 {
			cfg.MaxGenerations = inputConfig.MaxGenerations
		}
		if inputConfig.BranchesPerGen > 0 {
			cfg.BranchesPerGen = inputConfig.BranchesPerGen
		}
		if inputConfig.TargetScore > 0 && inputConfig.TargetScore <= 1.0 {
			cfg.TargetScore = inputConfig.TargetScore
		}
		if inputConfig.MaxToolCalls > 0 {
			cfg.MaxToolCalls = inputConfig.MaxToolCalls
		}
		if inputConfig.MaxLLMCalls > 0 {
			cfg.MaxLLMCalls = inputConfig.MaxLLMCalls
		}
	}

	// Create tool runner adapter
	var toolRunner ports.ToolRunner
	if g.toolService != nil {
		toolRunner = ports.ToolRunnerFunc(func(ctx context.Context, toolName string, arguments map[string]any) (any, error) {
			return g.toolService.ExecuteTool(ctx, toolName, arguments)
		})
	}

	return &paretoExecutionContext{
		config:        cfg,
		paretoArchive: prompt.NewPathParetoArchive(cfg.ParetoArchiveSize),
		tools:         tools,
		toolRunner:    toolRunner,
	}
}

// buildQueryWithContext builds the query with conversation context and memories.
func (g *ParetoResponseGenerator) buildQueryWithContext(
	userQuery string,
	messages []*models.Message,
	memories []*models.Memory,
	conversation *models.Conversation,
) string {
	var sb strings.Builder

	// Add system context
	sb.WriteString("You are Alicia, a helpful AI assistant with memory and tool capabilities.\n\n")

	// Add relevant memories
	if len(memories) > 0 {
		sb.WriteString("Relevant memories from previous conversations:\n")
		for i, memory := range memories {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, memory.Content))
		}
		sb.WriteString("\n")
	}

	// Add conversation history (last few messages)
	if len(messages) > 1 { // More than just the current message
		sb.WriteString("Recent conversation:\n")
		// Show at most last 5 messages for context
		start := 0
		if len(messages) > 5 {
			start = len(messages) - 5
		}
		for _, msg := range messages[start:] {
			if msg.Role == "system" {
				continue
			}
			role := string(msg.Role)
			content := msg.Contents
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("%s: %s\n", role, content))
		}
		sb.WriteString("\n")
	}

	// Add the current query
	sb.WriteString("User query: ")
	sb.WriteString(userQuery)

	return sb.String()
}

// runParetoSearch executes the GEPA evolutionary loop to find the best response.
func (g *ParetoResponseGenerator) runParetoSearch(
	ctx context.Context,
	query string,
	input *ParetoResponseInput,
	message *models.Message,
	execCtx *paretoExecutionContext,
) (*models.PathSearchResult, error) {
	// Generate run ID
	runID := g.idGenerator.GenerateOptimizationRunID()
	log.Printf("[ParetoSearch] Starting search runID=%s maxGenerations=%d branchesPerGen=%d targetScore=%.2f",
		runID, execCtx.config.MaxGenerations, execCtx.config.BranchesPerGen, execCtx.config.TargetScore)

	// Create seed candidate
	var seed *models.PathCandidate
	if input.SeedStrategy != "" {
		candidateID := g.idGenerator.GeneratePromptCandidateID()
		seed = models.NewPathCandidate(candidateID, runID, 0, nil, input.SeedStrategy, nil)
		log.Printf("[ParetoSearch] Created seed with custom strategy: %q", truncateForLog(input.SeedStrategy, 100))
	} else {
		candidateID := g.idGenerator.GeneratePromptCandidateID()
		seed = models.NewSeedCandidate(candidateID, runID)
		log.Printf("[ParetoSearch] Created default seed candidate id=%s", candidateID)
	}

	// Execute and evaluate seed
	log.Printf("[ParetoSearch] Executing seed path...")
	trace, err := g.executePath(ctx, query, seed, execCtx, input)
	if err != nil {
		log.Printf("[ParetoSearch] Seed execution failed: %v (using empty trace)", err)
		trace = &models.ExecutionTrace{
			Query:       query,
			ToolCalls:   []models.ToolCallRecord{},
			FinalAnswer: "",
			DurationMs:  0,
		}
	} else {
		log.Printf("[ParetoSearch] Seed execution complete: %d tool calls, %d ms, answer=%d chars",
			len(trace.ToolCalls), trace.DurationMs, len(trace.FinalAnswer))
	}
	seed.SetTrace(trace)

	log.Printf("[ParetoSearch] Evaluating seed candidate...")
	scores, feedback, err := g.evaluateCandidate(ctx, query, trace)
	if err != nil {
		log.Printf("[ParetoSearch] Seed evaluation failed: %v", err)
		scores = models.PathScores{}
		feedback = "Evaluation failed: " + err.Error()
	} else {
		log.Printf("[ParetoSearch] Seed scores: quality=%.3f efficiency=%.3f cost=%.3f robustness=%.3f latency=%.3f",
			scores.AnswerQuality, scores.Efficiency, scores.TokenCost, scores.Robustness, scores.Latency)
		log.Printf("[ParetoSearch] Seed feedback: %q", truncateForLog(feedback, 150))
	}
	seed.SetScores(scores)
	seed.SetFeedback(feedback)

	execCtx.paretoArchive.Add(seed)

	// Budget tracking
	toolCallsUsed := len(trace.ToolCalls)
	llmCallsUsed := 1
	log.Printf("[ParetoSearch] Initial budget: toolCalls=%d/%d llmCalls=%d/%d",
		toolCallsUsed, execCtx.config.MaxToolCalls, llmCallsUsed, execCtx.config.MaxLLMCalls)

	bestPath := seed
	log.Printf("[ParetoSearch] Initial best path: score=%.3f", bestPath.Scores.AnswerQuality)

	// Main evolutionary loop
	for gen := 0; gen < execCtx.config.MaxGenerations; gen++ {
		log.Printf("[ParetoSearch] === Generation %d/%d ===", gen+1, execCtx.config.MaxGenerations)
		log.Printf("[ParetoSearch] Budget status: toolCalls=%d/%d llmCalls=%d/%d archiveSize=%d bestScore=%.3f",
			toolCallsUsed, execCtx.config.MaxToolCalls, llmCallsUsed, execCtx.config.MaxLLMCalls,
			execCtx.paretoArchive.Size(), bestPath.Scores.AnswerQuality)

		// Check budget
		if toolCallsUsed >= execCtx.config.MaxToolCalls || llmCallsUsed >= execCtx.config.MaxLLMCalls {
			log.Printf("[ParetoSearch] Budget exhausted, stopping search")
			break
		}

		// Select candidates for mutation
		selected := execCtx.paretoArchive.SelectForMutation(execCtx.config.BranchesPerGen)
		log.Printf("[ParetoSearch] Selected %d candidates for mutation (requested %d)", len(selected), execCtx.config.BranchesPerGen)
		if len(selected) == 0 {
			if bestPath != nil {
				log.Printf("[ParetoSearch] No candidates selected, falling back to best path")
				selected = []*models.PathCandidate{bestPath}
			} else {
				log.Printf("[ParetoSearch] No candidates available, stopping search")
				break
			}
		}
		for i, cand := range selected {
			log.Printf("[ParetoSearch]   Candidate[%d]: id=%s gen=%d score=%.3f", i, cand.ID, cand.Generation, cand.Scores.AnswerQuality)
		}

		// Process branches
		if execCtx.config.EnableParallelBranches && len(selected) > 1 {
			log.Printf("[ParetoSearch] Processing %d branches in PARALLEL (max=%d)", len(selected), execCtx.config.MaxParallelBranches)
			result := g.processBranchesParallel(ctx, query, selected, feedback, toolCallsUsed, llmCallsUsed, bestPath, execCtx, input)
			toolCallsUsed = result.toolCallsUsed
			llmCallsUsed = result.llmCallsUsed
			bestPath = result.bestPath
			log.Printf("[ParetoSearch] Parallel processing complete: bestScore=%.3f earlyExit=%v", bestPath.Scores.AnswerQuality, result.earlyExit)

			if result.earlyExit {
				log.Printf("[ParetoSearch] EARLY EXIT: target score reached (%.3f >= %.3f)", bestPath.Scores.AnswerQuality, execCtx.config.TargetScore)
				return &models.PathSearchResult{
					BestPath:   result.bestPath,
					Answer:     result.bestPath.Trace.FinalAnswer,
					Score:      result.bestPath.Scores.AnswerQuality,
					Iterations: gen + 1,
				}, nil
			}
		} else {
			// Sequential processing
			log.Printf("[ParetoSearch] Processing %d branches SEQUENTIALLY", len(selected))
			for branchIdx, candidate := range selected {
				if toolCallsUsed >= execCtx.config.MaxToolCalls || llmCallsUsed >= execCtx.config.MaxLLMCalls {
					log.Printf("[ParetoSearch] Budget exhausted during sequential processing")
					break
				}

				log.Printf("[ParetoSearch] Branch[%d]: mutating candidate %s", branchIdx, candidate.ID)
				candidateFeedback := candidate.Feedback
				if candidateFeedback == "" {
					candidateFeedback = feedback
				}

				mutated, err := g.mutateCandidate(ctx, candidate, candidateFeedback)
				llmCallsUsed++
				if err != nil {
					log.Printf("[ParetoSearch] Branch[%d]: mutation failed: %v", branchIdx, err)
					continue
				}
				log.Printf("[ParetoSearch] Branch[%d]: mutation successful, new id=%s", branchIdx, mutated.ID)

				log.Printf("[ParetoSearch] Branch[%d]: executing mutated path...", branchIdx)
				mutatedTrace, err := g.executePath(ctx, query, mutated, execCtx, input)
				llmCallsUsed++
				if err != nil {
					log.Printf("[ParetoSearch] Branch[%d]: execution failed: %v", branchIdx, err)
					continue
				}
				mutated.SetTrace(mutatedTrace)
				toolCallsUsed += len(mutatedTrace.ToolCalls)
				log.Printf("[ParetoSearch] Branch[%d]: execution complete: %d tool calls, %d ms",
					branchIdx, len(mutatedTrace.ToolCalls), mutatedTrace.DurationMs)

				log.Printf("[ParetoSearch] Branch[%d]: evaluating...", branchIdx)
				mutatedScores, mutatedFeedback, err := g.evaluateCandidate(ctx, query, mutatedTrace)
				llmCallsUsed++
				if err != nil {
					log.Printf("[ParetoSearch] Branch[%d]: evaluation failed: %v", branchIdx, err)
					continue
				}
				mutated.SetScores(mutatedScores)
				mutated.SetFeedback(mutatedFeedback)
				log.Printf("[ParetoSearch] Branch[%d]: scores: quality=%.3f efficiency=%.3f", branchIdx, mutatedScores.AnswerQuality, mutatedScores.Efficiency)

				execCtx.paretoArchive.Add(mutated)

				if mutatedScores.AnswerQuality > bestPath.Scores.AnswerQuality {
					log.Printf("[ParetoSearch] Branch[%d]: NEW BEST PATH! %.3f > %.3f", branchIdx, mutatedScores.AnswerQuality, bestPath.Scores.AnswerQuality)
					bestPath = mutated
				}

				// Early exit if target reached
				if mutatedScores.AnswerQuality >= execCtx.config.TargetScore {
					log.Printf("[ParetoSearch] Branch[%d]: EARLY EXIT: target score reached (%.3f >= %.3f)",
						branchIdx, mutatedScores.AnswerQuality, execCtx.config.TargetScore)
					return &models.PathSearchResult{
						BestPath:   bestPath,
						Answer:     mutatedTrace.FinalAnswer,
						Score:      mutatedScores.AnswerQuality,
						Iterations: gen + 1,
					}, nil
				}
			}
		}

		// Crossover
		if execCtx.config.EnableCrossover && execCtx.paretoArchive.Size() >= 2 {
			log.Printf("[ParetoSearch] Attempting crossover (archiveSize=%d)", execCtx.paretoArchive.Size())
			parent1, parent2 := g.selectDiversePair(execCtx.paretoArchive)
			if parent1 != nil && parent2 != nil {
				log.Printf("[ParetoSearch] Crossover parents: %s (score=%.3f) x %s (score=%.3f)",
					parent1.ID, parent1.Scores.AnswerQuality, parent2.ID, parent2.Scores.AnswerQuality)
				child, err := g.crossoverCandidates(ctx, parent1, parent2)
				llmCallsUsed++
				if err == nil && child != nil {
					log.Printf("[ParetoSearch] Crossover produced child %s, executing...", child.ID)
					childTrace, err := g.executePath(ctx, query, child, execCtx, input)
					llmCallsUsed++
					if err == nil {
						child.SetTrace(childTrace)
						toolCallsUsed += len(childTrace.ToolCalls)
						childScores, childFeedback, _ := g.evaluateCandidate(ctx, query, childTrace)
						llmCallsUsed++
						child.SetScores(childScores)
						child.SetFeedback(childFeedback)
						execCtx.paretoArchive.Add(child)
						log.Printf("[ParetoSearch] Crossover child scores: quality=%.3f efficiency=%.3f", childScores.AnswerQuality, childScores.Efficiency)

						if childScores.AnswerQuality > bestPath.Scores.AnswerQuality {
							log.Printf("[ParetoSearch] Crossover child is NEW BEST PATH! %.3f > %.3f", childScores.AnswerQuality, bestPath.Scores.AnswerQuality)
							bestPath = child
						}
					} else {
						log.Printf("[ParetoSearch] Crossover child execution failed: %v", err)
					}
				} else if err != nil {
					log.Printf("[ParetoSearch] Crossover failed: %v", err)
				}
			} else {
				log.Printf("[ParetoSearch] Could not select diverse pair for crossover")
			}
		}

		log.Printf("[ParetoSearch] Generation %d complete: archiveSize=%d bestScore=%.3f budget(tools=%d/%d llm=%d/%d)",
			gen+1, execCtx.paretoArchive.Size(), bestPath.Scores.AnswerQuality,
			toolCallsUsed, execCtx.config.MaxToolCalls, llmCallsUsed, execCtx.config.MaxLLMCalls)
	}

	if bestPath == nil || bestPath.Trace == nil {
		log.Printf("[ParetoSearch] FAILED: no valid path found after %d generations", execCtx.config.MaxGenerations)
		return nil, fmt.Errorf("no valid path found after %d generations", execCtx.config.MaxGenerations)
	}

	log.Printf("[ParetoSearch] Search complete: finalScore=%.3f iterations=%d archiveSize=%d totalToolCalls=%d totalLLMCalls=%d",
		bestPath.Scores.AnswerQuality, execCtx.config.MaxGenerations, execCtx.paretoArchive.Size(), toolCallsUsed, llmCallsUsed)
	log.Printf("[ParetoSearch] Best path: id=%s gen=%d scores(q=%.3f e=%.3f c=%.3f r=%.3f l=%.3f)",
		bestPath.ID, bestPath.Generation, bestPath.Scores.AnswerQuality, bestPath.Scores.Efficiency,
		bestPath.Scores.TokenCost, bestPath.Scores.Robustness, bestPath.Scores.Latency)

	return &models.PathSearchResult{
		BestPath:   bestPath,
		Answer:     bestPath.Trace.FinalAnswer,
		Score:      bestPath.Scores.AnswerQuality,
		Iterations: execCtx.config.MaxGenerations,
	}, nil
}

// parallelResult holds the aggregated result from parallel branch execution.
type parallelResult struct {
	toolCallsUsed int
	llmCallsUsed  int
	bestPath      *models.PathCandidate
	earlyExit     bool
}

// branchResult holds the result of processing a single branch.
type branchResult struct {
	mutated   *models.PathCandidate
	toolCalls int
	llmCalls  int
	success   bool
	targetMet bool
}

// processBranchesParallel processes multiple branches concurrently.
func (g *ParetoResponseGenerator) processBranchesParallel(
	ctx context.Context,
	query string,
	candidates []*models.PathCandidate,
	fallbackFeedback string,
	initialToolCalls int,
	initialLLMCalls int,
	currentBest *models.PathCandidate,
	execCtx *paretoExecutionContext,
	input *ParetoResponseInput,
) parallelResult {
	log.Printf("[ParallelBranches] Starting parallel processing of %d candidates", len(candidates))

	result := parallelResult{
		toolCallsUsed: initialToolCalls,
		llmCallsUsed:  initialLLMCalls,
		bestPath:      currentBest,
		earlyExit:     false,
	}

	maxParallel := execCtx.config.MaxParallelBranches
	if maxParallel <= 0 {
		maxParallel = execCtx.config.BranchesPerGen
		if maxParallel > 5 {
			maxParallel = 5
		}
	}
	log.Printf("[ParallelBranches] Max parallel: %d", maxParallel)

	sem := make(chan struct{}, maxParallel)
	resultsChan := make(chan branchResult, len(candidates))
	var mu sync.Mutex
	var wg sync.WaitGroup
	var stopEarly bool

	for _, candidate := range candidates {
		mu.Lock()
		if stopEarly || result.toolCallsUsed >= execCtx.config.MaxToolCalls || result.llmCallsUsed >= execCtx.config.MaxLLMCalls {
			mu.Unlock()
			break
		}
		mu.Unlock()

		select {
		case <-ctx.Done():
			break
		default:
		}

		wg.Add(1)
		candidate := candidate

		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			mu.Lock()
			if stopEarly {
				mu.Unlock()
				return
			}
			mu.Unlock()

			candidateFeedback := candidate.Feedback
			if candidateFeedback == "" {
				candidateFeedback = fallbackFeedback
			}

			brResult := branchResult{llmCalls: 0, success: false}

			mutated, err := g.mutateCandidate(ctx, candidate, candidateFeedback)
			brResult.llmCalls++
			if err != nil {
				resultsChan <- brResult
				return
			}

			mutatedTrace, err := g.executePath(ctx, query, mutated, execCtx, input)
			brResult.llmCalls++
			if err != nil {
				resultsChan <- brResult
				return
			}
			mutated.SetTrace(mutatedTrace)
			brResult.toolCalls = len(mutatedTrace.ToolCalls)

			mutatedScores, mutatedFeedback, err := g.evaluateCandidate(ctx, query, mutatedTrace)
			brResult.llmCalls++
			if err != nil {
				resultsChan <- brResult
				return
			}
			mutated.SetScores(mutatedScores)
			mutated.SetFeedback(mutatedFeedback)

			brResult.mutated = mutated
			brResult.success = true
			brResult.targetMet = mutatedScores.AnswerQuality >= execCtx.config.TargetScore

			resultsChan <- brResult
		}()
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	completedCount := 0
	successCount := 0
	for brResult := range resultsChan {
		completedCount++
		mu.Lock()
		result.llmCallsUsed += brResult.llmCalls
		result.toolCallsUsed += brResult.toolCalls

		if brResult.success && brResult.mutated != nil {
			successCount++
			execCtx.paretoArchive.Add(brResult.mutated)
			log.Printf("[ParallelBranches] Branch completed successfully: id=%s score=%.3f",
				brResult.mutated.ID, brResult.mutated.Scores.AnswerQuality)

			if brResult.mutated.Scores.AnswerQuality > result.bestPath.Scores.AnswerQuality {
				log.Printf("[ParallelBranches] NEW BEST PATH: %.3f > %.3f",
					brResult.mutated.Scores.AnswerQuality, result.bestPath.Scores.AnswerQuality)
				result.bestPath = brResult.mutated
			}

			if brResult.targetMet {
				log.Printf("[ParallelBranches] Target score met! Triggering early exit")
				result.earlyExit = true
				stopEarly = true
			}
		} else {
			log.Printf("[ParallelBranches] Branch failed or produced no result")
		}
		mu.Unlock()
	}

	log.Printf("[ParallelBranches] Complete: %d/%d successful, bestScore=%.3f, earlyExit=%v",
		successCount, completedCount, result.bestPath.Scores.AnswerQuality, result.earlyExit)

	return result
}

// executePath runs the agent with the candidate's strategy and captures the execution trace.
func (g *ParetoResponseGenerator) executePath(
	ctx context.Context,
	query string,
	candidate *models.PathCandidate,
	execCtx *paretoExecutionContext,
	input *ParetoResponseInput,
) (*models.ExecutionTrace, error) {
	if candidate == nil {
		return nil, fmt.Errorf("candidate cannot be nil")
	}

	log.Printf("[ExecutePath] Starting execution for candidate %s (gen=%d)", candidate.ID, candidate.Generation)

	timeout := time.Duration(execCtx.config.ExecutionTimeoutMs) * time.Millisecond
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()

	// Build agent prompt
	agentPrompt := g.buildAgentPrompt(candidate, query, execCtx.tools)
	log.Printf("[ExecutePath] Built agent prompt (%d chars), %d lessons", len(agentPrompt), len(candidate.AccumulatedLessons))

	// Create initial messages
	messages := []ports.LLMMessage{
		{Role: "system", Content: agentPrompt},
		{Role: "user", Content: query},
	}

	// Run tool loop if tools available
	if execCtx.toolRunner != nil && len(execCtx.tools) > 0 {
		log.Printf("[ExecutePath] Executing with tool loop (tools=%d, maxIterations=%d, timeout=%v)",
			len(execCtx.tools), execCtx.config.MaxToolLoopIterations, timeout)
		return g.executePathWithToolLoop(timeoutCtx, messages, agentPrompt, query, startTime, execCtx, input)
	}

	// Single-turn execution
	log.Printf("[ExecutePath] Executing single-turn (no tools or toolRunner)")
	return g.executePathSingleTurn(timeoutCtx, messages, agentPrompt, query, startTime, execCtx)
}

// executePathSingleTurn performs single-turn execution without tool loop.
func (g *ParetoResponseGenerator) executePathSingleTurn(
	ctx context.Context,
	messages []ports.LLMMessage,
	agentPrompt, query string,
	startTime time.Time,
	execCtx *paretoExecutionContext,
) (*models.ExecutionTrace, error) {
	var response *ports.LLMResponse
	var err error

	if len(execCtx.tools) > 0 {
		response, err = g.llmService.ChatWithTools(ctx, messages, execCtx.tools)
	} else {
		response, err = g.llmService.Chat(ctx, messages)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	durationMs := time.Since(startTime).Milliseconds()

	// Parse tool calls (not executed)
	var toolCalls []models.ToolCallRecord
	if response != nil && len(response.ToolCalls) > 0 {
		for _, tc := range response.ToolCalls {
			toolCalls = append(toolCalls, models.ToolCallRecord{
				ToolName:  tc.Name,
				Arguments: tc.Arguments,
				Success:   true, // Optimistic
				Result:    nil,
				Error:     "",
			})
		}
	}

	return &models.ExecutionTrace{
		Query:          query,
		ToolCalls:      toolCalls,
		ReasoningSteps: []string{},
		FinalAnswer:    response.Content,
		TotalTokens:    g.estimateTokens(agentPrompt, response.Content),
		DurationMs:     durationMs,
	}, nil
}

// executePathWithToolLoop performs multi-turn execution with actual tool execution.
func (g *ParetoResponseGenerator) executePathWithToolLoop(
	ctx context.Context,
	messages []ports.LLMMessage,
	agentPrompt, query string,
	startTime time.Time,
	execCtx *paretoExecutionContext,
	input *ParetoResponseInput,
) (*models.ExecutionTrace, error) {
	currentMessages := make([]ports.LLMMessage, len(messages))
	copy(currentMessages, messages)

	var allToolCalls []models.ToolCallRecord
	var finalAnswer string
	var totalTokens int

	log.Printf("[ToolLoop] Starting tool loop (maxIterations=%d)", execCtx.config.MaxToolLoopIterations)

	for iteration := 0; iteration < execCtx.config.MaxToolLoopIterations; iteration++ {
		log.Printf("[ToolLoop] Iteration %d/%d: calling LLM...", iteration+1, execCtx.config.MaxToolLoopIterations)
		var response *ports.LLMResponse
		var err error

		if len(execCtx.tools) > 0 {
			response, err = g.llmService.ChatWithTools(ctx, currentMessages, execCtx.tools)
		} else {
			response, err = g.llmService.Chat(ctx, currentMessages)
		}

		if err != nil {
			log.Printf("[ToolLoop] LLM call failed on iteration %d: %v", iteration+1, err)
			return nil, fmt.Errorf("LLM call failed on iteration %d: %w", iteration, err)
		}

		totalTokens += g.estimateTokens("", response.Content)
		log.Printf("[ToolLoop] Iteration %d: got response (%d chars), %d tool calls",
			iteration+1, len(response.Content), len(response.ToolCalls))

		currentMessages = append(currentMessages, ports.LLMMessage{
			Role:    "assistant",
			Content: response.Content,
		})

		// If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			log.Printf("[ToolLoop] No tool calls, ending loop with final answer")
			finalAnswer = response.Content
			break
		}

		// Execute each tool call
		log.Printf("[ToolLoop] Executing %d tool calls...", len(response.ToolCalls))
		for tcIdx, tc := range response.ToolCalls {
			log.Printf("[ToolLoop]   Tool[%d]: %s", tcIdx, tc.Name)
			toolRecord := models.ToolCallRecord{
				ToolName:  tc.Name,
				Arguments: tc.Arguments,
				Success:   false,
				Result:    nil,
				Error:     "",
			}

			// Notify tool start
			if input.Notifier != nil {
				input.Notifier.NotifyToolUseStart(tc.ID, "", input.ConversationID, tc.Name, tc.Arguments)
			}

			// Execute tool
			result, execErr := execCtx.toolRunner.RunTool(ctx, tc.Name, tc.Arguments)

			if execErr != nil {
				log.Printf("[ToolLoop]   Tool[%d] %s FAILED: %v", tcIdx, tc.Name, execErr)
				toolRecord.Success = false
				toolRecord.Error = execErr.Error()
				currentMessages = append(currentMessages, ports.LLMMessage{
					Role:    "tool",
					Content: fmt.Sprintf("Error executing %s: %s", tc.Name, execErr.Error()),
				})
				if input.Notifier != nil {
					input.Notifier.NotifyToolUseComplete(tc.ID, tc.ID, input.ConversationID, false, nil, execErr.Error())
				}
			} else {
				log.Printf("[ToolLoop]   Tool[%d] %s SUCCESS", tcIdx, tc.Name)
				toolRecord.Success = true
				toolRecord.Result = result
				resultContent := fmt.Sprintf("%v", result)
				currentMessages = append(currentMessages, ports.LLMMessage{
					Role:    "tool",
					Content: resultContent,
				})
				if input.Notifier != nil {
					input.Notifier.NotifyToolUseComplete(tc.ID, tc.ID, input.ConversationID, true, result, "")
				}
			}

			allToolCalls = append(allToolCalls, toolRecord)
		}

		// If last iteration, use current response as answer
		if iteration == execCtx.config.MaxToolLoopIterations-1 {
			log.Printf("[ToolLoop] Max iterations reached, using last response as final answer")
			finalAnswer = response.Content
			if finalAnswer == "" {
				finalAnswer = "Max tool execution iterations reached."
			}
		}
	}

	durationMs := time.Since(startTime).Milliseconds()
	log.Printf("[ToolLoop] Complete: %d total tool calls, %d ms, finalAnswer=%d chars",
		len(allToolCalls), durationMs, len(finalAnswer))

	return &models.ExecutionTrace{
		Query:          query,
		ToolCalls:      allToolCalls,
		ReasoningSteps: []string{},
		FinalAnswer:    finalAnswer,
		TotalTokens:    totalTokens + g.estimateTokens(agentPrompt, ""),
		DurationMs:     durationMs,
	}, nil
}

// buildAgentPrompt constructs the full prompt for agent execution.
func (g *ParetoResponseGenerator) buildAgentPrompt(candidate *models.PathCandidate, query string, tools []*models.Tool) string {
	var sb strings.Builder

	// Add strategy prompt
	sb.WriteString(candidate.StrategyPrompt)
	sb.WriteString("\n\n")

	// Add accumulated lessons
	if len(candidate.AccumulatedLessons) > 0 {
		sb.WriteString("ACCUMULATED LESSONS FROM PREVIOUS ATTEMPTS:\n")
		for i, lesson := range candidate.AccumulatedLessons {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, lesson))
		}
		sb.WriteString("\n")
	}

	// Add tool descriptions
	if len(tools) > 0 {
		sb.WriteString("Available tools:\n")
		for _, tool := range tools {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Think step by step. Use tools as needed to find the best answer.\n")

	return sb.String()
}

// evaluateCandidate evaluates a path trace using the PathEvaluator.
func (g *ParetoResponseGenerator) evaluateCandidate(ctx context.Context, query string, trace *models.ExecutionTrace) (models.PathScores, string, error) {
	promptTrace := g.convertToPromptTrace(trace)

	promptScores, feedback, err := g.evaluator.Evaluate(ctx, query, promptTrace)
	if err != nil {
		return models.PathScores{}, "", fmt.Errorf("evaluation failed: %w", err)
	}

	modelScores := models.PathScores{
		AnswerQuality: promptScores.AnswerQuality,
		Efficiency:    promptScores.Efficiency,
		TokenCost:     promptScores.TokenCost,
		Robustness:    promptScores.Robustness,
		Latency:       promptScores.Latency,
	}

	return modelScores, feedback, nil
}

// mutateCandidate creates a mutated version of the candidate.
func (g *ParetoResponseGenerator) mutateCandidate(ctx context.Context, candidate *models.PathCandidate, feedback string) (*models.PathCandidate, error) {
	promptCandidate := g.convertToPromptCandidate(candidate)

	var promptTrace *prompt.ExecutionTrace
	if candidate.Trace != nil {
		promptTrace = g.convertToPromptTrace(candidate.Trace)
	} else {
		promptTrace = &prompt.ExecutionTrace{
			Query:       "",
			ToolCalls:   []prompt.ToolCallRecord{},
			FinalAnswer: "",
		}
	}

	mutatedPrompt, err := g.mutator.MutateStrategy(ctx, promptCandidate, promptTrace, feedback)
	if err != nil {
		return nil, fmt.Errorf("mutation failed: %w", err)
	}

	return g.convertFromPromptCandidate(mutatedPrompt), nil
}

// crossoverCandidates creates a child candidate by crossing over two parents.
func (g *ParetoResponseGenerator) crossoverCandidates(ctx context.Context, parent1, parent2 *models.PathCandidate) (*models.PathCandidate, error) {
	promptParent1 := g.convertToPromptCandidate(parent1)
	promptParent2 := g.convertToPromptCandidate(parent2)

	child, err := g.mutator.Crossover(ctx, promptParent1, promptParent2)
	if err != nil {
		return nil, fmt.Errorf("crossover failed: %w", err)
	}

	return g.convertFromPromptCandidate(child), nil
}

// selectDiversePair selects two diverse candidates for crossover.
func (g *ParetoResponseGenerator) selectDiversePair(archive *prompt.PathParetoArchive) (*models.PathCandidate, *models.PathCandidate) {
	front := archive.GetParetoFront()
	if len(front) < 2 {
		return nil, nil
	}

	var best1, best2 *models.PathCandidate
	maxDistance := 0.0

	for i := 0; i < len(front); i++ {
		for j := i + 1; j < len(front); j++ {
			distance := front[i].Scores.EuclideanDistance(front[j].Scores)
			if distance > maxDistance {
				maxDistance = distance
				best1 = front[i]
				best2 = front[j]
			}
		}
	}

	return best1, best2
}

// createToolUseFromTrace creates a ToolUse record from a tool call trace.
func (g *ParetoResponseGenerator) createToolUseFromTrace(ctx context.Context, messageID string, tc *models.ToolCallRecord, input *ParetoResponseInput) (*models.ToolUse, error) {
	if g.toolUseRepo == nil {
		return nil, nil
	}

	toolUseID := g.idGenerator.GenerateToolUseID()
	toolUse := models.NewToolUse(toolUseID, messageID, tc.ToolName, 0, tc.Arguments)

	if tc.Success {
		toolUse.Status = models.ToolStatusSuccess
		toolUse.Result = tc.Result
	} else {
		toolUse.Status = models.ToolStatusError
		toolUse.ErrorMessage = tc.Error
	}

	if err := g.toolUseRepo.Create(ctx, toolUse); err != nil {
		return nil, err
	}

	return toolUse, nil
}

// extractAndStoreMemories extracts memories from the response.
func (g *ParetoResponseGenerator) extractAndStoreMemories(ctx context.Context, message *models.Message, conversationID string) {
	if g.extractMemories == nil {
		return
	}

	output, err := g.extractMemories.Execute(ctx, &ExtractMemoriesInput{
		ConversationText: message.Contents,
		ConversationID:   conversationID,
		MessageID:        message.ID,
	})
	if err != nil {
		log.Printf("warning: failed to extract memories: %v", err)
		return
	}

	if len(output.CreatedMemories) > 0 {
		log.Printf("info: extracted %d memories from conversation", len(output.CreatedMemories))
	}
}

// estimateTokens provides a rough estimate of token usage.
func (g *ParetoResponseGenerator) estimateTokens(prompt, response string) int {
	totalChars := len(prompt) + len(response)
	return totalChars / 4
}

// --- Type conversion helpers ---

func (g *ParetoResponseGenerator) convertToPromptTrace(trace *models.ExecutionTrace) *prompt.ExecutionTrace {
	if trace == nil {
		return nil
	}

	toolCalls := make([]prompt.ToolCallRecord, len(trace.ToolCalls))
	for i, tc := range trace.ToolCalls {
		toolCalls[i] = prompt.ToolCallRecord{
			ToolName:  tc.ToolName,
			Arguments: tc.Arguments,
			Result:    tc.Result,
			Success:   tc.Success,
			Error:     tc.Error,
		}
	}

	return &prompt.ExecutionTrace{
		Query:          trace.Query,
		ToolCalls:      toolCalls,
		ReasoningSteps: trace.ReasoningSteps,
		FinalAnswer:    trace.FinalAnswer,
		TotalTokens:    trace.TotalTokens,
		DurationMs:     trace.DurationMs,
	}
}

func (g *ParetoResponseGenerator) convertToPromptCandidate(candidate *models.PathCandidate) *prompt.PathCandidate {
	if candidate == nil {
		return nil
	}

	var promptTrace *prompt.ExecutionTrace
	if candidate.Trace != nil {
		promptTrace = g.convertToPromptTrace(candidate.Trace)
	}

	return &prompt.PathCandidate{
		ID:                 candidate.ID,
		RunID:              candidate.RunID,
		Generation:         candidate.Generation,
		ParentIDs:          candidate.ParentIDs,
		StrategyPrompt:     candidate.StrategyPrompt,
		AccumulatedLessons: candidate.AccumulatedLessons,
		Trace:              promptTrace,
		Scores: prompt.PathScores{
			AnswerQuality: candidate.Scores.AnswerQuality,
			Efficiency:    candidate.Scores.Efficiency,
			TokenCost:     candidate.Scores.TokenCost,
			Robustness:    candidate.Scores.Robustness,
			Latency:       candidate.Scores.Latency,
		},
		CreatedAt: candidate.CreatedAt,
	}
}

func (g *ParetoResponseGenerator) convertFromPromptCandidate(candidate *prompt.PathCandidate) *models.PathCandidate {
	if candidate == nil {
		return nil
	}

	var modelTrace *models.ExecutionTrace
	if candidate.Trace != nil {
		toolCalls := make([]models.ToolCallRecord, len(candidate.Trace.ToolCalls))
		for i, tc := range candidate.Trace.ToolCalls {
			toolCalls[i] = models.ToolCallRecord{
				ToolName:  tc.ToolName,
				Arguments: tc.Arguments,
				Result:    tc.Result,
				Success:   tc.Success,
				Error:     tc.Error,
			}
		}
		modelTrace = &models.ExecutionTrace{
			Query:          candidate.Trace.Query,
			ToolCalls:      toolCalls,
			ReasoningSteps: candidate.Trace.ReasoningSteps,
			FinalAnswer:    candidate.Trace.FinalAnswer,
			TotalTokens:    candidate.Trace.TotalTokens,
			DurationMs:     candidate.Trace.DurationMs,
		}
	}

	return &models.PathCandidate{
		ID:                 candidate.ID,
		RunID:              candidate.RunID,
		Generation:         candidate.Generation,
		ParentIDs:          candidate.ParentIDs,
		StrategyPrompt:     candidate.StrategyPrompt,
		AccumulatedLessons: candidate.AccumulatedLessons,
		Trace:              modelTrace,
		Scores: models.PathScores{
			AnswerQuality: candidate.Scores.AnswerQuality,
			Efficiency:    candidate.Scores.Efficiency,
			TokenCost:     candidate.Scores.TokenCost,
			Robustness:    candidate.Scores.Robustness,
			Latency:       candidate.Scores.Latency,
		},
		CreatedAt: candidate.CreatedAt,
	}
}

// Ensure ParetoResponseGenerator implements the ports interface
var _ ports.ParetoResponseGenerator = (*ParetoResponseGenerator)(nil)
