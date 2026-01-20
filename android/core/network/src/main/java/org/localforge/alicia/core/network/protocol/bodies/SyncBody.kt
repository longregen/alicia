package org.localforge.alicia.core.network.protocol.bodies

/**
 * Sync request body (Type 17)
 * Request to sync conversation state
 */
data class SyncRequestBody(
    val conversationId: String,
    val fromSequence: Int? = null
)

/**
 * Sync response body (Type 18)
 * Response with sync data
 */
data class SyncResponseBody(
    val conversationId: String,
    val messages: List<Map<String, Any>> = emptyList(),
    val lastSequence: Int
)
