package org.localforge.alicia.core.network.protocol.bodies

/**
 * ErrorMessage (Type 1) conveys errors and exceptional conditions
 */
data class ErrorMessageBody(
    val id: String,
    val conversationId: String,
    val code: Int,
    val message: String,
    val severity: Severity,
    val recoverable: Boolean,
    val originatingId: String? = null
)

enum class Severity(val value: Int) {
    INFO(0),
    WARNING(1),
    ERROR(2),
    CRITICAL(3);

    companion object {
        private val map = values().associateBy(Severity::value)
        fun fromInt(value: Int) = map[value] ?: INFO
    }
}

// Error codes
object ErrorCodes {
    // Format and protocol errors (100-199)
    const val MALFORMED_DATA = 101
    const val UNKNOWN_TYPE = 102

    // Conversation errors (200-299)
    const val CONVERSATION_NOT_FOUND = 201
    const val INVALID_STATE = 202

    // Tool errors (300-399)
    const val TOOL_NOT_FOUND = 301
    const val TOOL_TIMEOUT = 304

    // Server errors (500-599)
    const val INTERNAL_ERROR = 501
    const val SERVICE_UNAVAILABLE = 503
    const val QUEUE_OVERFLOW = 504
}
