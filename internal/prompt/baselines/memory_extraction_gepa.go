package baselines

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/longregen/alicia/internal/prompt"
)

// MemoryExtractionSignature is the GEPA-optimizable signature for memory extraction
var MemoryExtractionSignature = prompt.MustParseSignature(
	"conversation_text, conversation_context -> extracted_facts, importance_scores, extraction_reasoning",
)

// MemoryExtractionSeedPrompt is the baseline seed prompt for GEPA optimization
var MemoryExtractionSeedPrompt = `You are a memory extraction specialist. Extract factual information from conversations that should be remembered for future reference.

EXTRACTION CRITERIA:
1. FACTUAL: Only extract objective facts, preferences, and explicit information
2. SIGNIFICANT: Only extract information that would be useful to recall later
3. SPECIFIC: Extract concrete details, not vague observations
4. PERSISTENT: Only extract information that remains true over time
5. USER-CENTRIC: Focus on facts about the user, their preferences, projects, and context

WHAT TO EXTRACT:
- Personal preferences (favorite foods, music, working styles)
- Biographical facts (names, birthdays, locations, roles)
- Project/work context (technologies used, deadlines, team members)
- Explicit instructions or rules the user sets
- Domain-specific knowledge relevant to the user's work
- Important relationships and connections

WHAT NOT TO EXTRACT:
- Conversational filler ("that's interesting", "I see", "thanks")
- Transient information (current time, weather, temporary states)
- Information that's only relevant to the current conversation
- Opinions about generic topics that aren't personal preferences
- Information already implied by context
- Trivial details with no future utility

RESPONSE FORMAT:
- extracted_facts: JSON array of fact strings, each fact should be self-contained and clear
- importance_scores: JSON array of floats (0.0-1.0) corresponding to each fact, where:
  * 0.9-1.0: Critical information (names, allergies, explicit rules)
  * 0.7-0.8: Important preferences or project context
  * 0.5-0.6: Useful context or secondary details
  * 0.3-0.4: Minor details that might be occasionally useful
  * 0.0-0.2: Rarely useful (should probably not be extracted)
- extraction_reasoning: Brief explanation of what was extracted and why, or why nothing was extracted

IMPORTANT: It is completely valid to extract NOTHING from conversational filler. Empty extraction is often the correct answer.`

// ToolResultMemorizationPrompt is used to analyze tool results for memorization worthiness
var ToolResultMemorizationPrompt = `You are a memory analysis specialist. Your task is to determine whether tool results contain information worth storing as long-term memories.

CRITERIA FOR MEMORIZATION:
1. USER-SPECIFIC: Information that is personalized or specific to the user
2. DURABLE: Facts that will remain relevant for future conversations
3. ACTIONABLE: Information that could inform future responses or decisions
4. NOVEL: Information not already implied by common knowledge

WHAT TO MEMORIZE:
- User preferences discovered through tool use (e.g., favorite restaurants from search)
- Account details or configurations retrieved via tools
- Resolved technical issues and their solutions
- Project-specific information from file or database queries
- Contact information or relationships discovered
- User-specific data patterns or behaviors

WHAT NOT TO MEMORIZE:
- Transient data (current weather, time, live prices)
- Generic information available to anyone
- Error messages or failed operations
- Data too large to meaningfully summarize
- Information specific only to the current request
- Temporary states that will change

RESPONSE: Provide a JSON object with your analysis.`

// ExtractedFact represents a single extracted fact with metadata
type ExtractedFact struct {
	Content    string  `json:"content"`
	Importance float64 `json:"importance"`
	Category   string  `json:"category,omitempty"`
}

// MemoryExtractionExample represents a training/validation example
type MemoryExtractionExample struct {
	ConversationText    string          `json:"conversation_text"`
	ConversationContext string          `json:"conversation_context"`
	ExpectedFacts       []ExtractedFact `json:"expected_facts"`
	Category            string          `json:"category"` // For stratified sampling
}

