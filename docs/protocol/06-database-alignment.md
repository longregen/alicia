## Database Alignment

The Alicia protocol is designed hand-in-hand with the Alicia conversation database schema. This ensures that every message in the protocol correlates directly with stored data, enabling consistency between live interactions and persistent records. This section outlines how the key protocol elements align with the database tables.

## Core Database Tables

### `alicia_conversations`

This table stores conversation metadata and LiveKit integration details.

**Key Fields:**
* `id` (NanoID) — The unique conversation identifier, also used as the LiveKit room name
* `user_id` — The user who owns this conversation
* `livekit_room_name` — The LiveKit room identifier (typically matches the conversation id)
* `created_at` — When the conversation started
* `updated_at` — Last activity timestamp
* `status` — Active, archived, or expired
* `language` — Preferred language for the conversation

**Protocol Mapping:**
* When a client sends a Configuration message with no `conversationId`, the server creates a new row in this table
* The generated `id` becomes the `conversationId` in all subsequent protocol messages
* The server creates a LiveKit room with `livekit_room_name` matching the conversation id
* When a client reconnects with a known `conversationId`, the server looks up this table to verify the conversation exists and retrieve the room name

### `alicia_messages`

This table stores all user and assistant messages in conversations.

**Key Fields:**
* `id` (NanoID) — The unique message identifier
* `conversation_id` — Foreign key to `alicia_conversations`
* `previous_message_id` — Foreign key to the previous message (forms the conversation chain)
* `role` — 'user' or 'assistant'
* `content` — The message text content
* `created_at` — Message timestamp
* `stanza_sequence` — Absolute value of stanzaId for ordering
* `status` — Active, edited, replaced, or deleted

**Protocol Mapping:**

**UserMessage (type 2):**
* Creates a new row with `role='user'`
* `id` field from protocol maps to `id` in database
* `previousId` field maps to `previous_message_id`
* `content` field maps to `content`
* `stanza_sequence` stores absolute value of the stanzaId

**AssistantMessage (type 3):**
* Creates a new row with `role='assistant'`
* `id` field from protocol maps to `id` in database
* `previousId` field maps to `previous_message_id` (references the UserMessage it responds to)
* `content` field maps to `content`

**StartAnswer (type 13) + AssistantSentence (type 16):**

When the server uses streaming:

1. **StartAnswer** triggers insertion of a row with `role='assistant'`, `id` from the message, `previous_message_id` referencing the user message, and `content` initially set to empty string or placeholder like "[streaming]"
2. As **AssistantSentence** chunks arrive, they are stored in `alicia_sentences` (see below) and the content field is built up
3. When the final AssistantSentence arrives (`isFinal=true`), the assembled full content replaces any placeholder in `alicia_messages.content`

This approach ensures the assistant message is visible in the database immediately, even during streaming.

### `alicia_sentences`

This table stores individual sentence chunks for streaming assistant responses.

**Key Fields:**
* `id` (auto-generated or derived from stanzaId)
* `message_id` — Foreign key to `alicia_messages` (the assistant message being streamed)
* `conversation_id` — Foreign key to `alicia_conversations`
* `sequence` — The sentence sequence number within this message
* `text` — The sentence content
* `is_final` — Boolean indicating if this is the final sentence
* `created_at` — Timestamp

**Protocol Mapping:**

Each **AssistantSentence (type 16)** message maps to a row:
* `previousId` in the protocol indicates which assistant message this sentence belongs to (maps to `message_id`)
* `sequence` field maps directly to `sequence`
* `text` field maps to `text`
* `isFinal` field maps to `is_final`

This table allows the system to:
* Replay streaming responses chunk by chunk on reconnection
* Analyze streaming patterns and latency
* Reconstruct the exact streaming sequence for debugging

### `alicia_memory`

This table stores memory items (facts, preferences, context) that can be retrieved during conversations.

**Key Fields:**
* `id` (NanoID) — The unique memory identifier
* `user_id` — The user this memory belongs to
* `memory_type` — Category: 'profile', 'fact', 'preference', 'context'
* `content` — The memory content (text)
* `embedding` — Vector embedding for semantic search
* `created_at` — When this memory was stored
* `accessed_at` — Last time this memory was retrieved

**Protocol Mapping:**

This table is queried during conversation processing (not directly created by protocol messages). However, when memory is used:

**MemoryTrace (type 14)** logs the usage:
* The server retrieves memory from this table during processing
* It sends a MemoryTrace message referencing the `memoryId` from this table
* This creates a record in `alicia_memory_used` (see below)

### `alicia_memory_used`

This table logs which memories were used in which conversations and messages.

