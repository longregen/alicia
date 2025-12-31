package baselines

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/longregen/alicia/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryExtractionSignature(t *testing.T) {
	assert.NotNil(t, MemoryExtractionSignature)
	assert.Equal(t, "conversation_text__conversation_context__to__extracted_facts__importance_scores__extraction_reasoning", MemoryExtractionSignature.Name)
}

func TestMemoryExtractionSeedPrompt(t *testing.T) {
	assert.NotEmpty(t, MemoryExtractionSeedPrompt)
	assert.Contains(t, MemoryExtractionSeedPrompt, "EXTRACTION CRITERIA")
	assert.Contains(t, MemoryExtractionSeedPrompt, "WHAT TO EXTRACT")
	assert.Contains(t, MemoryExtractionSeedPrompt, "WHAT NOT TO EXTRACT")
	assert.Contains(t, MemoryExtractionSeedPrompt, "importance_scores")
}

func TestMemoryExtractionExample_ToPromptExample(t *testing.T) {
	example := MemoryExtractionExample{
		ConversationText:    "My favorite color is blue",
		ConversationContext: "Personal preferences",
		ExpectedFacts: []ExtractedFact{
			{Content: "User's favorite color is blue", Importance: 0.7},
		},
		Category: "personal_preferences",
	}

	promptExample := example.ToPromptExample()

	assert.Equal(t, "My favorite color is blue", promptExample.Inputs["conversation_text"])
	assert.Equal(t, "Personal preferences", promptExample.Inputs["conversation_context"])

	var facts []string
	err := json.Unmarshal([]byte(promptExample.Outputs["extracted_facts"].(string)), &facts)
	require.NoError(t, err)
	assert.Equal(t, []string{"User's favorite color is blue"}, facts)

	var scores []float64
	err = json.Unmarshal([]byte(promptExample.Outputs["importance_scores"].(string)), &scores)
	require.NoError(t, err)
	assert.Equal(t, []float64{0.7}, scores)
}

func TestMemoryExtractionMetric_PerfectMatch(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text":    "I love jazz music",
			"conversation_context": "Music discussion",
		},
		Outputs: map[string]any{
			"extracted_facts":      `["User loves jazz music"]`,
			"importance_scores":    `[0.7]`,
			"extraction_reasoning": "Extracted user's music preference",
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"extracted_facts":      `["User loves jazz music"]`,
			"importance_scores":    `[0.7]`,
			"extraction_reasoning": "User expressed preference for jazz music",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	require.NoError(t, err)
	assert.Greater(t, result.Score, 0.9, "Perfect match should score very high")
	assert.Contains(t, result.Feedback, "valid")
}

func TestMemoryExtractionMetric_EmptyExtraction(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text":    "Thanks for your help!",
			"conversation_context": "Ending conversation",
		},
		Outputs: map[string]any{
			"extracted_facts":      `[]`,
			"importance_scores":    `[]`,
			"extraction_reasoning": "No extractable facts - conversational filler",
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"extracted_facts":      `[]`,
			"importance_scores":    `[]`,
			"extraction_reasoning": "This is conversational filler with no facts",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	require.NoError(t, err)
	assert.Greater(t, result.Score, 0.9, "Correctly extracting nothing should score high")
	assert.Contains(t, result.Feedback, "Correctly extracted nothing")
}

func TestMemoryExtractionMetric_Hallucination(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text":    "That's interesting",
			"conversation_context": "Casual chat",
		},
		Outputs: map[string]any{
			"extracted_facts":   `[]`,
			"importance_scores": `[]`,
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"extracted_facts":      `["User is interested in AI", "User likes technology"]`,
			"importance_scores":    `[0.6, 0.5]`,
			"extraction_reasoning": "Extracted user interests",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	require.NoError(t, err)
	// Precision is 0 (0/2 false positives), but recall is perfect (nothing to recall)
	// 0.35*0 + 0.35*1.0 + 0.20*? + 0.10*score = 0.35 + calibration + reasoning
	// Calibration fails because facts don't match, reasoning gets partial credit
	assert.Less(t, result.Score, 0.7, "Hallucinating facts should score low")
	assert.Contains(t, result.Feedback, "HALLUCINATED")
}

