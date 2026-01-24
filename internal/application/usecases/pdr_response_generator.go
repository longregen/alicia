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
)

// DefaultPDRResponseConfig returns sensible defaults for PDR response generation.
func DefaultPDRResponseConfig() *ports.PDRResponseConfig {
	return &ports.PDRResponseConfig{
		Rounds:                  2,     // Two refinement rounds
		ParallelDrafts:          3,     // Three parallel drafts per round
		WorkspaceTokenLimit:     1024,  // ~1K tokens for distilled workspace
		MaxToolLoopIterations:   5,     // Tool loop budget per draft
		MaxToolCalls:            50,    // Total tool call budget
		MaxLLMCalls:             30,    // Total LLM call budget
		ExecutionTimeoutMs:      90000, // 90 seconds
		EnableParallelExecution: true,
	}
}

// PDRResponseGenerator generates responses using the Parallel-Distill-Refine approach.
// Based on "Rethinking Thinking Tokens" (arXiv:2510.01123), it decouples sequential
// budget (latency) from total compute by:
//  1. Parallel: generating M drafts conditioned on a compact summary
//  2. Distill: compressing all outputs into a bounded workspace (≤κ tokens)
//  3. Refine: generating an improved response using the distilled workspace
type PDRResponseGenerator struct {
	llmService       ports.LLMService
	messageRepo      ports.MessageRepository
	conversationRepo ports.ConversationRepository
	toolRepo         ports.ToolRepository
	sentenceRepo     ports.SentenceRepository
	toolUseRepo      ports.ToolUseRepository
	reasoningRepo    ports.ReasoningStepRepository
	memoryUsageRepo  ports.MemoryUsageRepository
	toolService      ports.ToolService
	memoryService    ports.MemoryService
	idGenerator      ports.IDGenerator
	txManager        ports.TransactionManager
	config           *ports.PDRResponseConfig
	titleGenerator   *GenerateConversationTitle
	extractMemories  *ExtractMemories
}

// draftResult holds the output of a single parallel draft execution.
type draftResult struct {
	answer   string
	trace    *models.ExecutionTrace
	err      error
	draftIdx int
}

// NewPDRResponseGenerator creates a new PDR response generator.
func NewPDRResponseGenerator(
	llmService ports.LLMService,
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
	config *ports.PDRResponseConfig,
) *PDRResponseGenerator {
	if config == nil {
		config = DefaultPDRResponseConfig()
	}

	gen := &PDRResponseGenerator{
		llmService:       llmService,
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
		config:           config,
	}

	gen.titleGenerator = NewGenerateConversationTitle(conversationRepo, messageRepo, llmService, broadcaster)

	if memoryService != nil {
		gen.extractMemories = NewExtractMemories(memoryService, llmService, idGenerator)
	}

	return gen
}

