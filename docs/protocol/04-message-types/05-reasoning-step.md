### 5. ReasoningStep (Type 5)

**Purpose:** Exposes the assistant's intermediate reasoning and chain-of-thought process during response generation. This message provides transparency into how the Qwen3 LLM processes the user's request, showing internal thoughts, plans, and decisions before finalizing an answer or invoking tools. ReasoningStep is optional and primarily serves debugging, transparency, and advanced features.

**Typical Direction:** Server → Client

**Transport:** Messages flow over LiveKit data channels as MessagePack-encoded binary data.

**Fields:**

* `id` (Text, NanoID): A unique identifier for this reasoning step message, used when stored or referenced.
* `conversationId` (Text): The conversation identifier.
* `previousId` (Text, NanoID): The ID of the message this reasoning step relates to, typically the UserMessage that prompted this reasoning. When multiple reasoning steps occur in sequence, each step references the same user message ID, forming a logical chain of thoughts about answering that question.
* `content` (Text): The actual reasoning content from the Qwen3 LLM. This contains thoughts like "The user is asking about X, I should consider Y" or "I need to search the database before answering." The content is not necessarily shown in the normal UI but is transmitted for transparency, logging, and optional "show reasoning" modes.
* `stepIndex` (Int32): The sequence number of this step within the current reasoning chain (1, 2, 3, ...). This indicates the order of thoughts as the AI processes the user's query.
* `totalSteps` (Int32, optional): The total number of reasoning steps planned, if known in advance. Since reasoning is typically open-ended, this field is usually omitted. When provided, it enables the client to show progress indicators like "Step 2 of 5."

**MessagePack Representation (Example):**

```json
{
  "id": "rs1_zZ0",
  "conversationId": "conv_7H93k",
  "previousId": "msg_u1A2B",
  "content": "The user asks for Italian restaurants in NYC. I should search the restaurant database for top-rated Italian cuisine in New York City.",
  "stepIndex": 1
}
```

**Semantics:** The server sends ReasoningStep messages during response generation, between receiving the user's question and sending the final answer. A typical flow looks like:

1. Server receives UserMessage
2. Server sends ReasoningStep #1: "Analyzing the question..."
3. Server sends ReasoningStep #2: "Decided to look up relevant information..."
4. Server sends ToolUseRequest to search database
5. Server receives ToolResult
6. Server sends ReasoningStep #3: "Based on search results, I can now formulate an answer..."
7. Server sends StartAnswer and AssistantSentence messages

This sequence provides insight into the Qwen3 LLM's internal decision-making process. Multiple ReasoningStep messages can be sent for a single user query, each with an incrementing `stepIndex`.

**Configuration and Display:** The inclusion of reasoning steps is typically controlled by system configuration or user preferences. Some users want to see the AI's thought process, while others prefer just the final answer. The client may:

* Display reasoning steps in a debug console or "thinking" panel
* Show them only when "show reasoning" mode is enabled
* Completely ignore them in normal chat UI
* Store them for later review or analytics

The protocol transmits these messages regardless of client display preferences. Clients that don't support ReasoningStep simply ignore these messages without affecting the core conversation flow.

**Database Alignment:** ReasoningStep messages may be stored in a dedicated table like `alicia_reasoning_steps` or included as a special message type in the main messages table. Each reasoning step is linked to its conversation and the user message that triggered it via the `id` (NanoID) and `previousId` fields. Some implementations may not store reasoning steps permanently due to privacy or storage considerations, instead logging them transiently or only in debug environments.
