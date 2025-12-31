# optimized_tools

Optimized tool configurations with improved descriptions and schemas.

## Schema

```sql
CREATE TABLE optimized_tools (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ot'),
    tool_id TEXT NOT NULL REFERENCES alicia_tools(id) ON DELETE CASCADE,
    optimized_description TEXT NOT NULL,
    optimized_schema JSONB,
    result_template TEXT,
    examples JSONB,
    version INTEGER NOT NULL DEFAULT 1,
    score REAL,
    optimized_at TIMESTAMP NOT NULL DEFAULT NOW(),
    active BOOLEAN NOT NULL DEFAULT false,
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `ot_` prefix |
| `tool_id` | TEXT | FK to alicia_tools |
| `optimized_description` | TEXT | Improved tool description |
| `optimized_schema` | JSONB | Refined parameter schema |
| `result_template` | TEXT | Template for formatting results |
| `examples` | JSONB | Usage examples |
| `version` | INTEGER | Version number |
| `score` | REAL | Optimization score |
| `optimized_at` | TIMESTAMP | Optimization timestamp |
| `active` | BOOLEAN | Currently in use |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_optimized_tools_tool ON optimized_tools(tool_id, version DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimized_tools_active ON optimized_tools(tool_id, active) WHERE deleted_at IS NULL AND active = true;
CREATE INDEX idx_optimized_tools_score ON optimized_tools(score DESC NULLS LAST) WHERE deleted_at IS NULL;
```

## Relationships

- **Belongs to** [alicia_tools](./tools.md) via `tool_id`

## Purpose

Optimizes tool descriptions to improve:

1. **Selection accuracy**: LLM picks the right tool
2. **Parameter accuracy**: Correct arguments passed
3. **Result interpretation**: Better understanding of output

## Examples Field

```json
[
  {
    "intent": "search for weather in Paris",
    "arguments": {"location": "Paris, France"},
    "result_summary": "Current conditions: 18°C, partly cloudy"
  }
]
```

## Activation

Only one version per tool should be active:

```sql
-- Activate a new version
UPDATE optimized_tools SET active = false WHERE tool_id = 'at_xxx';
UPDATE optimized_tools SET active = true WHERE id = 'ot_new';
```

## Example Queries

```sql
-- Get active optimized tool
SELECT optimized_description, optimized_schema, examples
FROM optimized_tools
WHERE tool_id = 'at_xxx'
  AND active = true
  AND deleted_at IS NULL;

-- Version history
SELECT version, score, optimized_at
FROM optimized_tools
WHERE tool_id = 'at_xxx'
  AND deleted_at IS NULL
ORDER BY version DESC;
```

## See Also

- [Optimization System](../OPTIMIZATION_SYSTEM.md)
