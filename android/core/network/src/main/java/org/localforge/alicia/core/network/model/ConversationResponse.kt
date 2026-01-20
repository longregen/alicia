package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Response model for conversation data from the API
 *
 * @property createdAt ISO 8601 timestamp string. The API contract guarantees this field is always
 *                     present and non-null for all conversation responses.
 * @property updatedAt ISO 8601 timestamp string. The API contract guarantees this field is always
 *                     present and non-null for all conversation responses. Will match createdAt
 *                     if the conversation has never been modified.
 */
@JsonClass(generateAdapter = true)
data class ConversationResponse(
    @Json(name = "id")
    val id: String,

    @Json(name = "title")
    val title: String,

    @Json(name = "status")
    val status: String,

    @Json(name = "livekit_room_name")
    val liveKitRoomName: String? = null,

    @Json(name = "preferences")
    val preferences: ConversationPreferences? = null,

    @Json(name = "created_at")
    val createdAt: String,

    @Json(name = "updated_at")
    val updatedAt: String
)

/**
 * List response for conversations
 */
@JsonClass(generateAdapter = true)
data class ConversationListResponse(
    @Json(name = "conversations")
    val conversations: List<ConversationResponse>,

    @Json(name = "total")
    val total: Int,

    @Json(name = "limit")
    val limit: Int,

    @Json(name = "offset")
    val offset: Int
)

/**
 * Conversation preferences
 */
@JsonClass(generateAdapter = true)
data class ConversationPreferences(
    @Json(name = "voice_id")
    val voiceId: String? = null,

    @Json(name = "language")
    val language: String? = null,

    @Json(name = "enable_tools")
    val enableTools: Boolean? = null,

    @Json(name = "enable_memory")
    val enableMemory: Boolean? = null
)

/**
 * Request to update a conversation
 */
@JsonClass(generateAdapter = true)
data class UpdateConversationRequest(
    @Json(name = "title")
    val title: String? = null,

    @Json(name = "status")
    val status: String? = null,

    @Json(name = "preferences")
    val preferences: ConversationPreferences? = null
)
