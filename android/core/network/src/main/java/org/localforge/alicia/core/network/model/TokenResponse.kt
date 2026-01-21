package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class GenerateTokenRequest(
    @Json(name = "participant_id")
    val participantId: String,

    @Json(name = "participant_name")
    val participantName: String? = null
)

@JsonClass(generateAdapter = true)
data class TokenResponse(
    @Json(name = "token")
    val token: String,

    @Json(name = "expires_at")
    val expiresAt: Long,

    @Json(name = "room_name")
    val roomName: String,

    @Json(name = "participant_id")
    val participantId: String
)
