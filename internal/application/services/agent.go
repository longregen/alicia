package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// AgentServiceConfig holds configuration for the agent service
type AgentServiceConfig struct {
	// RAG configuration - no threshold, get more candidates for filtering
	RAGCandidateLimit int // Default: 15 - get more candidates, filter later
	MaxToolIterations int // Default: 5
}

// DefaultAgentServiceConfig returns the default configuration
func DefaultAgentServiceConfig() AgentServiceConfig {
	return AgentServiceConfig{
		RAGCandidateLimit: 15,
		MaxToolIterations: 5,
	}
}

// AgentService orchestrates the GEPA-optimized agent flow
type AgentService struct {
	// Core services
	llmService      ports.LLMService
	memoryService   ports.MemoryService
	toolService     ports.ToolService
	toolRepo        ports.ToolRepository
	messageRepo     ports.MessageRepository
	sentenceRepo    ports.SentenceRepository
	toolUseRepo     ports.ToolUseRepository
	reasoningRepo   ports.ReasoningStepRepository
	memoryUsageRepo ports.MemoryUsageRepository
	idGenerator     ports.IDGenerator
	txManager       ports.TransactionManager

	// GEPA-optimized services
	memoryFilter ports.MemoryFilterService
	toolSelector ports.ToolSelectionService

	// Configuration
	config AgentServiceConfig
}

// NewAgentService creates a new agent service with GEPA-optimized flow
func NewAgentService(
	llmService ports.LLMService,
	memoryService ports.MemoryService,
	toolService ports.ToolService,
	toolRepo ports.ToolRepository,
	messageRepo ports.MessageRepository,
	sentenceRepo ports.SentenceRepository,
	toolUseRepo ports.ToolUseRepository,
	reasoningRepo ports.ReasoningStepRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
	config AgentServiceConfig,
) *AgentService {
	return &AgentService{
		llmService:      llmService,
		memoryService:   memoryService,
		toolService:     toolService,
		toolRepo:        toolRepo,
		messageRepo:     messageRepo,
		sentenceRepo:    sentenceRepo,
		toolUseRepo:     toolUseRepo,
		reasoningRepo:   reasoningRepo,
		memoryUsageRepo: memoryUsageRepo,
		idGenerator:     idGenerator,
		txManager:       txManager,
		memoryFilter:    NewMemoryFilterService(llmService),
		toolSelector:    NewToolSelectionService(llmService),
		config:          config,
	}
}

