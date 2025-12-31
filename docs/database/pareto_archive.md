# pareto_archive

Elite Pareto-optimal solutions from GEPA optimization.

## Schema

```sql
CREATE TABLE pareto_archive (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('pa'),
    run_id TEXT NOT NULL REFERENCES prompt_optimization_runs(id) ON DELETE CASCADE,
    instructions TEXT NOT NULL,
    demos JSONB DEFAULT '[]',
    dimension_scores JSONB NOT NULL DEFAULT '{}',
    generation INT NOT NULL DEFAULT 0,
    coverage INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Note**: This table has no `deleted_at` or `updated_at` columns. Archive entries are immutable snapshots of Pareto-optimal solutions.

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `pa_` prefix |
| `run_id` | TEXT | FK to prompt_optimization_runs |
| `instructions` | TEXT | Prompt instructions |
| `demos` | JSONB | Few-shot examples |
| `dimension_scores` | JSONB | 7-dimension performance scores |
| `generation` | INTEGER | Generation when archived |
| `coverage` | INTEGER | Example coverage count |
| `created_at` | TIMESTAMP | Archive timestamp |

## Indexes

```sql
CREATE INDEX idx_pareto_archive_run ON pareto_archive(run_id);
CREATE INDEX idx_pareto_archive_scores ON pareto_archive USING GIN (dimension_scores);
```

## Relationships

- **Belongs to** [prompt_optimization_runs](./prompt_optimization_runs.md) via `run_id`

## Pareto Optimality

A solution is Pareto-optimal if no other solution is better in ALL dimensions. The archive maintains non-dominated solutions representing different trade-offs.

Example: Solution A (high quality, low efficiency) and Solution B (medium quality, high efficiency) can both be Pareto-optimal.

```
Quality
  ^
  |  A •
  |
  |    • B
  |
  +--------→ Efficiency
```

## 7-Dimension Scores

```json
{
  "successRate": 0.85,
  "quality": 0.92,
  "efficiency": 0.78,
  "robustness": 0.81,
  "generalization": 0.75,
  "diversity": 0.68,
  "innovation": 0.55
}
```

## Purpose

The Pareto archive:

1. **Preserves diversity**: Different trade-offs are maintained
2. **Prevents regression**: Good solutions aren't lost
3. **Enables selection**: Choose solution matching requirements
4. **Supports analysis**: Compare trade-offs across dimensions

## Example Queries

```sql
-- Pareto frontier for a run
SELECT id, dimension_scores, coverage
FROM pareto_archive
WHERE run_id = 'aor_xxx'
ORDER BY (dimension_scores->>'successRate')::float DESC;

-- High-quality solutions
SELECT instructions, dimension_scores
FROM pareto_archive
WHERE run_id = 'aor_xxx'
  AND (dimension_scores->>'quality')::float > 0.9;

-- Efficient solutions
SELECT instructions, dimension_scores
FROM pareto_archive
WHERE run_id = 'aor_xxx'
  AND (dimension_scores->>'efficiency')::float > 0.85;
```

## See Also

- [GEPA Primer](../GEPA_PRIMER.md) - Pareto selection explained
