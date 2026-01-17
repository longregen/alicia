package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// LLMClient handles communication with OpenAI-compatible LLM APIs
type LLMClient struct {
	baseURL      string
	apiKey       string
	model        string
	defaultMaxTk int
	httpClient   *http.Client
}

// NewLLMClient creates a new LLM client from environment variables
func NewLLMClient() *LLMClient {
	baseURL := os.Getenv("LLM_URL")
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	maxTokens := 2048 // default
	if v := os.Getenv("LLM_DEFAULT_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}

	return &LLMClient{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		apiKey:       apiKey,
		model:        model,
		defaultMaxTk: maxTokens,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// IsConfigured returns true if the LLM client has API credentials
func (c *LLMClient) IsConfigured() bool {
	return c.apiKey != ""
}

// ChatMessage represents a message in the chat completion API
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents the request body for chat completions
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatCompletionResponse represents the response from chat completions
type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete sends a chat completion request with default max tokens
func (c *LLMClient) Complete(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.CompleteWithMaxTokens(ctx, messages, c.defaultMaxTk)
}

// CompleteWithMaxTokens sends a chat completion request with custom max tokens
func (c *LLMClient) CompleteWithMaxTokens(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("LLM not configured")
	}

	if maxTokens <= 0 {
		maxTokens = c.defaultMaxTk
	}

	reqBody := ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: 0.3,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// GenerateSQLHint generates a helpful hint for a SQL error (uses fewer tokens)
func (c *LLMClient) GenerateSQLHint(ctx context.Context, sql string, errMsg string, schemaContext string) string {
	if !c.IsConfigured() {
		return extractHintFallback(errMsg)
	}

	systemPrompt := `You are a SQL debugging assistant for a PostgreSQL database.
Your job is to provide a single, actionable hint to fix SQL errors.

Rules:
- Be concise: one or two sentences max
- Be specific: mention exact column/table names when possible
- Suggest using describe_table to check schema
- If the error is about a missing column, suggest the correct column name if you can infer it from context`

	if schemaContext != "" {
		systemPrompt += "\n\nDatabase documentation:\n" + truncateForContext(schemaContext, 4000)
	}

	userPrompt := fmt.Sprintf(`SQL Query:
%s

Error:
%s

Provide a brief, actionable hint to fix this error.`, sql, errMsg)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	hintCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Hints should be short - use 256 tokens
	hint, err := c.CompleteWithMaxTokens(hintCtx, messages, 256)
	if err != nil {
		return extractHintFallback(errMsg)
	}

	return strings.TrimSpace(hint)
}

// AnswerSchemaQuestion uses the LLM to answer questions about the database schema
func (c *LLMClient) AnswerSchemaQuestion(ctx context.Context, question string, schemaContext string, maxTokens int) (string, error) {
	if !c.IsConfigured() {
		return schemaContext, nil
	}

	systemPrompt := `You are a database documentation assistant. Answer questions about the database schema based on the provided documentation.

Rules:
- Be accurate and specific
- Reference exact table and column names
- Include example SQL queries when helpful
- If the documentation doesn't contain the answer, say so and suggest using describe_table`

	if schemaContext != "" {
		systemPrompt += "\n\nDatabase documentation:\n" + schemaContext
	}

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: question},
	}

	return c.CompleteWithMaxTokens(ctx, messages, maxTokens)
}

func truncateForContext(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars] + "\n...[truncated]"
}

func extractHintFallback(errMsg string) string {
	lower := strings.ToLower(errMsg)

	if strings.Contains(lower, "does not exist") {
		if strings.Contains(lower, "column") {
			return "Column not found. Use describe_table to see available columns."
		}
		if strings.Contains(lower, "relation") || strings.Contains(lower, "table") {
			return "Table not found. Use describe_table to see available tables."
		}
	}
	if strings.Contains(lower, "syntax error") {
		return "Syntax error. Check SQL syntax."
	}
	if strings.Contains(lower, "ambiguous") {
		return "Ambiguous column. Qualify with table alias (e.g., t.column_name)."
	}
	if strings.Contains(lower, "type") || strings.Contains(lower, "cast") {
		return "Type mismatch. Use describe_table to check column types."
	}

	return "Check your SQL syntax and table/column names."
}
