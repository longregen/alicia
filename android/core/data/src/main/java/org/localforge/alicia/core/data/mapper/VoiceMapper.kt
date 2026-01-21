package org.localforge.alicia.core.data.mapper

import org.localforge.alicia.core.domain.model.Voice
import org.localforge.alicia.core.domain.model.VoiceGender
import org.localforge.alicia.core.network.model.VoiceResponse

fun VoiceResponse.toDomain(): Voice {
    return Voice(
        id = id,
        name = name,
        language = language,
        gender = parseGender(gender ?: "neutral"),
        description = style
    )
}

fun List<VoiceResponse>.toDomain(): List<Voice> {
    return map { it.toDomain() }
}

private fun parseGender(genderString: String): VoiceGender {
    return when (genderString.lowercase()) {
        "male", "m" -> VoiceGender.MALE
        "female", "f" -> VoiceGender.FEMALE
        "neutral", "n" -> VoiceGender.NEUTRAL
        else -> VoiceGender.NEUTRAL
    }
}
