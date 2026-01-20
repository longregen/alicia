package org.localforge.alicia.core.network.protocol.bodies

/**
 * Subscribe request body (Type 40)
 */
data class SubscribeBody(
    val conversationId: String,
    val fromSequence: Int? = null
)

/**
 * Unsubscribe request body (Type 41)
 */
data class UnsubscribeBody(
    val conversationId: String
)

/**
 * Subscribe acknowledgement body (Type 42)
 */
data class SubscribeAckBody(
    val conversationId: String,
    val success: Boolean,
    val error: String? = null,
    val missedMessages: Int? = null
)

/**
 * Unsubscribe acknowledgement body (Type 43)
 */
data class UnsubscribeAckBody(
    val conversationId: String,
    val success: Boolean
)
