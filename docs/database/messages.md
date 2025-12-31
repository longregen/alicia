# alicia_messages

Stores individual messages in conversations with support for offline sync and streaming completion.

## Schema

```sql
CREATE TABLE alicia_messages (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('am'),
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    sequence_number INTEGER NOT NULL,
    previous_id TEXT REFERENCES alicia_messages(id),
    message_role message_role NOT NULL,
    contents TEXT NOT NULL DEFAULT '',
    local_id TEXT,
    server_id TEXT,
    sync_status sync_status NOT NULL DEFAULT 'synced',
    synced_at TIMESTAMP,
    completion_status completion_status NOT NULL DEFAULT 'completed',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `am_` prefix |
| `conversation_id` | TEXT | FK to alicia_conversations |
| `sequence_number` | INTEGER | Order within conversation |
| `previous_id` | TEXT | Linked-list pointer to previous message |
| `message_role` | message_role | `user`, `assistant`, or `system` |
| `contents` | TEXT | Message text content |
| `local_id` | TEXT | Client-generated ID (offline support) |
| `server_id` | TEXT | Server-assigned canonical ID |
| `sync_status` | sync_status | `pending`, `synced`, or `conflict` |
| `synced_at` | TIMESTAMP | Last sync timestamp |
| `completion_status` | completion_status | `pending`, `streaming`, `completed`, `failed` |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_messages_conversation ON alicia_messages(conversation_id, sequence_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_previous ON alicia_messages(previous_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_local_id ON alicia_messages(local_id) WHERE local_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_messages_server_id ON alicia_messages(server_id) WHERE server_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_messages_sync_status ON alicia_messages(conversation_id, sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_completion_status ON alicia_messages(completion_status) WHERE deleted_at IS NULL AND completion_status != 'completed';
```

## Relationships

- **Belongs to** [alicia_conversations](./conversations.md) via `conversation_id`
- **Has many** [alicia_sentences](./sentences.md) via `message_id`
- **Has many** [alicia_tool_uses](./tool_uses.md) via `message_id`
- **Has many** [alicia_reasoning_steps](./reasoning_steps.md) via `message_id`
- **Has many** [alicia_memory_used](./memory_used.md) via `message_id`
- **Has many** [alicia_notes](./notes.md) via `message_id`
- **Referenced by** [alicia_audio](./audio.md) via `message_id`

## Dual-ID System

The `local_id`/`server_id` pair enables offline-first functionality:

1. Client creates message with `local_id` before connectivity
2. On sync, server assigns `server_id`
3. Both IDs remain for reconciliation

See [Offline Sync](../OFFLINE_SYNC.md) for details.

## Completion Status Flow

```
pending → streaming → completed
                  ↘ failed
```

- `pending`: Message created, not yet generating
- `streaming`: Response actively being generated
- `completed`: Generation finished successfully
- `failed`: Error during generation

## Example Queries

```sql
-- Get conversation history
SELECT id, message_role, contents, sequence_number
FROM alicia_messages
WHERE conversation_id = 'ac_xxx'
  AND deleted_at IS NULL
ORDER BY sequence_number ASC;

-- Find pending sync messages
SELECT id, local_id, contents
FROM alicia_messages
WHERE sync_status = 'pending'
  AND deleted_at IS NULL;
```

## See Also

- [Offline Sync](../OFFLINE_SYNC.md)
- [Protocol Database Alignment](../protocol/06-database-alignment.md)
