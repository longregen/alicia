### 16. AssistantSentence (Type 16)

**Purpose:** Delivers a segment (typically a sentence or small chunk) of the assistant's answer as part of a streamed response. This message type incrementally sends the assistant's reply in real-time, breaking the reply into manageable units (often sentences) that can be displayed or played as they arrive. For long or token-by-token generated answers, these chunks allow the user to start reading (or hearing) without waiting for the full answer.

Each AssistantSentence is an atomic unit of the assistant's answer with an ordering sequence number to reconstruct the full answer in order.

**Typical Direction:** Server → Client (via LiveKit data channel during answer streaming).

**Fields:**

* `id` (Text, NanoID, optional): Since each AssistantSentence is part of a larger assistant message, individual sentences may not need a unique NanoID in the database context (the whole answer has one ID given in StartAnswer). This field can be omitted, or a composite identifier like answerId + "-1" can be used if needed for storage or reference.
* `previousId` (Text, NanoID): References the StartAnswer's id (i.e., the assistant message id that this sentence is part of). This links the sentence to the overarching assistant answer, which is crucial for identifying which answer these sentences belong to, especially if multiple answers could theoretically be interleaved (though unusual in synchronous scenarios).
* `conversationId` (Text): Conversation ID.
* `sequence` (Int32): The sequence number of this sentence within the answer. The first sentence chunk is sequence 1, the next is 2, and so on, incrementing by 1. This is *not* the same as the envelope's stanzaId. For example, a StartAnswer might have stanzaId -2, then AssistantSentence #1 has stanzaId -3 (but sequence=1), the next has stanzaId -4 (sequence=2), etc. This field ensures the client can order sentences properly in case of any reordering or latency issues, though delivery order is typically preserved.
* `text` (Text): The text content of this sentence chunk, representing a portion of the assistant's answer. Ideally ends at a sentence or logical boundary, though exact segmentation varies. The intention is to break at coherent boundaries when possible.
* `audio` (Data or Track Reference, optional): Audio data or reference for this sentence, if the assistant's answer is provided in spoken form. When the assistant is speaking, each sentence may have a corresponding audio clip. This field can:
  * Contain binary audio data (e.g., WAV or Opus) for this sentence's speech
  * Reference an audio track on the LiveKit connection for synchronized playback
  * Be omitted for text-only responses

  If audio is present, the AssistantSentence may be delayed until TTS for that sentence is ready, ensuring text and audio remain synchronized.
* `isFinal` (Bool, optional): Indicates whether this is the final sentence of the answer. If true, the assistant has completed the answer with this chunk. If false or not present, the client knows more content is coming. This can be deduced from the absence of subsequent messages, but an explicit flag helps in cases like connection issues where the client needs to know if it received the final chunk.
* `sentiment` or `tone` (optional, advanced): Possibly conveys how the sentence is spoken (like tone or sentiment analysis), but typically not needed.

**MessagePack Representation (Informative):**

```
{
  "previousId": "msg_a9X8Y",
  "conversationId": "conv_7H93k",
  "sequence": 2,
  "text": "One popular spot is Luigi's Trattoria, which has a 4.5 star rating.",
  "isFinal": false
}
```

**Semantics:** The server sends a sequence of AssistantSentence messages after a StartAnswer:

1. StartAnswer (ID X) – stanza -2.
2. AssistantSentence (prevId = X, sequence=1, text="...first part...") – stanza -3.
3. AssistantSentence (prevId = X, sequence=2, text="...second part...") – stanza -4.
4. ... and so on, until:
5. AssistantSentence (prevId = X, sequence=N, text="...last part...", isFinal=true) – stanza -(N+2).

Optionally, after the final sentence, the server may send an Acknowledgement or simply consider the answer complete.

The client appends these texts in order to display the full answer progressively. If audio is included, the client plays each segment's audio in sequence (potentially with slight overlaps if desired, though usually sequential).

**Dealing with Partial Words:** If the model streams token by token, it may not have full sentences at each moment. The server buffers tokens until a sentence is complete (e.g., until it outputs a period or significant pause) then sends an AssistantSentence. This creates smoother output. If the model stops mid-sentence due to length limits, the incomplete text can be sent as the final chunk. Segmentation is implementation-dependent; the client simply assembles what it receives.

**User Interruption (Stop):** If the user interrupts, the server may end the sentence abruptly or send a final chunk with available text and isFinal=true (or send an ErrorMessage or special marker). If Stop is handled, the server stops sending further chunks and may mark the last one as final even if incomplete. The conversation then returns to user input.

**Relationship to AssistantMessage:** When not streaming, the server sends one AssistantMessage (type 3). When streaming, it uses StartAnswer + AssistantSentence. These modes are mutually exclusive for the same user query. A server could theoretically send a complete AssistantMessage at the very end with the entire content as a single message for integrity checking or storage, but this is redundant for the client. The database can reconstruct the full answer from the pieces, making a separate AssistantMessage unnecessary.

**Database Alignment:** AssistantSentence messages themselves are typically not individually stored in the database. Instead, the whole assembled answer is stored as one entry (the assistant message with id given in StartAnswer). Individual sentences might be stored in a separate table for debugging purposes, but this is not typical.

The database stores the answer id X with content representing the concatenation of all sentences. Any commentary or memory usage referencing that answer has the id available from the start (via StartAnswer).

AssistantSentence aligns with runtime streaming rather than being a persistent entity. The fields `sequence` and partial text typically do not appear in the database (except in debug logs if needed).

In summary, AssistantSentence allows the protocol to deliver the assistant's message in ordered pieces with optional synchronized audio, fulfilling the requirement for a "split text/audio unit with sequence number".
