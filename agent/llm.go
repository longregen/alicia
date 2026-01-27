package main

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/jsonutil"
	"github.com/longregen/alicia/shared/llm"
	openai "github.com/sashabaranov/go-openai"
)

type LLMClient struct {
	client         *llm.Client
	baseURL        string
	apiKey         string
	model          string
	embeddingModel string
	maxTokens      int
}

func NewLLMClient(baseURL, apiKey, model, embeddingModel string, maxTokens int) *LLMClient {
	client := llm.NewClient(baseURL, apiKey,
		llm.WithModel(model),
		llm.WithEmbeddingModel(embeddingModel),
		llm.WithMaxTokens(maxTokens),
		llm.WithTransport(otel.NewLangfuseTransport(http.DefaultTransport)),
	)
	return &LLMClient{
		client:         client,
		baseURL:        baseURL,
		apiKey:         apiKey,
		model:          model,
		embeddingModel: embeddingModel,
		maxTokens:      maxTokens,
	}
}

type ChatOptions struct {
	Temperature    *float32
	MaxTokens      int                                   // overrides client default if > 0
	ResponseFormat *openai.ChatCompletionResponseFormat  // structured output format
	ToolChoice     any                                   // "auto", "required", "none" or openai.ToolChoice struct
	GenerationName string                                // passed as metadata.generation_name for LiteLLM OTEL span naming
	PromptName     string
	PromptVersion  int
}

func float32Ptr(f float32) *float32 { return &f }

func (c *LLMClient) Chat(ctx context.Context, messages []LLMMessage, tools []Tool) (*LLMResponse, error) {
	return c.ChatWithOptions(ctx, messages, tools, ChatOptions{})
}

func (c *LLMClient) ChatWithOptions(ctx context.Context, messages []LLMMessage, tools []Tool, opts ChatOptions) (*LLMResponse, error) {
	msgs := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		msg := openai.ChatCompletionMessage{Role: m.Role, Content: m.Content}
		if m.ToolCallID != "" {
			msg.ToolCallID = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]openai.ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				msg.ToolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: jsonutil.MustJSON(tc.Arguments),
					},
				}
			}
		}
		msgs[i] = msg
	}

	maxTokens := c.maxTokens
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}

	req := openai.ChatCompletionRequest{
		Model:          c.model,
		Messages:       msgs,
		MaxTokens:      maxTokens,
		ResponseFormat: opts.ResponseFormat,
	}

	if opts.Temperature != nil {
		req.Temperature = *opts.Temperature
	}
	if opts.GenerationName != "" || opts.PromptName != "" {
		meta := map[string]string{}
		if opts.GenerationName != "" {
			meta["generation_name"] = opts.GenerationName
		}
		if opts.PromptName != "" {
			meta["observation.prompt.name"] = opts.PromptName
			if opts.PromptVersion > 0 {
				meta["observation.prompt.version"] = strconv.Itoa(opts.PromptVersion)
			}
		}
		req.Metadata = meta
	}

	if len(tools) > 0 {
		req.Tools = make([]openai.Tool, len(tools))
		for i, t := range tools {
			req.Tools[i] = openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Schema,
				},
			}
		}
		if opts.ToolChoice != nil {
			req.ToolChoice = opts.ToolChoice
		}
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		slog.WarnContext(ctx, "llm returned 0 choices")
		return &LLMResponse{}, nil
	}

	choice := resp.Choices[0]
	slog.InfoContext(ctx, "llm response", "content_length", len(choice.Message.Content), "tool_calls", len(choice.Message.ToolCalls), "finish_reason", choice.FinishReason)

	result := &LLMResponse{Content: choice.Message.Content, Reasoning: choice.Message.ReasoningContent}
	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, LLMToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: jsonutil.ParseJSON(tc.Function.Arguments),
		})
	}
	return result, nil
}

func (c *LLMClient) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(c.embeddingModel),
		Input: []string{text},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return resp.Data[0].Embedding, nil
}