// ToPromptExample converts to prompt.Example for GEPA
func (e *MemoryExtractionExample) ToPromptExample() prompt.Example {
	factContents := make([]string, len(e.ExpectedFacts))
	importanceScores := make([]float64, len(e.ExpectedFacts))

	for i, fact := range e.ExpectedFacts {
		factContents[i] = fact.Content
		importanceScores[i] = fact.Importance
	}

	factsJSON, _ := json.Marshal(factContents)
	scoresJSON, _ := json.Marshal(importanceScores)

	return prompt.Example{
		Inputs: map[string]any{
			"conversation_text":    e.ConversationText,
			"conversation_context": e.ConversationContext,
		},
		Outputs: map[string]any{
			"extracted_facts":   string(factsJSON),
			"importance_scores": string(scoresJSON),
		},
	}
}

// MemoryExtractionMetric provides GEPA-optimized feedback for memory extraction
type MemoryExtractionMetric struct {
	embedService any // Optional: for semantic similarity - ports.EmbeddingService - TODO: use correct interface
}

// NewMemoryExtractionMetric creates a new memory extraction metric
func NewMemoryExtractionMetric(embedService any) *MemoryExtractionMetric {
	return &MemoryExtractionMetric{
		embedService: embedService,
	}
}

// Score evaluates memory extraction with rich GEPA feedback
func (m *MemoryExtractionMetric) Score(
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
	return m.scoreSyntheticExample(ctx, gold, pred, trace)
}

// scoreVoteExample scores examples derived from user votes
func (m *MemoryExtractionMetric) scoreVoteExample(gold, pred prompt.Example, voteValue int) (prompt.ScoreWithFeedback, error) {
	if voteValue == -1 { // VoteValueDown
		// Downvote: low score + diagnostic feedback for GEPA reflection
		quickFeedback, _ := gold.Outputs["_quick_feedback"].(string)

		feedback := buildDiagnosticFeedbackForMemoryExtractionVote(quickFeedback)
		return prompt.ScoreWithFeedback{Score: 0.1, Feedback: feedback}, nil
	}

	// Upvote: high score (memory extraction votes are more about quality than exact matching)
	return prompt.ScoreWithFeedback{Score: 1.0, Feedback: "Memory extraction was marked as correct by the user."}, nil
}

// scoreSyntheticExample handles scoring for synthetic training examples
func (m *MemoryExtractionMetric) scoreSyntheticExample(ctx context.Context, gold, pred prompt.Example, trace *prompt.Trace) (prompt.ScoreWithFeedback, error) {
	expectedFacts := parseStringArray(toString(gold.Outputs["extracted_facts"]))
	predictedFacts := parseStringArray(toString(pred.Outputs["extracted_facts"]))
	expectedScores := parseFloatArray(toString(gold.Outputs["importance_scores"]))
	predictedScores := parseFloatArray(toString(pred.Outputs["importance_scores"]))
	conversationText := toString(gold.Inputs["conversation_text"])

	var feedbackParts []string
	var totalScore float64

	// Component 1: Precision - avoiding hallucinated facts (35% weight)
	precisionScore, precisionFeedback := m.scorePrecision(ctx, expectedFacts, predictedFacts, conversationText)
	totalScore += precisionScore * 0.35
	if precisionFeedback != "" {
		feedbackParts = append(feedbackParts, precisionFeedback)
	}

	// Component 2: Recall - not missing important facts (35% weight)
	recallScore, recallFeedback := m.scoreRecall(ctx, expectedFacts, predictedFacts)
	totalScore += recallScore * 0.35
	if recallFeedback != "" {
		feedbackParts = append(feedbackParts, recallFeedback)
	}

	// Component 3: Importance calibration - accurate scoring (20% weight)
	calibrationScore, calibrationFeedback := m.scoreImportanceCalibration(
		expectedFacts, predictedFacts, expectedScores, predictedScores)
	totalScore += calibrationScore * 0.20
	if calibrationFeedback != "" {
		feedbackParts = append(feedbackParts, calibrationFeedback)
	}

	// Component 4: Reasoning quality (10% weight)
	reasoning := toString(pred.Outputs["extraction_reasoning"])
	reasoningScore, reasoningFeedback := m.scoreReasoning(reasoning, predictedFacts, conversationText)
	totalScore += reasoningScore * 0.10
	if reasoningFeedback != "" {
		feedbackParts = append(feedbackParts, reasoningFeedback)
	}

	// Compile feedback
	feedback := strings.Join(feedbackParts, " ")

	// Add category-specific guidance
	if totalScore < 0.9 {
		feedback += m.getCategoryGuidance(gold, pred)
	}

	return prompt.ScoreWithFeedback{
		Score:    totalScore,
		Feedback: feedback,
	}, nil
}

