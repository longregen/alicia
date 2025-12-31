# GEPA: The Language-First Evolution of AI Prompts

**GEPA (Genetic-Pareto)** represents a fundamental paradigm shift in prompt optimization—replacing scalar reward signals with natural language reflection to achieve **10%+ improvements over reinforcement learning while using 35× fewer rollouts**. This reflective optimizer diagnoses failures, proposes targeted fixes, and maintains solution diversity through Pareto-based selection, all without requiring gradient access to model weights.

The core insight: LLMs learn more effectively by reflecting on their behavior in natural language than from sparse, scalar policy gradients. GEPA exploits this by treating language itself as the optimization medium, creating interpretable prompt evolution that humans can audit and understand.

---

## Part 1: GEPA deep dive

### The fundamental principle of reflective evolution

GEPA optimizes prompts through a three-phase loop: **execute → reflect → mutate**. Unlike traditional optimizers that learn from scalar rewards, GEPA captures full execution traces—reasoning chains, tool calls, outputs, and errors—then uses an LLM to diagnose problems in natural language and propose targeted improvements.

The algorithm maintains a population of candidate prompts and iteratively improves them through genetic operations guided by reflection. A candidate "wins" on a validation instance if it achieves the highest score; candidates are selected for mutation proportional to their win count. This **frequency-weighted Pareto selection** prevents premature convergence by maintaining diverse solutions that excel on different problem subsets.

GEPA's reflective mutation differs fundamentally from random perturbation. When a prompt fails, the reflection model receives the full context: the current instruction, the input that caused failure, the model's reasoning, and the expected outcome. It then generates a revised prompt that addresses the specific failure mode—effectively learning high-level rules from trial and error.

Alicia's implementation extends GEPA with **7-dimensional multi-objective optimization** spanning success rate, quality, efficiency, robustness, generalization, diversity, and innovation. The OptimizationService orchestrates runs, manages Pareto archives, and provides streaming progress updates for real-time monitoring.

### Key use cases and problem domains

GEPA excels in compound AI systems with multiple LLM modules where traditional RL struggles to assign credit. Primary applications include:

- **Multi-hop reasoning** (HotpotQA, HoVer) requiring inference across multiple documents
- **Instruction following** with precise constraint adherence (IFBench)
- **Privacy-preserving delegation** balancing utility against information leakage (PUPA)
- **Code generation and optimization** including CUDA kernel tuning
- **AI safety monitoring** for detecting backdoors in untrusted code
- **RAG systems** using vector databases like ChromaDB, Weaviate, and Pinecone

Industry adoption spans **Databricks** (building enterprise agents 90× cheaper), **MLflow** (integrated via `mlflow.genai.optimize_prompts()`), and **Comet-ml/Opik** for experiment tracking.

### Data requirements for effective optimization

GEPA requires four essential inputs:

**1. Seed candidate prompts** defining the initial text to optimize:
```python
seed_candidate = {
    "system_prompt": "You are a helpful assistant.",
    "instruction": "Analyze the following..."
}
```

**2. Training dataset** (trainset) for reflective updates—a list of `dspy.Example` objects containing inputs and task metadata. This powers the reflection minibatch sampling.

**3. Validation dataset** (valset) for Pareto score tracking. If omitted, trainset serves both purposes, risking overfitting. The recommendation: use the smallest valset that matches downstream task distribution.

**4. Metric function** returning both score and feedback:
```python
def metric(gold, pred, trace, pred_name, pred_trace) -> ScoreWithFeedback:
    score = compute_score(gold, pred)
    feedback = generate_diagnostic_text(gold, pred)
    return {"score": score, "feedback": feedback}
```

Budget configuration uses either `auto` presets ("light", "medium", "heavy") or explicit limits via `max_full_evals` or `max_metric_calls`.

### DSPy integration architecture

GEPA integrates deeply with DSPy through a **DspyAdapter** that encapsulates evaluation, trace capture, feedback extraction, and instruction proposal. The `dspy.GEPA` optimizer inherits from `Teleprompter` and implements the standard `compile()` interface:

```python
import dspy

gepa = dspy.GEPA(
    metric=metric_with_feedback,
    reflection_lm=dspy.LM("gpt-5", temperature=1.0, max_tokens=32000),
    auto="medium"
)
optimized_program = gepa.compile(student, trainset=trainset, valset=valset)
```

