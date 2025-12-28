package prompt

// FeedbackType represents different types of user feedback
type FeedbackType string

const (
	// Positive feedback
	FeedbackGreatAnswer   FeedbackType = "great_answer"
	FeedbackHelpful       FeedbackType = "helpful"
	FeedbackPerfect       FeedbackType = "perfect"

	// Negative feedback - general
	FeedbackWrongAnswer   FeedbackType = "wrong_answer"
	FeedbackNotHelpful    FeedbackType = "not_helpful"
	FeedbackMissingContext FeedbackType = "missing_context"
	FeedbackTooVerbose    FeedbackType = "too_verbose"

	// Performance feedback
	FeedbackTooSlow       FeedbackType = "too_slow"
	FeedbackInconsistent  FeedbackType = "inconsistent"

	// Creativity feedback
	FeedbackSameApproach  FeedbackType = "same_approach"
	FeedbackNotInnovative FeedbackType = "not_innovative"

	// Generalization feedback
	FeedbackDoesntFitCase FeedbackType = "doesnt_fit_case"

	// Tool-specific feedback
	FeedbackWrongTool     FeedbackType = "wrong_tool"
	FeedbackWrongParams   FeedbackType = "wrong_params"
	FeedbackUnnecessary   FeedbackType = "unnecessary"

	// Memory-specific feedback
	FeedbackNotRelevant   FeedbackType = "not_relevant"
	FeedbackCritical      FeedbackType = "critical"
	FeedbackOutdated      FeedbackType = "outdated"
	FeedbackTooGeneric    FeedbackType = "too_generic"

	// Reasoning-specific feedback
	FeedbackIncorrectAssumption FeedbackType = "incorrect_assumption"
	FeedbackMissedConsideration FeedbackType = "missed_consideration"
	FeedbackOvercomplicated     FeedbackType = "overcomplicated"
	FeedbackWrongDirection      FeedbackType = "wrong_direction"
)

// DimensionAdjustment represents changes to dimension weights based on feedback
type DimensionAdjustment struct {
	SuccessRate    float64
	Quality        float64
	Efficiency     float64
	Robustness     float64
	Generalization float64
	Diversity      float64
	Innovation     float64
}

// MapFeedbackToDimensions converts user feedback to dimension adjustments
// Positive adjustments indicate the dimension needs more focus
// Negative adjustments indicate the dimension is already well-optimized
func MapFeedbackToDimensions(feedback FeedbackType) DimensionAdjustment {
	switch feedback {
	// Positive feedback - indicates current optimization is good
	case FeedbackGreatAnswer, FeedbackHelpful:
		return DimensionAdjustment{
			Quality:     -0.05,
			SuccessRate: -0.05,
		}
	case FeedbackPerfect:
		return DimensionAdjustment{
			SuccessRate: -0.1,
			Quality:     -0.1,
		}

	// Wrong answer - focus on accuracy
	case FeedbackWrongAnswer:
		return DimensionAdjustment{
			SuccessRate: +0.15,
		}

	// Performance issues
	case FeedbackTooSlow:
		return DimensionAdjustment{
			Efficiency: +0.15,
			Quality:    -0.05,
		}
	case FeedbackTooVerbose:
		return DimensionAdjustment{
			Efficiency: +0.1,
			Quality:    -0.03,
		}

	// Consistency issues
	case FeedbackInconsistent:
		return DimensionAdjustment{
			Robustness: +0.15,
		}

	// Creativity/novelty issues
	case FeedbackSameApproach:
		return DimensionAdjustment{
			Diversity:  +0.1,
			Innovation: +0.05,
		}
	case FeedbackNotInnovative:
		return DimensionAdjustment{
			Innovation: +0.1,
			Diversity:  +0.05,
		}

	// Generalization issues
	case FeedbackDoesntFitCase:
		return DimensionAdjustment{
			Generalization: +0.15,
		}

	// Quality issues
	case FeedbackMissingContext:
		return DimensionAdjustment{
			Quality:    +0.1,
			Robustness: +0.05,
		}

	// Tool-specific feedback
	case FeedbackWrongTool:
		return DimensionAdjustment{
			SuccessRate: +0.1,
			Quality:     +0.05,
		}
	case FeedbackWrongParams:
		return DimensionAdjustment{
			SuccessRate: +0.1,
		}
	case FeedbackUnnecessary:
		return DimensionAdjustment{
			Efficiency: +0.1,
		}

	// Memory-specific feedback
	case FeedbackNotRelevant:
		return DimensionAdjustment{
			Quality:        +0.1,
			Generalization: +0.05,
		}
	case FeedbackCritical:
		return DimensionAdjustment{
			SuccessRate: -0.1,
			Robustness:  -0.05,
		}
	case FeedbackOutdated:
		return DimensionAdjustment{
			Quality: +0.1,
		}
	case FeedbackTooGeneric:
		return DimensionAdjustment{
			Quality:        +0.1,
			Generalization: -0.05,
		}

	// Reasoning-specific feedback
	case FeedbackIncorrectAssumption:
		return DimensionAdjustment{
			SuccessRate: +0.1,
			Quality:     +0.05,
		}
	case FeedbackMissedConsideration:
		return DimensionAdjustment{
			Quality:    +0.1,
			Robustness: +0.05,
		}
	case FeedbackOvercomplicated:
		return DimensionAdjustment{
			Efficiency: +0.1,
			Quality:    +0.05,
		}
	case FeedbackWrongDirection:
		return DimensionAdjustment{
			SuccessRate: +0.1,
		}

	default:
		return DimensionAdjustment{}
	}
}

