# alicia_memory

Long-term memory storage with vector embeddings for retrieval-augmented generation (RAG).

## Schema

```sql
CREATE TABLE alicia_memory (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('amem'),
    content TEXT NOT NULL,
    embeddings vector(1024),
    embeddings_info JSONB DEFAULT '{}',
    importance REAL DEFAULT 0.5,
    confidence REAL DEFAULT 1.0,
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    created_by TEXT,
    source_type TEXT,
    source_message_id VARCHAR(21) REFERENCES alicia_messages(id),
    source_info JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `amem_` prefix |
| `content` | TEXT | Memory content text |
| `embeddings` | vector(1024) | 1024-dimensional vector embedding |
| `embeddings_info` | JSONB | Embedding model and parameters |
| `importance` | REAL | Importance score (0-1) |
| `confidence` | REAL | Confidence in accuracy (0-1) |
| `user_rating` | INTEGER | User rating (1-5, optional) |
| `created_by` | TEXT | Creator identifier |
| `source_type` | TEXT | Source type (`conversation`, `document`, etc.) |
| `source_info` | JSONB | Source details (conversation ID, URL, etc.) |
| `tags` | TEXT[] | Categorization tags |
| `pinned` | BOOLEAN | Priority access flag |
| `archived` | BOOLEAN | Hidden from normal views |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_memory_embeddings ON alicia_memory USING ivfflat (embeddings vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_memory_importance ON alicia_memory(importance DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_memory_tags ON alicia_memory USING gin(tags) WHERE deleted_at IS NULL;
CREATE INDEX idx_memory_pinned ON alicia_memory(pinned) WHERE deleted_at IS NULL AND pinned = TRUE;
CREATE INDEX idx_memory_archived ON alicia_memory(archived) WHERE deleted_at IS NULL;
```

## Relationships

- **Has many** [alicia_memory_used](./memory_used.md) via `memory_id`
- **Target of** [alicia_votes](./votes.md) when `target_type = 'memory'`

## Vector Search

The `embeddings` field uses pgvector with IVFFlat indexing for efficient similarity search:

```sql
-- Semantic search for relevant memories
SELECT id, content, importance,
       1 - (embeddings <=> $1) as similarity
FROM alicia_memory
WHERE deleted_at IS NULL
  AND archived = FALSE
ORDER BY embeddings <=> $1
LIMIT 5;
```

## Pinned vs Archived

| Flag | Effect |
|------|--------|
| `pinned = TRUE` | Always included in retrieval, higher priority |
| `archived = TRUE` | Excluded from normal retrieval, visible in archive view |

## Embeddings Info

The `embeddings_info` JSONB field stores embedding metadata:

```json
{
  "model": "text-embedding-3-large",
  "dimensions": 1024,
  "created_at": "2025-01-01T00:00:00Z"
}
```

## Example Queries

```sql
-- Get pinned memories
SELECT id, content, importance
FROM alicia_memory
WHERE pinned = TRUE
  AND archived = FALSE
  AND deleted_at IS NULL
ORDER BY importance DESC;

-- Search by tag
SELECT id, content
FROM alicia_memory
WHERE 'preferences' = ANY(tags)
  AND deleted_at IS NULL;
```

## See Also

- [MemoryTrace Protocol](../protocol/04-message-types/14-memory-trace.md)
