### 14. MemoryTrace (Type 14)

**Purpose:** Provides transparency into the assistant's memory system by logging which long-term context and knowledge were retrieved or stored during response generation. This message shows what memories from the RAG (Retrieval-Augmented Generation) system are accessed to inform the current response, including facts about the user, summaries of previous interactions, and data from the knowledge base.

**Typical Direction:** Server → Client

**Transport:** Messages flow over LiveKit data channels as MessagePack-encoded binary data.

**Fields:**

* `id` (Text, NanoID): A unique identifier for this memory trace event, corresponding to a row in the `alicia_memory_used` table.
* `conversationId` (Text): The conversation identifier.
* `previousId` (Text, NanoID): The ID of the message that prompted this memory usage. This typically references the UserMessage that required memory retrieval, though it can also reference the StartAnswer or AssistantMessage ID if the memory is tied to answer generation.
* `memoryId` (Text): An identifier for the specific memory item that was accessed. This references a record in the memory database or vector store, enabling lookups of the full memory content.
* `memoryType` (Text, optional): The category of memory accessed, such as "long_term_note", "profile_attribute", "conversation_summary", "world_fact", or "user_preference".
* `action` (Text): How the memory was used. Values include:
  * `"retrieved"`: Memory was fetched and used for context
  * `"stored"`: New memory was saved
  * `"updated"`: Existing memory was modified
* `content` (Text): A snippet or summary of the memory content that was used. For example, "User's birthday is July 10" or "Prefers technical explanations with code examples." For large memories like documents, this field contains a truncated version or summary.
* `confidence` (Float, optional): When using similarity search or RAG retrieval, this represents the relevance score or confidence level of the retrieved memory (typically 0.0 to 1.0).
* `metadata` (Map, optional): Additional details such as:
  * `retrievalScore`: Similarity score from vector search
  * `tags`: Categories or labels associated with the memory
  * `source`: Origin of the memory (e.g., "user_profile", "conversation_2024_03")

**MessagePack Representation (Example):**

```json
{
  "id": "mem_001",
  "conversationId": "conv_7H93k",
  "previousId": "msg_u1A2B",
  "memoryId": "user_location_pref",
  "memoryType": "profile",
  "action": "retrieved",
  "content": "User is located in New York and prefers local restaurant recommendations",
  "confidence": 0.92,
  "metadata": {
    "tags": ["location", "user_profile"],
    "source": "user_profile"
  }
}
```

**Semantics:** MemoryTrace messages inform the client (and logs) about the assistant's interaction with its memory and knowledge systems. The server sends these messages during response generation, typically after StartAnswer but before the final AssistantSentence.

**Typical Flow:**

1. Server receives UserMessage: "What restaurants should I visit?"
2. Server sends StartAnswer
3. Server sends MemoryTrace #1: Retrieved user location from profile (action: "retrieved")
4. Server sends MemoryTrace #2: Retrieved user's cuisine preferences (action: "retrieved")
5. Server sends AssistantSentence with personalized recommendations

**Use Cases:**

* **Retrieval Transparency**: When the assistant retrieves "User prefers Italian food" from the profile, a MemoryTrace shows this memory was used to inform the answer.
* **Memory Updates**: When the assistant learns something new like "User mentioned they are vegetarian," a MemoryTrace with action "stored" logs this new memory.
* **RAG Context**: During semantic search, multiple MemoryTrace messages show which documents or facts were retrieved, along with their confidence scores.

**Multiple Memory Entries:** If multiple memories are accessed for a single query, the server sends multiple MemoryTrace messages, each with its own unique ID. All may reference the same `previousId` (the user's question).

**Client Handling:** Clients may display memory traces in a debug panel or "show context" feature, allowing users to understand what background information influenced the response. In production UIs, these messages might be hidden from end users but valuable for developers reviewing conversation quality.

**Database Alignment:** MemoryTrace fields directly correspond to the `alicia_memory_used` table with columns:

* `id` (primary key)
* `conversation_id`
* `message_id` (the triggering message, mapped to `previousId`)
* `memory_id` (the referenced memory item)
* `memory_type`
* `action` (retrieved/stored/updated)
* `content` (excerpt or summary)
* `confidence` (retrieval score)
* `created_at` (timestamp)
* `metadata` (JSON/JSONB)

This alignment ensures all memory operations are traceable and auditable through both the live protocol and database records.
