package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/longregen/alicia/internal/adapters/circuitbreaker"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

const (
	// LLMTimeout is the maximum time to wait for LLM responses
	LLMTimeout = 2 * time.Minute
)

// Service implements ports.LLMService using the OpenAI-compatible client
type Service struct {
	client  *Client
	breaker *circuitbreaker.CircuitBreaker
}

// NewService creates a new LLM service
func NewService(client *Client) *Service {
	return &Service{
		client:  client,
		breaker: circuitbreaker.New(5, 30*time.Second), // 5 failures, 30s timeout
	}
}

// Chat sends a non-streaming chat request
func (s *Service) Chat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	var result *ports.LLMResponse
	err := s.breaker.Execute(func() error {
		var err error
		result, err = s.doChat(ctx, messages)
		return err
	})
	return result, err
}

func (s *Service) doChat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	// Add timeout to prevent hanging on slow/failed LLM requests
	ctx, cancel := context.WithTimeout(ctx, LLMTimeout)
	defer cancel()

	chatMessages := s.convertMessages(messages)

	response, err := s.client.Chat(ctx, chatMessages)
	if err != nil {
		return nil, fmt.Errorf("chat request failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ports.LLMResponse{
		Content:   response.Choices[0].Message.Content,
		ToolCalls: s.convertToolCalls(response.Choices[0].Message.ToolCalls),
	}, nil
}

// ChatWithTools sends a non-streaming chat request with tools
func (s *Service) ChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	var result *ports.LLMResponse
	err := s.breaker.Execute(func() error {
		var err error
		result, err = s.doChatWithTools(ctx, messages, tools)
		return err
	})
	return result, err
}

func (s *Service) doChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	// Add timeout to prevent hanging on slow/failed LLM requests
	ctx, cancel := context.WithTimeout(ctx, LLMTimeout)
	defer cancel()

	chatMessages := s.convertMessages(messages)
	llmTools := s.convertTools(tools)

	response, err := s.client.ChatWithTools(ctx, chatMessages, llmTools)
	if err != nil {
		return nil, fmt.Errorf("chat with tools request failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ports.LLMResponse{
		Content:   response.Choices[0].Message.Content,
		ToolCalls: s.convertToolCalls(response.Choices[0].Message.ToolCalls),
	}, nil
}

// ChatStream sends a streaming chat request
func (s *Service) ChatStream(parentCtx context.Context, messages []ports.LLMMessage) (<-chan ports.LLMStreamChunk, error) {
	// Add timeout to prevent hanging on slow/failed LLM requests
	ctx, cancel := context.WithTimeout(parentCtx, LLMTimeout)

	chatMessages := s.convertMessages(messages)

	clientChan, err := s.client.ChatStream(ctx, chatMessages)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("chat stream request failed: %w", err)
	}

	outputChan := make(chan ports.LLMStreamChunk, 10)
	go func() {
		defer cancel()
		s.convertStreamChunks(ctx, clientChan, outputChan)
	}()

	return outputChan, nil
}

// ChatStreamWithTools sends a streaming chat request with tools
func (s *Service) ChatStreamWithTools(parentCtx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (<-chan ports.LLMStreamChunk, error) {
	// Add timeout to prevent hanging on slow/failed LLM requests
	ctx, cancel := context.WithTimeout(parentCtx, LLMTimeout)

	chatMessages := s.convertMessages(messages)
	llmTools := s.convertTools(tools)

	clientChan, err := s.client.ChatStreamWithTools(ctx, chatMessages, llmTools)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("chat stream with tools request failed: %w", err)
	}

	outputChan := make(chan ports.LLMStreamChunk, 10)
	go func() {
		defer cancel()
		s.convertStreamChunks(ctx, clientChan, outputChan)
	}()

	return outputChan, nil
}

// convertMessages converts ports.LLMMessage to ChatMessage
func (s *Service) convertMessages(messages []ports.LLMMessage) []ChatMessage {
	chatMessages := make([]ChatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return chatMessages
}

// convertTools converts domain tools to LLM tool format
func (s *Service) convertTools(tools []*models.Tool) []Tool {
	llmTools := make([]Tool, len(tools))
	for i, tool := range tools {
		llmTools[i] = Tool{
			Type: "function",
			Function: ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Schema,
			},
		}
	}
	return llmTools
}

// convertToolCalls converts LLM tool calls to ports.LLMToolCall
func (s *Service) convertToolCalls(toolCalls []ToolCall) []*ports.LLMToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	portToolCalls := make([]*ports.LLMToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		// Parse the arguments JSON string into a map
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			// If parsing fails, use an empty map
			args = make(map[string]any)
		}

		portToolCalls[i] = &ports.LLMToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		}
	}
	return portToolCalls
}

// convertStreamChunks converts client stream chunks to ports stream chunks
func (s *Service) convertStreamChunks(ctx context.Context, clientChan <-chan StreamChunk, outputChan chan<- ports.LLMStreamChunk) {
	defer close(outputChan)

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, exit gracefully
			outputChan <- ports.LLMStreamChunk{Error: ctx.Err()}
			return
		case chunk, ok := <-clientChan:
			if !ok {
				return // Channel closed normally
			}

			portChunk := ports.LLMStreamChunk{
				Content:   chunk.Content,
				Reasoning: chunk.Reasoning,
				Done:      chunk.Done,
				Error:     chunk.Error,
			}

			if chunk.ToolCall != nil {
				// Parse the arguments JSON string into a map
				var args map[string]any
				if err := json.Unmarshal([]byte(chunk.ToolCall.Function.Arguments), &args); err != nil {
					// If parsing fails, send an error chunk
					outputChan <- ports.LLMStreamChunk{
						Error: fmt.Errorf("failed to parse tool call arguments: %w", err),
					}
					continue
				}

				portChunk.ToolCall = &ports.LLMToolCall{
					ID:        chunk.ToolCall.ID,
					Name:      chunk.ToolCall.Function.Name,
					Arguments: args,
				}
			}

			outputChan <- portChunk
		}
	}
}
