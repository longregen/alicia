package org.localforge.alicia.core.network.protocol.bodies

/**
 * Commentary (Type 15) represents assistant's internal commentary
 */
data class CommentaryBody(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val content: String,
    val commentaryType: String? = null
)