// GenerateResponse generates a response using the GEPA-optimized flow
func (s *AgentService) GenerateResponse(ctx context.Context, input *ports.AgentInput) (*ports.AgentOutput, error) {
	// Step 1: Get the user message
	userMessage, err := s.messageRepo.GetByID(ctx, input.UserMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user message: %w", err)
	}

	// Step 2: Get conversation history for context
	messages, err := s.messageRepo.GetLatestByConversation(ctx, input.ConversationID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}
	conversationContext := s.buildConversationContext(messages)

	// Step 3: RAG - Retrieve memory candidates (no threshold filtering)
	memoryCandidates, err := s.retrieveMemoryCandidates(ctx, userMessage.Contents)
	if err != nil {
		log.Printf("warning: memory retrieval failed: %v\n", err)
		// Continue without memories
	}

	// Step 4: GEPA Memory Filter - Filter candidates to relevant ones
	var selectedMemories []*models.Memory
	if len(memoryCandidates) > 0 {
		filterResult, err := s.memoryFilter.FilterMemories(
			ctx,
			userMessage.Contents,
			conversationContext,
			memoryCandidates,
		)
		if err != nil {
			log.Printf("warning: memory filtering failed: %v\n", err)
			// Fallback to top 3 by similarity
			selectedMemories = s.fallbackMemorySelection(memoryCandidates, 3)
		} else {
			selectedMemories = filterResult.SelectedMemories
			log.Printf("GEPA filtered memories: selected %d, excluded %d. Reason: %s\n",
				len(filterResult.SelectedMemories),
				len(filterResult.ExcludedMemories),
				filterResult.Reasoning)
		}

		// Track memory usage
		s.trackMemoryUsage(ctx, input.ConversationID, input.UserMessageID, memoryCandidates, selectedMemories)
	}

	// Step 5: Get available tools
	var tools []*models.Tool
	var selectedToolNames []string
	if input.EnableTools {
		tools, err = s.toolRepo.ListEnabled(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get enabled tools: %w", err)
		}

		// Step 6: GEPA Tool Selection - Determine if/which tools to use
		if len(tools) > 0 {
			toolResult, err := s.toolSelector.SelectTool(
				ctx,
				userMessage.Contents,
				conversationContext,
				tools,
			)
			if err != nil {
				log.Printf("warning: tool selection failed: %v\n", err)
			} else if toolResult.SelectedTool != "none" {
				selectedToolNames = []string{toolResult.SelectedTool}
				log.Printf("GEPA selected tool: %s (confidence: %.2f). Reason: %s\n",
					toolResult.SelectedTool,
					toolResult.Confidence,
					toolResult.Reasoning)
			}
		}
	}

	// Step 7: Build LLM context with filtered memories and tool descriptions
	llmMessages := s.buildLLMContext(messages, selectedMemories, tools, selectedToolNames)

	// Step 8: Create assistant message
	sequenceNumber, err := s.messageRepo.GetNextSequenceNumber(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next sequence number: %w", err)
	}

	messageID := input.MessageID
	if messageID == "" {
		messageID = s.idGenerator.GenerateMessageID()
	}
	message := models.NewAssistantMessage(messageID, input.ConversationID, sequenceNumber, "")
	if input.PreviousID != "" {
		message.SetPreviousMessage(input.PreviousID)
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Step 9: Generate response
	output := &ports.AgentOutput{
		Message:       message,
		Sentences:     []*models.Sentence{},
		ToolUses:      []*models.ToolUse{},
		SelectedTools: selectedToolNames,
		UsedMemories:  selectedMemories,
	}

	if input.EnableStreaming {
		return s.executeStreaming(ctx, input, message, llmMessages, tools, output)
	}

	return s.executeWithToolLoop(ctx, input, message, llmMessages, tools, output)
}

// retrieveMemoryCandidates retrieves memory candidates via RAG (no threshold)
func (s *AgentService) retrieveMemoryCandidates(ctx context.Context, query string) ([]*ports.MemoryCandidate, error) {
	if s.memoryService == nil {
		return nil, nil
	}

	// Use SearchWithScores without threshold (pass 0 to get all)
	results, err := s.memoryService.SearchWithScores(ctx, query, 0.0, s.config.RAGCandidateLimit)
	if err != nil {
		return nil, err
	}

	candidates := make([]*ports.MemoryCandidate, len(results))
	now := time.Now()

	for i, r := range results {
		daysSinceAccess := 0
		if r.Memory.UpdatedAt.Before(now) {
			daysSinceAccess = int(now.Sub(r.Memory.UpdatedAt).Hours() / 24)
		}

		candidates[i] = &ports.MemoryCandidate{
			Memory:          r.Memory,
			SimilarityScore: r.Similarity,
			Importance:      r.Memory.Importance,
			DaysSinceAccess: daysSinceAccess,
		}
	}

	return candidates, nil
}

// fallbackMemorySelection selects top N memories by similarity
func (s *AgentService) fallbackMemorySelection(candidates []*ports.MemoryCandidate, n int) []*models.Memory {
	if len(candidates) <= n {
		memories := make([]*models.Memory, len(candidates))
		for i, c := range candidates {
			memories[i] = c.Memory
		}
		return memories
	}

	memories := make([]*models.Memory, n)
	for i := 0; i < n; i++ {
		memories[i] = candidates[i].Memory
	}
	return memories
}

// trackMemoryUsage records which memories were retrieved and selected
func (s *AgentService) trackMemoryUsage(
	ctx context.Context,
	conversationID, messageID string,
	candidates []*ports.MemoryCandidate,
	selected []*models.Memory,
) {
	selectedSet := make(map[string]bool)
	for _, m := range selected {
		selectedSet[m.ID] = true
	}

	for _, c := range candidates {
		if selectedSet[c.Memory.ID] {
			// Track as used
			_, err := s.memoryService.TrackUsage(ctx, c.Memory.ID, conversationID, messageID, c.SimilarityScore)
			if err != nil {
				log.Printf("warning: failed to track memory usage for %s: %v\n", c.Memory.ID, err)
			}
		}
	}
}

// buildConversationContext builds a context string from recent messages
func (s *AgentService) buildConversationContext(messages []*models.Message) string {
	if len(messages) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, msg := range messages {
		if msg.Role == "system" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, truncateString(msg.Contents, 200)))
	}
	return sb.String()
}