**Key Fields:**
* `id` (NanoID) — Unique usage record identifier
* `conversation_id` — Foreign key to `alicia_conversations`
* `message_id` — Foreign key to `alicia_messages` (which message triggered this retrieval)
* `memory_id` — Foreign key to `alicia_memory` (which memory was used)
* `memory_type` — Category of memory used
* `content_snippet` — Brief excerpt of the memory content
* `usage` — How it was used: 'retrieved', 'injected', 'referenced'
* `created_at` — Timestamp

**Protocol Mapping:**

Each **MemoryTrace (type 14)** message creates a row:
* `id` from protocol maps to `id`
* `previousId` indicates the message context (maps to `message_id`)
* `memoryId` maps to `memory_id`
* `memoryType` maps to `memory_type`
* `content` maps to `content_snippet`
* `usage` maps to `usage`

This enables:
* Tracking which memories influenced which responses
* Debugging memory retrieval issues
* Analyzing memory effectiveness over time

### `alicia_commentaries`

This table stores user feedback, system notes, and commentary on messages.

**Key Fields:**
* `id` (NanoID) — Unique commentary identifier
* `conversation_id` — Foreign key to `alicia_conversations`
* `message_id` — Foreign key to `alicia_messages` (the message being commented on)
* `author_role` — 'user' or 'system'
* `comment_type` — 'feedback', 'correction', 'system_note', 'evaluation'
* `content` — The commentary text
* `created_at` — Timestamp

**Protocol Mapping:**

Each **Commentary (type 15)** message creates a row:
* `id` from protocol maps to `id`
* `previousId` indicates the message being commented on (maps to `message_id`)
* `authorRole` maps to `author_role`
* `commentType` maps to `comment_type`
* `content` maps to `content`

This supports:
* User feedback collection ("This answer was helpful")
* System-generated evaluations of response quality
* Corrections or follow-up notes

### `alicia_tool_uses`

This table logs all tool invocations during conversations.

**Key Fields:**
* `id` (NanoID) — Unique tool use identifier
* `conversation_id` — Foreign key to `alicia_conversations`
* `message_id` — Foreign key to `alicia_messages` (which message triggered this tool use)
* `tool_name` — The tool that was invoked
* `needs_response` — Boolean indicating if response was expected
* `parameters` — JSON object with tool parameters
* `result_data` — JSON object with tool results (populated when result arrives)
* `success` — Boolean indicating if tool execution succeeded
* `status` — 'requested', 'completed', 'failed'
* `created_at` — Request timestamp
* `completed_at` — Result timestamp

**Protocol Mapping:**

**ToolUseRequest (type 6):**
* Creates a new row with `status='requested'`
* `id` from protocol maps to `id`
* `previousId` indicates context (maps to `message_id`)
* `toolName` maps to `tool_name`
* `needsResponse` maps to `needs_response`
* `parameters` maps to `parameters` (JSON)

**ToolUseResult (type 7):**
* Updates the existing row (matched by finding the request with matching `id` or by `previousId` referencing the request)
* `success` from protocol maps to `success`
* `data` maps to `result_data` (JSON)
* Sets `status='completed'` or `status='failed'`
* Sets `completed_at` to current timestamp

### `alicia_transcriptions`

This table stores transcription events for voice input.

**Key Fields:**
* `id` (auto-generated)
* `conversation_id` — Foreign key to `alicia_conversations`
* `user_message_id` — Foreign key to `alicia_messages` (the user message this contributes to, may be null for partials)
* `text` — The transcribed text
* `is_final` — Boolean indicating if this is the final transcription
* `confidence` — Transcription confidence score (0.0 to 1.0)
* `created_at` — Timestamp

**Protocol Mapping:**

Each **Transcription (type 9)** message creates a row:
* `text` field maps to `text`
* `isFinal` field maps to `is_final`
* When `isFinal=true`, the transcription is typically finalized into a UserMessage, which references this table entry

This table is useful for:
* Debugging ASR accuracy
* Showing real-time transcription progress in the UI
* Analyzing transcription latency

### `alicia_meta`

This table stores arbitrary key-value metadata for conversations and messages.

**Key Fields:**
* `id` (auto-generated)
* `conversation_id` — Foreign key to `alicia_conversations`
* `message_id` — Foreign key to `alicia_messages` (nullable, for conversation-level meta)
* `key` — The metadata key (e.g., "messaging.trace_id", "source", "model")
* `value` — The metadata value (text)
* `created_at` — Timestamp

**Protocol Mapping:**

The `meta` field in the protocol envelope maps to rows in this table:
* Each key-value pair in the `meta` map creates a row
* `conversation_id` is derived from the conversationId in the envelope
* `message_id` is derived from the message `id` if applicable
* Special keys like `messaging.trace_id` and `messaging.span_id` are stored here for distributed tracing

