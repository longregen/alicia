package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/longregen/alicia/pkg/langfuse"
	"github.com/longregen/alicia/shared/config"
	"github.com/longregen/alicia/shared/httpclient"
)

type LLMClient struct {
	baseURL      string
	apiKey       string
	model        string
	defaultMaxTk int
	httpClient   *http.Client
	langfuse     *langfuse.Client
}

func NewLLMClient() *LLMClient {
	baseURL := config.GetEnvWithFallback("LLM_URL", "OPENAI_BASE_URL", "https://api.openai.com/v1")
	apiKey := config.GetEnvWithFallback("LLM_API_KEY", "OPENAI_API_KEY", "")
	model := config.GetEnv("LLM_MODEL", "gpt-4o-mini")
	maxTokens := config.GetEnvInt("LLM_DEFAULT_MAX_TOKENS", 2048)

	host := config.GetEnv("LANGFUSE_HOST", "")
	publicKey := config.GetEnv("LANGFUSE_PUBLIC_KEY", "")
	secretKey := config.GetEnv("LANGFUSE_SECRET_KEY", "")
	var lf *langfuse.Client
	if host != "" && publicKey != "" && secretKey != "" {
		lf = langfuse.New(host, publicKey, secretKey)
		slog.Info("langfuse prompt management initialized", "host", host)
	} else {
		slog.Info("langfuse not configured, using fallback prompts")
	}

	return &LLMClient{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		apiKey:       apiKey,
		model:        model,
		defaultMaxTk: maxTokens,
		httpClient: httpclient.NewLong(),
		langfuse: lf,
	}
}

func (c *LLMClient) IsConfigured() bool {
	return c.apiKey != ""
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *LLMClient) CompleteWithMaxTokens(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
	return c.completeInternal(ctx, messages, maxTokens)
}

func (c *LLMClient) completeInternal(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
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

func (c *LLMClient) GenerateSQLHint(ctx context.Context, sql string, errMsg string, schemaContext string) string {
	if !c.IsConfigured() {
		return extractHintFallback(errMsg)
	}

	// Langfuse client has built-in fallbacks in pkg/langfuse/fallback.go
	sysPromptObj, err := c.langfuse.GetPrompt("alicia/garden/sql-debug-system", langfuse.WithLabel("production"))
	if err != nil {
		return extractHintFallback(errMsg)
	}
	systemPrompt := sysPromptObj.Compile(map[string]string{
		"db_docs": truncateForContext(schemaContext, 4000),
	})

	userPromptObj, err := c.langfuse.GetPrompt("alicia/garden/sql-debug-user", langfuse.WithLabel("production"))
	if err != nil {
		return extractHintFallback(errMsg)
	}
	userPrompt := userPromptObj.Compile(map[string]string{
		"sql":   sql,
		"error": errMsg,
	})

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	hintCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	hint, err := c.CompleteWithMaxTokens(hintCtx, messages, 256)
	if err != nil {
		return extractHintFallback(errMsg)
	}

	return strings.TrimSpace(hint)
}

func (c *LLMClient) AnswerSchemaQuestion(ctx context.Context, question string, schemaContext string, maxTokens int) (string, error) {
	if !c.IsConfigured() {
		return schemaContext, nil
	}

	// Langfuse client has built-in fallbacks in pkg/langfuse/fallback.go
	sysPromptObj, err := c.langfuse.GetPrompt("alicia/garden/schema-qa-system", langfuse.WithLabel("production"))
	if err != nil {
		return schemaContext, nil
	}
	systemPrompt := sysPromptObj.Compile(map[string]string{
		"db_docs": schemaContext,
	})

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: question},
	}

	return c.CompleteWithMaxTokens(ctx, messages, maxTokens)
}

// truncateForContext wraps langfuse.TruncateString with a newline-prefixed suffix
func truncateForContext(text string, maxChars int) string {
	return langfuse.TruncateString(text, maxChars, "\n...[truncated]")
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
