# DSPy/GEPA Optimization System Architecture

This document describes the implementation of Alicia's prompt optimization system, which uses DSPy and GEPA (Genetic Evolution of Prompt Archives) for automatic prompt improvement through multi-objective optimization.

## Overview

The optimization system implements a sophisticated pipeline for improving AI prompts through:

1. **Memory-Aware Few-Shot Learning** (Phase 4) - Retrieves relevant memories to enrich training examples
2. **HTTP API & CLI** (Phase 5) - Provides programmatic and command-line access to optimization
3. **Feedback Integration & Streaming** (Phase 6) - Real-time progress updates and user-driven dimension adjustments

## Architecture

### Core Components

```
┌─────────────────────────────────────────┐
│  OptimizationService                    │
│  - Manages optimization runs            │
│  - Coordinates GEPA optimizer           │
│  - Tracks dimension weights             │
└──────────────┬──────────────────────────┘
               │
               ├──> MemoryAwareModule (Phase 4)
               │    - Retrieves relevant memories
               │    - Converts to demonstrations
               │    - Enriches few-shot examples
               │
               ├──> GEPA Optimizer
               │    - Multi-objective evolution
               │    - Pareto archive management
               │    - Adaptive selection
               │
               └──> DimensionWeights System
                    - 7-dimension scoring
                    - Feedback-driven adjustment
                    - Weight normalization
```

## Phase 4: Memory-Aware Optimization

### Implementation

**Location**: `/home/usr/projects/alicia/internal/prompt/memory_aware_module.go`

The `MemoryAwareModule` wraps the base AliciaPredict module with memory-augmented few-shot learning:

```go
type MemoryAwareModule struct {
    *AliciaPredict
    memoryService ports.MemoryService
    maxDemos      int      // Default: 5
    threshold     float32  // Default: 0.7
}
```

### How It Works

1. **Memory Retrieval**: For each optimization iteration, the module:
   - Constructs a query from input fields (user_message, context, question)
   - Searches the memory service using semantic similarity
   - Filters results by similarity threshold (default 0.7)
   - Retrieves up to `maxDemos` relevant memories

2. **Memory Ranking**: Memories are ranked using multiple signals:
   - **Similarity Score** (50%): Semantic similarity to current input
   - **Importance Score** (30%): User-assigned importance
   - **Recency Score** (20%): Exponential decay based on age
   - **Category Bonus** (+20%): Extra weight for matching tags

3. **Demonstration Conversion**: Memories are converted to DSPy examples based on their category:
   - `preference`: User preferences for approach/style
   - `fact`: Factual knowledge to incorporate
   - `instruction`: Rules or guidelines to follow
   - `context`: Background information

4. **Few-Shot Injection**: Converted demonstrations are added to the module's few-shot examples before GEPA optimization

### Key Files

- `/home/usr/projects/alicia/internal/prompt/memory_aware_module.go` - Core memory-aware module
- `/home/usr/projects/alicia/internal/prompt/memory_converter.go` - Memory-to-demonstration conversion
- `/home/usr/projects/alicia/internal/application/services/memory.go` - Memory service implementation
- `/home/usr/projects/alicia/internal/application/services/optimization.go` - `OptimizeSignatureWithMemory()` method

### Usage

```go
// Create memory-aware module
memoryModule := prompt.NewMemoryAwareModule(
    signature,
    memoryService,
    prompt.WithMaxDemonstrations(5),
    prompt.WithSimilarityThreshold(0.7),
)

// Optimize with memory enrichment
run, err := optimizationService.OptimizeSignatureWithMemory(
    ctx,
    signature,
    trainset,
    valset,
    metric,
    memoryService,
)
```

## Seven-Dimension Optimization System

### Dimensions

**Location**: `/home/usr/projects/alicia/internal/prompt/dimensions.go`

Each prompt is evaluated across seven dimensions:

1. **Success Rate** (Default: 25%) - Correctness of outputs
2. **Quality** (Default: 20%) - Depth and completeness of responses
3. **Efficiency** (Default: 15%) - Speed and token usage
4. **Robustness** (Default: 15%) - Consistency across inputs
5. **Generalization** (Default: 10%) - Performance on varied examples
6. **Diversity** (Default: 10%) - Variety in approaches
7. **Innovation** (Default: 5%) - Novel solutions

### Dimension Weights

Weights are configurable and sum to 1.0:

