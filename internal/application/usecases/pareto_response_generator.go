package usecases

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Type aliases for backwards compatibility and convenience.
// These types are defined in ports and re-exported here for ease of use.
type (
	// ParetoResponseConfig configures the response generation.
	// See ports.ParetoResponseConfig for full documentation.
	ParetoResponseConfig = ports.ParetoResponseConfig

	// ParetoResponseInput contains the input parameters for response generation.
	// See ports.ParetoResponseInput for full documentation.
	ParetoResponseInput = ports.ParetoResponseInput

	// ParetoResponseOutput contains the result of response generation.
	// See ports.ParetoResponseOutput for full documentation.
	ParetoResponseOutput = ports.ParetoResponseOutput
)

// DefaultParetoResponseConfig returns sensible defaults for response generation.
func DefaultParetoResponseConfig() *ParetoResponseConfig {
	return &ParetoResponseConfig{
		MaxGenerations:         1,   // Single-pass generation
		BranchesPerGen:         1,   // Single path
		TargetScore:            0.8, // Target 80% quality score
		MaxToolCalls:           50,  // Budget: total tool calls
		MaxLLMCalls:            30,  // Budget: total LLM calls
		ParetoArchiveSize:      1,   // No archive needed
		EnableCrossover:        false,
		ExecutionTimeoutMs:     60000, // 60 seconds per execution
		EnableParallelBranches: false,
		MaxParallelBranches:    1,
		MaxToolLoopIterations:  5,
	}
}

// ParetoResponseGenerator generates responses using a straightforward approach.
// It executes tools and persists results.
type ParetoResponseGenerator struct {
	// Core LLM services
	llmService ports.LLMService

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

	// Default configuration
	config *ParetoResponseConfig

	// Title generator for new conversations
	titleGenerator *GenerateConversationTitle

	// Memory extraction
	extractMemories     *ExtractMemories
	memorizeFromToolUse *MemorizeFromToolUse
}

// executionContext holds per-execution state.
type paretoExecutionContext struct {
	config     *ParetoResponseConfig
	tools      []*models.Tool
	toolRunner ports.ToolRunner
}