Examples:
* `meta: {"source": "microphone"}` on a UserMessage creates a row with `key='source'`, `value='microphone'`
* `meta: {"model": "gpt-4", "responseTime": "123ms"}` on an AssistantMessage creates two rows

This flexible structure allows:
* Storing OpenTelemetry trace IDs for debugging
* Tracking model versions and parameters
* Recording client versions and platforms
* Custom application-specific metadata

### `alicia_audio_chunks`

This table stores audio chunks for voice conversations (optional, depending on whether audio is persisted).

**Key Fields:**
* `id` (auto-generated)
* `conversation_id` — Foreign key to `alicia_conversations`
* `message_id` — Foreign key to `alicia_messages` (if associated with a specific message)
* `sequence` — Chunk sequence number
* `audio_data` — Binary audio data (or reference to object storage)
* `format` — Audio format (e.g., "audio/pcm", "audio/opus")
* `sample_rate` — Sample rate in Hz
* `created_at` — Timestamp

**Protocol Mapping:**

Each **AudioChunk (type 10)** message can optionally create a row:
* `previousId` indicates context (maps to `message_id`)
* `data` field contains the binary audio
* `format` field maps to `format`
* `sampleRate` field maps to `sample_rate`

Note: Many implementations may not persist audio chunks to the database, instead processing them in real-time and only storing transcriptions. This table is optional.

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
1. Client queries local storage for `conversationId` and `lastSequenceSeen`
2. Client rejoins LiveKit room with name matching `conversationId`
3. LiveKit handles room reconnection and track restoration
4. Client sends Configuration message with `conversationId` and `lastSequenceSeen` over data channel
5. Server queries `alicia_messages` table: `SELECT * FROM alicia_messages WHERE conversation_id = ? AND stanza_sequence > ? ORDER BY stanza_sequence`
6. Server replays missed messages over the data channel
7. Audio tracks are already restored by LiveKit automatically

## Message Flow Example

Here's how a complete user question flows through protocol and database:

**1. User speaks into microphone (LiveKit audio track)**

Protocol: `AudioChunk` messages stream over LiveKit audio track
Database: Optionally stored in `alicia_audio_chunks`

**2. Server transcribes audio (STT)**

Protocol: `Transcription` messages with `isFinal=false` (partials), then `isFinal=true` (final)
Database: Rows created in `alicia_transcriptions`

**3. Final transcription becomes UserMessage**

Protocol: `UserMessage` with content from final transcription
Database: Row created in `alicia_messages` with `role='user'`

**4. Server retrieves relevant memories**

Protocol: `MemoryTrace` messages sent to client
Database:
* Query `alicia_memory` for relevant memories
* Insert rows in `alicia_memory_used` to log usage

**5. Server generates response (streaming)**

Protocol: `StartAnswer` followed by multiple `AssistantSentence` messages
Database:
* `StartAnswer` creates row in `alicia_messages` with `role='assistant'`
* Each `AssistantSentence` creates row in `alicia_sentences`
* Final sentence updates `alicia_messages.content` with full text

**6. Server performs tool call if needed**

Protocol: `ToolUseRequest` and `ToolUseResult`
Database:
* Request creates row in `alicia_tool_uses` with `status='requested'`
* Result updates row with results and `status='completed'`

**7. Server sends response audio (TTS)**

Protocol: `AudioChunk` messages stream over LiveKit audio track
Database: Optionally stored in `alicia_audio_chunks`

## Traceability and Debugging

By aligning protocol message IDs with database records, every event can be traced:

* **User reports incorrect answer:** Query `alicia_memory_used` by `message_id` to see which memories were retrieved
* **Debugging tool failures:** Query `alicia_tool_uses` by `conversation_id` to see all tool invocations and results
* **Performance analysis:** Query `alicia_sentences` with timestamps to analyze streaming latency
* **Distributed tracing:** Use `messaging.trace_id` from `alicia_meta` to correlate with backend spans in OpenTelemetry

## Summary

The Alicia protocol essentially serializes database interactions in real-time over LiveKit:

* **Insert** user message via `UserMessage`
* **Insert** assistant message via `StartAnswer` + `AssistantSentence` (streaming) or `AssistantMessage` (complete)
* **Log** memory usage via `MemoryTrace`
* **Log** tool calls via `ToolUseRequest` / `ToolUseResult`
* **Record** commentary via `Commentary`
* **Track** metadata via `meta` fields in envelopes

The database maintains a complete, queryable record of everything that happens in the conversation, while LiveKit provides the real-time transport layer for delivering these events instantly to connected clients.
