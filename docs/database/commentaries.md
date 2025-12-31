# alicia_user_conversation_commentaries

User feedback and comments about conversations or specific messages.

## Schema

```sql
CREATE TABLE alicia_user_conversation_commentaries (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('aucc'),
    content TEXT NOT NULL,
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    message_id TEXT REFERENCES alicia_messages(id) ON DELETE SET NULL,
    created_by TEXT,
    meta JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `aucc_` prefix |
| `content` | TEXT | Commentary text |
| `conversation_id` | TEXT | FK to alicia_conversations |
| `message_id` | TEXT | FK to alicia_messages (optional) |
| `created_by` | TEXT | User who created the comment |
| `meta` | JSONB | Additional metadata |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_commentaries_conversation ON alicia_user_conversation_commentaries(conversation_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commentaries_message ON alicia_user_conversation_commentaries(message_id) WHERE message_id IS NOT NULL AND deleted_at IS NULL;
```

## Relationships

- **Belongs to** [alicia_conversations](./conversations.md) via `conversation_id`
- **Optionally references** [alicia_messages](./messages.md) via `message_id`

## Usage

Commentaries provide freeform feedback on conversations. For structured feedback (voting), see:
- [alicia_votes](./votes.md) - Up/down voting
- [alicia_notes](./notes.md) - Categorized notes

## Example Queries

```sql
-- Get comments for a conversation
SELECT content, message_id, created_at
FROM alicia_user_conversation_commentaries
WHERE conversation_id = 'ac_xxx'
  AND deleted_at IS NULL
ORDER BY created_at DESC;
```

## See Also

- [Commentary Protocol](../protocol/04-message-types/15-commentary.md)
