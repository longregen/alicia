package org.localforge.alicia.core.network.protocol.bodies

data class ReasoningStepBody(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val sequence: Int,
    val content: String
)
