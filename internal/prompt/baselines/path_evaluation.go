package baselines

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
)

// PathEvaluator evaluates execution paths across multiple dimensions for GEPA Path Search.
type PathEvaluator struct {
	llm ports.LLMService
}

// NewPathEvaluator creates a new PathEvaluator.
func NewPathEvaluator(llm ports.LLMService) *PathEvaluator {
	return &PathEvaluator{llm: llm}
}

// Evaluate scores a path across 5 Pareto dimensions and returns feedback for mutation.
func (e *PathEvaluator) Evaluate(ctx context.Context, query string, trace *prompt.ExecutionTrace) (prompt.PathScores, string, error) {
	if trace == nil {
		return prompt.PathScores{}, "", fmt.Errorf("trace cannot be nil")
	}

	scores := prompt.PathScores{}

	// ──────────────────────────────────────────────────────────────────
	// STAGE 1: Fast heuristic screening
	// ──────────────────────────────────────────────────────────────────
	heuristicScore := e.heuristicScreen(trace)

	// ──────────────────────────────────────────────────────────────────
	// STAGE 2: LLM evaluation (only for promising candidates)
	// ──────────────────────────────────────────────────────────────────
	if heuristicScore >= 0.4 {
		// 2a. Answer quality (holistic)
		answerQuality, err := e.llmJudgeAnswerQuality(ctx, query, trace)
		if err != nil {
			// Fall back to heuristic on LLM error
			scores.AnswerQuality = heuristicScore
		} else {
			scores.AnswerQuality = answerQuality
		}

		// 2b. Hallucination check (CRITICAL - heavily penalize)
		hallucinated, err := e.llmCheckHallucinations(ctx, trace)
		if err == nil && hallucinated {
			scores.AnswerQuality *= 0.3 // Heavy penalty for hallucinations
		}

		// 2c. Specificity check (context-dependent)
		specificityMultiplier, err := e.llmJudgeSpecificity(ctx, query, trace)
		if err == nil {
			scores.AnswerQuality *= specificityMultiplier
		}
	} else {
		scores.AnswerQuality = heuristicScore
	}

	// ──────────────────────────────────────────────────────────────────
	// DIMENSION: Efficiency (heuristic)
	// ──────────────────────────────────────────────────────────────────
	toolCallCount := float64(len(trace.ToolCalls))
	scores.Efficiency = 1.0 - minFloat(1.0, toolCallCount/10.0)

	// ──────────────────────────────────────────────────────────────────
	// DIMENSION: Token cost (heuristic)
	// ──────────────────────────────────────────────────────────────────
	scores.TokenCost = 1.0 - minFloat(1.0, float64(trace.TotalTokens)/10000.0)

	// ──────────────────────────────────────────────────────────────────
	// DIMENSION: Robustness (errors + self-correction)
	// ──────────────────────────────────────────────────────────────────
	robustness, err := e.evaluateRobustness(ctx, trace)
	if err != nil {
		// Fall back to simple heuristic
		failedCalls := countFailedToolCalls(trace)
		if failedCalls == 0 {
			robustness = 1.0
		} else {
			robustness = maxFloat(0.0, 1.0-float64(failedCalls)*0.2)
		}
	}
	scores.Robustness = robustness

	// ──────────────────────────────────────────────────────────────────
	// DIMENSION: Latency (inverted: fast = high score)
	// ──────────────────────────────────────────────────────────────────
	scores.Latency = 1.0 - minFloat(1.0, float64(trace.DurationMs)/30000.0) // 30s max

	// Generate rich feedback for GEPA reflection
	feedback := e.generateFeedback(ctx, query, trace, scores)

	return scores, feedback, nil
}

// heuristicScreen provides fast initial screening.
func (e *PathEvaluator) heuristicScreen(trace *prompt.ExecutionTrace) float64 {
	score := 0.0

	// Has answer?
	if trace.FinalAnswer != "" && !isNonAnswer(trace.FinalAnswer) {
		score += 0.3
	}

	// Contains specific data (numbers, names)?
	if containsSpecificData(trace.FinalAnswer) {
		score += 0.2
	}

	// Successful tool calls?
	totalCalls := float64(len(trace.ToolCalls))
	if totalCalls > 0 {
		successRate := float64(countSuccessfulToolCalls(trace)) / totalCalls
		score += 0.3 * successRate
	} else {
		// No tool calls might be fine for simple queries
		score += 0.15
	}

	// Reasonable length?
	answerLen := len(trace.FinalAnswer)
	if answerLen > 20 && answerLen < 2000 {
		score += 0.2
	} else if answerLen > 0 && answerLen <= 20 {
		score += 0.1 // Short but present
	}

	return score
}

