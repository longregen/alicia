package baselines

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/longregen/alicia/internal/prompt"
)

// MemorySelectionSignature is the GEPA-optimizable signature for memory selection
var MemorySelectionSignature = prompt.MustParseSignature(
	"user_message, conversation_context, candidate_memories -> selected_memory_ids, relevance_reasoning",
)

// MemoryRankingSignature for re-ranking candidate memories
var MemoryRankingSignature = prompt.MustParseSignature(
	"user_message, memories_with_scores -> ranked_memory_ids, ranking_reasoning",
)

// MemorySelectionSeedPrompt is the baseline seed prompt for GEPA optimization
var MemorySelectionSeedPrompt = `You are a memory selection specialist. Given the user's message and candidate memories retrieved via semantic search, determine which memories are genuinely relevant and should be included in the conversation context.

SELECTION CRITERIA:
1. RELEVANCE: Does the memory directly relate to what the user is asking about?
2. RECENCY: Recent memories may be more relevant for ongoing topics
3. SPECIFICITY: Prefer specific facts over general information
4. CONTEXT FIT: Does the memory fit the current conversation flow?
5. AVOID NOISE: Exclude memories that are only superficially similar

FILTERING RULES:
- Exclude memories that share keywords but aren't conceptually related
- Exclude memories about different topics that happen to use similar language
- Exclude memories that would confuse rather than help the response
- Include memories that provide essential context the user expects you to know

RESPONSE FORMAT:
- selected_memory_ids: JSON array of memory IDs to include (e.g., ["mem_123", "mem_456"])
- relevance_reasoning: Brief explanation of why each memory was selected or excluded

Be conservative: it's better to include fewer highly-relevant memories than many tangentially-related ones.`

// MemoryRankingSeedPrompt for the ranking task
var MemoryRankingSeedPrompt = `You are a memory ranking specialist. Given candidate memories with their similarity scores, re-rank them by actual usefulness for answering the user's message.

RANKING FACTORS:
1. Semantic relevance (not just embedding similarity)
2. Information completeness - does this memory provide actionable info?
3. Recency - when was this learned?
4. User importance - was this explicitly marked as important?
5. Conversation coherence - does using this memory make sense in context?

The embedding similarity score is a starting point, but you should adjust rankings based on true semantic understanding.

RESPONSE FORMAT:
- ranked_memory_ids: JSON array of memory IDs in order of usefulness (most useful first)
- ranking_reasoning: Brief explanation of the ranking decisions`

// CandidateMemory represents a memory candidate for selection/ranking
type CandidateMemory struct {
	ID              string   `json:"id"`
	Content         string   `json:"content"`
	SimilarityScore float32  `json:"similarity_score"`
	Importance      float32  `json:"importance"`
	DaysSinceAccess int      `json:"days_since_access"`
	Tags            []string `json:"tags,omitempty"`
	Category        string   `json:"category,omitempty"`
}

// MemorySelectionExample represents a training/validation example
type MemorySelectionExample struct {
	UserMessage         string            `json:"user_message"`
	ConversationContext string            `json:"conversation_context"`
	CandidateMemories   []CandidateMemory `json:"candidate_memories"`
	ExpectedSelectedIDs []string          `json:"expected_selected_ids"`
	ExpectedExcludedIDs []string          `json:"expected_excluded_ids"`
	Category            string            `json:"category"` // For stratified sampling
}

// ToPromptExample converts to prompt.Example for GEPA
func (e *MemorySelectionExample) ToPromptExample() prompt.Example {
	memoriesJSON, err := json.Marshal(e.CandidateMemories)
	if err != nil {
		// Fallback to empty JSON array if marshaling fails
		memoriesJSON = []byte("[]")
	}
	selectedJSON, err := json.Marshal(e.ExpectedSelectedIDs)
	if err != nil {
		// Fallback to empty JSON array if marshaling fails
		selectedJSON = []byte("[]")
	}
	return prompt.Example{
		Inputs: map[string]any{
			"user_message":         e.UserMessage,
			"conversation_context": e.ConversationContext,
			"candidate_memories":   string(memoriesJSON),
		},
		Outputs: map[string]any{
			"selected_memory_ids": string(selectedJSON),
		},
	}
}

// MemorySelectionMetric provides GEPA-optimized feedback for memory selection
type MemorySelectionMetric struct {
	// memoryService ports.MemoryService // Optional: for actual memory lookups - TODO: define interface
}

// NewMemorySelectionMetric creates a new memory selection metric
func NewMemorySelectionMetric(memoryService any) *MemorySelectionMetric {
	return &MemorySelectionMetric{
		// memoryService: memoryService,
	}
}

