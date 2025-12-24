### 1. ErrorMessage (Type 1)

**Purpose:** Conveys errors and exceptional conditions during conversation processing. The server sends this message when it cannot process a request, when the conversation enters an invalid state, or when other failures occur. While the client can also send ErrorMessage to report local processing errors, the typical flow is server-to-client.

**Typical Direction:** Server → Client (primarily)

**Transport:** Messages flow over LiveKit data channels as MessagePack-encoded binary data.

**Fields:**

* `id` (Text, NanoID): A unique identifier for this error message, used when the error is recorded in the conversation log or error log.
* `conversationId` (Text): The conversation identifier this error pertains to.
* `code` (Int32): A standardized error code that classifies the error type:
  * **100-199**: Format and protocol errors (malformed messages, invalid types, parsing failures)
  * **200-299**: Conversation errors (invalid state, missing context, conversation not found)
  * **300-399**: Tool execution errors (tool not found, tool timeout, invalid tool parameters)
  * **500-599**: Server errors (internal failures, database errors, service unavailable)
* `message` (Text): A human-readable description of the error. This is shown in logs and may be displayed to users.
* `severity` (Int32): Severity level where 0=info, 1=warning, 2=error, 3=critical.
* `recoverable` (Bool): Indicates whether the conversation can continue after this error. When `false`, the client should terminate the conversation session.
* `originatingId` (Text, NanoID, optional): References the `id` of the message that caused this error, enabling cross-referencing with the database for debugging.

**MessagePack Representation (Example):**

```json
{
  "id": "err_abc123",
  "conversationId": "conv_xyz789",
  "code": 304,
  "message": "Tool execution timeout: search_database exceeded 30s limit",
  "severity": 2,
  "recoverable": true,
  "originatingId": "msg_def456"
}
```

**Error Code Examples:**

* `101`: Malformed MessagePack data
* `102`: Unknown message type
* `201`: Conversation not found
* `202`: Invalid conversation state
* `301`: Tool not found
* `304`: Tool execution timeout
* `501`: Internal server error
* `503`: Service temporarily unavailable

**Semantics:** When the client receives an ErrorMessage, it indicates a problem with processing a particular request or the entire conversation. The client should handle the error by:

* Alerting the user with the error message
* Taking corrective action if the error is recoverable
* Terminating the conversation session if `recoverable` is `false`

Non-recoverable errors (like critical server failures or conversation state corruption) signal that the conversation cannot continue and the client should close the connection.

Recoverable errors (like a single tool timeout) allow the conversation to continue, though the current request may have failed. The server continues processing subsequent user input normally.

**Database Alignment:** Error messages are stored in the conversation history with their unique ID. The `code` and `message` fields align with debugging and analytics tables. The `originatingId` creates a traceable link between errors and the messages that caused them, facilitating root cause analysis.