// llmJudgeAnswerQuality uses LLM for holistic answer quality assessment.
func (e *PathEvaluator) llmJudgeAnswerQuality(ctx context.Context, query string, trace *prompt.ExecutionTrace) (float64, error) {
	promptText := fmt.Sprintf(`Rate the quality of this answer on a scale of 0-10.

QUERY: %s

ANSWER: %s

Consider holistically:
- Relevance: Does it address what was asked?
- Accuracy: Is the information correct based on the available data?
- Completeness: Does it fully answer the question?
- Clarity: Is it well-organized and easy to understand?

Output format: SCORE: [0-10] REASON: [brief explanation]`, query, truncateForPrompt(trace.FinalAnswer, 1500))

	resp, err := e.llm.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: promptText}})
	if err != nil {
		return 0.0, fmt.Errorf("LLM call failed: %w", err)
	}

	score := parseScoreFromResponse(resp.Content)
	return score / 10.0, nil
}

// llmCheckHallucinations verifies answer claims against tool outputs (CRITICAL).
func (e *PathEvaluator) llmCheckHallucinations(ctx context.Context, trace *prompt.ExecutionTrace) (bool, error) {
	toolOutputs := formatToolOutputs(trace.ToolCalls)
	if toolOutputs == "" {
		// No tool outputs to verify against - can't determine hallucination
		return false, nil
	}

	promptText := fmt.Sprintf(`Check if this answer contains hallucinations (claims not supported by the tool outputs).

TOOL OUTPUTS:
%s

ANSWER:
%s

Does the answer make any specific factual claims (numbers, names, dates, etc.) that are NOT supported by the tool outputs above?
- Claims that are reasonable inferences from the data are OK
- Claims that contradict or go beyond the data are hallucinations

Output: HALLUCINATED: true/false REASON: [explanation]`, truncateForPrompt(toolOutputs, 2000), truncateForPrompt(trace.FinalAnswer, 1000))

	resp, err := e.llm.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: promptText}})
	if err != nil {
		return false, fmt.Errorf("LLM call failed: %w", err)
	}

	return strings.Contains(strings.ToLower(resp.Content), "hallucinated: true"), nil
}

// llmJudgeSpecificity checks if answer specificity is appropriate for the query.
func (e *PathEvaluator) llmJudgeSpecificity(ctx context.Context, query string, trace *prompt.ExecutionTrace) (float64, error) {
	promptText := fmt.Sprintf(`Is the specificity of this answer appropriate for the query?

QUERY: %s
ANSWER: %s

Evaluate:
- If query asks for specific data and answer is vague: return LOW score (0.5-0.7)
- If query is open-ended and answer is appropriately general: return HIGH score (0.9-1.0)
- If answer provides concrete data when expected: return HIGH score (0.9-1.0)
- If answer is overly specific when not needed: return MEDIUM score (0.8-0.9)

Output: SPECIFICITY_SCORE: [0.5-1.0]`, query, truncateForPrompt(trace.FinalAnswer, 1000))

	resp, err := e.llm.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: promptText}})
	if err != nil {
		return 1.0, fmt.Errorf("LLM call failed: %w", err)
	}

	score := parseSpecificityScore(resp.Content)
	if score < 0.5 {
		score = 0.5 // Floor at 0.5 since this is a multiplier
	}
	if score > 1.0 {
		score = 1.0
	}
	return score, nil
}

// evaluateRobustness scores error handling and self-correction.
func (e *PathEvaluator) evaluateRobustness(ctx context.Context, trace *prompt.ExecutionTrace) (float64, error) {
	score := 1.0

	// Count errors and recoveries
	errors := 0
	recoveries := 0
	for i, tc := range trace.ToolCalls {
		if !tc.Success {
			errors++
			// Check if next call shows recovery attempt (successful call after failure)
			if i+1 < len(trace.ToolCalls) && trace.ToolCalls[i+1].Success {
				recoveries++
			}
		}
	}

	// Penalize errors (context-aware via LLM for severity)
	if errors > 0 {
		severityPenalty, err := e.llmJudgeErrorSeverity(ctx, trace, errors)
		if err != nil {
			// Fallback: simple penalty
			severityPenalty = minFloat(0.5, float64(errors)*0.15)
		}
		score -= severityPenalty
	}

	// REWARD self-correction behavior
	if recoveries > 0 {
		score += 0.1 * float64(recoveries) // Bonus for recovery
	}

	return maxFloat(0.0, minFloat(1.0, score)), nil
}

