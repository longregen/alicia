## Message Types

The Alicia protocol defines a set of **message types** that cover all events and data exchanges in a conversation. Each message type has a name and a fixed numeric code (ID) used in the envelope's `type` field.

### Transport Layer

Protocol messages flow over **LiveKit data channels**, while voice audio travels over **dedicated audio tracks**. This separation ensures efficient handling of both structured protocol data and real-time voice streams.

### Message Categories

The 16 message types are organized into the following categories:

#### Conversation Messages (Types 2-3)
- [UserMessage](./02-user-message.md) – ID = 2 – User's text input
- [AssistantMessage](./03-assistant-message.md) – ID = 3 – Complete assistant response (non-streaming)

#### Audio Messages (Type 4, 9, 16)
- [AudioChunk](./04-audio-chunk.md) – ID = 4 – Raw audio data segment
- [Transcription](./09-transcription.md) – ID = 9 – Speech-to-text output
- [AssistantSentence](./16-assistant-sentence.md) – ID = 16 – Streaming assistant text with optional audio

#### Control Messages (Types 10-11, 13)
- [ControlStop](./10-control-stop.md) – ID = 10 – Stop current operation
- [ControlVariation](./11-control-variation.md) – ID = 11 – Edit/vary previous message
- [StartAnswer](./13-start-answer.md) – ID = 13 – Begin streaming response

#### Tool Messages (Types 5-7)
- [ReasoningStep](./05-reasoning-step.md) – ID = 5 – Internal reasoning trace
- [ToolUseRequest](./06-tool-use-request.md) – ID = 6 – Request to execute a tool
- [ToolUseResult](./07-tool-use-result.md) – ID = 7 – Tool execution result

#### Meta Messages (Types 1, 8, 12, 14-15)
- [ErrorMessage](./01-error-message.md) – ID = 1 – Error notification
- [Acknowledgement](./08-acknowledgement.md) – ID = 8 – Confirm receipt
- [Configuration](./12-configuration.md) – ID = 12 – Session configuration
- [MemoryTrace](./14-memory-trace.md) – ID = 14 – Memory retrieval log
- [Commentary](./15-commentary.md) – ID = 15 – Assistant's internal commentary

### Common Fields

Many message types share these fields for consistency:

* **`id` (Text, NanoID):** Unique message identifier as stored in the database. Present for messages recorded in conversation history (user messages, assistant messages, commentary, etc.).
* **`previousId` (Text, NanoID, optional):** NanoID of a related previous message. References the message to which this message is a direct response or is logically linked. For example, an AssistantMessage's `previousId` points to the UserMessage it responds to; a ToolUseResult's `previousId` points to the ToolUseRequest it fulfills.
* **`conversationId` (Text):** Associates the message with its conversation. Matches the envelope's conversationId to ensure proper routing even when processed out of context.
* **Timestamps or ordering info:** The protocol relies on stanzaId for ordering. Timestamps can be included via meta entries or in the message body if the database requires them.

### Database Alignment

The structure of each message type aligns with Alicia's database schema. Fields often correspond directly to columns in the conversation tables. Messages that represent transient control signals (like AudioChunk or ControlStop) may omit the `id` field or use it differently, as they do not map to stored records.

Each message type's detailed documentation specifies its fields, direction, and relationship to persistent storage.
