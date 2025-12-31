# alicia_reasoning_steps

Stores chain-of-thought reasoning steps from assistant responses.

## Schema

```sql
CREATE TABLE alicia_reasoning_steps (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ar'),
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    sequence_number INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `ar_` prefix |
| `message_id` | TEXT | FK to alicia_messages |
| `content` | TEXT | Reasoning step content |
| `sequence_number` | INTEGER | Order within message |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_reasoning_steps_message ON alicia_reasoning_steps(message_id, sequence_number) WHERE deleted_at IS NULL;
```

## Relationships

- **Belongs to** [alicia_messages](./messages.md) via `message_id`
- **Target of** [alicia_votes](./votes.md) when `target_type = 'reasoning'`

## Purpose

Captures the assistant's step-by-step reasoning process:

1. **Transparency**: Shows how conclusions are reached
2. **Debugging**: Identify where reasoning goes wrong
3. **Interleaving**: Reasoning can alternate with tool calls
4. **Feedback**: Users can vote on individual reasoning steps

## Example Queries

```sql
-- Get reasoning chain for a message
SELECT sequence_number, content
FROM alicia_reasoning_steps
WHERE message_id = 'am_xxx'
  AND deleted_at IS NULL
ORDER BY sequence_number;

-- Find messages with extensive reasoning
SELECT message_id, COUNT(*) as step_count
FROM alicia_reasoning_steps
WHERE deleted_at IS NULL
GROUP BY message_id
HAVING COUNT(*) > 5
ORDER BY step_count DESC;
```

## See Also

- [ReasoningStep Protocol](../protocol/04-message-types/05-reasoning-step.md)
