# alicia_sentences

Stores individual sentence chunks from messages with their TTS audio for streaming responses.

## Schema

```sql
CREATE TABLE alicia_sentences (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ams'),
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    sentence_sequence_number INTEGER NOT NULL,
    text TEXT NOT NULL DEFAULT '',
    audio_type audio_type,
    audio_format TEXT,
    duration_ms INTEGER,
    audio_bytesize INTEGER,
    audio_data BYTEA,
    meta JSONB DEFAULT '{}',
    completion_status completion_status NOT NULL DEFAULT 'completed',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `ams_` prefix |
| `message_id` | TEXT | FK to alicia_messages |
| `sentence_sequence_number` | INTEGER | Order within message |
| `text` | TEXT | Sentence text content |
| `audio_type` | audio_type | `input` or `output` |
| `audio_format` | TEXT | Audio format (`wav`, `pcm`, `opus`) |
| `duration_ms` | INTEGER | Audio duration in milliseconds |
| `audio_bytesize` | INTEGER | Size of audio data in bytes |
| `audio_data` | BYTEA | Raw audio data |
| `meta` | JSONB | Additional metadata (timing, prosody) |
| `completion_status` | completion_status | Streaming state |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_sentences_message ON alicia_sentences(message_id, sentence_sequence_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_sentences_completion_status ON alicia_sentences(completion_status) WHERE deleted_at IS NULL AND completion_status != 'completed';
CREATE INDEX idx_sentences_orphaned ON alicia_sentences(message_id, completion_status, created_at) WHERE deleted_at IS NULL AND completion_status IN ('pending', 'streaming', 'failed');
```

## Relationships

- **Belongs to** [alicia_messages](./messages.md) via `message_id`

## Purpose

Breaking messages into sentences enables:

1. **Progressive TTS**: Generate and stream audio sentence-by-sentence
2. **Lower latency**: Users hear the first sentence while later ones are still generating
3. **Granular replay**: Replay specific sentences without full message audio
4. **Interruption handling**: Stop generation mid-message cleanly

## Streaming Flow

```
1. Assistant begins response
2. First sentence detected → create sentence record (pending)
3. TTS generates audio → update with audio_data (streaming)
4. Audio sent via LiveKit → mark completed
5. Repeat for each sentence
```

## Example Queries

```sql
-- Get all sentences for a message
SELECT sentence_sequence_number, text, duration_ms
FROM alicia_sentences
WHERE message_id = 'am_xxx'
  AND deleted_at IS NULL
ORDER BY sentence_sequence_number ASC;

-- Find incomplete sentences (for cleanup)
SELECT id, message_id, completion_status
FROM alicia_sentences
WHERE completion_status IN ('pending', 'streaming')
  AND created_at < NOW() - INTERVAL '1 hour'
  AND deleted_at IS NULL;
```

## See Also

- [AssistantSentence Protocol](../protocol/04-message-types/16-assistant-sentence.md)