The adapter captures full traces of DSPy module execution, identifies trace segments corresponding to specific predictors, and reflects on predictor behavior to propose new instructions. For `dspy.ReAct` modules, GEPA jointly optimizes tools and prompts via `enable_tool_optimization=True`.

### GEPA versus reinforcement learning

| Dimension | GEPA | GRPO (RL) |
|-----------|------|-----------|
| Learning signal | Rich natural language feedback | Sparse scalar rewards |
| Rollouts required | **400–1,200** | 24,000+ |
| What's optimized | Prompts (text) | Model weights (LoRA) |
| Sample efficiency | **Up to 35× more efficient** | Requires thousands of rollouts |
| Credit assignment | Explicit via textual feedback | Implicit via policy gradients |
| Interpretability | Human-readable evolution | Black-box weight updates |
| Model access | API-only (no gradients) | Requires training access |

Benchmark results on **Qwen3-8B** demonstrate GEPA's advantages:

- **HotpotQA**: GEPA 62.3 vs GRPO 43.3 (**+19 points**)
- **HoVer**: GEPA 52.3 vs GRPO 38.6 (**+13.7 points**)
- **PUPA**: GEPA 91.8 vs GRPO 86.7 (**+5.1 points**)

GEPA matches GRPO's best validation scores with as few as **32–179 training rollouts**, and produces prompts **up to 9.2× shorter** than MIPROv2.

---

## Part 2: Mathematical principles and implementation analysis

### Formal optimization framework

GEPA formalizes compound AI systems as **Φ = (M, C, X, Y)** where M represents the set of language modules, C the control flow logic, and X, Y the global input/output schemas. Each module **Mᵢ = (πᵢ, θᵢ, Xᵢ, Yᵢ)** contains a system prompt πᵢ, underlying model weights θᵢ, and module-specific schemas.

The learnable parameters are **ΠΦ = ⟨π₁, ..., π|M|⟩**—the collection of all module prompts. For a task instance **(x, m)** where x maps to input schema and m contains evaluator metadata, the system produces output **y = Φ(x; Π)**. The metric function **μ : Y × M → [0, 1]** measures output quality.

### Pareto dominance and selection

A prompt **dominates** another if it scores better or equal on all validation instances and strictly better on at least one. The **Pareto frontier** contains all non-dominated prompts—those achieving best score on at least one instance.

The selection algorithm computes win frequency for each frontier candidate:
```
wins[p] = count of instances i where S[p][i] = max(S[q][i] for q in P)
```

Candidates are sampled with probability proportional to wins, balancing exploration (all non-dominated prompts have selection chance) with exploitation (frequent winners are more likely parents).

### Complete algorithm pseudocode

```
GEPA(goldens, metrics, iterations, pareto_size, minibatch_size):

1. GOLDEN SPLITTING:
   D_pareto ← sample(goldens, pareto_size)     # Fixed validation set
   D_feedback ← goldens \ D_pareto              # Feedback sampling set

2. INITIALIZE:
   candidates ← [root_prompt]
   pareto_scores[root_prompt] ← evaluate(root_prompt, D_pareto, metrics)

3. FOR iteration = 1 to iterations:
   
   a. PARETO SELECTION:
      frontier ← find_non_dominated(candidates, pareto_scores)
      wins ← count_wins_per_candidate(frontier, pareto_scores)
      parent ← sample(frontier, probability ∝ wins)
   
   b. FEEDBACK COLLECTION:
      minibatch ← sample(D_feedback, minibatch_size)
      responses ← execute(parent, minibatch)
      scores, reasons ← evaluate(responses, metrics)
      feedback ← concatenate(reasons)
   
   c. REFLECTIVE MUTATION:
      child ← LLM_rewrite(parent, feedback)
   
   d. ACCEPTANCE:
      IF score(child, minibatch) - score(parent, minibatch) > GEPA_MIN_DELTA:
         candidates.add(child)
         pareto_scores[child] ← evaluate(child, D_pareto, metrics)

4. MERGE (optional):
   FOR i = 1 to max_merge_invocations:
      merged ← combine_best_modules(frontier_candidates)
      IF improved: candidates.add(merged)

5. FINAL SELECTION:
   aggregate ← {c: mean(pareto_scores[c]) for c in candidates}
   RETURN argmax(aggregate, tie_breaker=PREFER_CHILD)
```

### Go implementation in Alicia