// buildLLMContext builds the LLM message context
func (s *AgentService) buildLLMContext(
	messages []*models.Message,
	memories []*models.Memory,
	tools []*models.Tool,
	selectedTools []string,
) []ports.LLMMessage {
	llmMessages := []ports.LLMMessage{}

	// Build system prompt
	var systemPrompt strings.Builder
	systemPrompt.WriteString("You are Alicia, a helpful AI assistant with memory and tool capabilities.")

	if len(memories) > 0 {
		systemPrompt.WriteString("\n\nRelevant memories from previous conversations:\n")
		for i, memory := range memories {
			systemPrompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, memory.Content))
		}
	}

	// Add tool descriptions to system prompt
	if len(tools) > 0 {
		systemPrompt.WriteString("\n\nAvailable tools:\n")
		for _, tool := range tools {
			systemPrompt.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		}
		systemPrompt.WriteString("\nConsider using these tools when they would improve your response quality.")
	}

	// Note preferred tools if GEPA selected any
	if len(selectedTools) > 0 {
		systemPrompt.WriteString(fmt.Sprintf("\n\nPreferred tools for this request: %s", strings.Join(selectedTools, ", ")))
	}

	llmMessages = append(llmMessages, ports.LLMMessage{
		Role:    "system",
		Content: systemPrompt.String(),
	})

	// Add conversation history
	for _, msg := range messages {
		role := string(msg.Role)
		if role == "system" {
			continue
		}
		llmMessages = append(llmMessages, ports.LLMMessage{
			Role:    role,
			Content: msg.Contents,
		})
	}

	return llmMessages
}

// executeWithToolLoop handles non-streaming response with tool execution
func (s *AgentService) executeWithToolLoop(
	ctx context.Context,
	input *ports.AgentInput,
	message *models.Message,
	llmMessages []ports.LLMMessage,
	tools []*models.Tool,
	output *ports.AgentOutput,
) (*ports.AgentOutput, error) {
	currentMessages := llmMessages

	for iteration := 0; iteration < s.config.MaxToolIterations; iteration++ {
		var response *ports.LLMResponse
		var err error

		if len(tools) > 0 {
			response, err = s.llmService.ChatWithTools(ctx, currentMessages, tools)
		} else {
			response, err = s.llmService.Chat(ctx, currentMessages)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to generate response: %w", err)
		}

		// Add assistant response to history
		currentMessages = append(currentMessages, ports.LLMMessage{
			Role:    "assistant",
			Content: response.Content,
		})

		// If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
				message.Contents = response.Content
				if err := s.messageRepo.Update(txCtx, message); err != nil {
					return fmt.Errorf("failed to update message: %w", err)
				}

				if input.EnableReasoning && response.Reasoning != "" {
					stepID := s.idGenerator.GenerateReasoningStepID()
					step := &models.ReasoningStep{
						ID:             stepID,
						MessageID:      message.ID,
						SequenceNumber: 0,
						Content:        response.Reasoning,
					}
					if err := s.reasoningRepo.Create(txCtx, step); err != nil {
						return fmt.Errorf("failed to create reasoning step: %w", err)
					}
					output.ReasoningSteps = append(output.ReasoningSteps, step)
				}
				return nil
			})

			if err != nil {
				return nil, err
			}

			// Extract memories async
			go s.extractAndStoreMemories(ctx, message, input.ConversationID)

			return output, nil
		}

		// Execute tool calls
		err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
			for _, toolCall := range response.ToolCalls {
				toolUse, err := s.toolService.CreateToolUse(txCtx, message.ID, toolCall.Name, toolCall.Arguments)
				if err != nil {
					return fmt.Errorf("failed to create tool use for %s: %w", toolCall.Name, err)
				}

				toolUse, err = s.toolService.ExecuteToolUse(txCtx, toolUse.ID)
				if err != nil {
					return fmt.Errorf("failed to execute tool %s: %w", toolCall.Name, err)
				}

				output.ToolUses = append(output.ToolUses, toolUse)

				// Add tool result to conversation
				resultContent := fmt.Sprintf("%v", toolUse.Result)
				currentMessages = append(currentMessages, ports.LLMMessage{
					Role:    "tool",
					Content: resultContent,
				})
			}
			return nil
		})

		if err != nil {
			currentMessages = append(currentMessages, ports.LLMMessage{
				Role:    "tool",
				Content: fmt.Sprintf("Error: %s", err.Error()),
			})
		}
	}

	// Max iterations reached
	message.Contents = "Max tool execution iterations reached."
	if err := s.messageRepo.Update(ctx, message); err != nil {
		log.Printf("ERROR: failed to update message %s after max iterations: %v", message.ID, err)
	}

	return output, nil
}

