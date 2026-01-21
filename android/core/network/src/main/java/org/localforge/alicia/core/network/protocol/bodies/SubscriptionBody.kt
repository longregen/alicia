package org.localforge.alicia.core.network.protocol.bodies

data class SubscribeBody(
    val conversationId: String,
    val fromSequence: Int? = null
)

data class UnsubscribeBody(
    val conversationId: String
)

data class SubscribeAckBody(
    val conversationId: String,
    val success: Boolean,
    val error: String? = null,
    val missedMessages: Int? = null
)

data class UnsubscribeAckBody(
    val conversationId: String,
    val success: Boolean
)
