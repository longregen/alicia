package org.localforge.alicia.core.domain.model

/**
 * Domain model representing a TTS voice option.
 */
data class Voice(
    val id: String,
    val name: String,
    val language: String,
    val gender: VoiceGender,
    val description: String? = null
)

/**
 * Represents the gender category of a TTS voice.
 * Used for voice selection and filtering in the UI.
 */
enum class VoiceGender {
    MALE,
    FEMALE,
    NEUTRAL
}
