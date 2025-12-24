package org.localforge.alicia.core.domain.model

/**
 * Protocol message types for displaying assistant reasoning, tool use, memory access, etc.
 * These match the protocol types from the web frontend.
 */

enum class Severity(val value: Int) {
    INFO(0),
    WARNING(1),
    ERROR(2),
    CRITICAL(3);

    companion object {
        fun fromInt(value: Int) = entries.firstOrNull { it.value == value } ?: INFO
    }
}

/**
 * Represents an error that occurred during conversation processing.
 * @property id Unique identifier for this error message
 * @property conversationId The conversation where the error occurred
 * @property code Error code for categorization
 * @property message Human-readable error description
 * @property severity Severity level of the error
 * @property recoverable Whether the error can be recovered from
 * @property originatingId ID of the message or request that caused this error
 */
data class ErrorMessage(
    val id: String,
    val conversationId: String,
    val code: Int,
    val message: String,
    val severity: Severity,
    val recoverable: Boolean,
    val originatingId: String? = null
)

/**
 * Represents a single step in the assistant's reasoning process.
 * @property id Unique identifier for this reasoning step
 * @property messageId The message this step belongs to
 * @property conversationId The conversation context
 * @property sequence Order of this step in the reasoning chain
 * @property content The reasoning content/thought
 */
data class ReasoningStep(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val sequence: Int,
    val content: String
)

/**
 * Represents a request to execute a tool during conversation.
 * @property id Unique identifier for this tool use request
 * @property messageId The message requesting tool use
 * @property conversationId The conversation context
 * @property toolName Name of the tool to execute
 * @property parameters Input parameters for the tool
 * @property execution Expected values: 'server', 'client', or 'either'. No compile-time validation - ensure correct values are passed.
 * @property timeoutMs Optional timeout in milliseconds
 */
data class ToolUseRequest(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val toolName: String,
    val parameters: Map<String, Any>,
    val execution: String, // "server" | "client" | "either"
    val timeoutMs: Int? = null
)

/**
 * Represents the result of a tool execution.
 * @property id Unique identifier for this result
 * @property requestId ID of the corresponding tool use request
 * @property conversationId The conversation context
 * @property success Whether the tool execution succeeded
 * @property result The tool's output data if successful
 * @property errorCode Error code if the execution failed
 * @property errorMessage Error description if the execution failed
 */
data class ToolUseResult(
    val id: String,
    val requestId: String,
    val conversationId: String,
    val success: Boolean,
    val result: Any? = null,
    val errorCode: String? = null,
    val errorMessage: String? = null
)

data class ToolUsage(
    val request: ToolUseRequest,
    val result: ToolUseResult? = null
)

/**
 * Represents a memory retrieval during conversation processing.
 * @property id Unique identifier for this memory trace
 * @property messageId The message that triggered memory retrieval
 * @property conversationId The conversation context
 * @property memoryId ID of the retrieved memory
 * @property content The memory content that was retrieved
 * @property relevance Relevance score (0.0 to 1.0) of this memory
 */
data class MemoryTrace(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val memoryId: String,
    val content: String,
    val relevance: Double
)

/**
 * Represents additional commentary or meta-information about a message.
 * @property id Unique identifier for this commentary
 * @property messageId The message being commented on
 * @property conversationId The conversation context
 * @property content The commentary text
 * @property commentaryType Optional categorization of the commentary
 */
data class Commentary(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val content: String,
    val commentaryType: String? = null
)
