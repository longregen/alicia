package org.localforge.alicia.core.domain.model

data class Voice(
    val id: String,
    val name: String,
    val language: String,
    val gender: VoiceGender,
    val description: String? = null
)

enum class VoiceGender {
    MALE,
    FEMALE,
    NEUTRAL
}