func TestMemoryExtractionMetric_MissingFacts(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text":    "My daughter Emma is 5 years old and starts school next month",
			"conversation_context": "Family discussion",
		},
		Outputs: map[string]any{
			"extracted_facts":   `["User has a daughter named Emma", "Emma is 5 years old", "Emma starts school next month"]`,
			"importance_scores": `[0.9, 0.8, 0.7]`,
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"extracted_facts":      `["User has a daughter named Emma"]`,
			"importance_scores":    `[0.9]`,
			"extraction_reasoning": "Extracted daughter's name",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	require.NoError(t, err)
	// Precision: 1.0 (1/1), Recall: 0.33 (1/3), Calibration: good, Reasoning: ok
	// 0.35*1.0 + 0.35*0.33 + 0.20*1.0 + 0.10*0.7 = 0.35 + 0.115 + 0.20 + 0.07 = 0.735
	assert.Less(t, result.Score, 0.8, "Missing important facts should reduce score")
	assert.Contains(t, result.Feedback, "MISSED")
}

func TestMemoryExtractionMetric_ImportanceCalibration(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text":    "I'm allergic to peanuts",
			"conversation_context": "Health info",
		},
		Outputs: map[string]any{
			"extracted_facts":   `["User is allergic to peanuts"]`,
			"importance_scores": `[1.0]`,
		},
	}

	t.Run("correct importance", func(t *testing.T) {
		pred := prompt.Example{
			Inputs: gold.Inputs,
			Outputs: map[string]any{
				"extracted_facts":      `["User is allergic to peanuts"]`,
				"importance_scores":    `[1.0]`,
				"extraction_reasoning": "Critical health information",
			},
		}

		result, err := metric.Score(context.Background(), gold, pred, nil)
		require.NoError(t, err)
		assert.Greater(t, result.Score, 0.9)
	})

	t.Run("incorrect importance - too low", func(t *testing.T) {
		pred := prompt.Example{
			Inputs: gold.Inputs,
			Outputs: map[string]any{
				"extracted_facts":      `["User is allergic to peanuts"]`,
				"importance_scores":    `[0.3]`,
				"extraction_reasoning": "Mentioned allergy",
			},
		}

		result, err := metric.Score(context.Background(), gold, pred, nil)
		require.NoError(t, err)
		assert.Less(t, result.Score, 0.9, "Miscalibrated importance should reduce score")
		assert.Contains(t, result.Feedback, "CALIBRATION")
	})

	t.Run("invalid importance score", func(t *testing.T) {
		pred := prompt.Example{
			Inputs: gold.Inputs,
			Outputs: map[string]any{
				"extracted_facts":      `["User is allergic to peanuts"]`,
				"importance_scores":    `[1.5]`,
				"extraction_reasoning": "Critical health info",
			},
		}

		result, err := metric.Score(context.Background(), gold, pred, nil)
		require.NoError(t, err)
		assert.Contains(t, result.Feedback, "invalid score")
	})
}

