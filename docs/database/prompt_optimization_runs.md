# prompt_optimization_runs

Tracks GEPA prompt optimization sessions.

## Schema

```sql
CREATE TABLE prompt_optimization_runs (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('aor'),
    signature_name VARCHAR(255) NOT NULL,
    status optimization_status NOT NULL DEFAULT 'pending',
    config JSONB DEFAULT '{}',
    best_score REAL,
    iterations INTEGER NOT NULL DEFAULT 0,
    dimension_weights JSONB DEFAULT '{
        "successRate": 0.25,
        "quality": 0.20,
        "efficiency": 0.15,
        "robustness": 0.15,
        "generalization": 0.10,
        "diversity": 0.10,
        "innovation": 0.05
    }',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `aor_` prefix |
| `signature_name` | VARCHAR(255) | Name of the DSPy signature being optimized |
| `status` | optimization_status | `pending`, `running`, `completed`, `failed` |
| `config` | JSONB | Hyperparameters and constraints |
| `best_score` | REAL | Best achieved score |
| `iterations` | INTEGER | Number of generations completed |
| `dimension_weights` | JSONB | Weights for 7-dimension scoring |
| `created_at` | TIMESTAMP | Run start timestamp |
| `completed_at` | TIMESTAMP | Run completion timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_optimization_runs_status ON prompt_optimization_runs(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimization_runs_signature ON prompt_optimization_runs(signature_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimization_runs_created ON prompt_optimization_runs(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimization_runs_completed ON prompt_optimization_runs(completed_at DESC) WHERE deleted_at IS NULL AND completed_at IS NOT NULL;
```

## Relationships

- **Has many** [prompt_candidates](./prompt_candidates.md) via `run_id`
- **Has many** [optimized_programs](./optimized_programs.md) via `run_id`
- **Has many** [pareto_archive](./pareto_archive.md) via `run_id`

## 7-Dimension Scoring

GEPA evaluates candidates across seven dimensions:

| Dimension | Default Weight | Description |
|-----------|----------------|-------------|
| `successRate` | 0.25 | Task completion rate |
| `quality` | 0.20 | Output quality |
| `efficiency` | 0.15 | Token/latency efficiency |
| `robustness` | 0.15 | Handling edge cases |
| `generalization` | 0.10 | Performance on novel inputs |
| `diversity` | 0.10 | Solution variety |
| `innovation` | 0.05 | Novel approaches |

## Status Flow

```
pending → running → completed
                ↘ failed
```

## Example Queries

```sql
-- Get recent optimization runs
SELECT id, signature_name, status, best_score, iterations
FROM prompt_optimization_runs
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 10;

-- Running optimizations
SELECT id, signature_name, iterations
FROM prompt_optimization_runs
WHERE status = 'running'
  AND deleted_at IS NULL;
```

## See Also

- [GEPA Primer](../GEPA_PRIMER.md) - Optimization concepts
- [Optimization System](../OPTIMIZATION_SYSTEM.md) - Architecture