// scorePrecision evaluates how many extracted facts are correct (avoiding hallucinations)
func (m *MemoryExtractionMetric) scorePrecision(
	ctx context.Context,
	expectedFacts, predictedFacts []string,
	conversationText string,
) (float64, string) {
	if len(predictedFacts) == 0 {
		if len(expectedFacts) == 0 {
			return 1.0, "Correctly extracted nothing from conversational filler."
		}
		return 1.0, "" // No false positives if nothing extracted
	}

	// Match predicted facts against expected facts
	matches := m.matchFacts(ctx, predictedFacts, expectedFacts)

	truePositives := 0
	falsePositives := []string{}

	for i, pred := range predictedFacts {
		if i < len(matches) && matches[i] != -1 {
			truePositives++
		} else {
			falsePositives = append(falsePositives, pred)
		}
	}

	precision := float64(truePositives) / float64(len(predictedFacts))

	if len(falsePositives) == 0 {
		return precision, "All extracted facts are valid."
	}

	truncated := falsePositives
	if len(falsePositives) > 2 {
		truncated = truncateFacts(falsePositives, 2)
	}
	feedback := fmt.Sprintf("HALLUCINATED FACTS: Extracted %d facts not present in conversation: %v. Only extract information explicitly stated.",
		len(falsePositives), truncated)

	return precision, feedback
}

// scoreRecall evaluates how many important facts were found (not missing information)
func (m *MemoryExtractionMetric) scoreRecall(
	ctx context.Context,
	expectedFacts, predictedFacts []string,
) (float64, string) {
	if len(expectedFacts) == 0 {
		return 1.0, "" // Nothing to recall
	}

	matches := m.matchFacts(ctx, predictedFacts, expectedFacts)

	truePositives := 0
	missedFacts := []string{}

	for i, expected := range expectedFacts {
		found := false
		for _, matchIdx := range matches {
			if matchIdx == i {
				found = true
				break
			}
		}
		if found {
			truePositives++
		} else {
			missedFacts = append(missedFacts, expected)
		}
	}

	recall := float64(truePositives) / float64(len(expectedFacts))

	if len(missedFacts) == 0 {
		return recall, "All important facts were extracted."
	}

	truncated := missedFacts
	if len(missedFacts) > 2 {
		truncated = truncateFacts(missedFacts, 2)
	}
	feedback := fmt.Sprintf("MISSED FACTS: Failed to extract %d important facts: %v. Look for significant information that should be remembered.",
		len(missedFacts), truncated)

	return recall, feedback
}

// scoreImportanceCalibration evaluates accuracy of importance scores
func (m *MemoryExtractionMetric) scoreImportanceCalibration(
	expectedFacts, predictedFacts []string,
	expectedScores, predictedScores []float64,
) (float64, string) {
	if len(predictedFacts) == 0 || len(predictedScores) != len(predictedFacts) {
		if len(expectedFacts) == 0 {
			return 1.0, "" // No calibration needed for empty extraction
		}
		return 0.0, "Importance scores don't match number of extracted facts."
	}

	// Match facts and compare scores
	ctx := context.Background()
	matches := m.matchFacts(ctx, predictedFacts, expectedFacts)

	var scoreErrors []float64
	var badScores []string

	for i, predScore := range predictedScores {
		// Validate score range
		if predScore < 0.0 || predScore > 1.0 {
			badScores = append(badScores, fmt.Sprintf("fact %d has invalid score %.2f", i, predScore))
			continue
		}

		if i >= len(matches) {
			continue
		}
		matchIdx := matches[i]
		if matchIdx != -1 && matchIdx < len(expectedScores) {
			expectedScore := expectedScores[matchIdx]
			error := math.Abs(predScore - expectedScore)
			scoreErrors = append(scoreErrors, error)

			// Flag significant calibration errors
			if error > 0.3 {
				badScores = append(badScores, fmt.Sprintf("'%s' scored %.2f but should be %.2f",
					truncate(predictedFacts[i], 40), predScore, expectedScore))
			}
		}
	}

	if len(badScores) > 0 {
		return 0.5, fmt.Sprintf("IMPORTANCE CALIBRATION: %s. Review importance scoring criteria.",
			strings.Join(badScores, "; "))
	}

	// Calculate average calibration error
	if len(scoreErrors) == 0 {
		return 1.0, "Importance scores well-calibrated."
	}

	avgError := 0.0
	for _, err := range scoreErrors {
		avgError += err
	}
	avgError /= float64(len(scoreErrors))

	// Convert error to score (0.0 error = 1.0 score, 1.0 error = 0.0 score)
	calibrationScore := 1.0 - avgError

	if avgError > 0.2 {
		return calibrationScore, fmt.Sprintf("Average importance score error: %.2f. Review importance criteria.", avgError)
	}

	return calibrationScore, ""
}