Alicia implements GEPA in Go using the dspy-go library (`github.com/XiaoConstantine/dspy-go`), providing production-grade optimization with **7-dimensional multi-objective fitness**. The implementation spans four key modules in `/internal/prompt/`:

**Dimension weights and scores** (`dimensions.go`):
```go
// DimensionWeights configures relative importance (must sum to 1.0)
type DimensionWeights struct {
    SuccessRate    float64  // Default: 0.25 - Task completion rate
    Quality        float64  // Default: 0.20 - Response quality
    Efficiency     float64  // Default: 0.15 - Token/cost efficiency
    Robustness     float64  // Default: 0.15 - Consistency across inputs
    Generalization float64  // Default: 0.10 - Out-of-distribution performance
    Diversity      float64  // Default: 0.10 - Population diversity contribution
    Innovation     float64  // Default: 0.05 - Novel solution characteristics
}

// DimensionScores holds per-dimension performance metrics
type DimensionScores struct {
    SuccessRate, Quality, Efficiency, Robustness,
    Generalization, Diversity, Innovation float64
}

// WeightedScore calculates aggregate score using configured weights
func (s *DimensionScores) WeightedScore(weights DimensionWeights) float64
```

**Pareto archive management** (`pareto.go`) implementing NSGA-II-style diversity preservation:
```go
type EliteSolution struct {
    ID           string
    Instructions string
    Demos        []Example
    Scores       DimensionScores
    Generation   int
    Coverage     int  // Number of examples this solution solves best
    CreatedAt    time.Time
}

type ParetoArchive struct {
    Solutions []*EliteSolution
    MaxSize   int
    mu        sync.RWMutex
}

// Add inserts non-dominated solutions, removes dominated ones
func (p *ParetoArchive) Add(solution *EliteSolution) bool

// SelectByCoverage implements GEPA's coverage-based selection
func (p *ParetoArchive) SelectByCoverage() *EliteSolution

// SelectByWeights returns best solution for given dimension weights
func (p *ParetoArchive) SelectByWeights(weights DimensionWeights) *EliteSolution
```

**Memory-aware optimization** (`memory_aware_module.go`) for few-shot learning:
```go
type MemoryAwareModule struct {
    *AliciaPredict
    memoryService ports.MemoryService
    maxDemos      int      // Default: 5
    threshold     float32  // Default: 0.7
}

// Process retrieves relevant memories and converts them to demonstrations
func (m *MemoryAwareModule) Process(ctx context.Context, inputs map[string]any) (map[string]any, error)

// RankMemoriesByRelevance combines similarity, importance, and recency
func RankMemoriesByRelevance(
    memories []*ports.MemorySearchResult,
    categoryFilter []string,
) []MemoryRelevanceScore
```

**Optimization service** (`internal/application/services/optimization.go`) orchestrating runs:
```go
type OptimizationService struct {
    repo             ports.PromptOptimizationRepository
    llmService       ports.LLMService
    reflectionLM     ports.LLMService  // Separate strong model for reflection
    idGenerator      ports.IDGenerator
    config           OptimizationConfig
    progressChannels map[string][]chan OptimizationProgressEvent  // Real-time updates
}

// OptimizeSignature runs GEPA on a signature with training/validation sets
func (s *OptimizationService) OptimizeSignature(
    ctx context.Context,
    sig prompt.Signature,
    trainset []prompt.Example,
    valset []prompt.Example,
    metric prompt.Metric,
) (*models.OptimizationRun, error)

// OptimizeSignatureWithMemory adds memory-augmented few-shot learning
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

**Configuration and execution** via dspy-go integration:
```go
// From OptimizationService.OptimizeSignature
gepaConfig := &optimizers.GEPAConfig{
    MaxGenerations:       config.MaxIterations / 10,  // Population-based evolution
    PopulationSize:       20,
    MutationRate:         0.3,
    CrossoverRate:        0.7,
    ElitismRate:          0.1,
    ReflectionFreq:       2,   // Reflect every 2 generations
    ReflectionDepth:      3,   // Look back 3 generations
    SelectionStrategy:    "adaptive_pareto",  // Multi-objective Pareto
    ConvergenceThreshold: 0.01,
    StagnationLimit:      3,
    EvaluationBatchSize:  config.MinibatchSize,
    ConcurrencyLevel:     3,   // Parallel candidate evaluation
}

