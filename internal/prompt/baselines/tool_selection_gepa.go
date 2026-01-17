package baselines

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/longregen/alicia/internal/prompt"
)

// ToolSelectionSignature is the GEPA-optimizable signature for tool selection
var ToolSelectionSignature = prompt.MustParseSignature(
	"user_message, context, available_tools -> selected_tool, arguments, reasoning",
)

// ToolSelectionSeedPrompt is the baseline seed prompt for GEPA optimization
var ToolSelectionSeedPrompt = `You are a tool selection specialist. Given a user's message and conversation context, determine the most appropriate tool to use.

SELECTION CRITERIA:
1. Match user intent to tool capabilities precisely
2. Consider conversation context for disambiguation
3. Prefer specificity: choose the most targeted tool for the task
4. When a tool could improve response quality, prefer using it over responding without tools
5. Extract all required arguments from the user message

RESPONSE FORMAT:
- selected_tool: The exact tool name, or "none" if no tool is needed
- arguments: JSON object with tool parameters extracted from the message
- reasoning: Brief explanation of why this tool was selected

IMPORTANT: Select a tool when it would provide value - such as current information, user-specific data, calculations, or web searches. Only skip tools for purely conversational messages.`

// ToolInfo represents a tool available for selection
type ToolInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters"` // param_name -> description
	Examples    []string          `json:"examples"`   // Example user messages that should trigger this tool
}

// ToolSelectionExample represents a training/validation example
type ToolSelectionExample struct {
	UserMessage    string         `json:"user_message"`
	Context        string         `json:"context"`
	AvailableTools []ToolInfo     `json:"available_tools"`
	ExpectedTool   string         `json:"expected_tool"`
	ExpectedArgs   map[string]any `json:"expected_args"`
	Category       string         `json:"category"` // For stratified sampling
}

// ToPromptExample converts to prompt.Example for GEPA
func (e *ToolSelectionExample) ToPromptExample() prompt.Example {
	toolsJSON, err := json.Marshal(e.AvailableTools)
	if err != nil {
		// Fallback to empty JSON array if marshaling fails
		toolsJSON = []byte("[]")
	}
	return prompt.Example{
		Inputs: map[string]any{
			"user_message":    e.UserMessage,
			"context":         e.Context,
			"available_tools": string(toolsJSON),
		},
		Outputs: map[string]any{
			"selected_tool": e.ExpectedTool,
			"arguments":     e.ExpectedArgs,
		},
	}
}

// ToolSelectionMetric provides GEPA-optimized feedback for tool selection
type ToolSelectionMetric struct {
	// toolPatternRepo ports.ToolUsagePatternRepository // Optional: for historical pattern matching - TODO: define interface
}

// NewToolSelectionMetric creates a new tool selection metric
func NewToolSelectionMetric(toolPatternRepo any) *ToolSelectionMetric {
	return &ToolSelectionMetric{
		// toolPatternRepo: toolPatternRepo,
	}
}

// Score evaluates tool selection with rich GEPA feedback
func (m *ToolSelectionMetric) Score(
	ctx context.Context,
	gold, pred prompt.Example,
	trace *prompt.Trace,
) (prompt.ScoreWithFeedback, error) {
	// Check if this is a vote-derived example
	if voteValue, ok := gold.Outputs["_vote_value"]; ok {
		if voteInt, ok := voteValue.(int); ok {
			return m.scoreVoteExample(gold, pred, voteInt)
		}
	}

	// Original scoring logic for synthetic examples
	return m.scoreSyntheticExample(gold, pred, trace, gold.Inputs["user_message"])
}

