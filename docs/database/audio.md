# alicia_audio

Stores audio recordings (user input and assistant output) with transcriptions.

## Schema

```sql
CREATE TABLE alicia_audio (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('aa'),
    message_id TEXT REFERENCES alicia_messages(id) ON DELETE SET NULL,
    audio_type audio_type NOT NULL,
    audio_format TEXT NOT NULL,
    audio_data BYTEA,
    duration_ms INTEGER,
    transcription TEXT,
    livekit_track_sid TEXT,
    transcription_meta JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `aa_` prefix |
| `message_id` | TEXT | FK to alicia_messages (nullable) |
| `audio_type` | audio_type | `input` (user) or `output` (assistant) |
| `audio_format` | TEXT | Format (`wav`, `pcm`, `opus`, etc.) |
| `audio_data` | BYTEA | Raw audio data |
| `duration_ms` | INTEGER | Duration in milliseconds |
| `transcription` | TEXT | Whisper transcription of audio |
| `livekit_track_sid` | TEXT | LiveKit track SID for correlation |
| `transcription_meta` | JSONB | Transcription metadata (confidence, language) |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_audio_message ON alicia_audio(message_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_audio_livekit_track ON alicia_audio(livekit_track_sid) WHERE livekit_track_sid IS NOT NULL AND deleted_at IS NULL;
```

## Relationships

- **Belongs to** [alicia_messages](./messages.md) via `message_id` (optional)

## Audio Types

| Type | Description |
|------|-------------|
| `input` | User voice input, transcribed via Whisper |
| `output` | Full assistant response audio (see also [sentences](./sentences.md) for chunked audio) |

## LiveKit Integration

The `livekit_track_sid` field correlates persisted audio with LiveKit real-time streams:

- Enables debugging of audio issues
- Supports analytics on audio quality
- Allows replay of conversations

## Transcription Metadata

The `transcription_meta` JSONB field may include:

```json
{
  "model": "whisper-1",
  "language": "en",
  "confidence": 0.95,
  "segments": [...],
  "vad_timestamps": [...]
}
```

## Example Queries

```sql
-- Get audio with transcription for a message
SELECT audio_type, duration_ms, transcription
FROM alicia_audio
WHERE message_id = 'am_xxx'
  AND deleted_at IS NULL;

-- Find audio by LiveKit track
SELECT id, message_id, transcription
FROM alicia_audio
WHERE livekit_track_sid = 'TR_xxx'
  AND deleted_at IS NULL;
```

## See Also

- [AudioChunk Protocol](../protocol/04-message-types/04-audio-chunk.md)
- [Transcription Protocol](../protocol/04-message-types/09-transcription.md)
