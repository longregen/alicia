package org.localforge.alicia.core.network.protocol.bodies

/**
 * ControlVariation (Type 11) requests a variation of a previous message
 */
data class ControlVariationBody(
    val conversationId: String,
    val targetId: String,
    val mode: VariationType,
    val newContent: String? = null
)

enum class VariationType {
    REGENERATE,
    EDIT,
    CONTINUE;

    companion object {
        fun fromString(value: String?): VariationType? {
            return when (value?.lowercase()) {
                "regenerate" -> REGENERATE
                "edit" -> EDIT
                "continue" -> CONTINUE
                else -> null
            }
        }
    }
}