// scoreVoteExample scores examples derived from user votes
func (m *ToolSelectionMetric) scoreVoteExample(gold, pred prompt.Example, voteValue int) (prompt.ScoreWithFeedback, error) {
	if voteValue == -1 { // VoteValueDown
		// Downvote: low score + diagnostic feedback for GEPA reflection
		quickFeedback, _ := gold.Outputs["_quick_feedback"].(string)
		toolName := toString(gold.Outputs["selected_tool"])
		args := toMap(gold.Outputs["arguments"])

		feedback := buildDiagnosticFeedbackForVote(quickFeedback, toolName, args)
		return prompt.ScoreWithFeedback{Score: 0.1, Feedback: feedback}, nil
	}

	// Upvote: high score if prediction matches
	predTool := toString(pred.Outputs["selected_tool"])
	goldTool := toString(gold.Outputs["selected_tool"])

	if strings.EqualFold(predTool, goldTool) {
		return prompt.ScoreWithFeedback{Score: 1.0, Feedback: "Correct tool selection."}, nil
	}

	return prompt.ScoreWithFeedback{
		Score:    0.5,
		Feedback: fmt.Sprintf("Selected '%s' but expected '%s'.", predTool, goldTool),
	}, nil
}

// scoreSyntheticExample handles scoring for synthetic training examples
func (m *ToolSelectionMetric) scoreSyntheticExample(gold, pred prompt.Example, trace *prompt.Trace, userMessageAny any) (prompt.ScoreWithFeedback, error) {
	expectedTool := toString(gold.Outputs["selected_tool"])
	predictedTool := toString(pred.Outputs["selected_tool"])
	expectedArgs := toMap(gold.Outputs["arguments"])
	predictedArgs := toMap(pred.Outputs["arguments"])
	userMessage := toString(userMessageAny)

	var feedbackParts []string
	var score float64

	// Component 1: Tool selection accuracy (50% weight)
	toolMatch := strings.EqualFold(expectedTool, predictedTool)
	if toolMatch {
		score += 0.5
		feedbackParts = append(feedbackParts, fmt.Sprintf("Correct tool: '%s'", expectedTool))
	} else {
		feedbackParts = append(feedbackParts,
			fmt.Sprintf("WRONG TOOL: Selected '%s' but expected '%s'. ", predictedTool, expectedTool))

		// Provide actionable feedback based on error type
		if predictedTool == "none" && expectedTool != "none" {
			feedbackParts = append(feedbackParts,
				fmt.Sprintf("The message '%s' requires the '%s' tool. Look for keywords or intent patterns that indicate tool usage.",
					truncate(userMessage, 50), expectedTool))
		} else if predictedTool != "none" && expectedTool == "none" {
			feedbackParts = append(feedbackParts,
				"This was a conversational message that doesn't require a tool. Be more conservative with tool selection.")
		} else {
			feedbackParts = append(feedbackParts,
				fmt.Sprintf("The '%s' tool handles this use case better than '%s'. Consider the specific capabilities of each tool.",
					expectedTool, predictedTool))
		}
	}

	// Component 2: Argument extraction accuracy (30% weight)
	if toolMatch && expectedTool != "none" {
		argScore, argFeedback := m.scoreArguments(expectedArgs, predictedArgs)
		score += argScore * 0.3
		feedbackParts = append(feedbackParts, argFeedback)
	} else if toolMatch && expectedTool == "none" {
		score += 0.3 // Full marks for correctly identifying no-tool case
	}
	// If tool doesn't match, no points for arguments

	// Component 3: Reasoning quality (20% weight)
	reasoning := toString(pred.Outputs["reasoning"])
	reasoningScore, reasoningFeedback := m.scoreReasoning(reasoning, expectedTool, userMessage)
	score += reasoningScore * 0.2
	if reasoningFeedback != "" {
		feedbackParts = append(feedbackParts, reasoningFeedback)
	}

	// Compile feedback
	feedback := strings.Join(feedbackParts, " ")

	// Add category-specific guidance for common failure patterns
	if !toolMatch {
		feedback += m.getCategoryGuidance(gold, pred)
	}

	return prompt.ScoreWithFeedback{
		Score:    score,
		Feedback: feedback,
	}, nil
}

