# DSPy + GEPA Implementation Plan for Alicia

## Executive Summary

This document outlines a plan to integrate DSPy-like prompt programming and GEPA optimization into Alicia. Rather than reimplementing from scratch, we recommend leveraging existing Go implementations (particularly `dspy-go`) and focusing on Alicia-specific integration patterns.

## Background

### What is DSPy?

DSPy (Declarative Self-improving Python) is Stanford NLP's framework for **programming language models** rather than prompting them. Key concepts:

1. **Signatures**: Declarative input/output specifications
   ```python
   "question -> answer"
   "context, question -> reasoning, answer"
   ```

2. **Modules**: Composable LLM building blocks
   - `Predict` - Basic LLM call
   - `ChainOfThought` - Adds reasoning steps
   - `ReAct` - Tool-using agent
   - Custom modules via composition

3. **Optimizers**: Automatic prompt improvement algorithms
   - `LabeledFewShot` - Use labeled examples
   - `BootstrapFewShot` - Generate demonstrations
   - `MIPROv2` - Joint instruction + example optimization
   - `GEPA` - State-of-the-art reflective evolution

### What is GEPA?

GEPA (Genetic-Pareto) is DSPy's most advanced optimizer:

- **Reflective Mutation**: Uses a strong LLM to analyze failures and propose improved instructions
- **Pareto Frontier**: Maintains diverse candidate pool (generalists + specialists)
- **Coverage-based Selection**: Samples candidates proportional to instances they solve best
- **Performance**: Outperforms RL by 20%, uses 35x fewer evaluations

**Algorithm Flow:**
```
1. Initialize with unoptimized program
2. Sample candidate from Pareto frontier (prob ∝ coverage)
3. Run on minibatch, collect traces + feedback
4. Reflect and propose new instruction
5. If improved, evaluate on full validation set
6. Update Pareto frontier
7. Repeat until budget exhausted
```

## Existing Go Implementations

### dspy-go (Recommended Foundation)

**Repository**: https://github.com/XiaoConstantine/dspy-go

Already implements:
- ✅ Core signatures and modules
- ✅ GEPA optimizer
- ✅ MIPRO optimizer
- ✅ BootstrapFewShot
- ✅ Multi-provider LLM support
- ✅ Structured outputs

### LangChainGo

**Repository**: https://github.com/tmc/langchaingo

Provides:
- Unified LLM interface (OpenAI, Anthropic, etc.)
- Embeddings and vector stores
- Chains and agents
- Already used by many Go projects

## Implementation Strategy

### Phase 1: Core Integration (Week 1-2)

#### 1.1 Add dspy-go Dependency

```bash
go get github.com/XiaoConstantine/dspy-go
```

#### 1.2 Create Prompt Abstraction Layer

New package: `internal/prompt/`

```go
// internal/prompt/signature.go
package prompt

import (
    "github.com/XiaoConstantine/dspy-go/core"
)

// Signature wraps dspy-go's signature with Alicia-specific features
type Signature struct {
    core.Signature
    Name        string
    Description string
    Version     int
}

// PredefinedSignatures for common Alicia use cases
var (
    ConversationResponse = MustParseSignature(
        "context, user_message, memories -> response",
    )

    ToolSelection = MustParseSignature(
        "user_intent, available_tools -> tool_name, reasoning",
    )

    MemoryExtraction = MustParseSignature(
        "conversation -> key_facts: list[str], importance: int",
    )
)
```

#### 1.3 Create Module Wrappers

```go
// internal/prompt/modules.go
package prompt

import (
    "context"
    "github.com/XiaoConstantine/dspy-go/modules"
)

// AliciaPredict wraps dspy-go Predict with Alicia integration
type AliciaPredict struct {
    *modules.Predict
    tracer   Tracer
    metrics  MetricsCollector
}

func NewAliciaPredict(sig Signature, opts ...Option) *AliciaPredict {
    return &AliciaPredict{
        Predict: modules.NewPredict(sig.Signature),
    }
}

func (p *AliciaPredict) Forward(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    // Pre-execution hooks
    span := p.tracer.StartSpan(ctx, "predict")
    defer span.End()

    // Execute
    outputs, err := p.Predict.Forward(ctx, inputs)

    // Post-execution metrics
    p.metrics.RecordExecution(span, inputs, outputs, err)

    return outputs, err
}
```