```go
type DimensionWeights struct {
    SuccessRate    float64 `json:"successRate"`
    Quality        float64 `json:"quality"`
    Efficiency     float64 `json:"efficiency"`
    Robustness     float64 `json:"robustness"`
    Generalization float64 `json:"generalization"`
    Diversity      float64 `json:"diversity"`
    Innovation     float64 `json:"innovation"`
}
```

The `DimensionScores` type holds actual scores for each dimension, and the weighted score is calculated as:

```go
score := scores.WeightedScore(weights)
// = successRate*w1 + quality*w2 + efficiency*w3 + ...
```

## Pareto Archive Management

**Location**: `/home/usr/projects/alicia/internal/prompt/pareto.go`

The Pareto archive maintains a collection of non-dominated solutions representing different trade-offs:

### Key Concepts

- **Elite Solution**: A prompt candidate with its multi-dimensional scores
- **Dominance**: Solution A dominates B if A is at least as good in all dimensions and strictly better in at least one
- **Pareto Front**: Set of all non-dominated solutions

### Operations

```go
archive := prompt.NewParetoArchive(50) // Max 50 solutions

// Add candidate (returns true if non-dominated)
added := archive.Add(solution)

// Select best for given weights
best := archive.SelectByWeights(weights)

// Select proportional to coverage (GEPA strategy)
selected := archive.SelectByCoverage()
```

### Diversity Pruning

When the archive exceeds max size, crowding distance is used to maintain diversity:

```go
func calculateCrowdingDistances(solutions []*EliteSolution) []float64
```

This implements NSGA-II style crowding distance calculation to prefer solutions in less-crowded regions of the objective space.

## Phase 6: Feedback Integration

### Feedback Mapping System

**Location**: `/home/usr/projects/alicia/internal/prompt/feedback_mapping.go`

User votes are mapped to dimension adjustments:

```go
type FeedbackType string

const (
    FeedbackGreatAnswer   = "great_answer"
    FeedbackWrongAnswer   = "wrong_answer"
    FeedbackTooVerbose    = "too_verbose"
    FeedbackInconsistent  = "inconsistent"
    // ... 20+ feedback types
)
```

Each feedback type maps to specific dimension adjustments:

| Feedback | Adjustments |
|----------|-------------|
| `wrong_answer` | `+0.15 successRate` |
| `too_verbose` | `+0.1 efficiency, -0.03 quality` |
| `too_slow` | `+0.15 efficiency, -0.05 quality` |
| `inconsistent` | `+0.15 robustness` |
| `same_approach` | `+0.1 diversity, +0.05 innovation` |
| `great_answer` | `-0.05 quality, -0.05 successRate` (reduce focus) |

### Frontend Vote Types

**Location**: `/home/usr/projects/alicia/frontend/src/stores/feedbackStore.ts`

```typescript
type VoteType = 'up' | 'down' | 'critical';
type VotableType = 'message' | 'tool_use' | 'memory' | 'reasoning';
```

Votes include optional quick feedback tags:
- `too_verbose`, `too_slow`, `wrong_params`, `not_relevant`, etc.

### Feedback API

**Location**: `/home/usr/projects/alicia/internal/adapters/http/handlers/feedback.go`

#### POST /api/v1/feedback
Submits user feedback and adjusts dimension weights.

**Request**:
```json
{
  "target_type": "message",
  "target_id": "msg_123",
  "vote": "down",
  "quick_feedback": "too_verbose"
}
```

**Response**:
```json
{
  "feedback_type": "too_verbose",
  "adjustment": {
    "efficiency": 0.1,
    "quality": -0.03
  },
  "new_weights": {
    "successRate": 0.25,
    "quality": 0.17,
    "efficiency": 0.25,
    ...
  }
}
```

#### GET /api/v1/feedback/dimensions
Returns current dimension weights.

#### PUT /api/v1/feedback/dimensions
Admin endpoint to manually set dimension weights.

### Adjustment Flow

```
User Vote (down + "too_verbose")
    ↓
VoteToFeedback() → FeedbackTooVerbose
    ↓
MapFeedbackToDimensions() → {efficiency: +0.1, quality: -0.03}
    ↓
ApplyAdjustment() → Update current weights
    ↓
Normalize() → Ensure weights sum to 1.0
    ↓
OptimizationService.ApplyFeedbackToWeights()
    ↓
Next optimization run uses updated weights
```

## Streaming Progress Updates

**Location**: `/home/usr/projects/alicia/internal/adapters/http/handlers/optimization_stream.go`