// scoreArguments evaluates argument extraction quality
func (m *ToolSelectionMetric) scoreArguments(expected, predicted map[string]any) (float64, string) {
	if len(expected) == 0 {
		if len(predicted) == 0 {
			return 1.0, "No arguments expected or provided."
		}
		return 0.8, "Extra arguments provided but none expected."
	}

	matchCount := 0
	var missingArgs, wrongArgs []string

	for key, expectedVal := range expected {
		if predVal, ok := predicted[key]; ok {
			if fmt.Sprintf("%v", expectedVal) == fmt.Sprintf("%v", predVal) {
				matchCount++
			} else {
				wrongArgs = append(wrongArgs, fmt.Sprintf("%s (got '%v', expected '%v')", key, predVal, expectedVal))
			}
		} else {
			missingArgs = append(missingArgs, key)
		}
	}

	score := float64(matchCount) / float64(len(expected))

	var feedback string
	if len(missingArgs) > 0 {
		feedback = fmt.Sprintf("Missing arguments: %s. Extract these from the user message.", strings.Join(missingArgs, ", "))
	}
	if len(wrongArgs) > 0 {
		if feedback != "" {
			feedback += " "
		}
		feedback += fmt.Sprintf("Incorrect arguments: %s.", strings.Join(wrongArgs, "; "))
	}
	if feedback == "" {
		feedback = "All arguments correctly extracted."
	}

	return score, feedback
}

// scoreReasoning evaluates reasoning quality
func (m *ToolSelectionMetric) scoreReasoning(reasoning, expectedTool, userMessage string) (float64, string) {
	if reasoning == "" {
		return 0.0, "No reasoning provided. Explain why this tool was selected."
	}

	score := 0.5 // Base score for providing reasoning

	// Check if reasoning mentions the tool
	if strings.Contains(strings.ToLower(reasoning), strings.ToLower(expectedTool)) {
		score += 0.25
	}

	// Check if reasoning references user intent
	intentKeywords := []string{"user wants", "intent", "request", "asking", "needs"}
	for _, kw := range intentKeywords {
		if strings.Contains(strings.ToLower(reasoning), kw) {
			score += 0.25
			break
		}
	}

	if score < 1.0 {
		return score, "Reasoning should explicitly connect user intent to tool capabilities."
	}
	return score, ""
}

// getCategoryGuidance provides category-specific improvement guidance
func (m *ToolSelectionMetric) getCategoryGuidance(gold, pred prompt.Example) string {
	category, ok := gold.Inputs["category"].(string)
	if !ok {
		return ""
	}

	switch category {
	case "memory":
		return " [MEMORY TOOLS] Look for keywords: 'remember', 'recall', 'what did I say', 'earlier', 'last time'."
	case "search":
		return " [SEARCH TOOLS] Look for keywords: 'find', 'search', 'look up', 'what is', 'who is'."
	case "calendar":
		return " [CALENDAR TOOLS] Look for keywords: 'schedule', 'meeting', 'appointment', 'remind me', 'when is'."
	case "conversation":
		return " [NO TOOL] Greetings, opinions, and casual chat don't require tools."
	case "ambiguous":
		return " [AMBIGUOUS] When uncertain, examine context for disambiguation. Prefer 'none' if truly unclear."
	default:
		return ""
	}
}