// Score evaluates memory selection with rich GEPA feedback
func (m *MemorySelectionMetric) Score(
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
	return m.scoreSyntheticExample(gold, pred, trace)
}

// scoreVoteExample scores examples derived from user votes
func (m *MemorySelectionMetric) scoreVoteExample(gold, pred prompt.Example, voteValue int) (prompt.ScoreWithFeedback, error) {
	if voteValue == -1 { // VoteValueDown
		// Downvote: low score + diagnostic feedback for GEPA reflection
		quickFeedback, _ := gold.Outputs["_quick_feedback"].(string)

		feedback := buildDiagnosticFeedbackForMemoryVote(quickFeedback)
		return prompt.ScoreWithFeedback{Score: 0.1, Feedback: feedback}, nil
	}

	// Upvote: high score if prediction matches
	predMemoryID := toString(pred.Outputs["selected_memory_id"])
	goldMemoryID := toString(gold.Outputs["selected_memory_id"])

	if predMemoryID == goldMemoryID {
		return prompt.ScoreWithFeedback{Score: 1.0, Feedback: "Correct memory selection."}, nil
	}

	return prompt.ScoreWithFeedback{
		Score:    0.5,
		Feedback: fmt.Sprintf("Selected memory '%s' but expected '%s'.", predMemoryID, goldMemoryID),
	}, nil
}

// scoreSyntheticExample handles scoring for synthetic training examples
func (m *MemorySelectionMetric) scoreSyntheticExample(gold, pred prompt.Example, trace *prompt.Trace) (prompt.ScoreWithFeedback, error) {
	expectedIDs := parseStringArray(toString(gold.Outputs["selected_memory_ids"]))
	predictedIDs := parseStringArray(toString(pred.Outputs["selected_memory_ids"]))
	userMessage := toString(gold.Inputs["user_message"])

	var feedbackParts []string
	var score float64

	// Create sets for comparison
	expectedSet := make(map[string]bool)
	for _, id := range expectedIDs {
		expectedSet[id] = true
	}

	predictedSet := make(map[string]bool)
	for _, id := range predictedIDs {
		predictedSet[id] = true
	}

	// Component 1: Precision - what fraction of selected memories were correct? (40% weight)
	truePositives := 0
	falsePositives := []string{}
	for _, id := range predictedIDs {
		if expectedSet[id] {
			truePositives++
		} else {
			falsePositives = append(falsePositives, id)
		}
	}

	precision := 0.0
	if len(predictedIDs) > 0 {
		precision = float64(truePositives) / float64(len(predictedIDs))
	} else if len(expectedIDs) == 0 {
		precision = 1.0 // Correctly selected nothing when nothing was expected
	}
	score += precision * 0.4

	// Component 2: Recall - what fraction of relevant memories were selected? (40% weight)
	falseNegatives := []string{}
	for _, id := range expectedIDs {
		if !predictedSet[id] {
			falseNegatives = append(falseNegatives, id)
		}
	}

	recall := 0.0
	if len(expectedIDs) > 0 {
		recall = float64(truePositives) / float64(len(expectedIDs))
	} else {
		recall = 1.0 // Nothing to recall - recall is perfect by definition
	}
	score += recall * 0.4

	// Component 3: Reasoning quality (20% weight)
	reasoning := toString(pred.Outputs["relevance_reasoning"])
	reasoningScore, reasoningFeedback := m.scoreReasoning(reasoning, expectedIDs, userMessage)
	score += reasoningScore * 0.2

	// Generate feedback
	if precision == 1.0 && recall == 1.0 {
		feedbackParts = append(feedbackParts, "Perfect memory selection!")
	} else {
		if len(falsePositives) > 0 {
			feedbackParts = append(feedbackParts,
				fmt.Sprintf("OVER-SELECTED: Included %v but these weren't relevant. These memories may share keywords but don't match the user's actual intent.",
					falsePositives))
		}
		if len(falseNegatives) > 0 {
			feedbackParts = append(feedbackParts,
				fmt.Sprintf("MISSED: Should have included %v. These memories contain information the user would expect you to know.",
					falseNegatives))
		}
		if len(predictedIDs) == 0 && len(expectedIDs) > 0 {
			feedbackParts = append(feedbackParts,
				"Selected no memories when relevant ones were available. The user's message references information you should recall.")
		}
		if len(expectedIDs) == 0 && len(predictedIDs) > 0 {
			feedbackParts = append(feedbackParts,
				"Selected memories when none were relevant. Be more conservative - not every message needs memory context.")
		}
	}

	if reasoningFeedback != "" {
		feedbackParts = append(feedbackParts, reasoningFeedback)
	}

	// Add context-specific guidance
	feedback := strings.Join(feedbackParts, " ")
	feedback += m.getCategoryGuidance(gold)

	return prompt.ScoreWithFeedback{
		Score:    score,
		Feedback: feedback,
	}, nil
}