func TestMemoryExtractionMetric_ReasoningQuality(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text":    "I prefer working in the morning",
			"conversation_context": "Work habits",
		},
		Outputs: map[string]any{
			"extracted_facts":   `["User prefers working in the morning"]`,
			"importance_scores": `[0.7]`,
		},
	}

	t.Run("good reasoning", func(t *testing.T) {
		pred := prompt.Example{
			Inputs: gold.Inputs,
			Outputs: map[string]any{
				"extracted_facts":      `["User prefers working in the morning"]`,
				"importance_scores":    `[0.7]`,
				"extraction_reasoning": "Extracted important work preference because user explicitly stated morning is preferred time",
			},
		}

		result, err := metric.Score(context.Background(), gold, pred, nil)
		require.NoError(t, err)
		assert.Greater(t, result.Score, 0.9)
	})

	t.Run("no reasoning", func(t *testing.T) {
		pred := prompt.Example{
			Inputs: gold.Inputs,
			Outputs: map[string]any{
				"extracted_facts":      `["User prefers working in the morning"]`,
				"importance_scores":    `[0.7]`,
				"extraction_reasoning": "",
			},
		}

		result, err := metric.Score(context.Background(), gold, pred, nil)
		require.NoError(t, err)
		assert.Less(t, result.Score, 0.95, "Missing reasoning should reduce score slightly")
		assert.Contains(t, result.Feedback, "No reasoning provided")
	})
}

func TestMemoryExtractionMetric_MismatchedScores(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text": "Test",
		},
		Outputs: map[string]any{
			"extracted_facts":   `["Fact 1", "Fact 2"]`,
			"importance_scores": `[0.8, 0.6]`,
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"extracted_facts":      `["Fact 1", "Fact 2"]`,
			"importance_scores":    `[0.8]`, // Mismatched length
			"extraction_reasoning": "Extracted facts",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	require.NoError(t, err)
	assert.Contains(t, result.Feedback, "don't match")
}