// ApplyAdjustment applies a dimension adjustment to weights
func ApplyAdjustment(weights DimensionWeights, adjustment DimensionAdjustment) DimensionWeights {
	result := DimensionWeights{
		SuccessRate:    clamp(weights.SuccessRate + adjustment.SuccessRate, 0.01, 0.5),
		Quality:        clamp(weights.Quality + adjustment.Quality, 0.01, 0.5),
		Efficiency:     clamp(weights.Efficiency + adjustment.Efficiency, 0.01, 0.5),
		Robustness:     clamp(weights.Robustness + adjustment.Robustness, 0.01, 0.5),
		Generalization: clamp(weights.Generalization + adjustment.Generalization, 0.01, 0.5),
		Diversity:      clamp(weights.Diversity + adjustment.Diversity, 0.01, 0.5),
		Innovation:     clamp(weights.Innovation + adjustment.Innovation, 0.01, 0.5),
	}
	result.Normalize()
	return result
}

// AggregateFeedback aggregates multiple feedback items into a single adjustment
func AggregateFeedback(feedbacks []FeedbackType) DimensionAdjustment {
	result := DimensionAdjustment{}

	for _, fb := range feedbacks {
		adj := MapFeedbackToDimensions(fb)
		result.SuccessRate += adj.SuccessRate
		result.Quality += adj.Quality
		result.Efficiency += adj.Efficiency
		result.Robustness += adj.Robustness
		result.Generalization += adj.Generalization
		result.Diversity += adj.Diversity
		result.Innovation += adj.Innovation
	}

	// Average the adjustments
	n := float64(len(feedbacks))
	if n > 0 {
		result.SuccessRate /= n
		result.Quality /= n
		result.Efficiency /= n
		result.Robustness /= n
		result.Generalization /= n
		result.Diversity /= n
		result.Innovation /= n
	}

	return result
}

// clamp restricts a value to the given range
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// QuickFeedbackToType converts quick feedback strings to FeedbackType
func QuickFeedbackToType(quickFeedback string) FeedbackType {
	mapping := map[string]FeedbackType{
		// Tool feedback
		"wrong_tool":      FeedbackWrongTool,
		"wrong_params":    FeedbackWrongParams,
		"unnecessary":     FeedbackUnnecessary,
		"missing_context": FeedbackMissingContext,
		"perfect":         FeedbackPerfect,

		// Memory feedback
		"outdated":      FeedbackOutdated,
		"wrong_context": FeedbackNotRelevant,
		"too_generic":   FeedbackTooGeneric,
		"incorrect":     FeedbackWrongAnswer,

		// Reasoning feedback
		"incorrect_assumption":  FeedbackIncorrectAssumption,
		"missed_consideration":  FeedbackMissedConsideration,
		"overcomplicated":       FeedbackOvercomplicated,
		"wrong_direction":       FeedbackWrongDirection,
	}

	if ft, ok := mapping[quickFeedback]; ok {
		return ft
	}
	return ""
}

// VoteToFeedback converts a vote (up/down/critical) with optional quick feedback to FeedbackType
func VoteToFeedback(vote string, quickFeedback string, targetType string) FeedbackType {
	// First check if there's specific quick feedback
	if quickFeedback != "" {
		if ft := QuickFeedbackToType(quickFeedback); ft != "" {
			return ft
		}
	}

	// Otherwise, map based on vote and target type
	switch vote {
	case "up":
		switch targetType {
		case "memory":
			return FeedbackHelpful
		case "tool_use":
			return FeedbackPerfect
		default:
			return FeedbackGreatAnswer
		}
	case "down":
		switch targetType {
		case "memory":
			return FeedbackNotRelevant
		case "tool_use":
			return FeedbackWrongTool
		case "reasoning":
			return FeedbackWrongDirection
		default:
			return FeedbackWrongAnswer
		}
	case "critical":
		return FeedbackCritical
	}

	return ""
}
