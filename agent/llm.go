package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/jsonutil"
	"github.com/longregen/alicia/shared/llm"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/trace"
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
	if opts.GenerationName != "" {
		req.Metadata = map[string]string{
			"generation_name": opts.GenerationName,
		}
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

	result := &LLMResponse{Content: choice.Message.Content, Reasoning: choice.Message.ReasoningContent, FinishReason: string(choice.FinishReason),
		PromptTokens: resp.Usage.PromptTokens, CompletionTokens: resp.Usage.CompletionTokens, TotalTokens: resp.Usage.TotalTokens}
	if resp.Usage.CompletionTokensDetails != nil {
		result.ReasoningTokens = resp.Usage.CompletionTokensDetails.ReasoningTokens
	}
	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, LLMToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: jsonutil.ParseJSON(tc.Function.Arguments),
		})
	}
	return result, nil
}

// MakeLLMCall is the unified entry point for LLM calls. It handles retry logic
// (unless NoRetry is set) and automatic Langfuse telemetry (unless NoTelemetry is set).
func MakeLLMCall(ctx context.Context, llm *LLMClient, msgs []LLMMessage, tools []Tool, opts LLMCallOptions) (*LLMResponse, error) {
	llmStart := time.Now()

	chatOpts := ChatOptions{
		Temperature:    opts.Temperature,
		MaxTokens:      opts.MaxTokens,
		ResponseFormat: opts.ResponseFormat,
		ToolChoice:     opts.ToolChoice,
		GenerationName: opts.GenerationName,
		PromptName:     opts.PromptName,
		PromptVersion:  opts.PromptVersion,
	}
	// Use prompt metadata if PromptName/PromptVersion not set directly
	if chatOpts.PromptName == "" && opts.Prompt.Name != "" {
		chatOpts.PromptName = opts.Prompt.Name
		chatOpts.PromptVersion = opts.Prompt.Version
	}

	var resp *LLMResponse
	var err error
	if opts.NoRetry {
		resp, err = llm.ChatWithOptions(ctx, msgs, tools, chatOpts)
	} else {
		resp, err = chatWithTokenRetry(ctx, llm, msgs, tools, chatOpts)
	}
	if err != nil {
		return nil, err
	}

	llmEnd := time.Now()

	if !opts.NoTelemetry {
		traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
		genID := fmt.Sprintf("%s-%d", opts.GenerationName, llmStart.UnixNano())

		input := any(msgs)
		if opts.InputOverride != nil {
			input = opts.InputOverride
		}
		output := any(resp.Content)
		if opts.OutputOverride != nil {
			output = opts.OutputOverride
		}

		go sendGenerationToLangfuse(LangfuseGeneration{
			TraceID: traceID, ID: genID, ParentObservationID: opts.ParentObservationID,
			ConvID: opts.ConvID, UserID: opts.UserID, Model: llm.model,
			TraceName: opts.TraceName, GenerationName: opts.GenerationName,
			Prompt: opts.Prompt, Input: input, Output: output,
			StartTime: llmStart, EndTime: llmEnd,
			PromptTokens: resp.PromptTokens, CompletionTokens: resp.CompletionTokens, TotalTokens: resp.TotalTokens,
			Temperature: opts.Temperature, MaxTokens: llm.maxTokens,
			Tools: toolNames(tools), ReasoningTokens: resp.ReasoningTokens,
			Reasoning: resp.Reasoning, FinishReason: resp.FinishReason,
		})
	}

	return resp, nil
}

func (c *LLMClient) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(c.embeddingModel),
		Input: []string{text},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
		slog.WarnContext(ctx, "embedding API returned empty vector", "input_length", len(text))
		return nil, nil
	}
	return resp.Data[0].Embedding, nil
}

