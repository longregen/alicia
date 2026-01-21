package org.localforge.alicia.core.network.protocol.bodies

data class ControlStopBody(
    val conversationId: String,
    val targetId: String? = null,
    val reason: String? = null,
    val stopType: StopType? = null
)

enum class StopType {
    GENERATION,
    SPEECH,
    ALL;

    companion object {
        fun fromString(value: String?): StopType? {
            return when (value?.lowercase()) {
                "generation" -> GENERATION
                "speech" -> SPEECH
                "all" -> ALL
                else -> null
            }
        }
    }
}
