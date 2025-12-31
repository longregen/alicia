# optimized_programs

Final optimized prompts ready for deployment.

## Schema

```sql
CREATE TABLE optimized_programs (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('op'),
    run_id TEXT NOT NULL REFERENCES prompt_optimization_runs(id) ON DELETE CASCADE,
    signature_name VARCHAR(255) NOT NULL,
    instructions TEXT NOT NULL,
    demos JSONB,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `op_` prefix |
| `run_id` | TEXT | FK to prompt_optimization_runs |
| `signature_name` | VARCHAR(255) | DSPy signature name |
| `instructions` | TEXT | Optimized prompt instructions |
| `demos` | JSONB | Optimized few-shot examples |
| `metadata` | JSONB | Performance metrics and config |
| `created_at` | TIMESTAMP | Creation timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_programs_run ON optimized_programs(run_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_programs_signature ON optimized_programs(signature_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_programs_created ON optimized_programs(created_at DESC) WHERE deleted_at IS NULL;
```

## Relationships

- **Belongs to** [prompt_optimization_runs](./prompt_optimization_runs.md) via `run_id`

## Metadata

```json
{
  "score": 0.92,
  "coverage": 47,
  "generation": 12,
  "parent_candidate_id": "apc_xxx",
  "dimension_scores": {...},
  "evaluation_count": 50
}
```

## Usage

Optimized programs are loaded at runtime:

```sql
-- Get latest optimized program for a signature
SELECT instructions, demos
FROM optimized_programs
WHERE signature_name = 'AnswerQuestion'
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1;
```

## Example Queries

```sql
-- All optimized programs
SELECT signature_name, created_at, metadata->>'score' as score
FROM optimized_programs
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- Compare versions
SELECT id, created_at, metadata->>'score' as score
FROM optimized_programs
WHERE signature_name = 'AnswerQuestion'
  AND deleted_at IS NULL
ORDER BY created_at DESC;
```

## See Also

- [GEPA Primer](../GEPA_PRIMER.md)
- [Optimization System](../OPTIMIZATION_SYSTEM.md)
