## Database Alignment

The Alicia protocol is designed hand-in-hand with the Alicia conversation database schema. This ensures that every message in the protocol correlates directly with stored data, enabling consistency between live interactions and persistent records. This section outlines how the key protocol elements align with the database tables.

## Core Database Tables

### `alicia_conversations`

This table stores conversation metadata and LiveKit integration details.

**Key Fields:**
* `id` (TEXT) — The unique conversation identifier, also used as the LiveKit room name
* `user_id` — The user who owns this conversation
* `livekit_room_name` — The LiveKit room identifier (typically matches the conversation id)
* `title` — Conversation title
* `status` — Conversation state: active, archived, or deleted
* `preferences` — JSON object containing conversation preferences
* `last_client_stanza_id` — Last stanza ID received from client (for reconnection)
* `last_server_stanza_id` — Last stanza ID sent by server (for reconnection)
* `created_at` — When the conversation started
* `updated_at` — Last activity timestamp
* `deleted_at` — Soft deletion timestamp (nullable)

**Protocol Mapping:**
* When a client sends a Configuration message with no `conversationId`, the server creates a new row in this table
* The generated `id` becomes the `conversationId` in all subsequent protocol messages
* The server creates a LiveKit room with `livekit_room_name` matching the conversation id
* When a client reconnects with a known `conversationId`, the server looks up this table to verify the conversation exists and retrieve the room name

### `alicia_messages`

This table stores all user and assistant messages in conversations.

**Key Fields:**
* `id` (TEXT) — The unique message identifier
* `conversation_id` — Foreign key to `alicia_conversations`
* `sequence_number` — Sequential ordering within conversation
* `previous_id` — Foreign key to the previous message (forms the conversation chain)
* `message_role` — 'user', 'assistant', or 'system'
* `contents` — The message text content
* `local_id` — Client-generated ID before server assignment (for offline support)
* `server_id` — Canonical server-assigned ID (for offline support)
* `sync_status` — Synchronization state: pending, synced, or conflict
* `synced_at` — Timestamp when message was last synced
* `completion_status` — Message completion state: pending, streaming, completed, or failed
* `created_at` — Message timestamp
* `updated_at` — Last modification timestamp
* `deleted_at` — Soft deletion timestamp (nullable)

**Protocol Mapping:**

**UserMessage (type 2):**
* Creates a new row with `message_role='user'`
* `id` field from protocol maps to `id` in database
* `previousId` field maps to `previous_id`
* `content` field maps to `contents`
* `completion_status` set to 'completed'

**AssistantMessage (type 3):**
* Creates a new row with `message_role='assistant'`
* `id` field from protocol maps to `id` in database
* `previousId` field maps to `previous_id` (references the UserMessage it responds to)
* `content` field maps to `contents`
* `completion_status` set to 'completed'

**StartAnswer (type 13) + AssistantSentence (type 16):**

When the server uses streaming:

1. **StartAnswer** triggers insertion of a row with `message_role='assistant'`, `id` from the message, `previous_id` referencing the user message, `contents` initially set to empty string, and `completion_status='streaming'`
2. As **AssistantSentence** chunks arrive, they are stored in `alicia_sentences` (see below) and the contents field is built up
3. When the final AssistantSentence arrives (indicated by the sentence being marked final), the assembled full content is stored in `alicia_messages.contents` and `completion_status` is set to 'completed'

This approach ensures the assistant message is visible in the database immediately, even during streaming.

### `alicia_sentences`

This table stores individual sentence chunks for streaming assistant responses, along with associated audio data.

**Key Fields:**
* `id` (TEXT) — The unique sentence identifier
* `message_id` — Foreign key to `alicia_messages` (the assistant message being streamed)
* `sentence_sequence_number` — The sentence sequence number within this message
* `text` — The sentence text content
* `audio_type` — Type of audio: 'input' or 'output'
* `audio_format` — Audio format specification (e.g., "pcm_s16le_24000")
* `duration_ms` — Audio duration in milliseconds
* `audio_bytesize` — Size of audio data in bytes
* `audio_data` — Binary audio data (BYTEA)
* `meta` — JSON object for additional metadata
* `completion_status` — Sentence completion state: pending, streaming, completed, or failed
* `created_at` — Creation timestamp
* `updated_at` — Last modification timestamp
* `deleted_at` — Soft deletion timestamp (nullable)

