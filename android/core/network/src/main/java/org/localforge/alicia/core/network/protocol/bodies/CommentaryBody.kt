package org.localforge.alicia.core.network.protocol.bodies

data class CommentaryBody(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val content: String,
    val commentaryType: String? = null
)
