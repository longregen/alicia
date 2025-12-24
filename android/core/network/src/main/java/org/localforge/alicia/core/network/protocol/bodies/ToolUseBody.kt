package org.localforge.alicia.core.network.protocol.bodies

/**
 * This file contains both ToolUseRequestBody and ToolUseResultBody classes,
 * which are related message types for tool execution.
 */

/**
 * ToolUseRequest (Type 6) represents a request to execute a tool
 */
data class ToolUseRequestBody(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val toolName: String,
    val parameters: Map<String, Any>,
    val execution: ToolExecution,
    val timeoutMs: Int? = null
)

/**
 * ToolUseResult (Type 7) represents a tool execution result
 */
data class ToolUseResultBody(
    val id: String,
    val requestId: String,
    val conversationId: String,
    val success: Boolean,
    val result: Any? = null,
    val errorCode: String? = null,
    val errorMessage: String? = null
)

enum class ToolExecution {
    SERVER,
    CLIENT,
    EITHER;

    companion object {
        fun fromString(value: String?): ToolExecution? {
            return when (value?.lowercase()) {
                "server" -> SERVER
                "client" -> CLIENT
                "either" -> EITHER
                else -> null
            }
        }
    }
}

/**
 * Default timeout for tool execution in milliseconds
 */
const val DEFAULT_TOOL_TIMEOUT = 30000