#### 1.4 Integrate with LLM Service

```go
// internal/prompt/llm_adapter.go
package prompt

import (
    "context"
    "github.com/XiaoConstantine/dspy-go/llms"
    "alicia/internal/ports"
)

// LLMServiceAdapter adapts Alicia's LLMService to dspy-go's LLM interface
type LLMServiceAdapter struct {
    service ports.LLMService
}

func NewLLMServiceAdapter(service ports.LLMService) *LLMServiceAdapter {
    return &LLMServiceAdapter{service: service}
}

func (a *LLMServiceAdapter) Generate(ctx context.Context, prompt string, opts ...llms.Option) (string, error) {
    messages := []ports.LLMMessage{
        {Role: "user", Content: prompt},
    }

    resp, err := a.service.Chat(ctx, messages)
    if err != nil {
        return "", err
    }

    return resp.Content, nil
}
```

### Phase 2: GEPA Optimization Service (Week 3-4)

#### 2.1 Define Optimization Repository

```go
// internal/ports/repositories.go (additions)

type PromptOptimizationRepository interface {
    // Optimization runs
    CreateRun(ctx context.Context, run *OptimizationRun) error
    GetRun(ctx context.Context, id string) (*OptimizationRun, error)
    UpdateRun(ctx context.Context, run *OptimizationRun) error

    // Candidates
    SaveCandidate(ctx context.Context, runID string, candidate *PromptCandidate) error
    GetCandidates(ctx context.Context, runID string) ([]*PromptCandidate, error)
    GetBestCandidate(ctx context.Context, runID string) (*PromptCandidate, error)

    // Evaluations
    SaveEvaluation(ctx context.Context, eval *PromptEvaluation) error
    GetEvaluations(ctx context.Context, candidateID string) ([]*PromptEvaluation, error)
}

type OptimizationRun struct {
    ID            string
    SignatureName string
    Status        OptimizationStatus
    Config        GEPAConfig
    BestScore     float64
    Iterations    int
    CreatedAt     time.Time
    CompletedAt   *time.Time
}

type PromptCandidate struct {
    ID           string
    RunID        string
    Instructions string
    Demos        []Example
    Coverage     int
    AvgScore     float64
    Generation   int
    ParentID     *string
    CreatedAt    time.Time
}

type PromptEvaluation struct {
    ID          string
    CandidateID string
    ExampleID   string
    Score       float64
    Feedback    string
    Trace       json.RawMessage
    CreatedAt   time.Time
}
```

#### 2.2 Create Optimization Service

```go
// internal/application/services/optimization.go
package services

import (
    "context"
    "github.com/XiaoConstantine/dspy-go/optimizers"
    "alicia/internal/ports"
    "alicia/internal/prompt"
)

type OptimizationService struct {
    repo         ports.PromptOptimizationRepository
    llmService   ports.LLMService
    reflectionLM ports.LLMService // Strong model for GEPA reflection
}

func NewOptimizationService(
    repo ports.PromptOptimizationRepository,
    llmService ports.LLMService,
    reflectionLM ports.LLMService,
) *OptimizationService {
    return &OptimizationService{
        repo:         repo,
        llmService:   llmService,
        reflectionLM: reflectionLM,
    }
}

// OptimizeSignature runs GEPA optimization on a signature
func (s *OptimizationService) OptimizeSignature(
    ctx context.Context,
    sig prompt.Signature,
    trainset []prompt.Example,
    valset []prompt.Example,
    metric prompt.Metric,
    config GEPAConfig,
) (*OptimizedProgram, error) {
    // Create optimization run record
    run := &ports.OptimizationRun{
        ID:            generateID(),
        SignatureName: sig.Name,
        Status:        StatusRunning,
        Config:        config,
    }
    if err := s.repo.CreateRun(ctx, run); err != nil {
        return nil, err
    }

    // Create dspy-go optimizer
    gepa := optimizers.NewGEPA(
        metric.ToGEPAMetric(),
        optimizers.WithAuto(config.Budget),
        optimizers.WithNumThreads(config.NumThreads),
        optimizers.WithReflectionLM(prompt.NewLLMServiceAdapter(s.reflectionLM)),
    )

    // Create student program
    student := prompt.NewAliciaPredict(sig)

    // Run optimization with progress tracking
    optimized, err := gepa.Compile(
        ctx,
        student,
        trainset,
        valset,
        optimizers.WithProgressCallback(func(progress Progress) {
            s.recordProgress(ctx, run.ID, progress)
        }),
    )
    if err != nil {
        run.Status = StatusFailed
        s.repo.UpdateRun(ctx, run)
        return nil, err
    }

    // Save result
    run.Status = StatusCompleted
    run.BestScore = optimized.BestScore()
    run.CompletedAt = timePtr(time.Now())
    s.repo.UpdateRun(ctx, run)

    return optimized, nil
}
```

