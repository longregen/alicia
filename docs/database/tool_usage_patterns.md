# tool_usage_patterns

Tool usage analytics for identifying optimization opportunities.

## Schema

```sql
CREATE TABLE tool_usage_patterns (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('tup'),
    tool_name VARCHAR(255) NOT NULL,
    user_intent_pattern TEXT NOT NULL,
    success_rate REAL,
    avg_result_quality REAL,
    common_arguments JSONB,
    sample_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `tup_` prefix |
| `tool_name` | VARCHAR(255) | Tool name |
| `user_intent_pattern` | TEXT | Pattern describing user intent |
| `success_rate` | REAL | Success rate for this pattern (0-1) |
| `avg_result_quality` | REAL | Average quality score |
| `common_arguments` | JSONB | Frequently used arguments |
| `sample_count` | INTEGER | Number of observations |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_usage_patterns_tool ON tool_usage_patterns(tool_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_usage_patterns_success ON tool_usage_patterns(tool_name, success_rate DESC NULLS LAST) WHERE deleted_at IS NULL;
CREATE INDEX idx_usage_patterns_updated ON tool_usage_patterns(updated_at DESC) WHERE deleted_at IS NULL;
```

## Purpose

Aggregates tool usage data to:

1. **Identify failure patterns**: Which intents fail frequently
2. **Optimize descriptions**: Improve for common use cases
3. **Suggest improvements**: Guide tool development
4. **Monitor health**: Track tool reliability over time

## User Intent Patterns

Patterns describe what users are trying to accomplish:

| Pattern | Example |
|---------|---------|
| `search_current_events` | "What happened in the news today?" |
| `lookup_documentation` | "How do I use the X API?" |
| `calculate_value` | "Convert 100 USD to EUR" |

## Common Arguments

```json
{
  "query": {
    "most_common": ["weather", "news", "price"],
    "avg_length": 12
  },
  "limit": {
    "most_common": [5, 10, 20],
    "default_usage": 0.65
  }
}
```

## Example Queries

```sql
-- Problematic patterns (low success rate)
SELECT tool_name, user_intent_pattern, success_rate, sample_count
FROM tool_usage_patterns
WHERE success_rate < 0.7
  AND sample_count > 10
  AND deleted_at IS NULL
ORDER BY success_rate;

-- Most common tool usage patterns
SELECT tool_name, user_intent_pattern, sample_count
FROM tool_usage_patterns
WHERE deleted_at IS NULL
ORDER BY sample_count DESC
LIMIT 20;

-- Tool health overview
SELECT tool_name,
       AVG(success_rate) as avg_success,
       SUM(sample_count) as total_samples
FROM tool_usage_patterns
WHERE deleted_at IS NULL
GROUP BY tool_name
ORDER BY avg_success;
```

## See Also

- [Optimization System](../OPTIMIZATION_SYSTEM.md)