// Execute generates a response using the Parallel-Distill-Refine approach.
// Implements ports.ParetoResponseGenerator for drop-in compatibility.
func (g *PDRResponseGenerator) Execute(ctx context.Context, input *ports.ParetoResponseInput) (*ports.ParetoResponseOutput, error) {
	log.Printf("[PDR] Execute called for conversation=%s, userMessage=%s", input.ConversationID, input.UserMessageID)

	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}
	if input.ConversationID == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}
	if input.UserMessageID == "" {
		return nil, fmt.Errorf("user message ID is required")
	}

	conversation, err := g.conversationRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	userMessage, err := g.messageRepo.GetByID(ctx, input.UserMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user message: %w", err)
	}

	messages, err := g.messageRepo.GetLatestByConversation(ctx, input.ConversationID, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	// Retrieve relevant memories
	var relevantMemories []*models.Memory
	if g.memoryService != nil {
		searchResults, err := g.memoryService.SearchWithScores(ctx, userMessage.Contents, 0.7, 5)
		if err != nil {
			log.Printf("[PDR] WARNING: failed to retrieve memories: %v", err)
		} else {
			relevantMemories = make([]*models.Memory, len(searchResults))
			for i, result := range searchResults {
				relevantMemories[i] = result.Memory
				if input.Notifier != nil {
					input.Notifier.NotifyMemoryRetrieved(
						input.UserMessageID,
						input.ConversationID,
						result.Memory.ID,
						result.Memory.Content,
						result.Similarity,
					)
				}
				_, _ = g.memoryService.TrackUsage(ctx, result.Memory.ID, input.ConversationID, input.UserMessageID, result.Similarity)
			}
		}
	}

	// Get available tools
	var tools []*models.Tool
	if input.EnableTools && g.toolService != nil {
		tools, err = g.toolService.ListAvailable(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get available tools: %w", err)
		}
	}

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

	if err := g.conversationRepo.UpdateTip(ctx, input.ConversationID, message.ID); err != nil {
		return nil, fmt.Errorf("failed to update conversation tip: %w", err)
	}

	if input.Notifier != nil {
		input.Notifier.NotifyGenerationStarted(message.ID, input.PreviousID, input.ConversationID)
	}

	// Generate thinking summary
	if input.Notifier != nil {
		go func() {
			summary := g.generateThinkingSummary(ctx, userMessage.Contents, tools, relevantMemories)
			if summary != "" {
				input.Notifier.NotifyThinkingSummary(message.ID, input.ConversationID, summary)
			}
		}()
	}

	// Build query context
	query := g.buildQueryWithContext(userMessage.Contents, messages, relevantMemories, conversation)

	// Create tool runner
	var toolRunner ports.ToolRunner
	if g.toolService != nil {
		toolRunner = ports.ToolRunnerFunc(func(ctx context.Context, toolName string, arguments map[string]any) (any, error) {
			return g.toolService.ExecuteTool(ctx, toolName, arguments)
		})
	}

	// Run the PDR cycle
	timeout := time.Duration(g.config.ExecutionTimeoutMs) * time.Millisecond
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()
	result, err := g.executePDR(timeoutCtx, query, tools, toolRunner, input)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("[PDR] Execution FAILED after %v: %v", duration, err)
		message.MarkAsFailed()
		_ = g.messageRepo.Update(ctx, message)
		if input.Notifier != nil {
			input.Notifier.NotifyGenerationFailed(message.ID, input.ConversationID, err)
		}
		return nil, fmt.Errorf("PDR execution failed: %w", err)
	}

	log.Printf("[PDR] Completed in %v: rounds=%d, answer=%d chars", duration, result.rounds, len(result.answer))

	// Update message
	message.Contents = strings.TrimSpace(result.answer)
	message.MarkAsCompleted()
	if err := g.messageRepo.Update(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	// Persist tool uses from the final trace
	var toolUses []*models.ToolUse
	if result.trace != nil {
		for _, tc := range result.trace.ToolCalls {
			toolUse, err := g.createToolUseFromTrace(ctx, message.ID, &tc)
			if err != nil {
				log.Printf("[PDR] WARNING: failed to create tool use record: %v", err)
				continue
			}
			toolUses = append(toolUses, toolUse)
		}
	}

	if input.Notifier != nil {
		input.Notifier.NotifyGenerationComplete(message.ID, input.ConversationID, message.Contents)
	}

	// Extract memories asynchronously
	go func() {
		memCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		g.extractAndStoreMemories(memCtx, message, input.ConversationID)
	}()

	g.titleGenerator.ExecuteAsync(ctx, input.ConversationID)

	return &ports.ParetoResponseOutput{
		Message:    message,
		ToolUses:   toolUses,
		Score:      1.0,
		Iterations: result.rounds,
	}, nil
}

// pdrResult holds the final output of the PDR cycle.
type pdrResult struct {
	answer string
	trace  *models.ExecutionTrace
	rounds int
}

