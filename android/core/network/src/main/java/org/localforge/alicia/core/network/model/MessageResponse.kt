package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Response model for message data from the API
 *
 * @property sequenceNumber The position of this message in the conversation sequence (0-based).
 * @property previousId The ID of the previous message in the sequence. Typically null for
 *                      the first message in a conversation. When null, this message starts
 *                      a new sequence chain.
 */
@JsonClass(generateAdapter = true)
data class MessageResponse(
    @Json(name = "id")
    val id: String,

    @Json(name = "conversation_id")
    val conversationId: String,

    @Json(name = "sequence_number")
    val sequenceNumber: Int,

    @Json(name = "previous_id")
    val previousId: String? = null,

    @Json(name = "role")
    val role: String,

    @Json(name = "contents")
    val contents: String,

    @Json(name = "created_at")
    val createdAt: String,

    @Json(name = "updated_at")
    val updatedAt: String
)

/**
 * List response for messages
 */
@JsonClass(generateAdapter = true)
data class MessageListResponse(
    @Json(name = "messages")
    val messages: List<MessageResponse>,

    @Json(name = "total")
    val total: Int
)

/**
 * Request model for sending a message
 */
@JsonClass(generateAdapter = true)
data class SendMessageRequest(
    @Json(name = "contents")
    val contents: String
)