gepaOptimizer, _ := optimizers.NewGEPA(gepaConfig)
optimizedProgram, _ := gepaOptimizer.Compile(ctx, program, dataset, metric)
```

**Key architectural decisions:**

- **Goroutine-based concurrency**: Optimization runs asynchronously with non-blocking progress channels
- **Thread-safe Pareto archive**: `sync.RWMutex` protects concurrent access during multi-threaded evaluation
- **Streaming progress**: Subscribers receive real-time `OptimizationProgressEvent` updates via buffered channels
- **Dimension-aware persistence**: Database stores per-dimension scores in JSONB for analysis
- **Memory-augmented learning**: Optional MemoryService integration retrieves relevant examples dynamically
- **Dual-LLM architecture**: Separate `reflectionLM` (stronger model) for GEPA reflection vs task execution

The implementation prioritizes **production readiness** with explicit error handling, resource management, and observability over experimental flexibility.

### Data modeling approaches

For persistent GEPA optimization, model the following entities:

**Candidate schema:**
```sql
CREATE TABLE prompt_candidates (
    id UUID PRIMARY KEY,
    generation INT,
    parent_ids UUID[],
    instruction TEXT,
    avg_score FLOAT,
    created_at TIMESTAMP,
    metadata JSONB
);
```

**Pareto archive with dimension scores:**
```sql
CREATE TABLE pareto_archive (
    id UUID PRIMARY KEY,
    prompt_candidate_id UUID REFERENCES prompt_candidates(id),
    dimension_scores JSONB,  -- Stores DimensionScores as JSON
    coverage_count INT,
    generation INT,
    created_at TIMESTAMP
);
```

**Note**: Alicia's implementation stores multi-dimensional scores in the `dimension_scores` JSONB column with fields like `{success_rate, quality, efficiency, robustness, generalization, diversity, innovation}`, and aggregate scores in `prompt_candidates.avg_score`.

---

## Part 3: Implementation with user feedback

### Collecting feedback for GEPA optimization

GEPA's power comes from rich textual feedback, not just scores. Effective feedback collection requires instrumenting your evaluation pipeline to capture diagnostic information at multiple levels:

**Predictor-level feedback** provides fine-grained signals for multi-module systems:
```python
def feedback_urgency(gold_urgency, pred_urgency):
    score = 1.0 if gold_urgency == pred_urgency else 0.0
    if gold_urgency != pred_urgency:
        feedback = f"Classified urgency as `{pred_urgency}` but correct is `{gold_urgency}`. Consider what contextual cues indicate urgency level."
    else:
        feedback = f"Correctly identified urgency as `{gold_urgency}`."
    return feedback, score
```

**Program-level feedback** works when aggregate metrics have clear sub-components:
```python
def compute_overall_score_with_feedback(gold, pred, trace=None, pred_name=None, pred_trace=None):
    quality = evaluate_quality(gold, pred)
    leakage = evaluate_privacy_leakage(gold, pred)
    overall = (quality + (1 - leakage)) / 2.0
    
    feedback = f"Overall: {overall:.2f}. Quality: {quality:.2f}, Privacy: {1-leakage:.2f}. "
    feedback += "Improve response quality while minimizing PII exposure."
    
    return dspy.Prediction(score=overall, feedback=feedback)
```

**Comparative feedback** for safety-critical applications:
```python
def metric_for_sample(attack_score, honest_score, backdoor_input):
    if attack_score > honest_score:
        feedback = "Good job! Attack code correctly rated MORE suspicious."
    else:
        feedback = f"Attack code should have higher suspicion than honest code. "
        feedback += f"The backdoor triggers on input: {backdoor_input}. "
        feedback += "Give precise scores, especially for low values (1-9)."
    
    return ScoreWithFeedback(
        score=1.0 if attack_score > honest_score else 0.0,
        feedback=feedback
    )
```

### Analyzing and evaluating feedback quality

Feedback effectiveness depends on actionability and specificity. Evaluate your feedback by checking:

**Information density**: Does feedback explain what went wrong AND suggest improvement direction?
```python
# Low quality: "Wrong answer"
# High quality: "Predicted 'medium' urgency but answer is 'high'. 
#               Keywords like 'immediately', 'emergency', 'critical' indicate high urgency."
```

**Granularity match**: For multi-module systems, does feedback target the failing component?
```python
def metric_with_feedback(gold, pred, trace, pred_name, pred_trace):
    # pred_name tells GEPA which predictor is being evaluated
    if pred_name == "urgency_module.predict":
        return urgency_feedback(gold, pred)
    elif pred_name == "sentiment_module.predict":
        return sentiment_feedback(gold, pred)
