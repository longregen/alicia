package org.localforge.alicia.core.network.protocol.bodies

data class StartAnswerBody(
    val id: String,
    val previousId: String,
    val conversationId: String,
    val answerType: AnswerType? = null,
    val plannedSentenceCount: Int? = null
)

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