// scoreReasoning evaluates reasoning quality
func (m *MemorySelectionMetric) scoreReasoning(reasoning string, expectedIDs []string, userMessage string) (float64, string) {
	if reasoning == "" {
		return 0.0, "No reasoning provided. Explain why each memory was selected or excluded."
	}

	score := 0.5 // Base score for providing reasoning

	// Check if reasoning mentions the selected memory IDs
	mentionsMemories := false
	for _, id := range expectedIDs {
		if strings.Contains(reasoning, id) {
			mentionsMemories = true
			break
		}
	}
	if mentionsMemories {
		score += 0.25
	}

	// Check if reasoning explains relevance criteria
	relevanceKeywords := []string{"relevant", "related", "context", "user", "asking", "topic"}
	for _, kw := range relevanceKeywords {
		if strings.Contains(strings.ToLower(reasoning), kw) {
			score += 0.25
			break
		}
	}

	if score < 1.0 {
		return score, "Reasoning should reference specific memories and explain their relevance to the user's message."
	}
	return score, ""
}

// getCategoryGuidance provides category-specific improvement guidance
func (m *MemorySelectionMetric) getCategoryGuidance(gold prompt.Example) string {
	memoriesJSON := toString(gold.Inputs["candidate_memories"])
	var memories []CandidateMemory
	if err := json.Unmarshal([]byte(memoriesJSON), &memories); err != nil {
		return ""
	}

	// Analyze the memories to provide targeted guidance
	var highSimilarityCount, lowImportanceCount int
	for _, mem := range memories {
		if mem.SimilarityScore > 0.8 {
			highSimilarityCount++
		}
		if mem.Importance < 0.3 {
			lowImportanceCount++
		}
	}

	var guidance string
	if highSimilarityCount > 3 {
		guidance = " [MANY SIMILAR] When multiple memories have high similarity, prioritize specificity and recency over raw scores."
	}
	if lowImportanceCount > len(memories)/2 {
		guidance += " [LOW IMPORTANCE] Many candidates are low-importance - be selective about including contextual noise."
	}

	return guidance
}

