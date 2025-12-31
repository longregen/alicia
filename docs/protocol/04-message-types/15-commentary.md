### 15. Commentary (Type 15)

**Purpose:** Enables users to provide feedback, annotations, and notes about the conversation or specific messages. Commentary messages support user ratings, explanations, and meta-discussion that sits alongside the main conversation flow. The system can also generate commentaries for internal analysis or to provide explanations when requested.

**Typical Direction:** Bidirectional
* **Client → Server**: User feedback, ratings, and annotations
* **Server → Client**: System-generated explanations or analysis (less common)

**Transport:** Messages flow over LiveKit data channels as MessagePack-encoded binary data.

**Fields:**

* `id` (Text, NanoID): A unique identifier for this commentary entry, as recorded in the `alicia_user_conversation_commentaries` table.
* `conversationId` (Text): The conversation identifier.
* `messageId` (Text, NanoID): The ID of the message this commentary refers to. Typically points to an AssistantMessage when users are providing feedback on an answer.
* `content` (Text): The text content of the commentary. Examples:
  * "This answer was very helpful because it provided specific code examples"
  * "The response missed the key point about authentication"
  * "Requesting more detail about the second step"
* `commentaryType` (Text, optional): Classification of the commentary:
  * `"feedback-positive"`: Positive user feedback
  * `"feedback-negative"`: Negative user feedback or issue report
  * `"explanation"`: Request for or provision of explanation
  * `"note"`: General annotation or reminder
  * `"correction"`: User correcting information in the response

**MessagePack Representation (Example - User Feedback):**

```json
{
  "id": "comm_FF99",
  "conversationId": "conv_7H93k",
  "messageId": "msg_a9X8Y",
  "content": "This answer was very helpful because it provided specific examples and explained the edge cases clearly.",
  "commentaryType": "feedback-positive"
}
```

**MessagePack Representation (Example - User Correction):**

```json
{
  "id": "comm_GH88",
  "conversationId": "conv_7H93k",
  "messageId": "msg_a9X8Y",
  "content": "The date mentioned should be 2024, not 2023",
  "commentaryType": "correction"
}
```

**Semantics:** Commentary messages are not part of the direct conversation turn sequence. They represent meta-information about the conversation that is stored and analyzed separately.

**Common Use Cases:**

* **Detailed Feedback**: Users provide written feedback explaining what was helpful or what was missing. This captures nuanced reactions.

* **Corrections**: Users point out factual errors or misunderstandings in the assistant's response, helping improve future performance.

* **Request for Explanation**: Users can ask "Why did you give this answer?" The client sends a Commentary requesting explanation.

* **Personal Notes**: Users add private notes or reminders about the conversation for their own reference.

**Typical Flow (User Feedback):**

1. Server sends AssistantMessage with answer
2. User writes feedback
3. Client sends Commentary message: messageId = assistant message ID, content = feedback text, commentaryType = type of feedback
4. Server stores Commentary in `alicia_user_conversation_commentaries` table
5. Server acknowledges receipt (optional)

**Client Handling:** Clients typically display commentary through:

* Feedback text boxes
* Comment threads or annotations on specific messages
* Separate feedback panels
* Hidden storage for developer review only

**Database Alignment:** Commentary fields directly correspond to the `alicia_user_conversation_commentaries` table with columns:

* `id` (primary key, NanoID)
* `conversation_id`
* `message_id` (the target message)
* `content` (the commentary text)
* `commentary_type` (category)
* `created_at` (timestamp)

This alignment ensures all user feedback and annotations are captured in both the live protocol stream and persistent storage, enabling quality analysis, training data collection, and user experience improvements.