// llmJudgeErrorSeverity determines how critical errors were.
func (e *PathEvaluator) llmJudgeErrorSeverity(ctx context.Context, trace *prompt.ExecutionTrace, errorCount int) (float64, error) {
	errorDetails := formatErrors(trace.ToolCalls)
	if errorDetails == "" {
		return 0.0, nil
	}

	hasAnswer := trace.FinalAnswer != "" && !isNonAnswer(trace.FinalAnswer)

	promptText := fmt.Sprintf(`Rate the severity of these errors (return a penalty from 0.0 to 0.5):

ERRORS (%d total):
%s

FINAL ANSWER ACHIEVED: %v

Severity guidelines:
- If errors were recoverable and didn't affect final answer: LOW penalty (0.0-0.1)
- If errors caused partial data loss but answer is still useful: MEDIUM penalty (0.1-0.3)
- If errors were critical and unrecoverable, preventing a good answer: HIGH penalty (0.3-0.5)

Output: SEVERITY_PENALTY: [0.0-0.5]`, errorCount, truncateForPrompt(errorDetails, 1000), hasAnswer)

	resp, err := e.llm.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: promptText}})
	if err != nil {
		return 0.0, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseSeverityPenalty(resp.Content), nil
}

// generateFeedback creates actionable feedback for strategy mutation.
func (e *PathEvaluator) generateFeedback(ctx context.Context, query string, trace *prompt.ExecutionTrace, scores prompt.PathScores) string {
	var feedbackParts []string

	// Answer quality issues
	if scores.AnswerQuality < 0.3 {
		feedbackParts = append(feedbackParts, "Answer quality is very low - the response may be incorrect, incomplete, or irrelevant.")
	} else if scores.AnswerQuality < 0.5 {
		feedbackParts = append(feedbackParts, "Answer quality is below average - consider improving accuracy or completeness.")
	} else if scores.AnswerQuality < 0.7 {
		feedbackParts = append(feedbackParts, "Answer quality is moderate - there's room for improvement.")
	}

	// Robustness issues
	failedTools := countFailedToolCalls(trace)
	if failedTools > 0 {
		feedbackParts = append(feedbackParts, fmt.Sprintf("%d tool call(s) failed - consider error handling or alternative approaches.", failedTools))
	}

	// Wasted effort (SOFT SIGNAL - not scored directly, but included in feedback)
	wastedCalls := countWastedToolCalls(trace)
	if wastedCalls > 0 {
		feedbackParts = append(feedbackParts,
			fmt.Sprintf("%d tool call(s) had results that weren't reflected in the answer - consider more focused exploration.", wastedCalls))
	}

	// Efficiency issues
	if len(trace.ToolCalls) > 5 && scores.AnswerQuality < 0.7 {
		feedbackParts = append(feedbackParts, "Many tool calls but moderate quality - the strategy may be inefficient.")
	} else if len(trace.ToolCalls) > 8 {
		feedbackParts = append(feedbackParts, "High number of tool calls - consider a more direct approach.")
	}

	// Token cost issues
	if scores.TokenCost < 0.3 {
		feedbackParts = append(feedbackParts, "Very high token usage - consider more concise prompts or fewer iterations.")
	} else if scores.TokenCost < 0.5 {
		feedbackParts = append(feedbackParts, "High token usage - there may be opportunities for optimization.")
	}

	// Latency issues
	if scores.Latency < 0.3 {
		feedbackParts = append(feedbackParts, "Very slow execution - consider a more direct approach or parallel operations.")
	} else if scores.Latency < 0.5 {
		feedbackParts = append(feedbackParts, "Slow execution - there may be opportunities to speed up the process.")
	}

	// Empty answer handling
	if trace.FinalAnswer == "" || isNonAnswer(trace.FinalAnswer) {
		feedbackParts = append(feedbackParts, "No meaningful answer was produced - the strategy needs significant revision.")
	}

	// Check for repeated failures
	if hasRepeatedFailures(trace) {
		feedbackParts = append(feedbackParts, "Multiple similar failures detected - consider a different approach entirely.")
	}

	if len(feedbackParts) == 0 {
		return "Path executed successfully with good results."
	}

	return strings.Join(feedbackParts, " ")
}