// SyntheticMemorySelectionDataset generates training/validation examples
func SyntheticMemorySelectionDataset() (trainset, valset []prompt.Example) {
	// Define diverse training scenarios
	trainingExamples := []MemorySelectionExample{
		// Direct reference - user explicitly asks about something they told us
		{
			UserMessage:         "What was my daughter's name again?",
			ConversationContext: "Casual conversation",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_001", Content: "User's daughter is named Emma", SimilarityScore: 0.92, Importance: 0.8},
				{ID: "mem_002", Content: "User mentioned their daughter started kindergarten", SimilarityScore: 0.78, Importance: 0.5},
				{ID: "mem_003", Content: "User's name is David", SimilarityScore: 0.45, Importance: 0.6},
			},
			ExpectedSelectedIDs: []string{"mem_001"},
			ExpectedExcludedIDs: []string{"mem_002", "mem_003"},
			Category:            "direct_reference",
		},

		// Context enrichment - user asks about topic we have background on
		{
			UserMessage:         "I'm thinking about my project again",
			ConversationContext: "User mentioned Phoenix project before",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_010", Content: "User is working on Phoenix project - a mobile app for fitness tracking", SimilarityScore: 0.85, Importance: 0.9},
				{ID: "mem_011", Content: "Phoenix project uses React Native", SimilarityScore: 0.72, Importance: 0.6},
				{ID: "mem_012", Content: "User prefers dark mode in apps", SimilarityScore: 0.35, Importance: 0.4},
			},
			ExpectedSelectedIDs: []string{"mem_010", "mem_011"},
			ExpectedExcludedIDs: []string{"mem_012"},
			Category:            "context_enrichment",
		},

		// False positives - high similarity but not relevant
		{
			UserMessage:         "I love hiking in the mountains!",
			ConversationContext: "Discussing hobbies",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_020", Content: "User enjoys hiking on weekends", SimilarityScore: 0.88, Importance: 0.5},
				{ID: "mem_021", Content: "User mentioned they like mountain biking", SimilarityScore: 0.75, Importance: 0.4}, // Similar topic but not same
				{ID: "mem_022", Content: "User's office is on Mountain View Road", SimilarityScore: 0.68, Importance: 0.3},   // False positive - keyword match
			},
			ExpectedSelectedIDs: []string{"mem_020"},
			ExpectedExcludedIDs: []string{"mem_021", "mem_022"},
			Category:            "filter_false_positives",
		},

		// No memories needed - conversational message
		{
			UserMessage:         "That's really interesting, tell me more!",
			ConversationContext: "User responding to explanation",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_030", Content: "User finds AI technology interesting", SimilarityScore: 0.52, Importance: 0.3},
				{ID: "mem_031", Content: "User asked about machine learning basics", SimilarityScore: 0.48, Importance: 0.4},
			},
			ExpectedSelectedIDs: []string{},
			ExpectedExcludedIDs: []string{"mem_030", "mem_031"},
			Category:            "no_memory_needed",
		},

		// Multiple relevant memories
		{
			UserMessage:         "Can you help me plan my trip to Japan?",
			ConversationContext: "Travel planning",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_040", Content: "User planning trip to Japan in March", SimilarityScore: 0.95, Importance: 0.9},
				{ID: "mem_041", Content: "User interested in visiting Kyoto temples", SimilarityScore: 0.82, Importance: 0.7},
				{ID: "mem_042", Content: "User is vegetarian", SimilarityScore: 0.45, Importance: 0.8},           // Relevant for food recommendations
				{ID: "mem_043", Content: "User visited Paris last year", SimilarityScore: 0.55, Importance: 0.4}, // Not relevant
			},
			ExpectedSelectedIDs: []string{"mem_040", "mem_041", "mem_042"},
			ExpectedExcludedIDs: []string{"mem_043"},
			Category:            "multiple_relevant",
		},

		// Preference recall
		{
			UserMessage:         "What kind of music should I listen to while working?",
			ConversationContext: "Productivity discussion",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_050", Content: "User prefers instrumental music while working", SimilarityScore: 0.78, Importance: 0.7},
				{ID: "mem_051", Content: "User's favorite band is Radiohead", SimilarityScore: 0.65, Importance: 0.4},
				{ID: "mem_052", Content: "User finds jazz relaxing", SimilarityScore: 0.55, Importance: 0.5},
			},
			ExpectedSelectedIDs: []string{"mem_050"},
			ExpectedExcludedIDs: []string{"mem_051", "mem_052"},
			Category:            "preference_recall",
		},

		// Time-sensitive context
		{
			UserMessage:         "How's the deadline looking?",
			ConversationContext: "Project discussion",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_060", Content: "Project deadline is December 15th", SimilarityScore: 0.75, Importance: 0.9, DaysSinceAccess: 2},
				{ID: "mem_061", Content: "User has a meeting with client on Monday", SimilarityScore: 0.45, Importance: 0.6, DaysSinceAccess: 5},
				{ID: "mem_062", Content: "Previous project deadline was missed", SimilarityScore: 0.55, Importance: 0.3, DaysSinceAccess: 60},
			},
			ExpectedSelectedIDs: []string{"mem_060"},
			ExpectedExcludedIDs: []string{"mem_061", "mem_062"},
			Category:            "time_sensitive",
		},

		// Greeting - no memory needed
		{
			UserMessage:         "Good morning! How are you today?",
			ConversationContext: "Start of conversation",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_070", Content: "User prefers morning meetings", SimilarityScore: 0.42, Importance: 0.3},
				{ID: "mem_071", Content: "User is in EST timezone", SimilarityScore: 0.38, Importance: 0.4},
			},
			ExpectedSelectedIDs: []string{},
			ExpectedExcludedIDs: []string{"mem_070", "mem_071"},
			Category:            "no_memory_needed",
		},

		// Implicit reference
		{
			UserMessage:         "I'm going to try that recipe tonight",
			ConversationContext: "Previous discussion about cooking",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_080", Content: "Shared pasta carbonara recipe with user", SimilarityScore: 0.72, Importance: 0.6, DaysSinceAccess: 1},
				{ID: "mem_081", Content: "User is allergic to shellfish", SimilarityScore: 0.35, Importance: 0.9},
				{ID: "mem_082", Content: "User's favorite cuisine is Italian", SimilarityScore: 0.55, Importance: 0.4},
			},
			ExpectedSelectedIDs: []string{"mem_080", "mem_081"},
			ExpectedExcludedIDs: []string{"mem_082"},
			Category:            "implicit_reference",
		},

		// Technical context
		{
			UserMessage:         "I'm getting that error again with the database",
			ConversationContext: "Technical troubleshooting",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_090", Content: "User had PostgreSQL connection timeout issues", SimilarityScore: 0.82, Importance: 0.7},
				{ID: "mem_091", Content: "User's project uses PostgreSQL 14", SimilarityScore: 0.68, Importance: 0.5},
				{ID: "mem_092", Content: "User prefers VS Code for development", SimilarityScore: 0.35, Importance: 0.3},
			},
			ExpectedSelectedIDs: []string{"mem_090", "mem_091"},
			ExpectedExcludedIDs: []string{"mem_092"},
			Category:            "technical_context",
		},
	}

	// Validation examples - different instances but similar patterns
	validationExamples := []MemorySelectionExample{
		{
			UserMessage:         "What's my son's birthday?",
			ConversationContext: "Family discussion",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_v01", Content: "User's son Alex has birthday on June 5th", SimilarityScore: 0.90, Importance: 0.8},
				{ID: "mem_v02", Content: "User mentioned their son is 8 years old", SimilarityScore: 0.72, Importance: 0.5},
				{ID: "mem_v03", Content: "User's birthday is in November", SimilarityScore: 0.55, Importance: 0.6},
			},
			ExpectedSelectedIDs: []string{"mem_v01"},
			ExpectedExcludedIDs: []string{"mem_v02", "mem_v03"},
			Category:            "direct_reference",
		},
		{
			UserMessage:         "Thanks for the help!",
			ConversationContext: "End of task",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_v10", Content: "User appreciates detailed explanations", SimilarityScore: 0.48, Importance: 0.4},
			},
			ExpectedSelectedIDs: []string{},
			ExpectedExcludedIDs: []string{"mem_v10"},
			Category:            "no_memory_needed",
		},
		{
			UserMessage:         "Let's continue with the marketing campaign",
			ConversationContext: "Work discussion",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_v20", Content: "User working on Q1 marketing campaign for SaaS product", SimilarityScore: 0.88, Importance: 0.8},
				{ID: "mem_v21", Content: "Campaign budget is $50,000", SimilarityScore: 0.75, Importance: 0.7},
				{ID: "mem_v22", Content: "User's company sells marketing software", SimilarityScore: 0.62, Importance: 0.5},
			},
			ExpectedSelectedIDs: []string{"mem_v20", "mem_v21"},
			ExpectedExcludedIDs: []string{"mem_v22"},
			Category:            "context_enrichment",
		},
		{
			UserMessage:         "I love coffee so much",
			ConversationContext: "Casual chat",
			CandidateMemories: []CandidateMemory{
				{ID: "mem_v30", Content: "User's favorite coffee shop is Blue Bottle", SimilarityScore: 0.72, Importance: 0.4},
				{ID: "mem_v31", Content: "User drinks 3 cups of coffee daily", SimilarityScore: 0.68, Importance: 0.3},
				{ID: "mem_v32", Content: "User mentioned loving espresso drinks", SimilarityScore: 0.65, Importance: 0.4},
			},
			ExpectedSelectedIDs: []string{}, // Just expressing opinion, no memory needed
			ExpectedExcludedIDs: []string{"mem_v30", "mem_v31", "mem_v32"},
			Category:            "no_memory_needed",
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

// parseStringArray parses a JSON array string into a string slice
func parseStringArray(s string) []string {
	if s == "" || s == "[]" || s == "null" {
		return []string{}
	}

	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		// Try splitting by comma if not valid JSON
		s = strings.Trim(s, "[]")
		if s == "" {
			return []string{}
		}
		parts := strings.Split(s, ",")
		for _, p := range parts {
			p = strings.TrimSpace(strings.Trim(p, `"`))
			if p != "" {
				result = append(result, p)
			}
		}
	}
	return result
}

// buildDiagnosticFeedbackForMemoryVote generates rich diagnostic feedback from quick_feedback for memory selection
func buildDiagnosticFeedbackForMemoryVote(quickFeedback string) string {
	switch quickFeedback {
	case "wrong_context":
		return "This memory was retrieved but wasn't relevant to the user's actual intent."
	case "too_generic":
		return "This memory was too generic to be useful. More specific memories should be prioritized."
	case "outdated":
		return "This memory contains outdated information. Consider recency when selecting memories."
	case "incorrect":
		return "This memory contains incorrect information that should not have been used."
	default:
		return "The memory selection was marked as incorrect by the user."
	}
}