// executePDR runs the full Parallel-Distill-Refine cycle for R rounds.
func (g *PDRResponseGenerator) executePDR(
	ctx context.Context,
	query string,
	tools []*models.Tool,
	toolRunner ports.ToolRunner,
	input *ports.ParetoResponseInput,
) (*pdrResult, error) {
	rounds := g.config.Rounds
	if rounds < 1 {
		rounds = 1
	}

	// workspace accumulates the distilled context across rounds
	workspace := ""
	var bestTrace *models.ExecutionTrace
	var bestAnswer string

	for round := 0; round < rounds; round++ {
		log.Printf("[PDR] Round %d/%d: generating %d parallel drafts", round+1, rounds, g.config.ParallelDrafts)

		// Phase 1: Parallel - generate M drafts
		drafts, err := g.parallelPhase(ctx, query, workspace, tools, toolRunner, input, round)
		if err != nil {
			// If we have a previous answer, use it instead of failing
			if bestAnswer != "" {
				log.Printf("[PDR] Round %d failed, using previous round's answer: %v", round+1, err)
				break
			}
			return nil, fmt.Errorf("parallel phase failed on round %d: %w", round+1, err)
		}

		// Collect successful drafts
		var successfulDrafts []draftResult
		for _, d := range drafts {
			if d.err == nil && d.answer != "" {
				successfulDrafts = append(successfulDrafts, d)
			}
		}

		if len(successfulDrafts) == 0 {
			if bestAnswer != "" {
				log.Printf("[PDR] No successful drafts in round %d, using previous answer", round+1)
				break
			}
			return nil, fmt.Errorf("no successful drafts in round %d", round+1)
		}

		log.Printf("[PDR] Round %d: %d/%d drafts succeeded", round+1, len(successfulDrafts), g.config.ParallelDrafts)

		// Phase 2: Distill - compress drafts into bounded workspace
		distilled, err := g.distillPhase(ctx, query, successfulDrafts)
		if err != nil {
			log.Printf("[PDR] Distill phase failed on round %d: %v, using best draft directly", round+1, err)
			// Fallback: use the longest draft as workspace
			for _, d := range successfulDrafts {
				if len(d.answer) > len(workspace) {
					workspace = d.answer
					bestTrace = d.trace
					bestAnswer = d.answer
				}
			}
		} else {
			workspace = distilled
		}

		// Phase 3: Refine - generate final answer using distilled workspace
		refined, trace, err := g.refinePhase(ctx, query, workspace, tools, toolRunner, input)
		if err != nil {
			log.Printf("[PDR] Refine phase failed on round %d: %v", round+1, err)
			// Use best draft from this round as fallback
			if bestAnswer == "" && len(successfulDrafts) > 0 {
				bestAnswer = successfulDrafts[0].answer
				bestTrace = successfulDrafts[0].trace
			}
		} else {
			bestAnswer = refined
			bestTrace = trace
		}

		log.Printf("[PDR] Round %d complete: refined answer=%d chars, workspace=%d chars",
			round+1, len(bestAnswer), len(workspace))
	}

	if bestAnswer == "" {
		return nil, fmt.Errorf("PDR produced no answer after %d rounds", rounds)
	}

	return &pdrResult{
		answer: bestAnswer,
		trace:  bestTrace,
		rounds: rounds,
	}, nil
}

// parallelPhase generates M parallel drafts conditioned on the current workspace.
func (g *PDRResponseGenerator) parallelPhase(
	ctx context.Context,
	query, workspace string,
	tools []*models.Tool,
	toolRunner ports.ToolRunner,
	input *ports.ParetoResponseInput,
	round int,
) ([]draftResult, error) {
	numDrafts := g.config.ParallelDrafts
	if numDrafts < 1 {
		numDrafts = 1
	}

	results := make([]draftResult, numDrafts)

	if g.config.EnableParallelExecution && numDrafts > 1 {
		// Execute drafts concurrently
		var wg sync.WaitGroup
		wg.Add(numDrafts)

		for i := 0; i < numDrafts; i++ {
			go func(idx int) {
				defer wg.Done()
				answer, trace, err := g.generateDraft(ctx, query, workspace, tools, toolRunner, input, round, idx)
				results[idx] = draftResult{
					answer:   answer,
					trace:    trace,
					err:      err,
					draftIdx: idx,
				}
			}(i)
		}

		wg.Wait()
	} else {
		// Execute drafts sequentially
		for i := 0; i < numDrafts; i++ {
			answer, trace, err := g.generateDraft(ctx, query, workspace, tools, toolRunner, input, round, i)
			results[i] = draftResult{
				answer:   answer,
				trace:    trace,
				err:      err,
				draftIdx: i,
			}
		}
	}

	return results, nil
}

