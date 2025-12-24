### 3. AssistantMessage (Type 3)

**Purpose:** Conveys a complete assistant response message in a single, non-streaming delivery. This is the counterpart to UserMessage for the assistant's turns. In streaming scenarios, the server uses *StartAnswer* and *AssistantSentence* messages (types 13 and 16) instead. AssistantMessage is used when the complete answer is available before sending or when the client does not support streaming.

**Typical Direction:** Server → Client (via LiveKit data channel).

**Fields:**

* `id` (Text, NanoID): The unique ID for this assistant message, as stored in the database.
* `previousId` (Text, NanoID, optional): The ID of the message to which this responds. Typically points to the user's last message ID, creating a reference from the assistant message record to the user message it answers.
* `conversationId` (Text): The conversation ID. Matches the envelope's conversationId and the related user message's conversationId.
* `content` (Text): The full text content of the assistant's reply. Can be a lengthy answer spanning multiple paragraphs.
* `timestamp` (optional): When this answer was generated/sent, if needed. Can also be conveyed via meta.
* `state` (optional): Indicates whether this message is complete or partial. In non-streaming contexts, an AssistantMessage is typically only sent when the answer is complete.

**MessagePack Representation (Informative):**

```
{
  "id": "msg_a9X8Y",
  "previousId": "msg_u1A2B",
  "conversationId": "conv_7H93k",
  "content": "I found several Italian restaurants in New York. Luigi's Trattoria has a 4.5 star rating and Pasta Palace has 4.3 stars. Would you like more details about either of these?",
  "timestamp": 1621459210000
}
```

**Semantics:** The AssistantMessage provides the user with the assistant's complete response. When streaming is not used, the server sends an AssistantMessage when the answer is ready, and the client displays it. When streaming *is* used, the server uses StartAnswer and AssistantSentence messages instead of sending a full AssistantMessage.

**Mutual Exclusion with StartAnswer:** A server MUST NOT send both AssistantMessage and StartAnswer for the same user query. The server MUST choose one of two approaches:

* **Non-streaming mode**: Send a complete AssistantMessage (type 3) with full content once the answer is ready.
* **Streaming mode**: Send a StartAnswer (type 13) followed by one or more AssistantSentence messages (type 16), with no separate AssistantMessage.

Clients MUST be able to handle receiving either approach. Sending both forms for the same answer creates ambiguity and is explicitly prohibited by this specification.

**When to Use Each Approach:**

* **Use AssistantMessage when:**
  * The client does not support streaming (indicated by absence of "streaming" or "partial_responses" in Configuration features)
  * The complete response is generated before sending (e.g., using non-streaming LLM APIs)
  * Latency is not a critical concern
  * The response is short and does not benefit from progressive display

* **Use StartAnswer + AssistantSentence when:**
  * The client supports streaming (indicated by "streaming" or "partial_responses" in Configuration features)
  * The response is generated progressively (e.g., using streaming LLM APIs)
  * Low time-to-first-token is important for user experience
  * The response may be long and benefits from progressive display
  * Voice synthesis needs to begin before the full text is ready

**Feature Negotiation:** During the Configuration handshake, clients SHOULD indicate their streaming support by including "streaming" or "partial_responses" in the `features` field. The server SHOULD use this information to determine which message format to use. If the client does not declare streaming support, the server SHOULD default to non-streaming mode (AssistantMessage only).

**Database Alignment:** Assistant messages are stored similarly to user messages in the conversation records (with a role indicating "assistant"). The `id` (NanoID) is the primary key for that message, and `previousId` links to the user message's id. In the Alicia database, assistant messages typically reside in the same table as user messages (with a role or sender column differentiating) or a parallel table. The content is stored along with timestamps. When streaming is used, the system stores the assistant's response in assembled form once it's complete (after streaming concludes, the final text is committed to the database). The *StartAnswer* message (type 13) carries the same `id` that is used for the assistant's message record, allowing early reference.
