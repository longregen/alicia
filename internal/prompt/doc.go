// Package prompt provides DSPy integration for Alicia, enabling prompt optimization
// using GEPA (Genetic-Pareto) optimization and other advanced techniques.
//
// This package wraps the dspy-go library with Alicia-specific abstractions and
// integrates with Alicia's existing services (LLM, embeddings, tools, memory).
//
// # Core Components
//
// Signature: Declarative input/output specifications for LLM modules
//
//	sig := prompt.MustParseSignature("question -> answer")
//	sig := prompt.ConversationResponse // Predefined signature
//
// Modules: Composable LLM building blocks with tracing and metrics
//
//	predict := prompt.NewAliciaPredict(sig,
//	    prompt.WithTracer(tracer),
//	    prompt.WithMetrics(metrics))
//	outputs, err := predict.Process(ctx, inputs)
//
// Metrics: Evaluation functions for optimization feedback
//
//	metric := &prompt.ExactMatchMetric{}
//	metric := prompt.NewSemanticSimilarityMetric(embedService, 0.8)
//	metric := prompt.NewLLMJudgeMetric(llmService, "helpfulness, accuracy")
//
// Dimensions: GEPA's 7 optimization dimensions for fine-grained control
//
//	weights := prompt.DefaultWeights()
//	weights.Efficiency = 0.3  // Prioritize speed
//	weights.Normalize()
//
// # Integration with Alicia Services
//
// LLM Service Adapter:
//
//	adapter := prompt.NewLLMServiceAdapter(llmService)
//	// adapter implements core.LLM interface
//
// # Usage Example
//
//	// 1. Define signature
//	sig := prompt.MustParseSignature("context, question -> answer, reasoning")
//
//	// 2. Create module
//	predict := prompt.NewAliciaPredict(sig)
//
//	// 3. Execute
//	inputs := map[string]any{
//	    "context": "Alicia is an AI assistant...",
//	    "question": "What are your capabilities?",
//	}
//	outputs, err := predict.Process(ctx, inputs)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	answer := outputs["answer"].(string)
//
// # Baseline Prompts
//
// The baselines subpackage provides hand-written initial prompts for common tasks:
//
//   - ConversationResponsePrompt: General conversational responses
//   - ToolSelectionPrompt: Tool selection decisions
//   - MemoryExtractionPrompt: Memory extraction from conversations
//   - ToolResultFormatterPrompt: Tool result formatting
//
// These serve as starting points for GEPA optimization.
//
// # Next Steps
//
// Phase 2 will add:
//   - OptimizationService for running GEPA
//   - Repository interfaces for storing optimization results
//   - Database schema for optimization data
//   - Configuration support for multi-LLM setups
//
// See docs/dspy-gepa-implementation-plan.md for the full roadmap.
package prompt