// generateDraft produces a single draft response, using the workspace as conditioning.
func (g *PDRResponseGenerator) generateDraft(
	ctx context.Context,
	query, workspace string,
	tools []*models.Tool,
	toolRunner ports.ToolRunner,
	input *ports.ParetoResponseInput,
	round, draftIdx int,
) (string, *models.ExecutionTrace, error) {
	startTime := time.Now()

	// Build draft-specific prompt incorporating workspace context
	systemPrompt := g.buildDraftPrompt(query, workspace, tools, round, draftIdx)

	messages := []ports.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}

	// If we have workspace from prior rounds, include it as context
	if workspace != "" {
		messages = append(messages[:1], append([]ports.LLMMessage{
			{Role: "assistant", Content: fmt.Sprintf("Previous analysis summary:\n%s", workspace)},
			{Role: "user", Content: fmt.Sprintf("Building on the above analysis, provide draft %d for: %s", draftIdx+1, query)},
		}, messages[2:]...)...)
	}

	// Run tool loop if tools are available
	if toolRunner != nil && len(tools) > 0 {
		trace, err := g.executeToolLoop(ctx, messages, tools, toolRunner, input, startTime)
		if err != nil {
			return "", nil, err
		}
		return trace.FinalAnswer, trace, nil
	}

	// Single-turn without tools
	response, err := g.llmService.Chat(ctx, messages)
	if err != nil {
		return "", nil, fmt.Errorf("draft LLM call failed: %w", err)
	}

	durationMs := time.Since(startTime).Milliseconds()
	trace := &models.ExecutionTrace{
		Query:       query,
		FinalAnswer: response.Content,
		TotalTokens: g.estimateTokens(systemPrompt, response.Content),
		DurationMs:  durationMs,
	}

	return response.Content, trace, nil
}

// distillPhase compresses multiple draft outputs into a bounded workspace.
// This is the key innovation: it keeps the sequential context (B_seq) bounded
// while allowing total compute (B_total) to grow with more drafts.
func (g *PDRResponseGenerator) distillPhase(
	ctx context.Context,
	query string,
	drafts []draftResult,
) (string, error) {
	if len(drafts) == 1 {
		// Single draft: just truncate to workspace limit
		return g.truncateToWorkspace(drafts[0].answer), nil
	}

	// Build distillation prompt
	var sb strings.Builder
	sb.WriteString("You are a synthesis assistant. Below are multiple draft responses to the same query. ")
	sb.WriteString("Extract and consolidate the key insights, facts, reasoning steps, and conclusions ")
	sb.WriteString("into a compact summary. Preserve all important information but remove redundancy.\n\n")
	sb.WriteString(fmt.Sprintf("Original query: %s\n\n", query))

	for i, draft := range drafts {
		sb.WriteString(fmt.Sprintf("=== Draft %d ===\n%s\n\n", i+1, draft.answer))
	}

	sb.WriteString(fmt.Sprintf("Provide a consolidated summary in at most %d words that captures ", g.config.WorkspaceTokenLimit*3/4))
	sb.WriteString("all unique insights from the drafts. Focus on:\n")
	sb.WriteString("1. Key facts and data points discovered\n")
	sb.WriteString("2. Reasoning steps that led to conclusions\n")
	sb.WriteString("3. Tool results and their implications\n")
	sb.WriteString("4. Areas of agreement and disagreement between drafts\n")
	sb.WriteString("5. The strongest conclusions supported by evidence\n")

	messages := []ports.LLMMessage{
		{Role: "user", Content: sb.String()},
	}

	response, err := g.llmService.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("distillation LLM call failed: %w", err)
	}

	return g.truncateToWorkspace(response.Content), nil
}

// refinePhase generates a final refined response using the distilled workspace.
func (g *PDRResponseGenerator) refinePhase(
	ctx context.Context,
	query, workspace string,
	tools []*models.Tool,
	toolRunner ports.ToolRunner,
	input *ports.ParetoResponseInput,
) (string, *models.ExecutionTrace, error) {
	startTime := time.Now()

	systemPrompt := g.buildRefinePrompt(query, workspace, tools)

	messages := []ports.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}

	// Run tool loop if tools are available for refinement
	if toolRunner != nil && len(tools) > 0 {
		trace, err := g.executeToolLoop(ctx, messages, tools, toolRunner, input, startTime)
		if err != nil {
			return "", nil, err
		}
		return trace.FinalAnswer, trace, nil
	}

	response, err := g.llmService.Chat(ctx, messages)
	if err != nil {
		return "", nil, fmt.Errorf("refine LLM call failed: %w", err)
	}

	durationMs := time.Since(startTime).Milliseconds()
	trace := &models.ExecutionTrace{
		Query:       query,
		FinalAnswer: response.Content,
		TotalTokens: g.estimateTokens(systemPrompt, response.Content),
		DurationMs:  durationMs,
	}

	return response.Content, trace, nil
}