**Protocol Mapping:**

Each **AssistantSentence (type 16)** message maps to a row:
* `previousId` in the protocol indicates which assistant message this sentence belongs to (maps to `message_id`)
* `sequence` field maps to `sentence_sequence_number`
* `text` field maps to `text`
* When marked as final, `completion_status` is set to 'completed'

Audio data from **AudioChunk (type 10)** messages is associated with sentences:
* `format` field maps to `audio_format`
* `duration` field maps to `duration_ms`
* Binary audio data maps to `audio_data`
* Audio size maps to `audio_bytesize`

This table allows the system to:
* Replay streaming responses chunk by chunk on reconnection
* Store and retrieve sentence-level audio for playback
* Analyze streaming patterns and latency
* Reconstruct the exact streaming sequence for debugging

### `alicia_memory`

This table stores memory items (facts, preferences, context) that can be retrieved during conversations using semantic search.

**Key Fields:**
* `id` (TEXT) — The unique memory identifier
* `content` — The memory content (text)
* `embeddings` — Vector embedding for semantic search (1024 dimensions)
* `embeddings_info` — JSON metadata about the embeddings
* `importance` — Importance score (0.0 to 1.0, default 0.5)
* `confidence` — Confidence score (0.0 to 1.0, default 1.0)
* `user_rating` — Optional user rating (1-5 stars)
* `created_by` — Identifier of who created this memory
* `source_type` — Type of source that generated this memory
* `source_info` — JSON metadata about the memory source
* `tags` — Array of tags for categorization
* `pinned` — Whether this memory is pinned for priority access
* `archived` — Whether this memory is archived and hidden from normal views
* `created_at` — When this memory was stored
* `updated_at` — Last modification timestamp
* `deleted_at` — Soft deletion timestamp (nullable)

**Protocol Mapping:**

This table is queried during conversation processing (not directly created by protocol messages). However, when memory is used:

**MemoryTrace (type 14)** logs the usage:
* The server retrieves memory from this table during processing
* It sends a MemoryTrace message referencing the `memoryId` from this table
* This creates a record in `alicia_memory_used` (see below)

### `alicia_memory_used`

This table logs which memories were retrieved and used in which conversations and messages.

**Key Fields:**
* `id` (TEXT) — Unique usage record identifier
* `conversation_id` — Foreign key to `alicia_conversations`
* `message_id` — Foreign key to `alicia_messages` (which message triggered this retrieval)
* `memory_id` — Foreign key to `alicia_memory` (which memory was used)
* `query_prompt` — The query prompt used to retrieve this memory
* `query_prompt_meta` — JSON metadata about the query prompt
* `similarity_score` — Cosine similarity score from vector search
* `meta` — JSON object for additional metadata
* `position_in_results` — Position of this memory in retrieval results
* `created_at` — Creation timestamp
* `updated_at` — Last modification timestamp
* `deleted_at` — Soft deletion timestamp (nullable)

**Protocol Mapping:**

Each **MemoryTrace (type 14)** message creates a row:
* `id` from protocol maps to `id`
* `messageId` indicates the message context (maps to `message_id`)
* `memoryId` maps to `memory_id`
* `relevance` score maps to `similarity_score`
* Retrieval context is stored in `meta`

This enables:
* Tracking which memories influenced which responses
* Debugging memory retrieval issues
* Analyzing memory effectiveness and relevance over time
* Understanding query patterns and memory access patterns

### `alicia_user_conversation_commentaries`

This table stores user notes and commentary on messages within conversations.

**Key Fields:**
* `id` (TEXT) — Unique commentary identifier
* `content` — The commentary text
* `conversation_id` — Foreign key to `alicia_conversations`
* `message_id` — Foreign key to `alicia_messages` (the message being commented on, nullable)
* `created_by` — Identifier of who created this commentary
* `meta` — JSON object for additional metadata
* `created_at` — Creation timestamp
* `updated_at` — Last modification timestamp
* `deleted_at` — Soft deletion timestamp (nullable)

**Protocol Mapping:**

User notes are created via **UserNote** protocol messages and stored in this table:
* `id` from protocol maps to `id`
* `content` maps to `content`
* `messageId` indicates the message being commented on (maps to `message_id`)
* User context is stored in `created_by`
* Additional metadata is stored in `meta`