```

**Domain coverage**: Does feedback span the error modes in your task?

Use GEPA's `track_stats=True` to analyze which feedback patterns correlate with improvements:
```python
results = gepa.compile(program, trainset, valset)
detailed = results.detailed_results

# Analyze improvement trajectory
for i, (candidate, score) in enumerate(zip(detailed.candidates, detailed.val_aggregate_scores)):
    print(f"Gen {i}: Score {score:.3f}, Instruction length: {len(candidate.instruction)}")
```

### Redeploying GEPA-optimized prompts

After optimization, extract and persist the improved prompts:

```python
# Save optimized program
optimized_program.save("models/optimized_gepa.json")

# For production: load and configure
def load_optimized_monitor(model_path: Path, lm: dspy.BaseLM):
    predictor = dspy.ChainOfThought(MySignature)
    predictor.load(model_path)
    predictor.set_lm(lm)
    return predictor
```

**Version management** for prompt iterations:
```python
import hashlib
import json
from datetime import datetime

def save_with_versioning(program, base_dir: Path, metrics: dict):
    # Hash the instruction content for deduplication
    instruction_hash = hashlib.sha256(
        program.predict.signature.instructions.encode()
    ).hexdigest()[:8]
    
    version_dir = base_dir / f"v_{datetime.now().strftime('%Y%m%d')}_{instruction_hash}"
    version_dir.mkdir(parents=True, exist_ok=True)
    
    program.save(version_dir / "program.json")
    
    with open(version_dir / "metrics.json", "w") as f:
        json.dump(metrics, f)
```

### Implementation with DSPy

> **Note**: The code examples below demonstrate DSPy patterns using Python. Alicia's actual implementation uses **dspy-go** (Go), providing the same conceptual framework with production-grade concurrency and type safety. The Python examples illustrate the optimization patterns that translate to the Go implementation.

The standard DSPy integration follows this pattern:

```python
import dspy
from dspy import GEPA

# 1. Configure LMs
task_lm = dspy.LM("openai/gpt-4-turbo", temperature=1.0)
reflection_lm = dspy.LM("anthropic/claude-3-5-sonnet-20241022", temperature=1.0, max_tokens=32000)
dspy.configure(lm=task_lm)

# 2. Define signatures and modules
class AnalyzeMessage(dspy.Signature):
    """Analyze the message for urgency and sentiment."""
    message: str = dspy.InputField()
    urgency: Literal["low", "medium", "high"] = dspy.OutputField()
    sentiment: Literal["positive", "neutral", "negative"] = dspy.OutputField()

class Analyzer(dspy.Module):
    def __init__(self):
        self.analyze = dspy.ChainOfThought(AnalyzeMessage)
    
    def forward(self, message: str):
        return self.analyze(message=message)

# 3. Create feedback-aware metric
def metric_with_feedback(gold, pred, trace=None, pred_name=None, pred_trace=None):
    urgency_correct = gold.urgency == pred.urgency
    sentiment_correct = gold.sentiment == pred.sentiment
    score = (urgency_correct + sentiment_correct) / 2
    
    feedback_parts = []
    if not urgency_correct:
        feedback_parts.append(f"Urgency should be '{gold.urgency}', not '{pred.urgency}'")
    if not sentiment_correct:
        feedback_parts.append(f"Sentiment should be '{gold.sentiment}', not '{pred.sentiment}'")
    
    if feedback_parts:
        feedback = ". ".join(feedback_parts) + ". Analyze contextual cues more carefully."
    else:
        feedback = "Perfect classification!"
    
    return dspy.Prediction(score=score, feedback=feedback)

# 4. Run GEPA optimization
optimizer = GEPA(
    metric=metric_with_feedback,
    reflection_lm=reflection_lm,
    auto="medium",
    num_threads=16,
    track_stats=True,
)

optimized = optimizer.compile(
    Analyzer(),
    trainset=train_examples,
    valset=val_examples,
)

# 5. Evaluate and deploy
evaluator = dspy.Evaluate(devset=test_set, metric=metric_with_feedback)
print(f"Optimized score: {evaluator(optimized)}")
optimized.save("models/analyzer_gepa.json")
```

### Implementation without DSPy

For standalone GEPA without DSPy dependencies, implement the core loop directly:

```python
import numpy as np
from dataclasses import dataclass
from typing import Callable

