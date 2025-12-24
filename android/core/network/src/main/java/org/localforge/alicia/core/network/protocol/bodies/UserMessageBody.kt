package org.localforge.alicia.core.network.protocol.bodies

/**
 * UserMessage (Type 2) carries a user's input message
 */
data class UserMessageBody(
    val id: String,
    val previousId: String? = null,
    val conversationId: String,
    val content: String,
    val timestamp: Long? = null
)
