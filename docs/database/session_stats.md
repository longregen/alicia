# alicia_session_stats

Session-level analytics for conversations.

## Schema

```sql
CREATE TABLE alicia_session_stats (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ass'),
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    message_count INTEGER NOT NULL DEFAULT 0,
    tool_call_count INTEGER NOT NULL DEFAULT 0,
    memories_used INTEGER NOT NULL DEFAULT 0,
    session_duration_seconds INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Note**: This table has no `deleted_at` column (permanent deletion). The `updated_at` column is automatically updated via trigger when stats are modified.

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `ass_` prefix |
| `conversation_id` | TEXT | FK to alicia_conversations |
| `message_count` | INTEGER | Total messages in session |
| `tool_call_count` | INTEGER | Total tool calls made |
| `memories_used` | INTEGER | Unique memories retrieved |
| `session_duration_seconds` | INTEGER | Session duration |
| `created_at` | TIMESTAMP | Session start timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |

## Indexes

```sql
CREATE INDEX idx_session_stats_conversation ON alicia_session_stats(conversation_id);
CREATE INDEX idx_session_stats_created_at ON alicia_session_stats(created_at DESC);
```

## Relationships

- **Belongs to** [alicia_conversations](./conversations.md) via `conversation_id`

## Purpose

Tracks aggregate session metrics for:

1. **Usage analytics**: Understand conversation patterns
2. **Performance monitoring**: Track tool usage and memory retrieval
3. **Resource planning**: Estimate compute requirements
4. **User engagement**: Measure session lengths

## Example Queries

```sql
-- Get session stats for a conversation
SELECT message_count, tool_call_count, memories_used, session_duration_seconds
FROM alicia_session_stats
WHERE conversation_id = 'ac_xxx';

-- Average session metrics
SELECT
    AVG(message_count) as avg_messages,
    AVG(tool_call_count) as avg_tools,
    AVG(session_duration_seconds) / 60.0 as avg_duration_minutes
FROM alicia_session_stats
WHERE created_at > NOW() - INTERVAL '30 days';

-- Most active sessions
SELECT conversation_id, message_count, tool_call_count
FROM alicia_session_stats
ORDER BY message_count DESC
LIMIT 10;
```