// SyntheticToolSelectionDataset generates training/validation examples
func SyntheticToolSelectionDataset() (trainset, valset []prompt.Example) {
	// Define available tools for the dataset
	tools := []ToolInfo{
		{
			Name:        "memory_search",
			Description: "Search user's memories and past conversations for relevant information",
			Parameters:  map[string]string{"query": "Search query", "time_range": "Optional time filter"},
			Examples:    []string{"What did I tell you about my project?", "Remember when I mentioned..."},
		},
		{
			Name:        "memory_save",
			Description: "Save important information to user's memory for future reference",
			Parameters:  map[string]string{"content": "Information to save", "category": "Memory category"},
			Examples:    []string{"Remember that my birthday is March 15", "Save this: I'm allergic to peanuts"},
		},
		{
			Name:        "web_search",
			Description: "Search the web for current information, news, or facts",
			Parameters:  map[string]string{"query": "Search query", "num_results": "Number of results"},
			Examples:    []string{"What's the weather in Tokyo?", "Latest news about AI"},
		},
		{
			Name:        "calendar_create",
			Description: "Create a calendar event or reminder",
			Parameters:  map[string]string{"title": "Event title", "datetime": "When", "duration": "How long"},
			Examples:    []string{"Schedule a meeting tomorrow at 3pm", "Remind me to call mom on Friday"},
		},
		{
			Name:        "calculator",
			Description: "Perform mathematical calculations",
			Parameters:  map[string]string{"expression": "Math expression to evaluate"},
			Examples:    []string{"What's 15% of 250?", "Calculate 2^10"},
		},
	}

	// Training examples - diverse and representative
	trainingExamples := []ToolSelectionExample{
		// Memory search examples
		{
			UserMessage:    "What did I tell you about my vacation plans last week?",
			Context:        "User has been discussing travel plans",
			AvailableTools: tools,
			ExpectedTool:   "memory_search",
			ExpectedArgs:   map[string]any{"query": "vacation plans", "time_range": "last week"},
			Category:       "memory",
		},
		{
			UserMessage:    "Do you remember my daughter's name?",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "memory_search",
			ExpectedArgs:   map[string]any{"query": "daughter's name"},
			Category:       "memory",
		},
		{
			UserMessage:    "What's the recipe I shared with you yesterday?",
			Context:        "Previous conversation about cooking",
			AvailableTools: tools,
			ExpectedTool:   "memory_search",
			ExpectedArgs:   map[string]any{"query": "recipe", "time_range": "yesterday"},
			Category:       "memory",
		},

		// Memory save examples
		{
			UserMessage:    "Remember that my favorite color is blue",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "memory_save",
			ExpectedArgs:   map[string]any{"content": "favorite color is blue", "category": "preferences"},
			Category:       "memory",
		},
		{
			UserMessage:    "Please save this: I'm working on the Phoenix project",
			Context:        "User discussing work",
			AvailableTools: tools,
			ExpectedTool:   "memory_save",
			ExpectedArgs:   map[string]any{"content": "working on the Phoenix project", "category": "work"},
			Category:       "memory",
		},

		// Web search examples
		{
			UserMessage:    "What's the current stock price of Apple?",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "web_search",
			ExpectedArgs:   map[string]any{"query": "Apple stock price"},
			Category:       "search",
		},
		{
			UserMessage:    "Find me information about quantum computing",
			Context:        "User is researching technology",
			AvailableTools: tools,
			ExpectedTool:   "web_search",
			ExpectedArgs:   map[string]any{"query": "quantum computing"},
			Category:       "search",
		},
		{
			UserMessage:    "What are the latest developments in renewable energy?",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "web_search",
			ExpectedArgs:   map[string]any{"query": "latest developments renewable energy"},
			Category:       "search",
		},

		// Calendar examples
		{
			UserMessage:    "Schedule a dentist appointment for next Tuesday at 2pm",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "calendar_create",
			ExpectedArgs:   map[string]any{"title": "dentist appointment", "datetime": "next Tuesday 2pm"},
			Category:       "calendar",
		},
		{
			UserMessage:    "Set a reminder to buy groceries tomorrow morning",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "calendar_create",
			ExpectedArgs:   map[string]any{"title": "buy groceries", "datetime": "tomorrow morning"},
			Category:       "calendar",
		},

		// Calculator examples
		{
			UserMessage:    "What's 18% tip on a $67 bill?",
			Context:        "At a restaurant",
			AvailableTools: tools,
			ExpectedTool:   "calculator",
			ExpectedArgs:   map[string]any{"expression": "67 * 0.18"},
			Category:       "calculation",
		},
		{
			UserMessage:    "Calculate the square root of 144",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "calculator",
			ExpectedArgs:   map[string]any{"expression": "sqrt(144)"},
			Category:       "calculation",
		},

		// No tool needed - conversational
		{
			UserMessage:    "Hello! How are you today?",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "none",
			ExpectedArgs:   nil,
			Category:       "conversation",
		},
		{
			UserMessage:    "Thanks for your help!",
			Context:        "Just completed a task",
			AvailableTools: tools,
			ExpectedTool:   "none",
			ExpectedArgs:   nil,
			Category:       "conversation",
		},
		{
			UserMessage:    "I think AI is really fascinating",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "none",
			ExpectedArgs:   nil,
			Category:       "conversation",
		},
		{
			UserMessage:    "What do you think about that?",
			Context:        "Discussing philosophy",
			AvailableTools: tools,
			ExpectedTool:   "none",
			ExpectedArgs:   nil,
			Category:       "conversation",
		},

		// Ambiguous cases - require careful context analysis
		{
			UserMessage:    "Can you help me with something?",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "none",
			ExpectedArgs:   nil,
			Category:       "ambiguous",
		},
		{
			UserMessage:    "Tell me about my project",
			Context:        "User previously saved project details",
			AvailableTools: tools,
			ExpectedTool:   "memory_search",
			ExpectedArgs:   map[string]any{"query": "project"},
			Category:       "ambiguous",
		},
	}

	// Validation examples - similar distribution but different instances
	validationExamples := []ToolSelectionExample{
		{
			UserMessage:    "What was that restaurant I mentioned liking?",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "memory_search",
			ExpectedArgs:   map[string]any{"query": "restaurant liked"},
			Category:       "memory",
		},
		{
			UserMessage:    "Note that I prefer morning meetings",
			Context:        "Discussing scheduling preferences",
			AvailableTools: tools,
			ExpectedTool:   "memory_save",
			ExpectedArgs:   map[string]any{"content": "prefer morning meetings", "category": "preferences"},
			Category:       "memory",
		},
		{
			UserMessage:    "Who won the 2024 Olympics gold in swimming?",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "web_search",
			ExpectedArgs:   map[string]any{"query": "2024 Olympics gold swimming"},
			Category:       "search",
		},
		{
			UserMessage:    "Book a 1-hour meeting with Sarah on Friday afternoon",
			Context:        "Work context",
			AvailableTools: tools,
			ExpectedTool:   "calendar_create",
			ExpectedArgs:   map[string]any{"title": "meeting with Sarah", "datetime": "Friday afternoon", "duration": "1 hour"},
			Category:       "calendar",
		},
		{
			UserMessage:    "How much is 45 euros in dollars if the rate is 1.08?",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "calculator",
			ExpectedArgs:   map[string]any{"expression": "45 * 1.08"},
			Category:       "calculation",
		},
		{
			UserMessage:    "That's a great idea!",
			Context:        "Brainstorming session",
			AvailableTools: tools,
			ExpectedTool:   "none",
			ExpectedArgs:   nil,
			Category:       "conversation",
		},
		{
			UserMessage:    "I need to check on that thing",
			Context:        "",
			AvailableTools: tools,
			ExpectedTool:   "none",
			ExpectedArgs:   nil,
			Category:       "ambiguous",
		},
	}

	// Convert to prompt.Example
	trainset = make([]prompt.Example, len(trainingExamples))
	for i, ex := range trainingExamples {
		trainset[i] = ex.ToPromptExample()
	}

	valset = make([]prompt.Example, len(validationExamples))
	for i, ex := range validationExamples {
		valset[i] = ex.ToPromptExample()
	}

	return trainset, valset
}

// Helper functions

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// buildDiagnosticFeedbackForVote generates rich diagnostic feedback from quick_feedback for GEPA reflection
func buildDiagnosticFeedbackForVote(quickFeedback string, toolName string, args map[string]any) string {
	switch quickFeedback {
	case "wrong_tool":
		return fmt.Sprintf("The selected tool '%s' was incorrect for this query. Consider what the user actually needed.", toolName)
	case "unnecessary":
		return "A tool was used when none was needed. The query could have been answered directly without tool use."
	case "wrong_params":
		return fmt.Sprintf("The tool '%s' was correct but the arguments %v were wrong. Review how to extract parameters from user intent.", toolName, args)
	case "missing_context":
		return "The tool selection lacked important context from the conversation. Consider the full conversation history."
	default:
		return "The tool selection was marked as incorrect by the user."
	}
}
