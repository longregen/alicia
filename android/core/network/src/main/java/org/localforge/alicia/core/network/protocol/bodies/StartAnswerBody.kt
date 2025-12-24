package org.localforge.alicia.core.network.protocol.bodies

/**
 * StartAnswer (Type 13) initiates a streaming assistant response
 */
data class StartAnswerBody(
    val id: String,
    val previousId: String,
    val conversationId: String,
    val answerType: AnswerType? = null,
    val plannedSentenceCount: Int? = null
)

// Note: Enum names use underscore (TEXT_VOICE) but are serialized with plus sign (text+voice)
enum class AnswerType {
    TEXT,
    VOICE,
    TEXT_VOICE;

    companion object {
        fun fromString(value: String?): AnswerType? {
            return when (value?.lowercase()) {
                "text" -> TEXT
                "voice" -> VOICE
                "text+voice" -> TEXT_VOICE
                else -> null
            }
        }
    }
}