// executeToolLoop performs multi-turn tool execution, shared by draft and refine phases.
func (g *PDRResponseGenerator) executeToolLoop(
	ctx context.Context,
	messages []ports.LLMMessage,
	tools []*models.Tool,
	toolRunner ports.ToolRunner,
	input *ports.ParetoResponseInput,
	startTime time.Time,
) (*models.ExecutionTrace, error) {
	currentMessages := make([]ports.LLMMessage, len(messages))
	copy(currentMessages, messages)

	var allToolCalls []models.ToolCallRecord
	var finalAnswer string
	var totalTokens int

	maxIterations := g.config.MaxToolLoopIterations
	if maxIterations < 1 {
		maxIterations = 5
	}

	for iteration := 0; iteration < maxIterations; iteration++ {
		response, err := g.llmService.ChatWithTools(ctx, currentMessages, tools)
		if err != nil {
			return nil, fmt.Errorf("tool loop LLM call failed on iteration %d: %w", iteration, err)
		}

		totalTokens += g.estimateTokens("", response.Content)

		currentMessages = append(currentMessages, ports.LLMMessage{
			Role:    "assistant",
			Content: response.Content,
		})

		if len(response.ToolCalls) == 0 {
			finalAnswer = response.Content
			break
		}

		for _, tc := range response.ToolCalls {
			toolRecord := models.ToolCallRecord{
				ToolName:  tc.Name,
				Arguments: tc.Arguments,
				Success:   false,
			}

			if input.Notifier != nil {
				input.Notifier.NotifyToolUseStart(tc.ID, "", input.ConversationID, tc.Name, tc.Arguments)
			}

			result, execErr := toolRunner.RunTool(ctx, tc.Name, tc.Arguments)
			if execErr != nil {
				toolRecord.Error = execErr.Error()
				currentMessages = append(currentMessages, ports.LLMMessage{
					Role:    "tool",
					Content: fmt.Sprintf("Error executing %s: %s", tc.Name, execErr.Error()),
				})
				if input.Notifier != nil {
					input.Notifier.NotifyToolUseComplete(tc.ID, tc.ID, input.ConversationID, false, nil, execErr.Error())
				}
			} else {
				toolRecord.Success = true
				toolRecord.Result = result
				currentMessages = append(currentMessages, ports.LLMMessage{
					Role:    "tool",
					Content: fmt.Sprintf("%v", result),
				})
				if input.Notifier != nil {
					input.Notifier.NotifyToolUseComplete(tc.ID, tc.ID, input.ConversationID, true, result, "")
				}
			}

			allToolCalls = append(allToolCalls, toolRecord)
		}

		if iteration == maxIterations-1 {
			finalAnswer = response.Content
			if finalAnswer == "" {
				finalAnswer = "Max tool execution iterations reached."
			}
		}
	}

	durationMs := time.Since(startTime).Milliseconds()

	return &models.ExecutionTrace{
		Query:       messages[len(messages)-1].Content,
		ToolCalls:   allToolCalls,
		FinalAnswer: finalAnswer,
		TotalTokens: totalTokens,
		DurationMs:  durationMs,
	}, nil
}

// buildDraftPrompt constructs the prompt for a parallel draft.
func (g *PDRResponseGenerator) buildDraftPrompt(query, workspace string, tools []*models.Tool, round, draftIdx int) string {
	var sb strings.Builder

	sb.WriteString("You are Alicia, a helpful AI assistant. ")
	sb.WriteString("Think step by step and provide a thorough, accurate answer.\n\n")

	// Diversity instruction to encourage different approaches across drafts
	approaches := []string{
		"Focus on being comprehensive and covering all aspects of the question.",
		"Focus on being concise and precise, prioritizing the most important information.",
		"Focus on creative problem-solving and considering alternative perspectives.",
		"Focus on systematic analysis, breaking the problem into clear steps.",
		"Focus on practical implications and actionable insights.",
	}
	if draftIdx < len(approaches) {
		sb.WriteString(fmt.Sprintf("Approach: %s\n\n", approaches[draftIdx]))
	}

	if workspace != "" {
		sb.WriteString("Context from previous analysis (use as reference, but develop your own answer):\n")
		sb.WriteString(workspace)
		sb.WriteString("\n\n")
	}

	if len(tools) > 0 {
		sb.WriteString("Available tools:\n")
		for _, tool := range tools {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		}
		sb.WriteString("\nUse tools as needed to find the best answer.\n")
	}

	return sb.String()
}

