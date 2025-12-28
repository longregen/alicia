# internal/prompt

DSPy integration package for Alicia, enabling prompt optimization using GEPA (Genetic-Pareto) optimization.

## Overview

This package wraps [dspy-go](https://github.com/XiaoConstantine/dspy-go) with Alicia-specific abstractions and integrates with existing Alicia services.

## Components

### Signatures (`signature.go`)

Declarative specifications of module inputs/outputs:

```go
// Parse from string
sig := prompt.MustParseSignature("question -> answer")

// Use predefined
sig := prompt.ConversationResponse
sig := prompt.ToolSelection
sig := prompt.MemoryExtraction
```

### Modules (`modules.go`)

Executable LLM components with tracing and metrics:

```go
predict := prompt.NewAliciaPredict(sig,
    prompt.WithTracer(tracer),
    prompt.WithMetrics(collector))

outputs, err := predict.Process(ctx, map[string]any{
    "question": "What is 2+2?",
})
```

### LLM Adapter (`llm_adapter.go`)

Adapts Alicia's `LLMService` to dspy-go's `core.LLM` interface:

```go
adapter := prompt.NewLLMServiceAdapter(llmService)
// Can now be used by dspy-go modules
```

### Metrics (`metrics.go`)

Evaluation functions for optimization:

```go
// Exact match
metric := &prompt.ExactMatchMetric{}

// Semantic similarity
metric := prompt.NewSemanticSimilarityMetric(embedService, 0.8)

// LLM-based evaluation
metric := prompt.NewLLMJudgeMetric(llmService, "helpfulness, accuracy")
```

### Dimensions (`dimensions.go`)

GEPA's 7 optimization dimensions:

```go
weights := prompt.DefaultWeights()
// Adjust priorities
weights.Efficiency = 0.3  // Prioritize speed
weights.Quality = 0.25    // Balance quality
weights.Normalize()       // Ensure sum = 1.0

// Calculate weighted score
score := dimensionScores.WeightedScore(weights)
```

The 7 dimensions are:
1. **Success Rate** (Accuracy) - How often correct results are produced
2. **Quality** - Overall output quality and coherence
3. **Efficiency** (Speed) - Resource usage and latency
4. **Robustness** (Reliability) - Stability across diverse inputs
5. **Generalization** (Adaptability) - Performance on new/unseen cases
6. **Diversity** (Creativity) - Variety in solution approaches
7. **Innovation** (Novelty) - Novel problem-solving strategies

### Baselines (`baselines/`)

Hand-written initial prompts for optimization:

```go
import "github.com/longregen/alicia/internal/prompt/baselines"

prompt := baselines.ConversationResponsePrompt
prompt := baselines.ToolSelectionPrompt
prompt := baselines.MemoryExtractionPrompt
```

## Usage Example

```go
package main

import (
    "context"
    "log"

    "github.com/longregen/alicia/internal/prompt"
)

func main() {
    ctx := context.Background()

    // 1. Create signature
    sig := prompt.MustParseSignature("context, question -> answer, reasoning")

    // 2. Create module
    predict := prompt.NewAliciaPredict(sig)

    // 3. Execute
    outputs, err := predict.Process(ctx, map[string]any{
        "context": "Alicia is an AI assistant with memory capabilities.",
        "question": "What are your main features?",
    })
    if err != nil {
        log.Fatal(err)
    }

    answer := outputs["answer"].(string)
    reasoning := outputs["reasoning"].(string)

    log.Printf("Answer: %s\nReasoning: %s", answer, reasoning)
}
```

## Testing

Run tests:

```bash
go test ./internal/prompt/...
```

All tests should pass:
- Signature parsing
- Dimension calculations
- Metric implementations

## Dependencies

- `github.com/XiaoConstantine/dspy-go v0.74.0` - DSPy implementation for Go
- Alicia's `internal/ports` interfaces

## Phase 1 Status

✅ **Complete** - Foundation ready for Phase 2

### What's Included

- [x] Signature abstractions
- [x] Module wrappers
- [x] LLM service adapter
- [x] Evaluation metrics
- [x] GEPA dimensions
- [x] Baseline prompts
- [x] Test suite
- [x] Package documentation

### What's Next (Phase 2)

- [ ] OptimizationService
- [ ] Repository interfaces
- [ ] Database schema
- [ ] GEPA integration
- [ ] Multi-LLM configuration

## References

- [Implementation Plan](../../docs/dspy-gepa-implementation-plan.md)
- [DSPy Documentation](https://dspy.ai/)
- [GEPA Paper](https://arxiv.org/abs/2507.19457)
- [dspy-go Repository](https://github.com/XiaoConstantine/dspy-go)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Alicia Application                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   ┌──────────────┐    ┌──────────────┐                 │
│   │   LLMService │    │ToolService   │                 │
│   └──────┬───────┘    └──────────────┘                 │
│          │                                               │
│          │ adapted via                                   │
│          ▼                                               │
│   ┌──────────────────┐       internal/prompt/           │
│   │LLMServiceAdapter │                                   │
│   └──────┬───────────┘                                   │
│          │                                               │
│          │ implements core.LLM                           │
│          ▼                                               │
│   ┌────────────────────────────────────┐                │
│   │        dspy-go (pkg/*)             │                │
│   │  ┌──────────┐   ┌──────────────┐  │                │
│   │  │ Modules  │   │  Optimizers  │  │                │
│   │  │ - Predict│   │  - GEPA      │  │                │
│   │  │ - CoT    │   │  - MIPRO     │  │                │
│   │  └──────────┘   └──────────────┘  │                │
│   └────────────────────────────────────┘                │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## License

Same as Alicia project.
