# alicia_tool_uses

Tracks tool executions with arguments, results, and status.

## Schema

```sql
CREATE TABLE alicia_tool_uses (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('atu'),
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    tool_arguments JSONB DEFAULT '{}',
    tool_result JSONB,
    status tool_status NOT NULL DEFAULT 'pending',
    error_message TEXT,
    sequence_number INTEGER NOT NULL,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `atu_` prefix |
| `message_id` | TEXT | FK to alicia_messages |
| `tool_name` | TEXT | Name of the tool |
| `tool_arguments` | JSONB | Arguments passed to tool |
| `tool_result` | JSONB | Result returned by tool |
| `status` | tool_status | Execution status |
| `error_message` | TEXT | Error message if failed |
| `sequence_number` | INTEGER | Order within message |
| `completed_at` | TIMESTAMP | Completion timestamp |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_tool_uses_message ON alicia_tool_uses(message_id, sequence_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_tool_uses_status ON alicia_tool_uses(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tool_uses_tool_name ON alicia_tool_uses(tool_name) WHERE deleted_at IS NULL;
```

## Relationships

- **Belongs to** [alicia_messages](./messages.md) via `message_id`
- **References** [alicia_tools](./tools.md) via `tool_name`
- **Target of** [alicia_votes](./votes.md) when `target_type = 'tool_use'`

## Status Flow

```
pending → running → success
                ↘ error
                ↘ cancelled
```

| Status | Description |
|--------|-------------|
| `pending` | Tool call requested, not started |
| `running` | Execution in progress |
| `success` | Completed successfully |
| `error` | Failed with error |
| `cancelled` | User cancelled execution |

## Example Queries

```sql
-- Get tool uses for a message
SELECT tool_name, tool_arguments, tool_result, status
FROM alicia_tool_uses
WHERE message_id = 'am_xxx'
  AND deleted_at IS NULL
ORDER BY sequence_number;

-- Find failed tool calls
SELECT id, tool_name, error_message, created_at
FROM alicia_tool_uses
WHERE status = 'error'
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 10;

-- Tool usage analytics
SELECT tool_name,
       COUNT(*) as total_calls,
       COUNT(*) FILTER (WHERE status = 'success') as successes,
       COUNT(*) FILTER (WHERE status = 'error') as errors
FROM alicia_tool_uses
WHERE deleted_at IS NULL
GROUP BY tool_name
ORDER BY total_calls DESC;
```

## See Also

- [ToolUseRequest Protocol](../protocol/04-message-types/06-tool-use-request.md)
- [ToolUseResult Protocol](../protocol/04-message-types/07-tool-use-result.md)
