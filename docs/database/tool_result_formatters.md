# tool_result_formatters

Learned result formatting rules for optimizing tool output presentation.

## Schema

```sql
CREATE TABLE tool_result_formatters (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('trf'),
    tool_name VARCHAR(255) NOT NULL UNIQUE,
    template TEXT NOT NULL,
    max_length INTEGER NOT NULL DEFAULT 2000,
    summarize_at INTEGER NOT NULL DEFAULT 1000,
    summary_prompt TEXT,
    key_fields JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `trf_` prefix |
| `tool_name` | VARCHAR(255) | Tool name (unique) |
| `template` | TEXT | Formatting template |
| `max_length` | INTEGER | Maximum output length |
| `summarize_at` | INTEGER | Character threshold for summarization |
| `summary_prompt` | TEXT | Prompt for summarizing long outputs |
| `key_fields` | JSONB | Important fields to preserve |
| `created_at` | TIMESTAMP | Creation timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_formatters_tool_name ON tool_result_formatters(tool_name) WHERE deleted_at IS NULL;
```

## Purpose

Manages how tool results are presented to the LLM:

1. **Token efficiency**: Reduce context usage
2. **Relevance**: Highlight important information
3. **Consistency**: Standardized output format

## Template Format

Templates use placeholder syntax:

```
{{#if error}}
Error: {{error}}
{{else}}
Result: {{result.summary}}
Details: {{result.details | truncate:500}}
{{/if}}
```

## Key Fields

Specifies which fields should always be preserved:

```json
{
  "always_include": ["status", "error", "summary"],
  "truncatable": ["details", "raw_output"],
  "omit_if_empty": ["warnings"]
}
```

## Summarization

When output exceeds `summarize_at` characters, the `summary_prompt` is used:

```
Summarize this tool result in 2-3 sentences, preserving:
- Success/failure status
- Key findings
- Any error messages
```

## Example Queries

```sql
-- Get formatter for a tool
SELECT template, max_length, summarize_at, key_fields
FROM tool_result_formatters
WHERE tool_name = 'web_search'
  AND deleted_at IS NULL;
```

## See Also

- [Optimization System](../OPTIMIZATION_SYSTEM.md)