// scoreReasoning evaluates reasoning quality
func (m *MemoryExtractionMetric) scoreReasoning(reasoning string, extractedFacts []string, conversationText string) (float64, string) {
	if reasoning == "" {
		return 0.0, "No reasoning provided. Explain what was extracted and why."
	}

	score := 0.4 // Base score for providing reasoning

	// Check if reasoning explains the extraction decision
	explainKeywords := []string{"extracted", "because", "since", "important", "significant"}
	for _, kw := range explainKeywords {
		if strings.Contains(strings.ToLower(reasoning), kw) {
			score += 0.3
			break
		}
	}

	// Check if reasoning addresses the empty extraction case
	if len(extractedFacts) == 0 {
		emptyKeywords := []string{"nothing", "no facts", "conversational", "filler", "no information"}
		for _, kw := range emptyKeywords {
			if strings.Contains(strings.ToLower(reasoning), kw) {
				score += 0.3
				break
			}
		}
	}

	if score < 1.0 {
		return score, "Reasoning should explain extraction decisions and criteria applied."
	}

	return score, ""
}

// getCategoryGuidance provides category-specific improvement guidance
func (m *MemoryExtractionMetric) getCategoryGuidance(gold, pred prompt.Example) string {
	category, ok := gold.Inputs["category"].(string)
	if !ok {
		return ""
	}

	switch category {
	case "personal_preferences":
		return " [PREFERENCES] Look for explicit statements like 'I prefer', 'my favorite', 'I like/dislike'."
	case "biographical":
		return " [BIOGRAPHICAL] Extract concrete facts: names, dates, locations, relationships."
	case "project_context":
		return " [PROJECT] Extract technical details, deadlines, team members, technologies used."
	case "instructions":
		return " [INSTRUCTIONS] Extract explicit rules or preferences the user sets for interaction."
	case "conversational_filler":
		return " [FILLER] This is conversational filler with no extractable facts. It's correct to extract nothing."
	case "mixed_content":
		return " [MIXED] Separate factual content from conversational filler. Only extract the facts."
	case "implicit_preferences":
		return " [IMPLICIT] Look for preferences implied by actions or repeated patterns."
	case "temporal_info":
		return " [TEMPORAL] Don't extract time-sensitive or transient information that won't be relevant later."
	case "dense_information":
		return " [DENSE] Break down complex information into individual, self-contained facts."
	default:
		return ""
	}
}

// matchFacts finds the best match for each fact in the source list within the target list
// Returns array where matchFacts[i] = index in target that matches source[i], or -1 if no match
func (m *MemoryExtractionMetric) matchFacts(ctx context.Context, source, target []string) []int {
	matches := make([]int, len(source))
	for i := range matches {
		matches[i] = -1
	}

	if len(source) == 0 || len(target) == 0 {
		return matches
	}

	// Try semantic matching if embedding service available
	// TODO: implement when embedService interface is properly defined
	// if m.embedService != nil {
	// 	allTexts := append(source, target...)
	// 	embeddings, err := m.embedService.EmbedBatch(ctx, allTexts)
	// 	if err == nil && len(embeddings) == len(allTexts) {
	// 		return m.semanticMatch(embeddings, len(source))
	// 	}
	// }

	// Fallback to string similarity
	return m.stringMatch(source, target)
}