This supports:
* User feedback and notes on specific messages
* Context annotations for improving responses
* Conversational metadata tracking

### `alicia_tool_uses`

This table logs all tool invocations during conversations.

**Key Fields:**
* `id` (TEXT) — Unique tool use identifier
* `message_id` — Foreign key to `alicia_messages` (which message triggered this tool use)
* `tool_name` — The tool that was invoked
* `tool_arguments` — JSON object with tool parameters
* `tool_result` — JSON object with tool results (populated when result arrives)
* `status` — Tool execution status: pending, running, success, error, or cancelled
* `error_message` — Error message if execution failed
* `sequence_number` — Sequential ordering of tool uses within a message
* `completed_at` — Result timestamp
* `created_at` — Request timestamp
* `updated_at` — Last modification timestamp
* `deleted_at` — Soft deletion timestamp (nullable)

**Protocol Mapping:**

**ToolUseRequest (type 6):**
* Creates a new row with `status='pending'`
* `id` from protocol maps to `id`
* `messageId` indicates context (maps to `message_id`)
* `toolName` maps to `tool_name`
* `parameters` maps to `tool_arguments` (JSON)

**ToolUseResult (type 7):**
* Updates the existing row (matched by `requestId` referencing the tool use `id`)
* If successful: `status` set to 'success', `result` maps to `tool_result` (JSON)
* If failed: `status` set to 'error', error information maps to `error_message`
* Sets `completed_at` to current timestamp

### `alicia_audio`

This table stores audio recordings and transcriptions for voice conversations.

**Key Fields:**
* `id` (TEXT) — Unique audio record identifier
* `message_id` — Foreign key to `alicia_messages` (nullable, the message associated with this audio)
* `audio_type` — Type of audio: 'input' (user speech) or 'output' (assistant speech)
* `audio_format` — Audio format specification (e.g., "pcm_s16le_24000", "opus_48000")
* `audio_data` — Binary audio data (BYTEA, nullable)
* `duration_ms` — Audio duration in milliseconds
* `transcription` — Transcribed text (for input audio)
* `livekit_track_sid` — LiveKit track session ID
* `transcription_meta` — JSON metadata about transcription (confidence, etc.)
* `created_at` — Creation timestamp
* `updated_at` — Last modification timestamp
* `deleted_at` — Soft deletion timestamp (nullable)

**Protocol Mapping:**

**Transcription (type 9)** messages update or create records:
* `text` field maps to `transcription`
* Confidence and metadata map to `transcription_meta`
* When final, the transcription is associated with a UserMessage via `message_id`

**AudioChunk (type 10)** messages can create records:
* `trackSid` maps to `livekit_track_sid`
* `format` field maps to `audio_format`
* `durationMs` field maps to `duration_ms`
* Binary audio data maps to `audio_data`

This table is useful for:
* Storing complete audio recordings for playback
* Debugging ASR accuracy
* Associating transcriptions with audio
* Analyzing audio quality and latency

### `alicia_meta`

This table stores arbitrary key-value metadata for any entity in the system using a generic reference system.