func TestSyntheticMemoryExtractionDataset(t *testing.T) {
	trainset, valset := SyntheticMemoryExtractionDataset()

	t.Run("dataset size", func(t *testing.T) {
		assert.GreaterOrEqual(t, len(trainset), 15, "Should have at least 15 training examples")
		assert.GreaterOrEqual(t, len(valset), 5, "Should have at least 5 validation examples")
		assert.LessOrEqual(t, len(trainset), 25, "Should have at most 25 training examples")
	})

	t.Run("all examples valid", func(t *testing.T) {
		for i, ex := range trainset {
			assert.NotEmpty(t, ex.Inputs["conversation_text"], "Example %d: conversation_text should not be empty", i)
			assert.NotNil(t, ex.Outputs["extracted_facts"], "Example %d: extracted_facts should not be nil", i)
			assert.NotNil(t, ex.Outputs["importance_scores"], "Example %d: importance_scores should not be nil", i)

			// Validate JSON parsing
			var facts []string
			err := json.Unmarshal([]byte(ex.Outputs["extracted_facts"].(string)), &facts)
			assert.NoError(t, err, "Example %d: extracted_facts should be valid JSON", i)

			var scores []float64
			err = json.Unmarshal([]byte(ex.Outputs["importance_scores"].(string)), &scores)
			assert.NoError(t, err, "Example %d: importance_scores should be valid JSON", i)

			// Validate lengths match
			assert.Equal(t, len(facts), len(scores), "Example %d: facts and scores should have same length", i)

			// Validate score ranges
			for j, score := range scores {
				assert.GreaterOrEqual(t, score, 0.0, "Example %d, score %d: should be >= 0.0", i, j)
				assert.LessOrEqual(t, score, 1.0, "Example %d, score %d: should be <= 1.0", i, j)
			}
		}
	})

	t.Run("category coverage", func(t *testing.T) {
		categories := map[string]int{
			"personal_preferences":  0,
			"biographical":          0,
			"project_context":       0,
			"instructions":          0,
			"conversational_filler": 0,
			"mixed_content":         0,
			"implicit_preferences":  0,
			"temporal_info":         0,
			"dense_information":     0,
			"domain_knowledge":      0,
		}

		// We can't directly access category from prompt.Example, but we can infer from structure
		for _, ex := range trainset {
			var facts []string
			json.Unmarshal([]byte(ex.Outputs["extracted_facts"].(string)), &facts)

			if len(facts) == 0 {
				categories["conversational_filler"]++
			} else if len(facts) >= 4 {
				categories["dense_information"]++
			} else {
				categories["other"]++
			}
		}

		// Should have examples with empty extractions (conversational filler)
		assert.Greater(t, categories["conversational_filler"], 0, "Should have conversational filler examples")
		// Should have examples with dense information
		assert.Greater(t, categories["dense_information"], 0, "Should have dense information examples")
	})

	t.Run("diverse fact types", func(t *testing.T) {
		allFacts := []string{}
		for _, ex := range trainset {
			var facts []string
			json.Unmarshal([]byte(ex.Outputs["extracted_facts"].(string)), &facts)
			allFacts = append(allFacts, facts...)
		}

		// Should have diverse content
		assert.Greater(t, len(allFacts), 20, "Should have many total facts across examples")

		// Check for variety in content
		hasName := false
		hasPreference := false
		hasTechnical := false

		for _, fact := range allFacts {
			lower := strings.ToLower(fact)
			if strings.Contains(lower, "name") || strings.Contains(lower, "named") {
				hasName = true
			}
			if strings.Contains(lower, "prefer") || strings.Contains(lower, "favorite") {
				hasPreference = true
			}
			if strings.Contains(lower, "use") || strings.Contains(lower, "project") {
				hasTechnical = true
			}
		}

		assert.True(t, hasName, "Should have facts about names")
		assert.True(t, hasPreference, "Should have facts about preferences")
		assert.True(t, hasTechnical, "Should have technical facts")
	})

	t.Run("importance score distribution", func(t *testing.T) {
		allScores := []float64{}
		for _, ex := range trainset {
			var scores []float64
			json.Unmarshal([]byte(ex.Outputs["importance_scores"].(string)), &scores)
			allScores = append(allScores, scores...)
		}

		if len(allScores) > 0 {
			// Should have diversity in importance scores
			hasCritical := false  // >= 0.9
			hasImportant := false // 0.7-0.8
			hasUseful := false    // 0.5-0.6

			for _, score := range allScores {
				if score >= 0.9 {
					hasCritical = true
				}
				if score >= 0.7 && score < 0.9 {
					hasImportant = true
				}
				if score >= 0.5 && score < 0.7 {
					hasUseful = true
				}
			}

			assert.True(t, hasCritical, "Should have critical importance facts (>= 0.9)")
			assert.True(t, hasImportant, "Should have important facts (0.7-0.8)")
			assert.True(t, hasUseful, "Should have useful facts (0.5-0.6)")
		}
	})

	t.Run("validation set different from training", func(t *testing.T) {
		trainTexts := make(map[string]bool)
		for _, ex := range trainset {
			trainTexts[ex.Inputs["conversation_text"].(string)] = true
		}

		for i, ex := range valset {
			text := ex.Inputs["conversation_text"].(string)
			assert.False(t, trainTexts[text], "Validation example %d should not duplicate training text", i)
		}
	})
}

func TestMemoryExtractionMetric_CategoryGuidance(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	testCases := []struct {
		category         string
		expectedGuidance string
	}{
		{"personal_preferences", "PREFERENCES"},
		{"biographical", "BIOGRAPHICAL"},
		{"project_context", "PROJECT"},
		{"instructions", "INSTRUCTIONS"},
		{"conversational_filler", "FILLER"},
		{"mixed_content", "MIXED"},
		{"implicit_preferences", "IMPLICIT"},
		{"temporal_info", "TEMPORAL"},
		{"dense_information", "DENSE"},
	}

	for _, tc := range testCases {
		t.Run(tc.category, func(t *testing.T) {
			gold := prompt.Example{
				Inputs: map[string]any{
					"conversation_text": "Test",
					"category":          tc.category,
				},
				Outputs: map[string]any{
					"extracted_facts":   `["Expected fact"]`,
					"importance_scores": `[0.8]`,
				},
			}

			pred := prompt.Example{
				Inputs: gold.Inputs,
				Outputs: map[string]any{
					"extracted_facts":      `["Wrong fact"]`,
					"importance_scores":    `[0.5]`,
					"extraction_reasoning": "test",
				},
			}

			result, err := metric.Score(context.Background(), gold, pred, nil)
			require.NoError(t, err)
			assert.Contains(t, result.Feedback, tc.expectedGuidance,
				"Category %s should include guidance keyword %s", tc.category, tc.expectedGuidance)
		})
	}
}

