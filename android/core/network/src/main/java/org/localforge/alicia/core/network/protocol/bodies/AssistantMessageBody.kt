package org.localforge.alicia.core.network.protocol.bodies

/**
 * AssistantMessage (Type 3) conveys a complete assistant response (non-streaming)
 */
data class AssistantMessageBody(
    val id: String,
    val previousId: String? = null,
    val conversationId: String,
    val content: String,
    val timestamp: Long? = null
)
