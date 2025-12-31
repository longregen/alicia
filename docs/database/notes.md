# alicia_notes

Structured notes on messages for detailed feedback.

## Schema

```sql
CREATE TABLE alicia_notes (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('an'),
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('improvement', 'correction', 'context', 'general')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `an_` prefix |
| `message_id` | TEXT | FK to alicia_messages |
| `content` | TEXT | Note content |
| `category` | TEXT | Note category |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_notes_message ON alicia_notes(message_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_category ON alicia_notes(category) WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_created_at ON alicia_notes(created_at DESC) WHERE deleted_at IS NULL;
```

## Relationships

- **Belongs to** [alicia_messages](./messages.md) via `message_id`

## Categories

| Category | Purpose | Example |
|----------|---------|---------|
| `improvement` | Suggestion for better response | "Should include code examples" |
| `correction` | Factual error | "The API was deprecated in 2024" |
| `context` | Missing context | "User prefers TypeScript" |
| `general` | Other feedback | "This was helpful" |

## Usage

Notes provide detailed feedback beyond votes:
- **Votes** (see [alicia_votes](./votes.md)) are quick up/down signals
- **Notes** are longer explanations for training and improvement

## Example Queries

```sql
-- Get notes for a message
SELECT category, content, created_at
FROM alicia_notes
WHERE message_id = 'am_xxx'
  AND deleted_at IS NULL
ORDER BY created_at;

-- Recent corrections
SELECT n.content, m.contents as message
FROM alicia_notes n
JOIN alicia_messages m ON n.message_id = m.id
WHERE n.category = 'correction'
  AND n.deleted_at IS NULL
ORDER BY n.created_at DESC
LIMIT 10;
```

## See Also

- [alicia_votes](./votes.md) - Quick voting