func TestMemoryExtractionMetric_StringMatching(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text": "Test",
		},
		Outputs: map[string]any{
			"extracted_facts":   `["User's favorite color is blue"]`,
			"importance_scores": `[0.7]`,
		},
	}

	t.Run("exact match", func(t *testing.T) {
		pred := prompt.Example{
			Inputs: gold.Inputs,
			Outputs: map[string]any{
				"extracted_facts":      `["User's favorite color is blue"]`,
				"importance_scores":    `[0.7]`,
				"extraction_reasoning": "test",
			},
		}

		result, err := metric.Score(context.Background(), gold, pred, nil)
		require.NoError(t, err)
		assert.Greater(t, result.Score, 0.9)
	})

	t.Run("semantically similar", func(t *testing.T) {
		pred := prompt.Example{
			Inputs: gold.Inputs,
			Outputs: map[string]any{
				"extracted_facts":      `["User prefers the color blue"]`,
				"importance_scores":    `[0.7]`,
				"extraction_reasoning": "test",
			},
		}

		result, err := metric.Score(context.Background(), gold, pred, nil)
		require.NoError(t, err)
		// String similarity matching with threshold 0.6
		// "User's favorite color is blue" vs "User prefers the color blue"
		// Jaccard similarity is moderate (shares "user", "color", "blue")
		// But not high enough to match. Score will be low due to no match.
		assert.Greater(t, result.Score, 0.2, "Some partial credit even without match due to other components")
	})

	t.Run("completely different", func(t *testing.T) {
		pred := prompt.Example{
			Inputs: gold.Inputs,
			Outputs: map[string]any{
				"extracted_facts":      `["User lives in Seattle"]`,
				"importance_scores":    `[0.8]`,
				"extraction_reasoning": "test",
			},
		}

		result, err := metric.Score(context.Background(), gold, pred, nil)
		require.NoError(t, err)
		assert.Less(t, result.Score, 0.7, "Different facts should score lower")
	})
}

func TestParseFloatArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []float64
	}{
		{"valid array", `[0.5, 0.7, 1.0]`, []float64{0.5, 0.7, 1.0}},
		{"empty array", `[]`, []float64{}},
		{"null", `null`, []float64{}},
		{"empty string", ``, []float64{}},
		{"single value", `[0.8]`, []float64{0.8}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFloatArray(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseStringArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"valid array", `["fact1", "fact2"]`, []string{"fact1", "fact2"}},
		{"empty array", `[]`, []string{}},
		{"null", `null`, []string{}},
		{"empty string", ``, []string{}},
		{"single value", `["fact"]`, []string{"fact"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseStringArray(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleStringSimilarity(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		minScore float64
	}{
		{"User loves jazz", "User loves jazz", 1.0},
		{"User loves jazz music", "User enjoys jazz music", 0.6},
		{"User's favorite color is blue", "User prefers blue color", 0.25},
		{"Completely different", "Nothing alike", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.a+" vs "+tt.b, func(t *testing.T) {
			score := simpleStringSimilarity(tt.a, tt.b)
			assert.GreaterOrEqual(t, score, tt.minScore)
		})
	}
}

func TestTruncateFacts(t *testing.T) {
	facts := []string{
		"This is a very long fact that should be truncated to a reasonable length",
		"Another long fact with lots of detail",
		"Short fact",
	}

	result := truncateFacts(facts, 2)
	assert.Equal(t, 2, len(result))
	assert.LessOrEqual(t, len(result[0]), 53) // 50 chars + "..."
}