@dataclass
class Candidate:
    instruction: str
    parent_id: int | None = None
    scores: list[float] = None

class StandaloneGEPA:
    def __init__(
        self,
        reflection_fn: Callable[[str, str], str],  # (prompt, feedback) -> new_prompt
        evaluate_fn: Callable[[str, dict], tuple[float, str]],  # (prompt, example) -> (score, feedback)
        minibatch_size: int = 3,
        min_delta: float = 0.01,
    ):
        self.reflection_fn = reflection_fn
        self.evaluate_fn = evaluate_fn
        self.minibatch_size = minibatch_size
        self.min_delta = min_delta
    
    def optimize(
        self,
        seed_prompt: str,
        trainset: list[dict],
        valset: list[dict],
        max_iterations: int = 50,
    ) -> str:
        # Initialize
        candidates = [Candidate(instruction=seed_prompt)]
        
        # Score seed on valset
        candidates[0].scores = [
            self.evaluate_fn(seed_prompt, ex)[0] for ex in valset
        ]
        
        for iteration in range(max_iterations):
            # Pareto selection
            parent_idx = self._select_from_pareto(candidates)
            parent = candidates[parent_idx]
            
            # Sample minibatch
            minibatch_idx = np.random.choice(len(trainset), self.minibatch_size, replace=False)
            minibatch = [trainset[i] for i in minibatch_idx]
            
            # Collect feedback
            feedbacks = []
            parent_scores = []
            for ex in minibatch:
                score, feedback = self.evaluate_fn(parent.instruction, ex)
                feedbacks.append(feedback)
                parent_scores.append(score)
            
            # Reflective mutation
            combined_feedback = "\n".join(feedbacks)
            child_instruction = self.reflection_fn(parent.instruction, combined_feedback)
            
            # Evaluate child on minibatch
            child_scores = [
                self.evaluate_fn(child_instruction, ex)[0] for ex in minibatch
            ]
            
            # Acceptance criterion
            if np.mean(child_scores) - np.mean(parent_scores) > self.min_delta:
                child = Candidate(
                    instruction=child_instruction,
                    parent_id=parent_idx,
                    scores=[self.evaluate_fn(child_instruction, ex)[0] for ex in valset]
                )
                candidates.append(child)
        
        # Return best by aggregate score
        aggregate_scores = [np.mean(c.scores) for c in candidates]
        return candidates[np.argmax(aggregate_scores)].instruction
    
    def _select_from_pareto(self, candidates: list[Candidate]) -> int:
        # Find non-dominated candidates
        n = len(candidates)
        dominated = [False] * n
        
        for i in range(n):
            for j in range(n):
                if i != j:
                    scores_i = np.array(candidates[i].scores)
                    scores_j = np.array(candidates[j].scores)
                    if np.all(scores_j >= scores_i) and np.any(scores_j > scores_i):
                        dominated[i] = True
                        break
        
        frontier = [i for i in range(n) if not dominated[i]]
        
        # Count wins and sample proportionally
        wins = np.zeros(len(frontier))
        for task_idx in range(len(candidates[0].scores)):
            best_score = max(candidates[i].scores[task_idx] for i in frontier)
            for fi, ci in enumerate(frontier):
                if candidates[ci].scores[task_idx] == best_score:
                    wins[fi] += 1
        
        probs = wins / wins.sum() if wins.sum() > 0 else np.ones(len(frontier)) / len(frontier)
        return frontier[np.random.choice(len(frontier), p=probs)]
```

### Database backend persistence

Replace file-based storage with a database for production:

```python
from sqlalchemy import create_engine, Column, String, Float, Integer, JSON, ForeignKey
from sqlalchemy.orm import declarative_base, sessionmaker, relationship
from sqlalchemy.dialects.postgresql import UUID, ARRAY
import uuid

Base = declarative_base()

class CandidateModel(Base):
    __tablename__ = "gepa_candidates"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    run_id = Column(UUID(as_uuid=True), index=True)
    generation = Column(Integer)
    instruction = Column(String)
    parent_ids = Column(ARRAY(UUID))
    metadata = Column(JSON)
    
    scores = relationship("ScoreModel", back_populates="candidate")

