package main

import (
	"context"
	"log/slog"

)

const maxTokenRetries = 3

// chatWithTokenRetry calls ChatWithOptions and retries with exponentially
// increasing MaxTokens when the response is truncated (finish_reason="length").
// This handles reasoning models that consume tokens for thinking before emitting
// the actual tool call JSON.
func chatWithTokenRetry(ctx context.Context, llm *LLMClient, msgs []LLMMessage, tools []Tool, opts ChatOptions) (*LLMResponse, error) {
	const initialMaxTokens = 2048
	maxTokens := initialMaxTokens

	for attempt := 0; attempt <= maxTokenRetries; attempt++ {
		opts.MaxTokens = maxTokens
		resp, err := llm.ChatWithOptions(ctx, msgs, tools, opts)
		if err != nil {
			return nil, err
		}

		if resp.FinishReason != "length" {
			return resp, nil
		}

		// Truncated — double max_tokens and retry
		nextTokens := maxTokens * 2
		slog.WarnContext(ctx, "response truncated (finish_reason=length), retrying with more tokens",
			"attempt", attempt+1,
			"prev_max_tokens", maxTokens,
			"next_max_tokens", nextTokens,
		)
		maxTokens = nextTokens
	}

	// Final attempt already returned above; if we exhaust retries, do one last call
	opts.MaxTokens = maxTokens
	return llm.ChatWithOptions(ctx, msgs, tools, opts)
}

// FinalAnswerToolName is the special tool for returning responses to the user.
const FinalAnswerToolName = "answer_user"

// FinalAnswerTool returns the tool definition for final_answer.
// This tool forces all responses through the function calling API.
func FinalAnswerTool() Tool {
	return Tool{
		Name:        FinalAnswerToolName,
		Description: "Send your final response to the user. You MUST use this tool to respond - never write responses as plain text.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "Your complete response to the user",
				},
			},
			"required": []string{"content"},
		},
	}
}

// IsFinalAnswerCall checks if a tool call is the final_answer tool.
func IsFinalAnswerCall(tc LLMToolCall) bool {
	return tc.Name == FinalAnswerToolName
}

// ExtractFinalAnswer extracts the content from a final_answer tool call.
func ExtractFinalAnswer(tc LLMToolCall) string {
	if content, ok := tc.Arguments["content"].(string); ok {
		return content
	}
	return ""
}