// semanticMatch uses embeddings for matching
// TODO: uncomment when embedService interface is properly defined
// func (m *MemoryExtractionMetric) semanticMatch(embeddings []*ports.EmbeddingResult, sourceLen int) []int {
// 	matches := make([]int, sourceLen)
// 	targetLen := len(embeddings) - sourceLen
//
// 	for i := 0; i < sourceLen; i++ {
// 		bestScore := float32(0.0)
// 		bestMatch := -1
//
// 		for j := 0; j < targetLen; j++ {
// 			targetIdx := sourceLen + j
// 			similarity := cosineSimilarity(embeddings[i].Embedding, embeddings[targetIdx].Embedding)
//
// 			if similarity > bestScore && similarity > 0.7 { // Threshold for semantic match
// 				bestScore = similarity
// 				bestMatch = j
// 			}
// 		}
//
// 		matches[i] = bestMatch
// 	}
//
// 	return matches
// }

// stringMatch uses string similarity for matching
func (m *MemoryExtractionMetric) stringMatch(source, target []string) []int {
	matches := make([]int, len(source))

	for i, src := range source {
		bestScore := 0.0
		bestMatch := -1

		for j, tgt := range target {
			similarity := simpleStringSimilarity(src, tgt)

			if similarity > bestScore && similarity > 0.6 { // Threshold for string match
				bestScore = similarity
				bestMatch = j
			}
		}

		matches[i] = bestMatch
	}

	return matches
}