// Helper functions

// containsSpecificData checks if the answer contains specific data like numbers, names, dates.
func containsSpecificData(answer string) bool {
	if answer == "" {
		return false
	}

	// Check for numbers (including decimals and percentages)
	numberRegex := regexp.MustCompile(`\d+\.?\d*%?`)
	if numberRegex.MatchString(answer) {
		return true
	}

	// Check for capitalized proper nouns (potential names)
	words := strings.Fields(answer)
	capitalizedCount := 0
	for _, word := range words {
		if len(word) > 1 {
			runes := []rune(word)
			if unicode.IsUpper(runes[0]) && !isCommonWord(strings.ToLower(word)) {
				capitalizedCount++
			}
		}
	}
	if capitalizedCount >= 2 {
		return true
	}

	// Check for date patterns
	dateRegex := regexp.MustCompile(`\d{1,2}[/-]\d{1,2}[/-]\d{2,4}|\d{4}[/-]\d{1,2}[/-]\d{1,2}|(?i)(january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2}`)
	if dateRegex.MatchString(answer) {
		return true
	}

	return false
}

// countSuccessfulToolCalls counts the number of successful tool calls.
func countSuccessfulToolCalls(trace *prompt.ExecutionTrace) int {
	count := 0
	for _, tc := range trace.ToolCalls {
		if tc.Success {
			count++
		}
	}
	return count
}

// countFailedToolCalls counts the number of failed tool calls.
func countFailedToolCalls(trace *prompt.ExecutionTrace) int {
	count := 0
	for _, tc := range trace.ToolCalls {
		if !tc.Success {
			count++
		}
	}
	return count
}

// countWastedToolCalls estimates tool calls whose results weren't used in the answer.
func countWastedToolCalls(trace *prompt.ExecutionTrace) int {
	if trace.FinalAnswer == "" {
		return len(trace.ToolCalls) // All wasted if no answer
	}

	answerLower := strings.ToLower(trace.FinalAnswer)
	wastedCount := 0

	for _, tc := range trace.ToolCalls {
		if !tc.Success {
			continue // Failed calls don't count as wasted
		}

		// Check if any part of the result appears in the answer
		resultStr := fmt.Sprintf("%v", tc.Result)
		if resultStr == "" || resultStr == "<nil>" {
			continue
		}

		// Extract key terms from result
		keyTerms := extractKeyTerms(resultStr)
		foundInAnswer := false
		for _, term := range keyTerms {
			if strings.Contains(answerLower, strings.ToLower(term)) {
				foundInAnswer = true
				break
			}
		}

		if !foundInAnswer {
			wastedCount++
		}
	}

	return wastedCount
}

// formatToolOutputs formats tool call results for LLM consumption.
func formatToolOutputs(toolCalls []prompt.ToolCallRecord) string {
	var sb strings.Builder

	for i, tc := range toolCalls {
		if tc.Success && tc.Result != nil {
			resultStr := fmt.Sprintf("%v", tc.Result)
			if len(resultStr) > 500 {
				resultStr = resultStr[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("[Tool %d: %s]\n%s\n\n", i+1, tc.ToolName, resultStr))
		}
	}

	return sb.String()
}

// formatErrors formats error information from tool calls.
func formatErrors(toolCalls []prompt.ToolCallRecord) string {
	var sb strings.Builder

	for i, tc := range toolCalls {
		if !tc.Success {
			sb.WriteString(fmt.Sprintf("[Error %d: %s]\n", i+1, tc.ToolName))
			if tc.Error != "" {
				sb.WriteString(fmt.Sprintf("  Error: %s\n", tc.Error))
			}
			sb.WriteString(fmt.Sprintf("  Arguments: %v\n\n", tc.Arguments))
		}
	}

	return sb.String()
}

// parseScoreFromResponse extracts a numeric score from LLM response.
func parseScoreFromResponse(response string) float64 {
	// Look for "SCORE: X" pattern
	scoreRegex := regexp.MustCompile(`(?i)SCORE:\s*(\d+(?:\.\d+)?)`)
	matches := scoreRegex.FindStringSubmatch(response)
	if len(matches) > 1 {
		score, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return score
		}
	}

	// Fallback: look for any number at the start
	numRegex := regexp.MustCompile(`^(\d+(?:\.\d+)?)`)
	matches = numRegex.FindStringSubmatch(strings.TrimSpace(response))
	if len(matches) > 1 {
		score, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return score
		}
	}

	return 5.0 // Default middle score if parsing fails
}