### Phase 3: Memory-Aware Optimization (Week 5-6)

Leverage Alicia's existing memory system for in-context learning:

```go
// internal/prompt/memory_integration.go
package prompt

import (
    "context"
    "alicia/internal/ports"
)

// MemoryAwareModule injects relevant memories as demonstrations
type MemoryAwareModule struct {
    base          Module
    memoryService ports.MemoryService
    embedService  ports.EmbeddingService
    maxDemos      int
}

func NewMemoryAwareModule(
    base Module,
    memoryService ports.MemoryService,
    embedService ports.EmbeddingService,
) *MemoryAwareModule {
    return &MemoryAwareModule{
        base:          base,
        memoryService: memoryService,
        embedService:  embedService,
        maxDemos:      5,
    }
}

func (m *MemoryAwareModule) Forward(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    // Search for relevant memories as potential demonstrations
    query := extractQuery(inputs)
    memories, err := m.memoryService.SearchMemoriesWithRelevance(ctx, query, m.maxDemos)
    if err != nil {
        // Continue without memories on error
        return m.base.Forward(ctx, inputs)
    }

    // Convert memories to demonstrations
    demos := memoriesToDemos(memories)

    // Inject demonstrations into module
    enrichedInputs := inputs
    enrichedInputs["_demonstrations"] = demos

    return m.base.Forward(ctx, enrichedInputs)
}
```

### Phase 4: Evaluation & Metrics (Week 7-8)

```go
// internal/prompt/metrics.go
package prompt

import (
    "context"
)

// Metric defines an evaluation function for prompt optimization
type Metric interface {
    // Score evaluates a prediction against gold truth
    // Returns score (0-1) and optional feedback for GEPA reflection
    Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error)
}

// ScoreWithFeedback combines numeric score with textual feedback for GEPA
type ScoreWithFeedback struct {
    Score    float64
    Feedback string
}

// Common metrics
type ExactMatchMetric struct{}

func (m *ExactMatchMetric) Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error) {
    expected := gold.Outputs["answer"]
    actual := pred.Outputs["answer"]

    if expected == actual {
        return ScoreWithFeedback{Score: 1.0, Feedback: "Correct!"}, nil
    }

    return ScoreWithFeedback{
        Score: 0.0,
        Feedback: fmt.Sprintf("Expected: %v, Got: %v", expected, actual),
    }, nil
}

// SemanticSimilarityMetric uses embeddings for soft matching
type SemanticSimilarityMetric struct {
    embedService ports.EmbeddingService
    threshold    float64
}

func (m *SemanticSimilarityMetric) Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error) {
    expected := gold.Outputs["answer"].(string)
    actual := pred.Outputs["answer"].(string)

    // Get embeddings
    embeddings, err := m.embedService.EmbedTexts(ctx, []string{expected, actual})
    if err != nil {
        return ScoreWithFeedback{}, err
    }

    // Calculate cosine similarity
    similarity := cosineSimilarity(embeddings[0], embeddings[1])

    feedback := fmt.Sprintf(
        "Semantic similarity: %.2f\nExpected: %s\nActual: %s",
        similarity, expected, actual,
    )

    return ScoreWithFeedback{Score: similarity, Feedback: feedback}, nil
}

// LLMJudgeMetric uses an LLM to evaluate response quality
type LLMJudgeMetric struct {
    llmService ports.LLMService
    criteria   string
}

func (m *LLMJudgeMetric) Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error) {
    prompt := fmt.Sprintf(`Evaluate this response based on: %s

