// Package llm provides a shared OpenAI-compatible client factory.
package llm

import (
	"context"
	"net/http"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.GetTracerProvider().Tracer("shared/llm")

// Config holds the configuration for the LLM client.
type Config struct {
	BaseURL        string
	APIKey         string
	Model          string
	EmbeddingModel string
	MaxTokens      int
	HTTPClient     *http.Client
	Transport      http.RoundTripper
	Timeout        time.Duration
}

// Option configures a Config.
type Option func(*Config)

// WithModel sets the default model for chat completions.
func WithModel(model string) Option {
	return func(c *Config) {
		c.Model = model
	}
}

// WithEmbeddingModel sets the default model for embeddings.
func WithEmbeddingModel(model string) Option {
	return func(c *Config) {
		c.EmbeddingModel = model
	}
}

// WithMaxTokens sets the default max tokens for completions.
func WithMaxTokens(maxTokens int) Option {
	return func(c *Config) {
		c.MaxTokens = maxTokens
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = client
	}
}

// WithTransport sets a custom HTTP transport (e.g., for OTEL/Langfuse tracing).
// This is ignored if WithHTTPClient is also used.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *Config) {
		c.Transport = rt
	}
}

// WithTimeout sets the HTTP client timeout.
// This is ignored if WithHTTPClient is also used.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = d
	}
}

// Client wraps the OpenAI client with configuration metadata.
type Client struct {
	*openai.Client
	BaseURL        string
	APIKey         string
	Model          string
	EmbeddingModel string
	MaxTokens      int
}

// NewClient creates an OpenAI-compatible client with the given configuration.
// BaseURL should be the full API base URL (e.g., "https://api.openai.com/v1").
func NewClient(baseURL, apiKey string, opts ...Option) *Client {
	cfg := &Config{
		BaseURL:   strings.TrimSuffix(baseURL, "/"),
		APIKey:    apiKey,
		Model:     "gpt-4o-mini",
		MaxTokens: 4096,
		Timeout:   60 * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	openaiCfg := openai.DefaultConfig(cfg.APIKey)
	openaiCfg.BaseURL = cfg.BaseURL

	// Set up HTTP client
	if cfg.HTTPClient != nil {
		openaiCfg.HTTPClient = cfg.HTTPClient
	} else {
		transport := cfg.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		openaiCfg.HTTPClient = &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		}
	}

	return &Client{
		Client:         openai.NewClientWithConfig(openaiCfg),
		BaseURL:        cfg.BaseURL,
		APIKey:         cfg.APIKey,
		Model:          cfg.Model,
		EmbeddingModel: cfg.EmbeddingModel,
		MaxTokens:      cfg.MaxTokens,
	}
}

// CreateChatCompletion wraps the OpenAI client's CreateChatCompletion with an OTel span.
func (c *Client) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	ctx, span := tracer.Start(ctx, "llm.chat", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	span.SetAttributes(
		attribute.String("llm.model", req.Model),
		attribute.Int("llm.request.max_tokens", req.MaxTokens),
		attribute.Int("llm.request.tools", len(req.Tools)),
		attribute.Int("llm.request.messages", len(req.Messages)),
	)
	if req.Temperature > 0 {
		span.SetAttributes(attribute.Float64("llm.request.temperature", float64(req.Temperature)))
	}

	resp, err := c.Client.CreateChatCompletion(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return resp, err
	}

	span.SetAttributes(
		attribute.Int("llm.usage.input_tokens", resp.Usage.PromptTokens),
		attribute.Int("llm.usage.output_tokens", resp.Usage.CompletionTokens),
		attribute.Int("llm.usage.total_tokens", resp.Usage.TotalTokens),
	)
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		span.SetAttributes(
			attribute.String("llm.response.finish_reason", string(choice.FinishReason)),
			attribute.Int("llm.response.tool_calls", len(choice.Message.ToolCalls)),
			attribute.Int("llm.response.content_length", len(choice.Message.Content)),
		)
	} else {
		span.SetAttributes(attribute.Int("llm.response.choices", 0))
	}

	return resp, nil
}

// CreateEmbeddings wraps the OpenAI client's CreateEmbeddings with an OTel span.
func (c *Client) CreateEmbeddings(ctx context.Context, req openai.EmbeddingRequest) (openai.EmbeddingResponse, error) {
	ctx, span := tracer.Start(ctx, "llm.embeddings", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	span.SetAttributes(attribute.String("llm.model", string(req.Model)))
	if inputs, ok := req.Input.([]string); ok {
		span.SetAttributes(attribute.Int("llm.request.inputs", len(inputs)))
	}

	resp, err := c.Client.CreateEmbeddings(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return resp, err
	}

	span.SetAttributes(
		attribute.Int("llm.usage.input_tokens", resp.Usage.PromptTokens),
		attribute.Int("llm.usage.total_tokens", resp.Usage.TotalTokens),
		attribute.Int("llm.response.embeddings", len(resp.Data)),
	)

	return resp, nil
}