// buildRefinePrompt constructs the prompt for the refinement phase.
func (g *PDRResponseGenerator) buildRefinePrompt(query, workspace string, tools []*models.Tool) string {
	var sb strings.Builder

	sb.WriteString("You are Alicia, a helpful AI assistant. ")
	sb.WriteString("You have access to a consolidated analysis from multiple perspectives. ")
	sb.WriteString("Use this to produce a final, high-quality response.\n\n")

	sb.WriteString("Consolidated analysis:\n")
	sb.WriteString(workspace)
	sb.WriteString("\n\n")

	sb.WriteString("Instructions:\n")
	sb.WriteString("1. Synthesize the key insights from the analysis above\n")
	sb.WriteString("2. Resolve any contradictions by reasoning carefully\n")
	sb.WriteString("3. Produce a clear, accurate, and well-structured final answer\n")
	sb.WriteString("4. If the analysis is incomplete, use tools to fill gaps\n\n")

	if len(tools) > 0 {
		sb.WriteString("Available tools:\n")
		for _, tool := range tools {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// truncateToWorkspace truncates content to fit within the workspace token limit.
func (g *PDRResponseGenerator) truncateToWorkspace(content string) string {
	// Approximate: 1 token ≈ 4 characters
	maxChars := g.config.WorkspaceTokenLimit * 4
	if len(content) <= maxChars {
		return content
	}
	return content[:maxChars]
}

// buildQueryWithContext builds the query with conversation context and memories.
func (g *PDRResponseGenerator) buildQueryWithContext(
	userQuery string,
	messages []*models.Message,
	memories []*models.Memory,
	conversation *models.Conversation,
) string {
	var sb strings.Builder

	sb.WriteString("You are Alicia, a helpful AI assistant with memory and tool capabilities.\n\n")

	if len(memories) > 0 {
		sb.WriteString("Relevant memories from previous conversations:\n")
		for i, memory := range memories {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, memory.Content))
		}
		sb.WriteString("\n")
	}

	if len(messages) > 1 {
		sb.WriteString("Recent conversation:\n")
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

	sb.WriteString("User query: ")
	sb.WriteString(userQuery)

	return sb.String()
}

// generateThinkingSummary generates a brief summary of what the agent is about to do.
func (g *PDRResponseGenerator) generateThinkingSummary(ctx context.Context, userQuery string, tools []*models.Tool, memories []*models.Memory) string {
	var toolNames []string
	for _, t := range tools {
		if t != nil {
			toolNames = append(toolNames, t.Name)
		}
	}

	memoryHint := ""
	if len(memories) > 0 {
		memoryHint = fmt.Sprintf(" I have %d relevant memories to consider.", len(memories))
	}

	toolHint := ""
	if len(toolNames) > 0 {
		toolHint = fmt.Sprintf(" Available tools: %s.", strings.Join(toolNames, ", "))
	}

	prompt := fmt.Sprintf(`Given this user question, write a single short sentence (max 15 words) describing what you're about to do to answer it. Be specific and action-oriented. Don't use "I will" - just describe the action.

User question: %s
%s%s

Response (one short sentence):`, userQuery, memoryHint, toolHint)

	summaryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	llmMessages := []ports.LLMMessage{
		{Role: "user", Content: prompt},
	}

	response, err := g.llmService.Chat(summaryCtx, llmMessages)
	if err != nil {
		return ""
	}

	if response == nil || response.Content == "" {
		return ""
	}

	summary := strings.TrimSpace(response.Content)
	summary = strings.Trim(summary, "\"'")
	if len(summary) > 100 {
		summary = summary[:97] + "..."
	}

	return summary
}

// createToolUseFromTrace creates a ToolUse record from a tool call trace.
func (g *PDRResponseGenerator) createToolUseFromTrace(ctx context.Context, messageID string, tc *models.ToolCallRecord) (*models.ToolUse, error) {
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
func (g *PDRResponseGenerator) extractAndStoreMemories(ctx context.Context, message *models.Message, conversationID string) {
	if g.extractMemories == nil {
		return
	}

	output, err := g.extractMemories.Execute(ctx, &ExtractMemoriesInput{
		ConversationText: message.Contents,
		ConversationID:   conversationID,
		MessageID:        message.ID,
	})
	if err != nil {
		log.Printf("[PDR] WARNING: failed to extract memories: %v", err)
		return
	}

	if len(output.CreatedMemories) > 0 {
		log.Printf("[PDR] extracted %d memories from conversation", len(output.CreatedMemories))
	}
}

// estimateTokens provides a rough estimate of token usage.
func (g *PDRResponseGenerator) estimateTokens(prompt, response string) int {
	return (len(prompt) + len(response)) / 4
}

// Ensure PDRResponseGenerator implements the ports interface
var _ ports.ParetoResponseGenerator = (*PDRResponseGenerator)(nil)
