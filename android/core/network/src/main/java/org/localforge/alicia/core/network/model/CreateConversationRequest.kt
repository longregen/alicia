package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class CreateConversationRequest(
    @Json(name = "title")
    val title: String,

    @Json(name = "preferences")
    val preferences: ConversationPreferences? = null
)
