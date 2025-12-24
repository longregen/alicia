package org.localforge.alicia.core.network.protocol.bodies

/**
 * ReasoningStep (Type 5) represents internal reasoning trace
 */
data class ReasoningStepBody(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val sequence: Int,
    val content: String
)
