### 10. ControlStop (Type 10)

**Purpose:** A control command to halt the assistant's current action, typically used to stop a response that is in progress. This is equivalent to the user pressing a "Stop" or "Cancel" button mid-response to stop LLM generation and/or TTS playback.

**Typical Direction:** Client → Server (via LiveKit data channel). The user's client issues this command to instruct the server to cease whatever it is doing for the conversation (commonly, stop generating more AssistantSentence messages or stop TTS audio playback).

**Fields:**

* `conversationId` (Text): Conversation ID (maps to LiveKit room name).
* `targetId` (Text, NanoID, optional): The ID of the message or activity to stop. This points to the `id` of a StartAnswer or AssistantMessage that is in progress. If provided, it clarifies which generation to stop (useful if multiple answers or threads are in progress, though this is uncommon in a single conversation). If omitted, the latest assistant message is implicitly the target.
* `reason` (Text, optional): A short description or code explaining why to stop. For example, "user_clicked_stop", "timeout", or "content_filter_triggered". If the user manually triggered the stop, this may not be necessary, but if the stop is automated (like a safety cutoff), the reason should be provided.
* `stopType` (Enum, optional): A code indicating what to stop. Possible values:
  * **`"generation"`** – Stop the AI from generating more text. If it is currently streaming an answer, end it.
  * **`"speech"`** – Stop audio playback (if using TTS).
  * **`"all"`** – Halt all activity completely (both generation and speech).

  If omitted, the default is to stop content generation.

**MessagePack Representation (Informative):**

```
{
  "conversationId": "conv_7H93k",
  "targetId": "msg_a9X8Y",
  "reason": "user_clicked_stop",
  "stopType": "generation"
}
```

**Semantics:** When the server receives ControlStop, it immediately attempts to stop sending any further content for the current answer. This involves interrupting the generation process (if the assistant AI is mid-sentence, it aborts the process). The server then finalizes the assistant's answer at that point. It may send an AssistantSentence marked as final (if partial answer was already streaming) or simply cease sending further sentences.

The server SHOULD send an Acknowledgement (type 8) in response to confirm that it received and processed the stop command. This provides the user with assurance that the stop took effect.

The server SHOULD ensure that any partial answer already sent remains properly marked in the conversation. In the database, the assistant message is stored as whatever was generated so far, marked as incomplete or truncated. Alternatively, if the implementation does not store incomplete answers, it may discard the partial content, but typically partial content is saved.

From the client's perspective, after sending ControlStop, it expects the assistant to cease output. The server confirms this by sending an Acknowledgement or simply by silence followed by readiness for the user to speak again.

**If the server cannot stop immediately** (for example, if the model cannot be instantly interrupted), it stops as soon as possible and may send a final chunk that was already nearly completed. In any case, after a stop, the conversation turn is considered ended prematurely.

**Database Alignment:** There is not necessarily a dedicated table entry for a stop command in the Alicia schema. It is a control event, not message content. Implementations may log it via the metadata table or as a special event in the conversation record. Some systems log a "stop" event for audit purposes. The specification includes it as a message type so that it is part of the protocol. If logged, it has an id, conversationId, and may reference what it stopped. However, it is not typically shown in conversation history to end-users.
