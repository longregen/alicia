package org.localforge.alicia.core.network.protocol.bodies

data class SyncRequestBody(
    val conversationId: String,
    val fromSequence: Int? = null
)

data class SyncResponseBody(
    val conversationId: String,
    val messages: List<Map<String, Any>> = emptyList(),
    val lastSequence: Int
)
