# prompt_candidates

Candidate prompts generated during GEPA optimization.

## Schema

```sql
CREATE TABLE prompt_candidates (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('apc'),
    run_id TEXT NOT NULL REFERENCES prompt_optimization_runs(id) ON DELETE CASCADE,
    instructions TEXT NOT NULL,
    demos JSONB,
    coverage INTEGER NOT NULL DEFAULT 0,
    avg_score REAL,
    generation INTEGER NOT NULL DEFAULT 0,
    parent_id TEXT REFERENCES prompt_candidates(id) ON DELETE SET NULL,
    dimension_scores JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `apc_` prefix |
| `run_id` | TEXT | FK to prompt_optimization_runs |
| `instructions` | TEXT | Prompt instructions |
| `demos` | JSONB | Few-shot examples |
| `coverage` | INTEGER | Examples solved successfully |
| `avg_score` | REAL | Average score across evaluations |
| `generation` | INTEGER | Generation number |
| `parent_id` | TEXT | Parent candidate (for mutation tracking) |
| `dimension_scores` | JSONB | Per-dimension scores |
| `created_at` | TIMESTAMP | Creation timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_candidates_run ON prompt_candidates(run_id, generation) WHERE deleted_at IS NULL;
CREATE INDEX idx_candidates_score ON prompt_candidates(run_id, avg_score DESC NULLS LAST) WHERE deleted_at IS NULL;
CREATE INDEX idx_candidates_parent ON prompt_candidates(parent_id) WHERE parent_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_candidates_coverage ON prompt_candidates(run_id, coverage DESC) WHERE deleted_at IS NULL;
```

## Relationships

- **Belongs to** [prompt_optimization_runs](./prompt_optimization_runs.md) via `run_id`
- **Has many** [prompt_evaluations](./prompt_evaluations.md) via `candidate_id`
- **Self-references** via `parent_id` (mutation lineage)

## Genetic Lineage

The `parent_id` field tracks mutation history:

```
gen0_candidate → gen1_mutant → gen2_mutant
                           ↘ gen2_mutant_b
```

## Coverage-Based Selection

GEPA uses coverage to ensure diverse solutions. A candidate "covers" an example if it achieves the best score on that example. Candidates are selected proportional to their coverage.

## Dimension Scores

```json
{
  "successRate": 0.85,
  "quality": 0.78,
  "efficiency": 0.92,
  "robustness": 0.65,
  "generalization": 0.70,
  "diversity": 0.80,
  "innovation": 0.45
}
```

## Example Queries

```sql
-- Best candidates from a run
SELECT id, instructions, avg_score, coverage
FROM prompt_candidates
WHERE run_id = 'aor_xxx'
  AND deleted_at IS NULL
ORDER BY avg_score DESC NULLS LAST
LIMIT 5;

-- Candidate lineage
WITH RECURSIVE lineage AS (
    SELECT id, parent_id, generation, instructions
    FROM prompt_candidates
    WHERE id = 'apc_xxx'
    UNION ALL
    SELECT c.id, c.parent_id, c.generation, c.instructions
    FROM prompt_candidates c
    JOIN lineage l ON c.id = l.parent_id
)
SELECT * FROM lineage ORDER BY generation;
```

## See Also

- [GEPA Primer](../GEPA_PRIMER.md)