// NewParetoResponseGenerator creates a new response generator.
func NewParetoResponseGenerator(
	llmService ports.LLMService,
	reflectionLLM ports.LLMService, // kept for API compatibility, unused
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

	gen := &ParetoResponseGenerator{
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

	// Initialize title generator
	gen.titleGenerator = NewGenerateConversationTitle(conversationRepo, messageRepo, llmService, broadcaster)

	// Initialize memory extraction
	if memoryService != nil {
		gen.extractMemories = NewExtractMemories(memoryService, llmService, idGenerator)
		gen.memorizeFromToolUse = NewMemorizeFromToolUse(llmService, memoryService, gen.extractMemories)
	}

	return gen
}

// Execute generates a response using a straightforward approach.
// This is the main entry point for response generation.
func (g *ParetoResponseGenerator) Execute(ctx context.Context, input *ParetoResponseInput) (*ParetoResponseOutput, error) {
	log.Printf("[ResponseGenerator] Execute called for conversation=%s, userMessage=%s, previousID=%s", input.ConversationID, input.UserMessageID, input.PreviousID)

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
	log.Printf("[ResponseGenerator] Loaded conversation: %s (title=%q)", conversation.ID, conversation.Title)

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
	log.Printf("[ResponseGenerator] Retrieving relevant memories...")
	var relevantMemories []*models.Memory
	if g.memoryService != nil {
		searchResults, err := g.memoryService.SearchWithScores(ctx, userMessage.Contents, 0.7, 5)
		if err != nil {
			log.Printf("[ResponseGenerator] WARNING: failed to retrieve memories: %v", err)
		} else {
			log.Printf("[ResponseGenerator] Found %d relevant memories", len(searchResults))
			relevantMemories = make([]*models.Memory, len(searchResults))
			for i, result := range searchResults {
				relevantMemories[i] = result.Memory
				log.Printf("[ResponseGenerator]   Memory[%d]: id=%s similarity=%.3f content=%q",
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
		log.Printf("[ResponseGenerator] Memory service not available, skipping memory retrieval")
	}

	// Get available tools (only those with registered executors)
	log.Printf("[ResponseGenerator] Loading tools (enableTools=%v)...", input.EnableTools)
	var tools []*models.Tool
	if input.EnableTools && g.toolService != nil {
		tools, err = g.toolService.ListAvailable(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get available tools: %w", err)
		}
		log.Printf("[ResponseGenerator] Loaded %d available tools (with executors)", len(tools))
		for i, tool := range tools {
			log.Printf("[ResponseGenerator]   Tool[%d]: %s", i, tool.Name)
		}
	} else {
		log.Printf("[ResponseGenerator] Tools disabled or toolService not available")
	}

	// Create per-execution context
	execCtx := g.newExecutionContext(input.Config, tools)

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
	log.Printf("[ResponseGenerator] Creating assistant message: id=%s, previousID=%s (input.PreviousID=%s)", message.ID, message.PreviousID, input.PreviousID)

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

	// Generate and send thinking summary
	if input.Notifier != nil {
		go func() {
			summary := g.generateThinkingSummary(ctx, userMessage.Contents, tools, relevantMemories)
			if summary != "" {
				input.Notifier.NotifyThinkingSummary(message.ID, input.ConversationID, summary)
			}
		}()
	}

	// Build the query from user message and context
	query := g.buildQueryWithContext(userMessage.Contents, messages, relevantMemories, conversation)
	log.Printf("[ResponseGenerator] Built query context (%d chars) for response generation", len(query))

	// Generate response
	log.Printf("[ResponseGenerator] Starting response generation...")
	startTime := time.Now()
	result, err := g.generateResponse(ctx, query, input, message, execCtx)
	duration := time.Since(startTime)
	if err != nil {
		log.Printf("[ResponseGenerator] Response generation FAILED after %v: %v", duration, err)
		// Mark message as failed
		message.MarkAsFailed()
		_ = g.messageRepo.Update(ctx, message)
		if input.Notifier != nil {
			input.Notifier.NotifyGenerationFailed(message.ID, input.ConversationID, err)
		}
		return nil, fmt.Errorf("response generation failed: %w", err)
	}
	log.Printf("[ResponseGenerator] Response generation completed in %v: answer=%d chars",
		duration, len(result.Answer))

	// Update message with response
	message.Contents = strings.TrimSpace(result.Answer)
	message.MarkAsCompleted()
	if err := g.messageRepo.Update(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	// Create tool uses from the trace
	var toolUses []*models.ToolUse
	if result.BestPath != nil && result.BestPath.Trace != nil {
		log.Printf("[ResponseGenerator] Path has %d tool calls", len(result.BestPath.Trace.ToolCalls))
		for i, tc := range result.BestPath.Trace.ToolCalls {
			log.Printf("[ResponseGenerator]   ToolCall[%d]: %s success=%v", i, tc.ToolName, tc.Success)
			toolUse, err := g.createToolUseFromTrace(ctx, message.ID, &tc, input)
			if err != nil {
				log.Printf("[ResponseGenerator] WARNING: failed to create tool use record: %v", err)
				continue
			}
			toolUses = append(toolUses, toolUse)
		}
	} else {
		log.Printf("[ResponseGenerator] No tool calls in path")
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

	log.Printf("[ResponseGenerator] Execute complete: messageID=%s toolUses=%d",
		message.ID, len(toolUses))

	return &ParetoResponseOutput{
		Message:    message,
		ToolUses:   toolUses,
		Score:      1.0,
		Iterations: 1,
	}, nil
}

// newExecutionContext creates a per-execution context.
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
		if inputConfig.MaxToolCalls > 0 {
			cfg.MaxToolCalls = inputConfig.MaxToolCalls
		}
		if inputConfig.MaxLLMCalls > 0 {
			cfg.MaxLLMCalls = inputConfig.MaxLLMCalls
		}
		if inputConfig.ExecutionTimeoutMs > 0 {
			cfg.ExecutionTimeoutMs = inputConfig.ExecutionTimeoutMs
		}
		if inputConfig.MaxToolLoopIterations > 0 {
			cfg.MaxToolLoopIterations = inputConfig.MaxToolLoopIterations
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
		config:     cfg,
		tools:      tools,
		toolRunner: toolRunner,
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

// generateThinkingSummary generates a brief summary of what the agent is about to do.
func (g *ParetoResponseGenerator) generateThinkingSummary(ctx context.Context, userQuery string, tools []*models.Tool, memories []*models.Memory) string {
	// Build a list of available tool names
	var toolNames []string
	for _, t := range tools {
		if t != nil {
			toolNames = append(toolNames, t.Name)
		}
	}

	// Build memory context hint
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

	// Use a short timeout for thinking summary - it should be fast
	summaryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Use Chat with a single user message
	messages := []ports.LLMMessage{
		{Role: "user", Content: prompt},
	}

	response, err := g.llmService.Chat(summaryCtx, messages)
	if err != nil {
		log.Printf("[ResponseGenerator] Failed to generate thinking summary: %v", err)
		return ""
	}

	if response == nil || response.Content == "" {
		return ""
	}

	// Clean up the response - remove quotes, trim whitespace
	summary := strings.TrimSpace(response.Content)
	summary = strings.Trim(summary, "\"'")

	// Limit length
	if len(summary) > 100 {
		summary = summary[:97] + "..."
	}

	return summary
}

// generateResponse generates a response using the LLM with tool loop.
func (g *ParetoResponseGenerator) generateResponse(
	ctx context.Context,
	query string,
	input *ParetoResponseInput,
	message *models.Message,
	execCtx *paretoExecutionContext,
) (*models.PathSearchResult, error) {
	timeout := time.Duration(execCtx.config.ExecutionTimeoutMs) * time.Millisecond
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()

	// Build agent prompt
	agentPrompt := g.buildAgentPrompt(query, execCtx.tools)

	// Create initial messages
	messages := []ports.LLMMessage{
		{Role: "system", Content: agentPrompt},
		{Role: "user", Content: query},
	}

	// Run tool loop if tools available
	if execCtx.toolRunner != nil && len(execCtx.tools) > 0 {
		log.Printf("[ResponseGenerator] Executing with tool loop (tools=%d, maxIterations=%d, timeout=%v)",
			len(execCtx.tools), execCtx.config.MaxToolLoopIterations, timeout)
		trace, err := g.executeWithToolLoop(timeoutCtx, messages, agentPrompt, query, startTime, execCtx, input)
		if err != nil {
			return nil, err
		}
		return &models.PathSearchResult{
			BestPath: &models.PathCandidate{
				Trace: trace,
			},
			Answer:     trace.FinalAnswer,
			Score:      1.0,
			Iterations: 1,
		}, nil
	}

	// Single-turn execution without tools
	log.Printf("[ResponseGenerator] Executing single-turn (no tools or toolRunner)")
	trace, err := g.executeSingleTurn(timeoutCtx, messages, agentPrompt, query, startTime, execCtx)
	if err != nil {
		return nil, err
	}
	return &models.PathSearchResult{
		BestPath: &models.PathCandidate{
			Trace: trace,
		},
		Answer:     trace.FinalAnswer,
		Score:      1.0,
		Iterations: 1,
	}, nil
}

// buildAgentPrompt constructs the full prompt for agent execution.
func (g *ParetoResponseGenerator) buildAgentPrompt(query string, tools []*models.Tool) string {
	var sb strings.Builder

	sb.WriteString("You are Alicia, a helpful AI assistant. Think step by step and provide clear, accurate answers.\n\n")

	// Add tool descriptions
	if len(tools) > 0 {
		sb.WriteString("Available tools:\n")
		for _, tool := range tools {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		}
		sb.WriteString("\nUse tools as needed to find the best answer.\n")
	}

	return sb.String()
}

// executeSingleTurn performs single-turn execution without tool loop.
func (g *ParetoResponseGenerator) executeSingleTurn(
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

// executeWithToolLoop performs multi-turn execution with actual tool execution.
func (g *ParetoResponseGenerator) executeWithToolLoop(
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

		// Execute tool calls
		log.Printf("[ToolLoop] Executing %d tool calls...", len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
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
			if execCtx.toolRunner == nil {
				toolRecord.Error = "tool runner not available"
			} else {
				result, execErr := execCtx.toolRunner.RunTool(ctx, tc.Name, tc.Arguments)
				if execErr != nil {
					log.Printf("[ToolLoop] Tool %s FAILED: %v", tc.Name, execErr)
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
					log.Printf("[ToolLoop] Tool %s SUCCESS", tc.Name)
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

// Ensure ParetoResponseGenerator implements the ports interface
var _ ports.ParetoResponseGenerator = (*ParetoResponseGenerator)(nil)
