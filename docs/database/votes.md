# alicia_votes

Structured feedback voting on messages, tool uses, memories, and reasoning.

## Schema

```sql
CREATE TABLE alicia_votes (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('av'),
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    message_id TEXT REFERENCES alicia_messages(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL CHECK (target_type IN ('message', 'sentence', 'tool_use', 'memory', 'reasoning', 'memory_usage', 'memory_extraction')),
    target_id TEXT NOT NULL,
    vote TEXT NOT NULL CHECK (vote IN ('up', 'down', 'critical')),
    quick_feedback TEXT CHECK (quick_feedback IN (
        'wrong_tool', 'wrong_params', 'unnecessary', 'missing_context',
        'outdated', 'wrong_context', 'too_generic', 'incorrect',
        'incorrect_assumption', 'missed_consideration', 'overcomplicated', 'wrong_direction'
    )),
    note TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `av_` prefix |
| `conversation_id` | TEXT | FK to alicia_conversations |
| `message_id` | TEXT | FK to alicia_messages (context) |
| `target_type` | TEXT | Type of entity being voted on |
| `target_id` | TEXT | ID of the target entity |
| `vote` | TEXT | `up`, `down`, or `critical` |
| `quick_feedback` | TEXT | Predefined feedback category |
| `note` | TEXT | Optional freeform note |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_votes_conversation ON alicia_votes(conversation_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_votes_message ON alicia_votes(message_id) WHERE message_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_votes_target ON alicia_votes(target_type, target_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_votes_type_vote ON alicia_votes(target_type, vote) WHERE deleted_at IS NULL;
```

## Relationships

- **Belongs to** [alicia_conversations](./conversations.md) via `conversation_id`
- **References** [alicia_messages](./messages.md) via `message_id`

## Vote Types

| Vote | Meaning |
|------|---------|
| `up` | Positive feedback |
| `down` | Negative feedback |
| `critical` | Essential/important (for memories) |

## Target Types

| Type | Target ID | Description |
|------|-----------|-------------|
| `message` | alicia_messages.id | Vote on message |
| `sentence` | alicia_sentences.id | Vote on sentence |
| `tool_use` | alicia_tool_uses.id | Vote on tool call |
| `memory` | alicia_memory.id | Vote on memory |
| `reasoning` | alicia_reasoning_steps.id | Vote on reasoning step |
| `memory_usage` | alicia_memory_used.id | Vote on memory usage |
| `memory_extraction` | (custom ID) | Vote on memory extraction |

## Quick Feedback Categories

For fast structured feedback without typing:

| Category | Applicable To |
|----------|---------------|
| `wrong_tool` | tool_use |
| `wrong_params` | tool_use |
| `unnecessary` | tool_use, memory |
| `missing_context` | message, memory |
| `outdated` | memory |
| `wrong_context` | memory |
| `too_generic` | message, memory |
| `incorrect` | message, reasoning |
| `incorrect_assumption` | reasoning |
| `missed_consideration` | reasoning |
| `overcomplicated` | reasoning |
| `wrong_direction` | message |

## Example Queries

```sql
-- Get votes for a message
SELECT vote, quick_feedback, note
FROM alicia_votes
WHERE target_type = 'message'
  AND target_id = 'am_xxx'
  AND deleted_at IS NULL;

-- Feedback analytics
SELECT target_type, vote, COUNT(*)
FROM alicia_votes
WHERE deleted_at IS NULL
GROUP BY target_type, vote
ORDER BY target_type, vote;
```

## See Also

- [alicia_notes](./notes.md) - Longer form feedback
