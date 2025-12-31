### 7. ToolUseResult (Type 7)

**Purpose:** Conveys the result or outcome of a previously requested tool execution. This message provides the data retrieved or the outcome of an action that the assistant requested via ToolUseRequest. The assistant uses this information to continue its reasoning or formulate a final answer.

**Direction:** Bidirectional (via LiveKit data channel)

* **Client → Server**: When the client executes a tool (in response to a ToolUseRequest with `execution: "client"`), the client sends the ToolUseResult back to the server with the execution outcome.

* **Server → Client**: When the server executes a tool internally (ToolUseRequest with `execution: "server"`), the server sends the ToolUseResult to the client for transparency and logging purposes.

**Fields:**

* `id` (Text, NanoID): References the `id` of the ToolUseRequest that this result fulfills. This ties the result to its originating request, enabling correlation in the conversation flow.

* `success` (Bool): Indicates whether the tool execution was successful. `true` means the tool ran and returned a result. `false` means there was a failure (in which case, `errorCode` and `errorMessage` fields provide details).

* `result` (Map, optional): The data returned by the tool, if successful. This is a structured object containing the result data from the tool execution. In MessagePack, this is represented as a map where keys are field names and values are the result data (strings, numbers, booleans, arrays, or nested maps).

  Examples:
  * For a web search tool: `{"results": [...], "totalResults": 42}`
  * For a calculator: `{"answer": 42.5}`
  * For a file read: `{"content": "file contents here", "size": 1024}`

  If `success` is `false`, this field MAY be omitted or set to `null`.

* `errorCode` (Text, optional): A machine-readable error code when `success` is `false`. Standard error codes include:
  * `"unknown_tool"` – The tool name is not recognized or supported
  * `"timeout"` – Tool execution exceeded the specified timeout
  * `"execution_error"` – The tool threw an error during execution
  * `"invalid_parameters"` – The parameters were malformed or invalid

  Custom error codes may be used for tool-specific failures.

* `errorMessage` (Text, optional): A human-readable error description when `success` is `false`. This provides context about what went wrong (e.g., "Network timeout", "File not found", "Permission denied").

**MessagePack Representation (Examples):**

Successful result example:
```json
{
  "id": "toolreq_abc123",
  "success": true,
  "result": {
    "results": [
      {"name": "Luigi's Trattoria", "rating": 4.5, "address": "123 Main St"},
      {"name": "Pasta Palace", "rating": 4.3, "address": "456 Broadway"}
    ],
    "totalResults": 42
  }
}
```

Error result example:
```json
{
  "id": "toolreq_xyz789",
  "success": false,
  "errorCode": "execution_error",
  "errorMessage": "File not found: /Users/alice/documents/notes.txt"
}
```

Timeout error example:
```json
{
  "id": "toolreq_def456",
  "success": false,
  "errorCode": "timeout",
  "errorMessage": "Tool execution exceeded timeout of 5000ms"
}
```

**Semantics:**

Upon receiving a ToolUseResult, the assistant (server) incorporates the result into its next steps. The typical flow is:

1. **Client execution flow**: When the client executes a tool (because it received a ToolUseRequest with `execution: "client"`), it sends the ToolUseResult back to the server. The server waits for this result before continuing the conversation, then resumes answer generation using the tool's output.

2. **Server execution flow**: When the server executes a tool internally (`execution: "server"`), it sends the ToolUseResult to the client for transparency. The client may display intermediate information (e.g., "Assistant searched the web: found 5 results") or simply log it for the conversation record.

The `id` field ensures proper correlation between requests and results, even when multiple tool requests are in flight simultaneously.

**Error Handling:**

If the tool execution fails (`success: false`), the assistant decides how to proceed:
* Attempt a different approach or alternative tool
* Ask the user for clarification or additional input
* Gracefully handle the missing information in the response
* Report the error to the user if it affects the answer quality

Tool failures are normal operations and are conveyed through ToolUseResult messages rather than ErrorMessage messages (Type 1). ErrorMessage is reserved for protocol-level failures or severe system errors.

**Client Handling:**

When the client receives a ToolUseResult from the server (indicating the server executed a tool), the client:
* May display intermediate information to the user for transparency
* Logs the result as part of the conversation record
* Updates the UI to reflect the assistant's progress
* Does not need to send any response (this is informational only)

**LiveKit Integration:**

ToolUseResult messages flow over LiveKit data channels as MessagePack-encoded payloads. The reliable data channel ensures that results are delivered in order and the `id` correlation remains intact. Both client and server use this correlation to maintain conversation state and track which tool executions have completed.