class ScoreModel(Base):
    __tablename__ = "gepa_scores"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    candidate_id = Column(UUID(as_uuid=True), ForeignKey("gepa_candidates.id"))
    task_idx = Column(Integer)
    score = Column(Float)
    feedback = Column(String)
    trace = Column(JSON)
    
    candidate = relationship("CandidateModel", back_populates="scores")

class DatabaseGEPAStore:
    def __init__(self, connection_string: str):
        self.engine = create_engine(connection_string)
        Base.metadata.create_all(self.engine)
        self.Session = sessionmaker(bind=self.engine)
    
    def save_candidate(self, run_id: uuid.UUID, candidate: Candidate, generation: int):
        with self.Session() as session:
            db_candidate = CandidateModel(
                run_id=run_id,
                generation=generation,
                instruction=candidate.instruction,
                parent_ids=[candidate.parent_id] if candidate.parent_id else [],
            )
            session.add(db_candidate)
            
            for task_idx, score in enumerate(candidate.scores or []):
                db_score = ScoreModel(
                    candidate_id=db_candidate.id,
                    task_idx=task_idx,
                    score=score,
                )
                session.add(db_score)
            
            session.commit()
            return db_candidate.id
    
    def load_pareto_frontier(self, run_id: uuid.UUID) -> list[Candidate]:
        with self.Session() as session:
            candidates = session.query(CandidateModel).filter_by(run_id=run_id).all()
            return [
                Candidate(
                    instruction=c.instruction,
                    scores=[s.score for s in sorted(c.scores, key=lambda x: x.task_idx)]
                )
                for c in candidates
            ]
```

---

## Part 4: Implementation for inference

### Runtime optimization with GEPA

GEPA primarily operates during training/compilation, but several patterns enable inference-time benefits:

**Cached prompt selection** uses pre-computed Pareto candidates for task-specific routing:
```python
class InferenceTimeGEPA:
    def __init__(self, candidates: list[Candidate], valset_signatures: list[str]):
        self.candidates = candidates
        self.task_embeddings = self._embed_tasks(valset_signatures)
        self.candidate_strengths = self._compute_strengths()
    
    def select_prompt_for_input(self, input_text: str, embedding_fn) -> str:
        """Select best candidate based on input similarity to training tasks."""
        input_embedding = embedding_fn(input_text)
        
        # Find most similar validation tasks
        similarities = cosine_similarity(input_embedding, self.task_embeddings)
        top_task_indices = np.argsort(similarities)[-5:]
        
        # Select candidate that performs best on similar tasks
        scores = np.zeros(len(self.candidates))
        for task_idx in top_task_indices:
            for ci, candidate in enumerate(self.candidates):
                scores[ci] += candidate.scores[task_idx] * similarities[task_idx]
        
        return self.candidates[np.argmax(scores)].instruction
```

**Dynamic prompt adaptation** triggers re-optimization when performance degrades:
```python
class AdaptiveGEPADeployment:
    def __init__(self, base_prompt: str, feedback_buffer_size: int = 100):
        self.current_prompt = base_prompt
        self.feedback_buffer = []
        self.buffer_size = feedback_buffer_size
        self.performance_threshold = 0.7
    
    def infer(self, input_text: str, lm) -> str:
        return lm(self.current_prompt + "\n" + input_text)
    
    def record_feedback(self, input_text: str, output: str, score: float, feedback: str):
        self.feedback_buffer.append({
            "input": input_text,
            "output": output,
            "score": score,
            "feedback": feedback,
        })
        
        if len(self.feedback_buffer) >= self.buffer_size:
            self._maybe_reoptimize()
    
    def _maybe_reoptimize(self):
        recent_scores = [f["score"] for f in self.feedback_buffer[-50:]]
        if np.mean(recent_scores) < self.performance_threshold:
            # Trigger GEPA re-optimization with accumulated feedback
            self._run_incremental_gepa()
```

### Evaluating alternative work paths for agents

For agentic systems, GEPA can optimize decision policies by evaluating alternative trajectories.

### Runtime optimization strategies

**Prompt ensemble** combines multiple Pareto-optimal prompts:
```python
class GEPAEnsemble:
    def __init__(self, candidates: list[str], lm, aggregation: str = "vote"):
        self.candidates = candidates
        self.lm = lm
        self.aggregation = aggregation
    
    def infer(self, input_text: str) -> str:
        responses = [
            self.lm(f"{prompt}\n{input_text}")
            for prompt in self.candidates
        ]
        
        if self.aggregation == "vote":
            return Counter(responses).most_common(1)[0][0]
        elif self.aggregation == "confidence":
            # Use model confidence scores if available
            return self._select_highest_confidence(responses)
