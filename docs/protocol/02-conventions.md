## Conventions and Terminology

### LiveKit Terminology

**Room:** A LiveKit room represents a single conversation session. Each room has a unique identifier and contains participants exchanging messages and audio. Conversations map one-to-one with rooms.

**Participant:** An entity within a room, either a **client** (the end-user application) or an **agent** (the AI assistant service). Participants communicate via data channels and audio tracks.

**Data Channel:** LiveKit's reliable, ordered message transport mechanism. The Alicia protocol sends all text messages, tool invocations, and control signals through data channels as MessagePack-encoded binary data.

**Audio Track:** LiveKit's real-time audio streaming transport. Clients publish audio tracks for voice input; agents publish audio tracks for voice output (text-to-speech). Audio tracks enable low-latency voice interaction.

### Protocol Terminology

**Conversation:** A dialogue session between a user and the assistant, corresponding to a LiveKit room. Each conversation has a unique `conversationId` (typically a NanoID or UUID) that identifies it in the Alicia database and enables conversation resumption.

**Message:** A single unit of communication within a conversation. Each message has a **type** (user input, assistant response, tool request, etc.) and carries specific data. Every message is wrapped in an **Envelope** structure.

**Envelope:** A binary frame that wraps each message with metadata and identifiers. The envelope includes:
- A message type code
- A **stanza ID** for ordering
- Optional metadata in a `meta` field
- Optional tracing information (`otel_span`, `otel_span_counter`)

All messages MUST be sent within an envelope. The envelope format is detailed in the next section.

**Stanza ID:** A signed 32-bit integer that uniquely identifies a message within a conversation and determines its order. Stanza IDs are **monotonically increasing** (in absolute value) per conversation:

- **Client messages** use **positive stanza IDs** (1, 3, 5, ...)
- **Agent messages** use **negative stanza IDs** (-2, -4, -6, ...)

The absolute value increases by 1 for each new message, ensuring total ordering while encoding sender role in the sign. Clients MUST assign increasing positive stanza IDs; agents MUST assign increasing negative stanza IDs.

**Message ID (NanoID):** Each conversation message has a stable unique identifier called the **Alicia message ID**. Alicia uses NanoIDs (typically 21-character secure random strings) for message identifiers in its database. Protocol messages that correspond to stored conversation entries (user messages, assistant answers, tool results, etc.) MUST include their NanoID in the message content and (when applicable) a `previousId` field referencing the prior related message's NanoID. This allows mapping protocol messages to database records and linking them via `previousId` as defined in the Alicia schema.

NanoIDs provide durable identity in the database; stanza IDs provide transient ordering for real-time communication.

**Metadata (Meta):** The `meta` field within an envelope carries arbitrary key-value pairs alongside a message. Metadata can include timestamps, client device information, or custom application data. The Alicia database stores these key-value pairs in the `alicia_meta` table. Two specific metadata fields support distributed tracing: `otel_span` and `otel_span_counter`.

**MessagePack:** The binary encoding format for all messages. MessagePack is a schema-less, efficient serialization system that produces compact binary representations. All messages and envelopes are serialized using MessagePack before transmission through LiveKit data channels. Implementations MUST use MessagePack to encode and decode messages according to the structures defined in this specification.

### Message Direction Notation

This specification uses arrow notation to indicate message direction:

- **C→S** (Client to Server): Messages sent from client to agent
- **S→C** (Server to Client): Messages sent from agent to client

Examples:
- `UserMessage (C→S)`: User input from client to agent
- `AssistantMessage (S→C)`: Assistant response from agent to client
- `ToolUseRequest (S→C)`: Tool invocation request from agent to client

### RFC 2119 Keywords

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Other Terminology

**Tool:** An external function or service that the assistant invokes during a conversation (e.g., web search, database query, calculator). `ToolUseRequest` and `ToolUseResult` messages handle tool invocations and results.

**Reasoning Step:** A step in the assistant's reasoning or chain-of-thought that may be exposed via a message for transparency or debugging.

**Conversation Resume:** Re-establishing a conversation session after disconnection. The client provides a `conversationId` and last seen message indices to continue from the appropriate point in the conversation history.

All multi-word field names in this document use `camelCase` for consistency with typical code conventions.
