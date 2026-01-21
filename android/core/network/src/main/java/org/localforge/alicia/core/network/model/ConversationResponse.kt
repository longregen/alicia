package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

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

@JsonClass(generateAdapter = true)
data class UpdateConversationRequest(
    @Json(name = "title")
    val title: String? = null,

    @Json(name = "status")
    val status: String? = null,

    @Json(name = "preferences")
    val preferences: ConversationPreferences? = null
)
