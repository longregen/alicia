package org.localforge.alicia.core.network.protocol.bodies

data class UserMessageBody(
    val id: String,
    val previousId: String? = null,
    val conversationId: String,
    val content: String,
    val timestamp: Long? = null
)
