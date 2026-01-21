package org.localforge.alicia.core.domain.model

enum class Severity(val value: Int) {
    INFO(0),
    WARNING(1),
    ERROR(2),
    CRITICAL(3);

    companion object {
        fun fromInt(value: Int) = entries.firstOrNull { it.value == value } ?: INFO
    }
}

data class ErrorMessage(
    val id: String,
    val conversationId: String,
    val code: Int,
    val message: String,
    val severity: Severity,
    val recoverable: Boolean,
    val originatingId: String? = null
)

data class ReasoningStep(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val sequence: Int,
    val content: String
)

data class ToolUseRequest(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val toolName: String,
    val parameters: Map<String, Any>,
    val execution: String,
    val timeoutMs: Int? = null
)

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

data class MemoryTrace(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val memoryId: String,
    val content: String,
    val relevance: Double
)

data class Commentary(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val content: String,
    val commentaryType: String? = null
)