### SSE Stream Endpoint

#### GET /api/v1/optimizations/{id}/stream
Server-Sent Events stream for real-time optimization progress.

**Event Types**:

1. **Connected Event**:
```json
{
  "type": "connected",
  "run_id": "aor_abc123",
  "status": "running",
  "timestamp": "2025-12-30T10:00:00Z"
}
```

2. **Progress Event**:
```json
{
  "type": "progress",
  "run_id": "aor_abc123",
  "iteration": 15,
  "max_iterations": 100,
  "current_score": 0.78,
  "best_score": 0.82,
  "dimension_scores": {
    "successRate": 0.85,
    "quality": 0.80,
    "efficiency": 0.75,
    ...
  },
  "status": "running"
}
```

3. **Completed Event**:
```json
{
  "type": "completed",
  "run_id": "aor_abc123",
  "iteration": 100,
  "best_score": 0.89,
  "dimension_scores": { ... },
  "status": "completed"
}
```

### Frontend Integration

```typescript
const eventSource = new EventSource(`/api/v1/optimizations/${runId}/stream`);

eventSource.addEventListener('message', (event) => {
  const data = JSON.parse(event.data);

  switch (data.type) {
    case 'progress':
      dimensionStore.updateScores(data.dimension_scores);
      break;
    case 'completed':
      eventSource.close();
      break;
  }
});
```

## Optimization Service API

**Location**: `/home/usr/projects/alicia/internal/application/services/optimization.go`

### Key Methods

#### StartOptimizationRun
```go
func (s *OptimizationService) StartOptimizationRun(
    ctx context.Context,
    name string,
    promptType string,
    baselinePrompt string,
) (*models.OptimizationRun, error)
```

Creates a new optimization run with configuration.

#### OptimizeSignature
```go
func (s *OptimizationService) OptimizeSignature(
    ctx context.Context,
    sig prompt.Signature,
    trainset []prompt.Example,
    valset []prompt.Example,
    metric prompt.Metric,
) (*models.OptimizationRun, error)
```

Runs GEPA optimization on a signature. Executes asynchronously in a goroutine.

#### OptimizeSignatureWithMemory
```go
func (s *OptimizationService) OptimizeSignatureWithMemory(
    ctx context.Context,
    sig prompt.Signature,
    trainset []prompt.Example,
    valset []prompt.Example,
    metric prompt.Metric,
    memoryService ports.MemoryService,
    memoryOptions ...prompt.MemoryAwareOption,
) (*models.OptimizationRun, error)
```

Memory-aware variant that enriches training with retrieved memories.

#### RecordEvaluationWithDimensions
```go
func (s *OptimizationService) RecordEvaluationWithDimensions(
    ctx context.Context,
    candidateID string,
    runID string,
    input string,
    output string,
    dimScores prompt.DimensionScores,
    success bool,
    latencyMs int64,
) (*models.PromptEvaluation, error)
```

Records evaluation with per-dimension scores.

### Configuration

```go
type OptimizationConfig struct {
    MaxIterations     int                      // Default: 100
    MinibatchSize     int                      // Default: 5
    SkipPerfectScore  bool                     // Default: true
    ParetoArchiveSize int                      // Default: 50
    DimensionWeights  prompt.DimensionWeights  // Configurable
}
```

### GEPA Configuration

The service maps its config to GEPA parameters:

```go
gepaConfig := &optimizers.GEPAConfig{
    MaxGenerations:       maxIterations / 10,
    PopulationSize:       20,
    MutationRate:         0.3,
    CrossoverRate:        0.7,
    ElitismRate:          0.1,
    ReflectionFreq:       2,
    ReflectionDepth:      3,
    SelectionStrategy:    "adaptive_pareto",
    EvaluationBatchSize:  minibatchSize,
    ConcurrencyLevel:     3,
}
```

## HTTP API Summary

### Optimization Endpoints

**Location**: `/home/usr/projects/alicia/internal/adapters/http/handlers/optimization.go`

- `POST /api/v1/optimizations` - Start optimization run
- `GET /api/v1/optimizations/{id}` - Get run status
- `GET /api/v1/optimizations` - List runs with filtering
- `GET /api/v1/optimizations/{id}/candidates` - Get all candidates
- `GET /api/v1/optimizations/{id}/best` - Get best candidate

### Feedback Endpoints