Question: %v
Expected Answer: %v
Actual Response: %v

Provide a score from 0.0 to 1.0 and explain your reasoning.
Format: SCORE: X.X
REASONING: ...`,
        m.criteria,
        gold.Inputs["question"],
        gold.Outputs["answer"],
        pred.Outputs["answer"],
    )

    resp, err := m.llmService.Chat(ctx, []ports.LLMMessage{
        {Role: "user", Content: prompt},
    })
    if err != nil {
        return ScoreWithFeedback{}, err
    }

    score, reasoning := parseJudgeResponse(resp.Content)
    return ScoreWithFeedback{Score: score, Feedback: reasoning}, nil
}
```

### Phase 5: HTTP API & CLI (Week 9-10)

```go
// internal/adapters/http/handlers/optimization.go
package handlers

type OptimizationHandler struct {
    optService *services.OptimizationService
}

// POST /api/v1/optimizations
func (h *OptimizationHandler) CreateOptimization(w http.ResponseWriter, r *http.Request) {
    var req CreateOptimizationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, err)
        return
    }

    // Start optimization in background
    go func() {
        ctx := context.Background()
        h.optService.OptimizeSignature(
            ctx,
            req.Signature,
            req.TrainSet,
            req.ValSet,
            req.Metric,
            req.Config,
        )
    }()

    respondJSON(w, http.StatusAccepted, map[string]string{
        "status": "optimization_started",
    })
}

// GET /api/v1/optimizations/:id
func (h *OptimizationHandler) GetOptimization(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    run, err := h.optService.GetRun(r.Context(), id)
    if err != nil {
        respondError(w, http.StatusNotFound, err)
        return
    }
    respondJSON(w, http.StatusOK, run)
}
```

## Database Schema

```sql
-- Optimization runs
CREATE TABLE prompt_optimization_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signature_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    config JSONB NOT NULL,
    best_score FLOAT,
    iterations INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Prompt candidates (Pareto frontier)
CREATE TABLE prompt_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES prompt_optimization_runs(id),
    instructions TEXT NOT NULL,
    demos JSONB,
    coverage INT DEFAULT 0,
    avg_score FLOAT,
    generation INT NOT NULL DEFAULT 0,
    parent_id UUID REFERENCES prompt_candidates(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_candidates_run ON prompt_candidates(run_id);
CREATE INDEX idx_candidates_score ON prompt_candidates(avg_score DESC);

-- Evaluation results
CREATE TABLE prompt_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id UUID NOT NULL REFERENCES prompt_candidates(id),
    example_id VARCHAR(255) NOT NULL,
    score FLOAT NOT NULL,
    feedback TEXT,
    trace JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_evaluations_candidate ON prompt_evaluations(candidate_id);

-- Optimized programs (final results)
CREATE TABLE optimized_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES prompt_optimization_runs(id),
    signature_name VARCHAR(255) NOT NULL,
    instructions TEXT NOT NULL,
    demos JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_optimized_signature ON optimized_programs(signature_name);
```

## Configuration

```yaml
# config.yaml additions
prompt_optimization:
  enabled: true

  gepa:
    default_budget: "medium"  # light, medium, heavy
    num_threads: 8
    reflection_minibatch_size: 5
    skip_perfect_score: true
    use_merge: true

  reflection_model:
    provider: "anthropic"
    model: "claude-3-5-sonnet-20241022"

  student_model:
    provider: "openai"
    model: "gpt-4o-mini"

  storage:
    type: "postgres"
    cache_optimized_programs: true
```

