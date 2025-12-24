package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Response model for available voices
 *
 * @property sampleRate The audio sample rate in Hz. Common values include 8000, 16000, 24000,
 *                      and 48000 Hz. Null indicates the server's default sample rate will be used.
 *                      Higher sample rates provide better audio quality but require more bandwidth.
 */
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