- `POST /api/v1/feedback` - Submit feedback, adjust weights
- `GET /api/v1/feedback/dimensions` - Get current weights
- `PUT /api/v1/feedback/dimensions` - Set weights (admin)

### Streaming Endpoints

- `GET /api/v1/optimizations/{id}/stream` - SSE progress stream

### Deployment Endpoints

**Location**: `/home/usr/projects/alicia/internal/adapters/http/handlers/deployment.go`

- `POST /api/v1/deployments` - Deploy optimized prompt
- `GET /api/v1/deployments/{type}/active` - Get active deployment
- `GET /api/v1/deployments/{type}/history` - Deployment history
- `DELETE /api/v1/deployments/{id}` - Rollback deployment

## Database Schema

Optimization data is stored in PostgreSQL:

### optimization_runs
```sql
CREATE TABLE optimization_runs (
    id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL,
    prompt_type VARCHAR NOT NULL,
    status VARCHAR NOT NULL,
    max_iterations INT NOT NULL,
    iterations INT DEFAULT 0,
    best_score FLOAT DEFAULT 0,
    best_dim_scores JSONB,
    dimension_weights JSONB,
    config JSONB,
    meta JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);
```

### prompt_candidates
```sql
CREATE TABLE prompt_candidates (
    id VARCHAR PRIMARY KEY,
    run_id VARCHAR REFERENCES optimization_runs(id),
    iteration INT NOT NULL,
    prompt_text TEXT NOT NULL,
    avg_score FLOAT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### prompt_evaluations
```sql
CREATE TABLE prompt_evaluations (
    id VARCHAR PRIMARY KEY,
    candidate_id VARCHAR REFERENCES prompt_candidates(id),
    run_id VARCHAR REFERENCES optimization_runs(id),
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    score FLOAT NOT NULL,
    dimension_scores JSONB,
    success BOOLEAN NOT NULL,
    latency_ms BIGINT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Testing

### Unit Tests

Key test files:
- `/home/usr/projects/alicia/internal/prompt/memory_aware_module_test.go`
- `/home/usr/projects/alicia/internal/prompt/pareto_test.go`
- `/home/usr/projects/alicia/internal/prompt/dimensions_test.go`
- `/home/usr/projects/alicia/internal/prompt/feedback_mapping_test.go`
- `/home/usr/projects/alicia/internal/application/services/optimization_test.go`

### Integration Tests

Example test workflow:
```go
// 1. Start optimization
run, err := service.StartOptimizationRun(ctx, "test", "conversation", "")

// 2. Add candidates and evaluations
candidate, err := service.AddCandidate(ctx, run.ID, promptText, 1)
eval, err := service.RecordEvaluationWithDimensions(
    ctx, candidate.ID, run.ID, input, output, dimScores, true, 100,
)

// 3. Apply feedback
service.ApplyFeedbackToWeights(prompt.FeedbackTooVerbose)

// 4. Get updated weights
weights := service.GetDimensionWeights()

// 5. Complete run
err = service.CompleteRun(ctx, run.ID, bestScore)
```

## Extension Points

### Adding New Feedback Types

1. Define new `FeedbackType` constant in `feedback_mapping.go`
2. Add mapping in `MapFeedbackToDimensions()` function
3. Add to `VoteToFeedback()` mapping if user-facing
4. Update frontend quick feedback options

### Custom Memory Ranking

Implement custom `MemoryRelevanceScore` calculation:

```go
func RankMemoriesByRelevance(
    memories []*ports.MemorySearchResult,
    categoryFilter []string,
) []MemoryRelevanceScore {
    // Custom ranking logic
}
```

### Custom Metrics

Implement the `prompt.Metric` interface:

```go
type CustomMetric struct{}

func (m *CustomMetric) Score(
    ctx context.Context,
    expected prompt.Example,
    predicted prompt.Example,
    trace map[string]any,
) (prompt.ScoreWithFeedback, error) {
    // Calculate custom score and dimension breakdown
}
```

## Related Documentation

- `/home/usr/projects/alicia/docs/PHASE_6_INTEGRATION.md` - Detailed Phase 6 integration guide
- `/home/usr/projects/alicia/docs/DATABASE.md` - Database schema details
- `/home/usr/projects/alicia/docs/ARCHITECTURE.md` - Overall system architecture

## See Also

- [GEPA Primer](GEPA_PRIMER.md) - Detailed GEPA algorithm explanation
- [Database Schema](DATABASE.md) - Optimization tables
