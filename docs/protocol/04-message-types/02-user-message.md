### 2. UserMessage (Type 2)

**Purpose:** Carries a user's input message in the conversation. This is the primary message type for user text input, representing what the user says or asks.

**Typical Direction:** Client → Server (via LiveKit data channel).

**Fields:**

* `id` (Text, NanoID): The unique Alicia message ID for this user message. This ID is used to reference the message in the database and by subsequent messages (e.g., an AssistantMessage references this via `previousId`).
* `previousId` (Text, NanoID, optional): The ID of a prior message to which this user message is related. In a typical alternating conversation, `previousId` points to the last AssistantMessage's id. For the first user message in a conversation, `previousId` is null or omitted. When a UserMessage replaces an earlier message (via a ControlVariation/edit flow), `previousId` may point to the message it replaces.
* `conversationId` (Text): The conversation this message belongs to. Matches the envelope's conversationId and corresponds to an existing conversation record (if resuming) or a newly created one.
* `content` (Text): The actual text content of the user's message. Can be a question, command, or any user input, potentially spanning multiple sentences or paragraphs.
* `timestamp` (Int64 or DateTime, optional): A timestamp of when the message was created/sent (epoch milliseconds). May also be conveyed via meta data.
* `attachments` (optional, structure): If the protocol supports non-text attachments in user messages (e.g., images), they can be included here or via meta. The core specification assumes text-only content; attachments are not explicitly covered.

**MessagePack Representation (Informative):**

```
{
  "id": "msg_u1A2B",
  "previousId": "msg_a9X8Y",
  "conversationId": "conv_7H93k",
  "content": "Hello, can you help me find a good Italian restaurant in New York?",
  "timestamp": 1621459200000
}
```

**Semantics:** When the server receives a UserMessage over the LiveKit data channel, it treats it as a new user turn in the conversation. The server responds by sending either an AssistantMessage or a StartAnswer/AssistantSentence sequence. The `stanzaId` for UserMessage (in the envelope) is positive, indicating it originates from the client. The server uses the `id` (NanoID) to log this message in the database (`alicia_user_messages` table or a unified messages table). The `previousId` maintains linkage in the conversation thread.

**Voice Input Flow:** When the user provides input via voice rather than text:

* The client sends a *Configuration* (type 12) or indicator to begin voice input.
* The client streams *AudioChunk* messages (type 4) containing raw audio data over the LiveKit connection.
* The server processes the audio and sends *Transcription* messages (type 9) with recognized text.
* Once speech is fully transcribed, the server (or client) generates a UserMessage containing the final text. The exact responsibility depends on implementation, but ultimately a UserMessage with text content triggers the assistant's reply.

**Database Alignment:** User messages are stored in Alicia's conversation records. The `id` is stored as a primary key (NanoID) in the `alicia_messages` table with a role field indicating "user". The `content` is the text content stored. The `previousId` corresponds to a `previous_message_id` field, referencing another message's NanoID. The conversationId ties the message to the conversation record. Commentary or memory usage related to this message is logged in separate tables. Any tool usage or system actions triggered by this message reference its id.