## Use Cases for Alicia

### 1. Response Quality Optimization

```go
// Optimize Alicia's conversational responses
sig := prompt.MustParseSignature("context, user_message, memories -> response")

trainset := loadConversationExamples("data/conversations_train.json")
valset := loadConversationExamples("data/conversations_val.json")

metric := &prompt.LLMJudgeMetric{
    llmService: reflectionLLM,
    criteria: "helpfulness, accuracy, conversational tone, use of memories",
}

optimized, _ := optService.OptimizeSignature(ctx, sig, trainset, valset, metric, config)
```

### 2. Tool Selection Optimization

```go
// Optimize when to use which tools
sig := prompt.MustParseSignature("user_intent, available_tools -> selected_tool, reasoning")

metric := &prompt.CompositeMetric{
    Components: []prompt.WeightedMetric{
        {Metric: &prompt.ExactMatchMetric{}, Weight: 0.7},
        {Metric: &prompt.ReasoningQualityMetric{}, Weight: 0.3},
    },
}
```

### 3. Memory Extraction Optimization

```go
// Optimize what to remember from conversations
sig := prompt.MustParseSignature("conversation -> key_facts: list[str], importance: int")

metric := &prompt.RecallMetric{
    // Measures if extracted facts cover expected key points
}
```

## Testing Strategy

### Unit Tests

```go
func TestGEPAOptimization(t *testing.T) {
    // Mock LLM service
    mockLLM := &MockLLMService{
        responses: map[string]string{
            "What is 2+2?": "4",
        },
    }

    sig := prompt.MustParseSignature("question -> answer")
    student := prompt.NewAliciaPredict(sig)

    trainset := []prompt.Example{
        {Inputs: map[string]any{"question": "What is 2+2?"}, Outputs: map[string]any{"answer": "4"}},
    }

    gepa := NewGEPA(metric, WithAuto("light"))
    optimized, err := gepa.Compile(ctx, student, trainset, trainset)

    assert.NoError(t, err)
    assert.NotNil(t, optimized)
}
```

### Integration Tests

```go
func TestOptimizationServiceE2E(t *testing.T) {
    // Use test database
    db := setupTestDB(t)
    defer db.Close()

    repo := postgres.NewOptimizationRepository(db)
    optService := services.NewOptimizationService(repo, llmService, reflectionLLM)

    sig := prompt.MustParseSignature("question -> answer")
    trainset := loadTestExamples(t, "testdata/qa_train.json")
    valset := loadTestExamples(t, "testdata/qa_val.json")

    result, err := optService.OptimizeSignature(ctx, sig, trainset, valset, metric, config)

    assert.NoError(t, err)
    assert.Greater(t, result.BestScore, 0.8)
}
```

## Estimated Effort

| Phase | Duration | Description |
|-------|----------|-------------|
| Phase 1 | 2 weeks | Core integration with dspy-go |
| Phase 2 | 2 weeks | GEPA optimization service |
| Phase 3 | 2 weeks | Memory-aware optimization |
| Phase 4 | 2 weeks | Evaluation & metrics |
| Phase 5 | 2 weeks | HTTP API & CLI |
| **Total** | **10 weeks** | Full implementation |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| dspy-go API changes | Pin version, contribute upstream |
| Performance bottlenecks | Goroutines for parallel eval, caching |
| LLM rate limits | Backoff/retry, batching, caching |
| Complex metrics | Start simple, iterate based on needs |

## Next Steps

1. **Immediate**: Add dspy-go dependency, create proof-of-concept
2. **Week 1-2**: Implement Phase 1 (core integration)
3. **Week 3-4**: Implement Phase 2 (GEPA service)
4. **Ongoing**: Iterate based on evaluation results

## References

- [DSPy Documentation](https://dspy.ai/)
- [GEPA Paper](https://arxiv.org/abs/2507.19457)
- [dspy-go Repository](https://github.com/XiaoConstantine/dspy-go)
- [LangChainGo](https://github.com/tmc/langchaingo)
