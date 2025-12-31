### 6. ToolUseRequest (Type 6)

**Purpose:** Represents a request for external tool invocation. When the AI assistant needs to perform an operation beyond its native capabilities—such as making an API call, querying a database, or accessing local resources—it sends a ToolUseRequest. This message encapsulates what tool to invoke and the parameters needed for execution.

**Direction:** Server → Client (via LiveKit data channel)

The server sends ToolUseRequest messages to the client over LiveKit data channels. Depending on the `execution` field, either the client executes the tool and returns a result, or the server handles execution internally while sending the request to the client for transparency.

**Fields:**

* `id` (Text, NanoID): Unique identifier for this tool request. The corresponding ToolUseResult references this ID to correlate the response.

* `messageId` (Text, NanoID): The ID of the message that triggered this tool use, typically the AssistantMessage being generated that requires this tool. This links the tool use back to the conversation context.

* `toolName` (Text): The name or identifier of the tool to invoke. Examples include `"web_search"`, `"calculator"`, `"read_local_file"`, or `"database_query"`. Both client and server must share a common understanding of available tool names.

* `parameters` (Map): The parameters for the tool invocation. This is a structured object containing the tool-specific parameters needed for execution. In MessagePack, this is represented as a map where keys are parameter names and values are the parameter values (strings, numbers, booleans, arrays, or nested maps).

  Examples:
  * For `"web_search"`: `{"query": "search terms", "limit": 5}`
  * For `"calculator"`: `{"expression": "2 + 2"}`
  * For `"read_local_file"`: `{"filePath": "/path/to/file.txt"}`

* `execution` (Text, REQUIRED): Specifies who executes this tool. This field is REQUIRED and MUST contain one of the following values:

  * **`"server"`** – The server executes this tool. The client receives this message for informational/transparency purposes only and MUST NOT attempt execution. The server generates and sends the ToolUseResult.

  * **`"client"`** – The client MUST execute this tool. The server delegates tool execution to the client (typically for tools requiring client-side resources, local file access, or user interaction). The client MUST send back a ToolUseResult message with the outcome. If the client cannot execute the tool, it MUST send a ToolUseResult with `success: false` and appropriate error information.

  * **`"either"`** – Either server or client may execute this tool. This enables flexible architectures where tools can be executed by whoever has the capability. If the client supports the tool, it SHOULD execute it and return a ToolUseResult. If the client does not support the tool, the server executes it. This mode requires coordination to avoid duplicate execution.

  If this field is omitted or contains an unrecognized value, the receiver MUST reject the message with an error.

* `timeoutMs` (Int32, optional): The maximum time in milliseconds that the executor should spend attempting to run this tool before giving up. If omitted, a default timeout of **30000 ms (30 seconds)** applies. The executor (server or client, depending on `execution` field) SHOULD respect this timeout. If tool execution exceeds the timeout, a ToolUseResult MUST be sent with `success: false` and `errorCode: "timeout"`.

**MessagePack Representation (Examples):**

Server-executed tool example:
```json
{
  "id": "toolreq_abc123",
  "messageId": "msg_a9X8Y",
  "toolName": "web_search",
  "execution": "server",
  "parameters": {
    "query": "best Italian restaurants in New York City",
    "limit": 5
  },
  "timeoutMs": 30000
}
```

Client-executed tool example:
```json
{
  "id": "toolreq_xyz789",
  "messageId": "msg_a9X8Y",
  "toolName": "read_local_file",
  "execution": "client",
  "parameters": {
    "filePath": "/Users/alice/documents/notes.txt"
  },
  "timeoutMs": 5000
}
```

**Semantics:**

When the server sends a ToolUseRequest, it pauses its answering process to obtain external information or perform an action. The `execution` field determines the flow:

* **Server execution (`execution: "server"`)**: The server handles the tool internally and sends the request to the client for transparency. The client receives the message but does not execute anything. The server subsequently sends a ToolUseResult with the outcome.

* **Client execution (`execution: "client"`)**: The client receives the request and MUST execute the specified tool. The client performs the action (if it has the capability and permission) and sends back a ToolUseResult over the LiveKit data channel.

* **Either execution (`execution: "either"`)**: The client evaluates whether it can execute the tool. If yes, it executes and returns a result. If no, the server handles execution.

**Client Error Handling:**

When a client receives a ToolUseRequest with `execution: "client"`, it MUST handle the following error scenarios:

1. **Unknown Tool**: If the client does not recognize or support the `toolName`, it MUST send a ToolUseResult with:
   * `success: false`
   * `errorCode: "unknown_tool"`
   * `errorMessage: "Tool 'toolName' is not supported by this client"`

2. **Invalid Parameters**: If the parameters are malformed, missing required fields, or contain invalid values, the client MUST send a ToolUseResult with:
   * `success: false`
   * `errorCode: "invalid_parameters"`
   * `errorMessage`: Human-readable description of the parameter issue

3. **Execution Error**: If the tool execution fails (e.g., network error, permission denied, file not found), the client MUST send a ToolUseResult with:
   * `success: false`
   * `errorCode: "execution_error"`
   * `errorMessage`: Human-readable description of the error
   * `result`: MAY include partial results or diagnostic information

4. **Timeout**: If tool execution exceeds `timeoutMs`, the client MUST:
   * Abort or cancel the tool execution if possible
   * Send a ToolUseResult with:
     * `success: false`
     * `errorCode: "timeout"`
     * `errorMessage: "Tool execution exceeded timeout of {timeoutMs}ms"`

The client MUST NOT silently ignore tool requests. Every ToolUseRequest with `execution: "client"` MUST receive a corresponding ToolUseResult, even if it's an error result.

**Timeout Guidance:**

* **Default Timeout**: 30 seconds (30000 ms)
* **Recommended Timeouts by Tool Type**:
  * Quick computations (calculator, text processing): 1000-5000 ms
  * Local file operations: 5000-10000 ms
  * Network requests (API calls, web search): 10000-30000 ms
  * Heavy computations or long-running tasks: 60000 ms or more
* The requester SHOULD set appropriate timeouts based on the expected tool execution time
* The executor SHOULD enforce the timeout and return an error if exceeded
* For tools without a specified timeout, the 30-second default MUST be used

**LiveKit Integration:**

ToolUseRequest messages flow over LiveKit data channels as MessagePack-encoded payloads. The reliable data channel ensures that tool requests are delivered in order and without loss. Both client and server maintain the correlation between requests (via `id`) and results (via `id` in ToolUseResult) to track the conversation flow and tool execution state.
