# alicia_conversations

Top-level container for conversations, mapping each to a LiveKit room for real-time communication.

## Schema

```sql
CREATE TABLE alicia_conversations (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ac'),
    user_id TEXT NOT NULL DEFAULT 'default_user',
    title TEXT NOT NULL DEFAULT '',
    status conversation_status NOT NULL DEFAULT 'active',
    livekit_room_name TEXT,
    tip_message_id VARCHAR(21) REFERENCES alicia_messages(id),
    preferences JSONB DEFAULT '{}',
    last_client_stanza_id INTEGER NOT NULL DEFAULT 0,
    last_server_stanza_id INTEGER NOT NULL DEFAULT -1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `ac_` prefix |
| `user_id` | TEXT | User identifier for multi-user support |
| `title` | TEXT | Descriptive title for the conversation |
| `status` | conversation_status | `active`, `archived`, or `deleted` |
| `livekit_room_name` | TEXT | LiveKit room name for real-time communication |
| `preferences` | JSONB | User preferences (TTS voice, language, etc.) |
| `last_client_stanza_id` | INTEGER | Last stanza ID received from client (reconnection) |
| `last_server_stanza_id` | INTEGER | Last stanza ID sent by server (negative values) |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_conversations_status ON alicia_conversations(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_conversations_livekit_room ON alicia_conversations(livekit_room_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_conversations_created_at ON alicia_conversations(created_at DESC);
CREATE INDEX idx_conversations_user_created ON alicia_conversations(user_id, created_at DESC) WHERE deleted_at IS NULL;
```

## Relationships

- **Has many** [alicia_messages](./messages.md) via `conversation_id`
- **Has many** [alicia_votes](./votes.md) via `conversation_id`
- **Has many** [alicia_memory_used](./memory_used.md) via `conversation_id`
- **Has many** [alicia_session_stats](./session_stats.md) via `conversation_id`
- **Has many** [alicia_user_conversation_commentaries](./commentaries.md) via `conversation_id`

## LiveKit Integration

Each conversation maps to a LiveKit room. When a user reconnects:

1. Retrieve conversation by ID
2. Use `livekit_room_name` to rejoin the room
3. Use stanza IDs to resume message flow

The stanza ID system enables reliable reconnection - the client reports its last received server stanza, and the server replays any missed messages.

## Example Queries

```sql
-- Find active conversations for a user
SELECT id, title, livekit_room_name
FROM alicia_conversations
WHERE user_id = 'user_123'
  AND status = 'active'
  AND deleted_at IS NULL
ORDER BY updated_at DESC;

-- Reconnect to a conversation
SELECT id, livekit_room_name, last_client_stanza_id, last_server_stanza_id
FROM alicia_conversations
WHERE id = 'ac_xxx'
  AND deleted_at IS NULL;
```

## See Also

- [Reconnection Semantics](../protocol/05-reconnection-semantics.md)
- [Offline Sync](../OFFLINE_SYNC.md)