**Key Fields:**
* `id` (TEXT) — Unique metadata record identifier
* `ref` — Reference to the entity this metadata belongs to (can reference any table's ID)
* `key` — The metadata key (e.g., "messaging.trace_id", "source", "model")
* `value` — The metadata value (text)
* `created_at` — Creation timestamp
* `updated_at` — Last modification timestamp
* `deleted_at` — Soft deletion timestamp (nullable)

**Protocol Mapping:**

The `meta` field in protocol envelopes maps to rows in this table:
* Each key-value pair in the `meta` map creates a row
* `ref` is set to the relevant entity ID (conversation ID, message ID, etc.)
* Special keys like `messaging.trace_id` and `messaging.span_id` are stored here for distributed tracing

Examples:
* `meta: {"source": "microphone"}` on a UserMessage creates a row with `ref=<message_id>`, `key='source'`, `value='microphone'`
* `meta: {"model": "claude-3-opus", "responseTime": "123ms"}` on an AssistantMessage creates two rows with `ref=<message_id>`

This flexible structure allows:
* Storing OpenTelemetry trace IDs for debugging
* Tracking model versions and parameters
* Recording client versions and platforms
* Custom application-specific metadata
* Associating metadata with any entity type

Note: Audio data is primarily stored in two locations:
1. **Message-level audio**: Complete audio recordings are stored in the `alicia_audio` table (described above)
2. **Sentence-level audio**: Audio for individual streaming sentences is stored directly in the `alicia_sentences` table

The system does not use a separate `alicia_audio_chunks` table. Audio chunks from the protocol are processed and stored in the appropriate table based on context (either as complete audio recordings or as sentence-specific audio data).

## LiveKit Integration

The database schema includes LiveKit-specific fields to support the real-time communication layer:

### Conversation to Room Mapping

Each conversation has a one-to-one mapping with a LiveKit room:

```
alicia_conversations.id → LiveKit room name
alicia_conversations.livekit_room_name → LiveKit room identifier (typically same as id)
```

When a client connects:
1. Client obtains LiveKit token for room matching conversationId
2. Client joins LiveKit room
3. Server looks up conversation by id to verify and load context
4. Protocol messages flow over LiveKit data channel

### Reconnection Flow

When a client reconnects:
1. Client queries local storage for `conversationId` and last seen stanza ID
2. Client rejoins LiveKit room with name matching `conversationId`
3. LiveKit handles room reconnection and track restoration
4. Client sends Configuration message with `conversationId` and last seen stanza ID over data channel
5. Server queries conversation's stanza tracking fields (`last_client_stanza_id`, `last_server_stanza_id`) to determine what messages need to be replayed
6. Server replays missed messages over the data channel
7. Audio tracks are already restored by LiveKit automatically

## Message Flow Example

Here's how a complete user question flows through protocol and database:

**1. User speaks into microphone (LiveKit audio track)**

Protocol: Audio streams over LiveKit audio track
Database: Optionally stored in `alicia_audio` with `audio_type='input'`

**2. Server transcribes audio (STT)**

Protocol: `Transcription` messages with `final=false` (partials), then `final=true` (final)
Database: Transcription text stored in `alicia_audio.transcription` field

**3. Final transcription becomes UserMessage**

Protocol: `UserMessage` with content from final transcription
Database: Row created in `alicia_messages` with `message_role='user'` and `completion_status='completed'`

**4. Server retrieves relevant memories**

Protocol: `MemoryTrace` messages sent to client
Database:
* Query `alicia_memory` for relevant memories
* Insert rows in `alicia_memory_used` to log usage

**5. Server generates response (streaming)**

Protocol: `StartAnswer` followed by multiple `AssistantSentence` messages
Database:
* `StartAnswer` creates row in `alicia_messages` with `message_role='assistant'` and `completion_status='streaming'`
* Each `AssistantSentence` creates row in `alicia_sentences`
* Final sentence updates `alicia_messages.contents` with full text and sets `completion_status='completed'`

**6. Server performs tool call if needed**

Protocol: `ToolUseRequest` and `ToolUseResult`
Database:
* Request creates row in `alicia_tool_uses` with `status='pending'`
* Result updates row with results and `status='success'` or `status='error'`

**7. Server sends response audio (TTS)**

Protocol: Audio streams over LiveKit audio track, with `AudioChunk` messages providing metadata
Database: Audio data stored in `alicia_sentences` table (linked to individual sentences) or `alicia_audio` table (for complete recordings)

## Traceability and Debugging

By aligning protocol message IDs with database records, every event can be traced:

* **User reports incorrect answer:** Query `alicia_memory_used` by `message_id` to see which memories were retrieved
* **Debugging tool failures:** Query `alicia_tool_uses` by `conversation_id` to see all tool invocations and results
* **Performance analysis:** Query `alicia_sentences` with timestamps to analyze streaming latency
* **Distributed tracing:** Use `messaging.trace_id` from `alicia_meta` to correlate with backend spans in OpenTelemetry

## Summary

The Alicia protocol serializes database interactions in real-time over LiveKit:

* **Insert** user messages via `UserMessage`
* **Insert** assistant messages via `StartAnswer` + `AssistantSentence` (streaming) or `AssistantMessage` (complete)
* **Log** memory usage via `MemoryTrace`
* **Log** tool calls via `ToolUseRequest` / `ToolUseResult`
* **Record** user notes via `UserNote`
* **Track** metadata via `meta` fields in envelopes

The database maintains a complete, queryable record of all conversation activity, while LiveKit provides the real-time transport layer for delivering these events instantly to connected clients.