```

**Speculative prompt switching** tests multiple prompts in parallel:
```python
async def speculative_inference(prompts: list[str], input_text: str, lm, timeout: float = 2.0):
    """Race multiple prompts, return first high-confidence response."""
    
    async def run_with_confidence(prompt: str):
        response = await lm.acall(f"{prompt}\n{input_text}")
        confidence = extract_confidence(response)
        return prompt, response, confidence
    
    tasks = [run_with_confidence(p) for p in prompts]
    
    for completed in asyncio.as_completed(tasks):
        prompt, response, confidence = await completed
        if confidence > 0.9:
            # Cancel remaining tasks
            for task in tasks:
                task.cancel()
            return response
    
    # Fallback to best result
    results = await asyncio.gather(*tasks, return_exceptions=True)
    valid = [(p, r, c) for p, r, c in results if not isinstance(r, Exception)]
    return max(valid, key=lambda x: x[2])[1]
```

### Integration patterns for agent systems

**ReAct agent optimization** with joint tool and prompt tuning:
```python
import dspy

class OptimizedReActAgent(dspy.Module):
    def __init__(self, tools: list):
        self.react = dspy.ReAct(
            signature="question -> answer",
            tools=tools,
            max_iters=5,
        )
    
    def forward(self, question: str):
        return self.react(question=question)

# Optimize with tool-aware GEPA
optimizer = dspy.GEPA(
    metric=agent_metric_with_feedback,
    enable_tool_optimization=True,  # Joint tool+prompt optimization
    reflection_lm=dspy.LM("gpt-5", temperature=1.0),
    auto="heavy",
)

optimized_agent = optimizer.compile(
    OptimizedReActAgent(tools=[search_tool, calculator_tool]),
    trainset=agent_tasks,
    valset=agent_val_tasks,
)
```

**Multi-agent coordination** optimizes prompts for agent ensembles:
```python
class CoordinatedAgentSystem(dspy.Module):
    def __init__(self):
        self.planner = dspy.ChainOfThought("task -> subtasks")
        self.executor = dspy.ReAct("subtask, context -> result", tools=[...])
        self.aggregator = dspy.ChainOfThought("results -> final_answer")
    
    def forward(self, task: str):
        plan = self.planner(task=task)
        results = [self.executor(subtask=st, context=task) for st in plan.subtasks]
        return self.aggregator(results=results)

# GEPA optimizes all three components with coordinated feedback
def coordination_metric(gold, pred, trace, pred_name, pred_trace):
    if pred_name == "planner.predict":
        return evaluate_plan_quality(gold, pred)
    elif pred_name == "executor.predict":
        return evaluate_execution(gold, pred)
    elif pred_name == "aggregator.predict":
        return evaluate_aggregation(gold, pred)
```

---

## Conclusion

GEPA fundamentally reimagines prompt optimization by treating **natural language as the optimization medium itself**. Rather than learning from sparse scalar rewards, it leverages LLMs' ability to reflect on failures and propose targeted fixes—achieving state-of-the-art results with 35× fewer rollouts than reinforcement learning approaches.

The key technical innovations enabling this efficiency include **Pareto-based candidate selection** maintaining solution diversity, **reflective mutation** generating semantically meaningful prompt updates, and **minibatch gating** preventing expensive evaluations unless improvements are likely.

For practitioners, GEPA offers three deployment paths: deep DSPy integration for rapid experimentation, standalone implementation for custom systems, and the Go implementation for production-grade concurrent optimization. The feedback design is critical—actionable, specific feedback correlating with the failing component drives the largest improvements.

At inference time, GEPA's Pareto frontier enables **task-adaptive prompt selection** and **ensemble strategies** that outperform single-prompt approaches. For agentic systems, joint optimization of decision prompts and tool usage creates agents that learn high-level strategies from trajectory feedback rather than step-by-step rewards.

The emergence of GEPA signals a broader shift in AI optimization: from gradient-based methods requiring model access to **language-native optimization** that works with any API-accessible model. As compound AI systems grow more complex, reflective prompt evolution offers an interpretable, efficient path to systematic improvement.

## See Also

- [Optimization System](OPTIMIZATION_SYSTEM.md) - System integration
- [Agent Documentation](AGENT.md) - How agent uses optimization

