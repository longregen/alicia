### 13. StartAnswer (Type 13)

**Purpose:** Initiates a streaming assistant response. This message serves as a preamble indicating the assistant is beginning to formulate a response to a user message. It allocates an answer ID and provides initial metadata before content streams via *AssistantSentence* chunks. The message signals "Answer is starting now".

**Typical Direction:** Server → Client (via LiveKit data channel, sent just before the first chunk of the assistant's answer).

**Fields:**

* `id` (Text, NanoID): The unique ID for the assistant's answer/message that is being started. This is the assistant message's ID, even though the full content is not yet available. By sending it upfront, the server allows the client to know the upcoming answer's identifier. This id is used in the AssistantMessage record in the database and allows linking any commentary or memory uses to this answer.
* `previousId` (Text, NanoID): The ID of the UserMessage that this answer is responding to. Points to the user's message ID.
* `conversationId` (Text): Conversation ID.
* `answerType` (Text or Enum, optional): Describes the format or style of the answer. Examples: "text", "voice", "text+voice", "visual". If omitted, defaults to text. Helps the client prepare (e.g., if voice, the client shows a speaker icon or prepares an audio player).
* `plannedSentenceCount` (Int32, optional): The number of sentence chunks that will be sent, if known. Not common, as generation is streaming and open-ended. Can be used for progress indicators when the system uses pre-segmented text.
* `additionalContext` (optional): Placeholder for any other information like "source of answer" or initial partial content. Usually not needed.

**MessagePack Representation (Informative):**

```
{
  "id": "msg_a9X8Y",
  "previousId": "msg_u1A2B",
  "conversationId": "conv_7H93k",
  "answerType": "text+voice",
  "plannedSentenceCount": 4
}
```

**Semantics:** Upon receiving StartAnswer, the client knows the assistant's response is starting. The client can display a typing indicator ("Assistant is responding…") or allocate a message bubble in the UI ready to be filled by subsequent content.

The StartAnswer is immediately followed by one or more AssistantSentence messages (type 16) that contain the actual content in segments. Each AssistantSentence references this StartAnswer via its `id` and carries sequential partial content.

**Mutual Exclusion with AssistantMessage:** A server MUST NOT send both StartAnswer and AssistantMessage (type 3) for the same user query. When StartAnswer is used to begin an answer, the complete response MUST be delivered through AssistantSentence messages only. No separate AssistantMessage should be sent afterward. See the AssistantMessage (type 3) documentation for detailed guidance on when to use each approach.

If the assistant cannot complete an answer after sending StartAnswer (e.g., a tool fails catastrophically or the user stops it), the StartAnswer stands as an empty answer placeholder. The client may show nothing or skip the incomplete answer.

In some implementations, StartAnswer also triggers the client to clear any older suggestions or begin timers.

**Why StartAnswer Exists:**

* **Separation of concerns**: The ID for the answer is allocated and communicated before content is fully generated. This is valuable for logging and linking (e.g., memory retrieval or commentary tied to this answer can reference its id even as content streams).
* **Metadata delivery**: Allows sending metadata about the answer (like answerType or which model produced it) before content arrives.
* **Turn demarcation**: Clearly marks the boundary between the user turn and assistant turn in the message flow.

**Database Alignment:** The StartAnswer corresponds to the creation of a new assistant message entry in the conversation before content is available. In Alicia's database, when the assistant begins answering, a new message record is inserted with the given ID and initially empty or null content, to be updated as content arrives or once complete. This allows linking memory usage or commentary to the message record even while it's being formed.

Alternatively, the database may wait to insert until the answer is complete. However, since commentary and memory traces may arrive during answer generation (e.g., a memoryTrace is logged when the assistant gathers context), having the message id upfront helps attach those logs.

StartAnswer's fields align with the message table: an entry with id, conversationId, and previousId (pointing to the user message) is established. The content remains empty until finalized, or partial content is updated progressively depending on database design.
