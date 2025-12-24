package org.localforge.alicia.core.data.mapper

import org.localforge.alicia.core.domain.model.Voice
import org.localforge.alicia.core.domain.model.VoiceGender
import org.localforge.alicia.core.network.model.VoiceResponse

/**
 * Convert VoiceResponse to Voice domain model.
 */
fun VoiceResponse.toDomain(): Voice {
    return Voice(
        id = id,
        name = name,
        language = language,
        gender = parseGender(gender ?: "neutral"),
        // Map VoiceResponse.style to Voice.description field
        description = style
    )
}

/**
 * Convert list of VoiceResponses to list of Voice domain models.
 */
fun List<VoiceResponse>.toDomain(): List<Voice> {
    return map { it.toDomain() }
}

/**
 * Parse gender string to VoiceGender enum.
 */
private fun parseGender(genderString: String): VoiceGender {
    return when (genderString.lowercase()) {
        "male", "m" -> VoiceGender.MALE
        "female", "f" -> VoiceGender.FEMALE
        "neutral", "n" -> VoiceGender.NEUTRAL
        else -> VoiceGender.NEUTRAL
    }
}
