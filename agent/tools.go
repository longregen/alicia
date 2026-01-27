package main

// FinalAnswerToolName is the special tool for returning responses to the user.
const FinalAnswerToolName = "final_answer"

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
