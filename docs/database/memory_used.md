# alicia_memory_used

Tracks which memories were retrieved and used during conversations.

## Schema

```sql
CREATE TABLE alicia_memory_used (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('amu'),
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    memory_id TEXT NOT NULL REFERENCES alicia_memory(id) ON DELETE CASCADE,
    query_prompt TEXT,
    query_prompt_meta JSONB DEFAULT '{}',
    similarity_score REAL,
    meta JSONB DEFAULT '{}',
    position_in_results INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `amu_` prefix |
| `conversation_id` | TEXT | FK to alicia_conversations |
| `message_id` | TEXT | FK to alicia_messages |
| `memory_id` | TEXT | FK to alicia_memory |
| `query_prompt` | TEXT | Query used for retrieval |
| `query_prompt_meta` | JSONB | Query metadata |
| `similarity_score` | REAL | Cosine similarity score |
| `meta` | JSONB | Additional usage metadata |
| `position_in_results` | INTEGER | Rank in search results |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_memory_used_conversation ON alicia_memory_used(conversation_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_memory_used_message ON alicia_memory_used(message_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_memory_used_memory ON alicia_memory_used(memory_id) WHERE deleted_at IS NULL;
```

## Relationships

- **Belongs to** [alicia_conversations](./conversations.md) via `conversation_id`
- **Belongs to** [alicia_messages](./messages.md) via `message_id`
- **Belongs to** [alicia_memory](./memory.md) via `memory_id`

## Purpose

This junction table enables:

1. **Retrieval analytics**: Which memories are most useful
2. **Relevance tracking**: How similarity scores correlate with helpfulness
3. **Memory improvement**: Identify low-performing memories
4. **Debugging**: Understand why certain responses were generated

## Example Queries

```sql
-- Memory usage frequency
SELECT m.content, COUNT(*) as usage_count, AVG(mu.similarity_score) as avg_similarity
FROM alicia_memory_used mu
JOIN alicia_memory m ON mu.memory_id = m.id
WHERE mu.deleted_at IS NULL
GROUP BY m.id, m.content
ORDER BY usage_count DESC
LIMIT 10;

-- Memories used in a specific message
SELECT m.content, mu.similarity_score, mu.position_in_results
FROM alicia_memory_used mu
JOIN alicia_memory m ON mu.memory_id = m.id
WHERE mu.message_id = 'am_xxx'
  AND mu.deleted_at IS NULL
ORDER BY mu.position_in_results;
```

## See Also

- [MemoryTrace Protocol](../protocol/04-message-types/14-memory-trace.md)
