package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

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

@JsonClass(generateAdapter = true)
data class MessageListResponse(
    @Json(name = "messages")
    val messages: List<MessageResponse>,

    @Json(name = "total")
    val total: Int
)

@JsonClass(generateAdapter = true)
data class SendMessageRequest(
    @Json(name = "contents")
    val contents: String
)

@JsonClass(generateAdapter = true)
data class SwitchBranchRequest(
    @Json(name = "tip_message_id")
    val tipMessageId: String
)
