package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class VoiceResponse(
    @Json(name = "id")
    val id: String,

    @Json(name = "name")
    val name: String,

    @Json(name = "language")
    val language: String,

    @Json(name = "gender")
    val gender: String? = null,

    @Json(name = "style")
    val style: String? = null,

    @Json(name = "sample_rate")
    val sampleRate: Int? = null
)