// parseSpecificityScore extracts specificity score from LLM response.
func parseSpecificityScore(response string) float64 {
	scoreRegex := regexp.MustCompile(`(?i)SPECIFICITY_SCORE:\s*(\d+(?:\.\d+)?)`)
	matches := scoreRegex.FindStringSubmatch(response)
	if len(matches) > 1 {
		score, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return score
		}
	}
	return 1.0 // Default to no penalty if parsing fails
}

// parseSeverityPenalty extracts severity penalty from LLM response.
func parseSeverityPenalty(response string) float64 {
	penaltyRegex := regexp.MustCompile(`(?i)SEVERITY_PENALTY:\s*(\d+(?:\.\d+)?)`)
	matches := penaltyRegex.FindStringSubmatch(response)
	if len(matches) > 1 {
		penalty, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return minFloat(0.5, maxFloat(0.0, penalty))
		}
	}
	return 0.1 // Default modest penalty if parsing fails
}

// isNonAnswer checks if the answer is essentially a non-answer.
func isNonAnswer(answer string) bool {
	answer = strings.ToLower(strings.TrimSpace(answer))
	nonAnswerPhrases := []string{
		"unable to determine",
		"i don't know",
		"i cannot",
		"i can't",
		"no information available",
		"insufficient data",
		"cannot answer",
		"unable to answer",
		"i'm not sure",
		"i am not sure",
	}
	for _, phrase := range nonAnswerPhrases {
		if strings.Contains(answer, phrase) {
			return true
		}
	}
	return false
}

// isCommonWord checks if a word is a common English word (not a proper noun).
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true,
		"i": true, "you": true, "he": true, "she": true, "it": true,
		"we": true, "they": true, "what": true, "which": true, "who": true,
		"when": true, "where": true, "why": true, "how": true,
		"this": true, "that": true, "these": true, "those": true,
		"and": true, "but": true, "or": true, "nor": true, "for": true,
		"yet": true, "so": true, "if": true, "then": true, "else": true,
		"however": true, "therefore": true, "thus": true, "hence": true,
	}
	return commonWords[word]
}

// extractKeyTerms extracts key terms from a string for matching.
func extractKeyTerms(s string) []string {
	// Extract numbers, capitalized words, and quoted strings
	var terms []string

	// Numbers
	numRegex := regexp.MustCompile(`\d+\.?\d*`)
	terms = append(terms, numRegex.FindAllString(s, -1)...)

	// Quoted strings
	quotedRegex := regexp.MustCompile(`"([^"]+)"`)
	for _, match := range quotedRegex.FindAllStringSubmatch(s, -1) {
		if len(match) > 1 {
			terms = append(terms, match[1])
		}
	}

	// Capitalized words (potential proper nouns)
	words := strings.Fields(s)
	for _, word := range words {
		if len(word) > 2 {
			runes := []rune(word)
			if unicode.IsUpper(runes[0]) && !isCommonWord(strings.ToLower(word)) {
				// Clean punctuation
				word = strings.Trim(word, ".,;:!?\"'()[]{}")
				if len(word) > 2 {
					terms = append(terms, word)
				}
			}
		}
	}

	return terms
}

// hasRepeatedFailures checks if there are multiple similar failures.
func hasRepeatedFailures(trace *prompt.ExecutionTrace) bool {
	if len(trace.ToolCalls) < 2 {
		return false
	}

	failedTools := make(map[string]int)
	for _, tc := range trace.ToolCalls {
		if !tc.Success {
			failedTools[tc.ToolName]++
		}
	}

	for _, count := range failedTools {
		if count >= 2 {
			return true
		}
	}
	return false
}

// truncateForPrompt truncates a string for inclusion in a prompt.
func truncateForPrompt(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}

// minFloat returns the minimum of two float64 values.
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// maxFloat returns the maximum of two float64 values.
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