// executeStreaming handles streaming response
func (s *AgentService) executeStreaming(
	ctx context.Context,
	input *ports.AgentInput,
	message *models.Message,
	llmMessages []ports.LLMMessage,
	tools []*models.Tool,
	output *ports.AgentOutput,
) (*ports.AgentOutput, error) {
	var streamChan <-chan ports.LLMStreamChunk
	var err error

	if len(tools) > 0 {
		streamChan, err = s.llmService.ChatStreamWithTools(ctx, llmMessages, tools)
	} else {
		streamChan, err = s.llmService.ChatStream(ctx, llmMessages)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to start streaming: %w", err)
	}

	message.MarkAsStreaming()
	if err := s.messageRepo.Update(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to mark message as streaming: %w", err)
	}

	outputChan := make(chan *ports.ResponseStreamChunk, 10)
	go s.processStream(ctx, message, streamChan, outputChan, input.EnableReasoning, input.ConversationID)

	output.StreamChannel = outputChan
	return output, nil
}

// processStream handles the streaming response processing
func (s *AgentService) processStream(
	ctx context.Context,
	message *models.Message,
	inputChan <-chan ports.LLMStreamChunk,
	outputChan chan<- *ports.ResponseStreamChunk,
	enableReasoning bool,
	conversationID string,
) {
	defer close(outputChan)

	var fullContent strings.Builder
	sentenceSequence := 0

	for chunk := range inputChan {
		if chunk.Error != nil {
			outputChan <- &ports.ResponseStreamChunk{Error: chunk.Error}
			return
		}

		if chunk.Content != "" {
			fullContent.WriteString(chunk.Content)
			// Stream content directly
			outputChan <- &ports.ResponseStreamChunk{
				Text:    chunk.Content,
				IsFinal: false,
			}
		}

		if chunk.ToolCall != nil {
			toolUseID := s.idGenerator.GenerateToolUseID()
			toolUse := models.NewToolUse(
				toolUseID,
				message.ID,
				chunk.ToolCall.Name,
				sentenceSequence,
				chunk.ToolCall.Arguments,
			)

			if err := s.toolUseRepo.Create(ctx, toolUse); err != nil {
				outputChan <- &ports.ResponseStreamChunk{Error: err}
				continue
			}

			outputChan <- &ports.ResponseStreamChunk{
				ToolCall:  chunk.ToolCall,
				ToolUseID: toolUseID,
			}

			if s.toolService != nil {
				_, err := s.toolService.ExecuteToolUse(ctx, toolUseID)
				if err != nil {
					log.Printf("Tool execution failed: %v", err)
				}
			}
		}

		if enableReasoning && chunk.Reasoning != "" {
			outputChan <- &ports.ResponseStreamChunk{
				Reasoning: chunk.Reasoning,
			}
		}

		if chunk.Done {
			message.Contents = fullContent.String()
			message.MarkAsCompleted()
			if err := s.messageRepo.Update(ctx, message); err != nil {
				log.Printf("ERROR: failed to update message %s after stream completion: %v", message.ID, err)
			}

			outputChan <- &ports.ResponseStreamChunk{
				IsFinal: true,
			}

			go s.extractAndStoreMemories(ctx, message, conversationID)
			return
		}
	}
}

func (s *AgentService) extractAndStoreMemories(ctx context.Context, message *models.Message, conversationID string) {
	if s.memoryService == nil || len(message.Contents) < 50 {
		return
	}

	memCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	extractionPrompt := []ports.LLMMessage{
		{
			Role: "system",
			Content: `Extract important, factual information from this response that should be remembered.
Output each piece as "MEMORY: <content>" on a separate line. Output "NONE" if nothing important.`,
		},
		{
			Role:    "user",
			Content: message.Contents,
		},
	}

	extraction, err := s.llmService.Chat(memCtx, extractionPrompt)
	if err != nil || extraction.Content == "NONE" {
		return
	}

	for _, line := range strings.Split(extraction.Content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "MEMORY:") {
			continue
		}

		content := strings.TrimSpace(strings.TrimPrefix(line, "MEMORY:"))
		if len(content) < 10 {
			continue
		}

		memory, err := s.memoryService.CreateFromConversation(memCtx, content, conversationID, message.ID)
		if err != nil {
			log.Printf("warning: failed to create memory: %v\n", err)
			continue
		}

		importance := float32(0.5)
		if len(content) > 100 {
			importance = 0.7
		}
		if _, err := s.memoryService.SetImportance(memCtx, memory.ID, importance); err != nil {
			log.Printf("warning: failed to set memory importance for %s: %v", memory.ID, err)
		}
	}
}

// GetMemoryFilterService returns the memory filter for configuration
func (s *AgentService) GetMemoryFilterService() ports.MemoryFilterService {
	return s.memoryFilter
}

// GetToolSelectionService returns the tool selector for configuration
func (s *AgentService) GetToolSelectionService() ports.ToolSelectionService {
	return s.toolSelector
}

// truncateString truncates a string to maxLen
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Ensure AgentService implements the interface
var _ ports.AgentService = (*AgentService)(nil)
