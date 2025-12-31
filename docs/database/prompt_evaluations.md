# prompt_evaluations

Evaluation results for candidate prompts against test examples.

## Schema

```sql
CREATE TABLE prompt_evaluations (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ape'),
    candidate_id TEXT NOT NULL REFERENCES prompt_candidates(id) ON DELETE CASCADE,
    example_id VARCHAR(255) NOT NULL,
    score REAL NOT NULL,
    feedback TEXT,
    trace JSONB,
    dimension_scores JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `ape_` prefix |
| `candidate_id` | TEXT | FK to prompt_candidates |
| `example_id` | VARCHAR(255) | Test example identifier |
| `score` | REAL | Evaluation score (0-1) |
| `feedback` | TEXT | Evaluation feedback |
| `trace` | JSONB | Execution trace |
| `dimension_scores` | JSONB | Per-dimension scores |
| `created_at` | TIMESTAMP | Evaluation timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_evaluations_candidate ON prompt_evaluations(candidate_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_evaluations_example ON prompt_evaluations(example_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_evaluations_score ON prompt_evaluations(candidate_id, score DESC) WHERE deleted_at IS NULL;
```

## Relationships

- **Belongs to** [prompt_candidates](./prompt_candidates.md) via `candidate_id`

## Trace Format

The `trace` JSONB field captures the full execution:

```json
{
  "input": "...",
  "reasoning": ["step1", "step2"],
  "tool_calls": [...],
  "output": "...",
  "expected": "...",
  "latency_ms": 1234,
  "tokens_used": 567
}
```

## Reflective Feedback

The `feedback` field contains natural language analysis of failures:

> "The candidate failed because it didn't consider edge case X. The prompt should explicitly mention handling empty inputs."

This feedback drives GEPA's reflective mutation.

## Example Queries

```sql
-- Evaluations for a candidate
SELECT example_id, score, feedback
FROM prompt_evaluations
WHERE candidate_id = 'apc_xxx'
  AND deleted_at IS NULL
ORDER BY score;

-- Failed evaluations for analysis
SELECT e.example_id, e.score, e.feedback, c.instructions
FROM prompt_evaluations e
JOIN prompt_candidates c ON e.candidate_id = c.id
WHERE e.score < 0.5
  AND e.deleted_at IS NULL
ORDER BY e.created_at DESC
LIMIT 20;
```

## See Also

- [GEPA Primer](../GEPA_PRIMER.md)
