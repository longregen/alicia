package tools

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

const (
	// MaxToolIterations is the maximum number of tool use iterations
	// to prevent infinite loops
	MaxToolIterations = 5
)

// Coordinator orchestrates the tool execution loop during LLM interactions
type Coordinator struct {
	toolService ports.ToolService
	llmService  ports.LLMService
	idGenerator ports.IDGenerator
}

// NewCoordinator creates a new tool coordinator
func NewCoordinator(
	toolService ports.ToolService,
	llmService ports.LLMService,
	idGenerator ports.IDGenerator,
) *Coordinator {
	return &Coordinator{
		toolService: toolService,
		llmService:  llmService,
		idGenerator: idGenerator,
	}
}

// ExecutionResult contains the result of a tool execution loop
type ExecutionResult struct {
	FinalResponse string
	ToolUses      []*models.ToolUse
	Messages      []ports.LLMMessage
	Iterations    int
	StoppedReason string // "max_iterations", "no_tool_calls", "error"
}

// ExecuteWithTools runs the LLM with tool support and handles the tool use loop
func (c *Coordinator) ExecuteWithTools(
	ctx context.Context,
	messages []ports.LLMMessage,
	tools []*models.Tool,
	messageID string,
) (*ExecutionResult, error) {
	result := &ExecutionResult{
		ToolUses: make([]*models.ToolUse, 0),
		Messages: messages,
	}

	for i := 0; i < MaxToolIterations; i++ {
		result.Iterations = i + 1

		// Call LLM with current message history
		response, err := c.llmService.ChatWithTools(ctx, result.Messages, tools)
		if err != nil {
			result.StoppedReason = "error"
			return nil, fmt.Errorf("LLM request failed: %w", err)
		}

		// Add assistant response to message history
		result.Messages = append(result.Messages, ports.LLMMessage{
			Role:    "assistant",
			Content: response.Content,
		})

		// If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			result.FinalResponse = response.Content
			result.StoppedReason = "no_tool_calls"
			return result, nil
		}

		// Execute each tool call
		for _, toolCall := range response.ToolCalls {
			// Create and execute tool use
			toolUse, err := c.executeToolCall(ctx, messageID, toolCall)
			if err != nil {
				// On error, add error message to conversation and continue
				result.Messages = append(result.Messages, ports.LLMMessage{
					Role:    "tool",
					Content: fmt.Sprintf("Error executing tool %s: %s", toolCall.Name, err.Error()),
				})
				continue
			}

			result.ToolUses = append(result.ToolUses, toolUse)

			// Add tool result to message history
			resultContent := fmt.Sprintf("%v", toolUse.Result)
			result.Messages = append(result.Messages, ports.LLMMessage{
				Role:    "tool",
				Content: resultContent,
			})
		}
	}

	// If we hit max iterations, return the last response
	result.StoppedReason = "max_iterations"
	return result, nil
}

// ExecuteWithToolsStreaming runs the LLM with tool support in streaming mode
func (c *Coordinator) ExecuteWithToolsStreaming(
	ctx context.Context,
	messages []ports.LLMMessage,
	tools []*models.Tool,
	messageID string,
) (<-chan ToolExecutionChunk, error) {
	outputChan := make(chan ToolExecutionChunk, 10)

	go func() {
		defer close(outputChan)

		currentMessages := messages
		iterations := 0

		for iterations < MaxToolIterations {
			iterations++

			// Call LLM with current message history
			streamChan, err := c.llmService.ChatStreamWithTools(ctx, currentMessages, tools)
			if err != nil {
				outputChan <- ToolExecutionChunk{
					Error: fmt.Errorf("LLM stream request failed: %w", err),
				}
				return
			}

			// Collect the response
			var fullContent string
			var toolCalls []*ports.LLMToolCall

			for chunk := range streamChan {
				if chunk.Error != nil {
					outputChan <- ToolExecutionChunk{Error: chunk.Error}
					return
				}

				// Forward content chunks to output
				if chunk.Content != "" {
					fullContent += chunk.Content
					outputChan <- ToolExecutionChunk{
						Content:   chunk.Content,
						Reasoning: chunk.Reasoning,
					}
				}

				// Collect tool calls
				if chunk.ToolCall != nil {
					toolCalls = append(toolCalls, chunk.ToolCall)
				}

				if chunk.Done {
					break
				}
			}

			// Add assistant response to message history
			currentMessages = append(currentMessages, ports.LLMMessage{
				Role:    "assistant",
				Content: fullContent,
			})

			// If no tool calls, we're done
			if len(toolCalls) == 0 {
				outputChan <- ToolExecutionChunk{
					Done:          true,
					StoppedReason: "no_tool_calls",
				}
				return
			}

			// Execute each tool call
			for _, toolCall := range toolCalls {
				// Notify about tool use
				outputChan <- ToolExecutionChunk{
					ToolCall: toolCall,
				}

				// Create and execute tool use
				toolUse, err := c.executeToolCall(ctx, messageID, toolCall)
				if err != nil {
					// Send error and add to conversation
					outputChan <- ToolExecutionChunk{
						ToolResult: &ToolResult{
							ToolCallID: toolCall.ID,
							Success:    false,
							Error:      err.Error(),
						},
					}
					currentMessages = append(currentMessages, ports.LLMMessage{
						Role:    "tool",
						Content: fmt.Sprintf("Error executing tool %s: %s", toolCall.Name, err.Error()),
					})
					continue
				}

				// Send success result
				resultContent := fmt.Sprintf("%v", toolUse.Result)
				outputChan <- ToolExecutionChunk{
					ToolResult: &ToolResult{
						ToolCallID: toolCall.ID,
						ToolName:   toolCall.Name,
						Success:    true,
						Result:     toolUse.Result,
					},
				}

				// Add tool result to message history
				currentMessages = append(currentMessages, ports.LLMMessage{
					Role:    "tool",
					Content: resultContent,
				})
			}
		}

		// Max iterations reached
		outputChan <- ToolExecutionChunk{
			Done:          true,
			StoppedReason: "max_iterations",
		}
	}()

	return outputChan, nil
}

// executeToolCall executes a single tool call and returns the ToolUse
func (c *Coordinator) executeToolCall(
	ctx context.Context,
	messageID string,
	toolCall *ports.LLMToolCall,
) (*models.ToolUse, error) {
	// Create tool use record
	toolUse, err := c.toolService.CreateToolUse(ctx, messageID, toolCall.Name, toolCall.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool use: %w", err)
	}

	// Execute the tool
	toolUse, err = c.toolService.ExecuteToolUse(ctx, toolUse.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tool: %w", err)
	}

	return toolUse, nil
}

// ToolExecutionChunk represents a chunk in the streaming tool execution
type ToolExecutionChunk struct {
	Content       string
	Reasoning     string
	ToolCall      *ports.LLMToolCall
	ToolResult    *ToolResult
	Done          bool
	StoppedReason string
	Error         error
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string
	ToolName   string
	Success    bool
	Result     any
	Error      string
}
