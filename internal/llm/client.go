package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/adapters/retry"
)

// ChatMessage represents a message in the OpenAI chat format
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall represents a function call from the LLM
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function details
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Tool represents a function definition for the LLM
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction represents the function metadata
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Client is an OpenAI-compatible LLM client
type Client struct {
	baseURL     string
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
	retryConfig retry.BackoffConfig
}

// NewClient creates a new LLM client
func NewClient(baseURL, apiKey, model string, maxTokens int, temperature float64) *Client {
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1")

	return &Client{
		baseURL:     baseURL,
		apiKey:      apiKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Default request timeout
		},
		retryConfig: retry.HTTPConfig(),
	}
}

// ChatCompletionRequest represents the request to the chat completions API
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
	Tools       []Tool        `json:"tools,omitempty"`
	ToolChoice  string        `json:"tool_choice,omitempty"` // "auto", "none", or specific tool
}

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Content      string
	Reasoning    string
	ToolCall     *ToolCall
	FinishReason string
	Error        error
	Done         bool
}

// ChatCompletionResponse represents the response from the chat completions API
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat sends a non-streaming chat completion request
func (c *Client) Chat(ctx context.Context, messages []ChatMessage) (*ChatCompletionResponse, error) {
	return c.chat(ctx, messages, nil)
}

// ChatWithTools sends a non-streaming chat completion request with tools
func (c *Client) ChatWithTools(ctx context.Context, messages []ChatMessage, tools []Tool) (*ChatCompletionResponse, error) {
	return c.chat(ctx, messages, tools)
}

func (c *Client) chat(ctx context.Context, messages []ChatMessage, tools []Tool) (*ChatCompletionResponse, error) {
	if len(messages) == 0 || messages[0].Role != "system" {
		systemMsg := ChatMessage{
			Role:    "system",
			Content: "You are Alicia, a helpful AI assistant. Respond concisely and helpfully.",
		}
		messages = append([]ChatMessage{systemMsg}, messages...)
	}

	req := ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
		Stream:      false,
	}

	if len(tools) > 0 {
		req.Tools = tools
		req.ToolChoice = "auto"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var respBody []byte
	var statusCode int

	err = retry.WithBackoffHTTP(ctx, c.retryConfig, func() (int, error) {
		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return 0, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return statusCode, fmt.Errorf("failed to read body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return statusCode, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
		}

		return statusCode, nil
	})

	if err != nil {
		return nil, err
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// ChatStream sends a streaming chat completion request
func (c *Client) ChatStream(ctx context.Context, messages []ChatMessage) (<-chan StreamChunk, error) {
	return c.chatStream(ctx, messages, nil)
}

// ChatStreamWithTools sends a streaming chat completion request with tools
func (c *Client) ChatStreamWithTools(ctx context.Context, messages []ChatMessage, tools []Tool) (<-chan StreamChunk, error) {
	return c.chatStream(ctx, messages, tools)
}

func (c *Client) chatStream(ctx context.Context, messages []ChatMessage, tools []Tool) (<-chan StreamChunk, error) {
	if len(messages) == 0 || messages[0].Role != "system" {
		systemMsg := ChatMessage{
			Role:    "system",
			Content: "You are Alicia, a helpful AI assistant. Respond concisely and helpfully.",
		}
		messages = append([]ChatMessage{systemMsg}, messages...)
	}

	req := ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
		Stream:      true,
	}

	if len(tools) > 0 {
		req.Tools = tools
		req.ToolChoice = "auto"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *http.Response
	var statusCode int

	// For streaming, we retry the initial connection, but not the stream itself
	err = retry.WithBackoffHTTP(ctx, c.retryConfig, func() (int, error) {
		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err = c.httpClient.Do(httpReq)
		if err != nil {
			return 0, fmt.Errorf("failed to send request: %w", err)
		}

		statusCode = resp.StatusCode

		if resp.StatusCode != http.StatusOK {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return statusCode, fmt.Errorf("API error: %s (failed to read body: %w)", resp.Status, readErr)
			}
			return statusCode, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
		}

		return statusCode, nil
	})

	if err != nil {
		return nil, err
	}

	chunks := make(chan StreamChunk, 10)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		// Check context before starting
		select {
		case <-ctx.Done():
			chunks <- StreamChunk{Error: ctx.Err()}
			return
		default:
		}

		reader := bufio.NewReader(resp.Body)
		var currentToolCall *ToolCall

		for {
			select {
			case <-ctx.Done():
				chunks <- StreamChunk{Error: ctx.Err()}
				return
			default:
			}

			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					chunks <- StreamChunk{Error: err}
				}
				chunks <- StreamChunk{Done: true}
				return
			}

			lineStr := strings.TrimSpace(string(line))
			if lineStr == "" {
				continue
			}

			if !strings.HasPrefix(lineStr, "data: ") {
				continue
			}

			data := strings.TrimPrefix(lineStr, "data: ")
			if data == "[DONE]" {
				chunks <- StreamChunk{Done: true}
				return
			}

			var response struct {
				Choices []struct {
					Delta struct {
						Content          string `json:"content"`
						ReasoningContent string `json:"reasoning_content"`
						ToolCalls        []struct {
							Index    int    `json:"index"`
							ID       string `json:"id"`
							Type     string `json:"type"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &response); err != nil {
				continue
			}

			if len(response.Choices) == 0 {
				continue
			}

			choice := response.Choices[0]

			// Handle tool calls (accumulate across chunks)
			if len(choice.Delta.ToolCalls) > 0 {
				tc := choice.Delta.ToolCalls[0]
				if tc.ID != "" {
					// New tool call
					if currentToolCall != nil {
						// Send the previous complete tool call
						chunks <- StreamChunk{ToolCall: currentToolCall}
					}
					currentToolCall = &ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				} else if currentToolCall != nil {
					// Accumulate arguments
					currentToolCall.Function.Arguments += tc.Function.Arguments
				}
			}

			chunk := StreamChunk{
				Content:      choice.Delta.Content,
				Reasoning:    choice.Delta.ReasoningContent,
				FinishReason: choice.FinishReason,
			}

			// If we have a finish reason, send any pending tool call
			if choice.FinishReason != "" {
				if currentToolCall != nil {
					chunks <- StreamChunk{ToolCall: currentToolCall}
					currentToolCall = nil
				}
				chunk.Done = true
			}

			if chunk.Content != "" || chunk.Reasoning != "" || chunk.FinishReason != "" {
				chunks <- chunk
			}
		}
	}()

	return chunks, nil
}