// SyntheticMemoryExtractionDataset generates training/validation examples
func SyntheticMemoryExtractionDataset() (trainset, valset []prompt.Example) {
	// Training examples covering diverse scenarios
	trainingExamples := []MemoryExtractionExample{
		// Personal preferences - explicit
		{
			ConversationText:    "I love listening to jazz music while I work. It helps me concentrate.",
			ConversationContext: "Discussion about work habits",
			ExpectedFacts: []ExtractedFact{
				{Content: "User prefers listening to jazz music while working", Importance: 0.7},
				{Content: "Jazz music helps user concentrate", Importance: 0.6},
			},
			Category: "personal_preferences",
		},
		{
			ConversationText:    "My favorite programming language is Go. I also really enjoy Python for data work.",
			ConversationContext: "Technical discussion",
			ExpectedFacts: []ExtractedFact{
				{Content: "User's favorite programming language is Go", Importance: 0.7},
				{Content: "User enjoys Python for data work", Importance: 0.6},
			},
			Category: "personal_preferences",
		},

		// Biographical facts
		{
			ConversationText:    "My daughter Emma starts kindergarten next month. She's really excited about it!",
			ConversationContext: "Family conversation",
			ExpectedFacts: []ExtractedFact{
				{Content: "User has a daughter named Emma", Importance: 0.9},
				{Content: "Emma is starting kindergarten", Importance: 0.7},
			},
			Category: "biographical",
		},
		{
			ConversationText:    "I live in Seattle and work as a software engineer at a startup.",
			ConversationContext: "Introduction",
			ExpectedFacts: []ExtractedFact{
				{Content: "User lives in Seattle", Importance: 0.8},
				{Content: "User works as a software engineer at a startup", Importance: 0.8},
			},
			Category: "biographical",
		},

		// Project/technical context
		{
			ConversationText:    "We're using React Native for the Phoenix mobile app, and the deadline is March 15th.",
			ConversationContext: "Project discussion",
			ExpectedFacts: []ExtractedFact{
				{Content: "Phoenix mobile app uses React Native", Importance: 0.8},
				{Content: "Phoenix project deadline is March 15th", Importance: 0.9},
			},
			Category: "project_context",
		},
		{
			ConversationText:    "The API is built with Go and we're using PostgreSQL 14 for the database. Sarah is the tech lead.",
			ConversationContext: "Technical architecture",
			ExpectedFacts: []ExtractedFact{
				{Content: "API is built with Go", Importance: 0.7},
				{Content: "Project uses PostgreSQL 14 database", Importance: 0.7},
				{Content: "Sarah is the tech lead", Importance: 0.8},
			},
			Category: "project_context",
		},

		// Explicit instructions
		{
			ConversationText:    "Please always ask me before making database schema changes. I want to review them first.",
			ConversationContext: "Setting preferences",
			ExpectedFacts: []ExtractedFact{
				{Content: "User wants to review database schema changes before they are made", Importance: 0.95},
			},
			Category: "instructions",
		},
		{
			ConversationText:    "I prefer detailed explanations over brief summaries. Don't worry about being too verbose.",
			ConversationContext: "Communication preferences",
			ExpectedFacts: []ExtractedFact{
				{Content: "User prefers detailed explanations over brief summaries", Importance: 0.9},
			},
			Category: "instructions",
		},

		// Domain-specific knowledge
		{
			ConversationText:    "I'm allergic to peanuts and shellfish, so I have to be really careful when eating out.",
			ConversationContext: "Health discussion",
			ExpectedFacts: []ExtractedFact{
				{Content: "User is allergic to peanuts", Importance: 1.0},
				{Content: "User is allergic to shellfish", Importance: 1.0},
			},
			Category: "domain_knowledge",
		},

		// Conversational filler - should extract NOTHING
		{
			ConversationText:    "That's really interesting! I see what you mean.",
			ConversationContext: "Casual conversation",
			ExpectedFacts:       []ExtractedFact{},
			Category:            "conversational_filler",
		},
		{
			ConversationText:    "Thanks for your help! Have a great day.",
			ConversationContext: "Ending conversation",
			ExpectedFacts:       []ExtractedFact{},
			Category:            "conversational_filler",
		},
		{
			ConversationText:    "Hmm, let me think about that for a moment.",
			ConversationContext: "Thinking aloud",
			ExpectedFacts:       []ExtractedFact{},
			Category:            "conversational_filler",
		},

		// Mixed content - facts among noise
		{
			ConversationText:    "Yeah, I think that's a good idea. By the way, my birthday is June 15th if you want to remember that.",
			ConversationContext: "Casual conversation",
			ExpectedFacts: []ExtractedFact{
				{Content: "User's birthday is June 15th", Importance: 0.9},
			},
			Category: "mixed_content",
		},
		{
			ConversationText:    "That makes sense! Oh, and I should mention I'm working remotely from Toronto now.",
			ConversationContext: "Work discussion",
			ExpectedFacts: []ExtractedFact{
				{Content: "User is working remotely from Toronto", Importance: 0.8},
			},
			Category: "mixed_content",
		},

		// Implicit preferences
		{
			ConversationText:    "I always start my day by checking emails and then move to deep work in the afternoon.",
			ConversationContext: "Productivity discussion",
			ExpectedFacts: []ExtractedFact{
				{Content: "User checks emails first thing in the morning", Importance: 0.5},
				{Content: "User does deep work in the afternoon", Importance: 0.6},
			},
			Category: "implicit_preferences",
		},

		// Dense information
		{
			ConversationText:    "I'm Michael Chen, CTO at DataFlow Inc. We're a B2B SaaS company based in Austin with 50 employees.",
			ConversationContext: "Professional introduction",
			ExpectedFacts: []ExtractedFact{
				{Content: "User's name is Michael Chen", Importance: 0.95},
				{Content: "User is CTO at DataFlow Inc", Importance: 0.9},
				{Content: "DataFlow Inc is a B2B SaaS company", Importance: 0.7},
				{Content: "DataFlow Inc is based in Austin", Importance: 0.6},
				{Content: "DataFlow Inc has 50 employees", Importance: 0.5},
			},
			Category: "dense_information",
		},

		// Temporal/transient info - should extract NOTHING or very little
		{
			ConversationText:    "I'm feeling tired today. The weather is nice though.",
			ConversationContext: "Small talk",
			ExpectedFacts:       []ExtractedFact{},
			Category:            "temporal_info",
		},
		{
			ConversationText:    "I'm currently reading a book about machine learning, it's pretty good so far.",
			ConversationContext: "Casual conversation",
			ExpectedFacts:       []ExtractedFact{}, // Transient activity
			Category:            "temporal_info",
		},
	}

	// Validation examples - different instances but similar patterns
	validationExamples := []MemoryExtractionExample{
		{
			ConversationText:    "I prefer working early in the morning when it's quiet. That's when I'm most productive.",
			ConversationContext: "Work habits",
			ExpectedFacts: []ExtractedFact{
				{Content: "User prefers working early in the morning", Importance: 0.7},
				{Content: "User is most productive in the morning when it's quiet", Importance: 0.6},
			},
			Category: "personal_preferences",
		},
		{
			ConversationText:    "My son Jake plays soccer on the weekends. He's on the varsity team.",
			ConversationContext: "Family discussion",
			ExpectedFacts: []ExtractedFact{
				{Content: "User has a son named Jake", Importance: 0.9},
				{Content: "Jake plays soccer on varsity team", Importance: 0.7},
			},
			Category: "biographical",
		},
		{
			ConversationText:    "The mobile redesign uses Flutter and we're aiming to launch in Q2.",
			ConversationContext: "Project planning",
			ExpectedFacts: []ExtractedFact{
				{Content: "Mobile redesign uses Flutter", Importance: 0.8},
				{Content: "Mobile redesign planned to launch in Q2", Importance: 0.9},
			},
			Category: "project_context",
		},
		{
			ConversationText:    "Always run tests before committing code changes. This is important to me.",
			ConversationContext: "Development workflow",
			ExpectedFacts: []ExtractedFact{
				{Content: "User wants tests run before committing code changes", Importance: 0.95},
			},
			Category: "instructions",
		},
		{
			ConversationText:    "Okay, sounds good to me!",
			ConversationContext: "Agreement",
			ExpectedFacts:       []ExtractedFact{},
			Category:            "conversational_filler",
		},
		{
			ConversationText:    "Got it, thanks! Also, I'm lactose intolerant so no dairy for me.",
			ConversationContext: "Food discussion",
			ExpectedFacts: []ExtractedFact{
				{Content: "User is lactose intolerant", Importance: 1.0},
			},
			Category: "mixed_content",
		},
		{
			ConversationText:    "I usually take a walk after lunch to clear my head before afternoon meetings.",
			ConversationContext: "Daily routine",
			ExpectedFacts: []ExtractedFact{
				{Content: "User takes a walk after lunch", Importance: 0.5},
			},
			Category: "implicit_preferences",
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

func parseFloatArray(s string) []float64 {
	if s == "" || s == "[]" || s == "null" {
		return []float64{}
	}

	var result []float64
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return []float64{}
	}
	return result
}

func truncateFacts(facts []string, maxCount int) []string {
	if len(facts) <= maxCount {
		return facts
	}
	truncated := make([]string, maxCount)
	for i := 0; i < maxCount; i++ {
		truncated[i] = truncate(facts[i], 50)
	}
	return truncated
}


// simpleStringSimilarity provides a basic string similarity score
func simpleStringSimilarity(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	if a == b {
		return 1.0
	}

	// Simple Jaccard similarity on words
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	setA := make(map[string]bool)
	for _, word := range wordsA {
		setA[word] = true
	}

	setB := make(map[string]bool)
	for _, word := range wordsB {
		setB[word] = true
	}

	intersection := 0
	for word := range setA {
		if setB[word] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// buildDiagnosticFeedbackForMemoryExtractionVote generates rich diagnostic feedback from quick_feedback for memory extraction
func buildDiagnosticFeedbackForMemoryExtractionVote(quickFeedback string) string {
	switch quickFeedback {
	case "too_generic":
		return "The extracted memory was too generic to be useful. Extract more specific, actionable facts."
	case "incorrect":
		return "The extracted information was incorrect. Only extract facts that are explicitly stated."
	case "not_factual":
		return "The extracted content was not factual. Focus on objective facts, not opinions or interpretations."
	case "missing_context":
		return "The extraction missed important context. Include relevant details that make the fact meaningful."
	default:
		return "The memory extraction was marked as incorrect by the user."
	}
}
