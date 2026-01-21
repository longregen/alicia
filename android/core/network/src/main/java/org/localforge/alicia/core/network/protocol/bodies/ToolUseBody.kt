package org.localforge.alicia.core.network.protocol.bodies

data class ToolUseRequestBody(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val toolName: String,
    val parameters: Map<String, Any>,
    val execution: ToolExecution,
    val timeoutMs: Int? = null
)

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

const val DEFAULT_TOOL_TIMEOUT = 30000
