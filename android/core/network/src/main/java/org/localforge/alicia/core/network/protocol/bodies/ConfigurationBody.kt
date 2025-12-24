package org.localforge.alicia.core.network.protocol.bodies

/**
 * Configuration (Type 12) initializes and configures the connection
 */
data class ConfigurationBody(
    val conversationId: String? = null,
    val lastSequenceSeen: Int? = null,
    val clientVersion: String? = null,
    val preferredLanguage: String? = null,
    val device: String? = null,
    val features: List<String>? = null
)

// Common features
object Features {
    const val STREAMING = "streaming"
    const val PARTIAL_RESPONSES = "partial_responses"
    const val AUDIO_OUTPUT = "audio_output"
    const val REASONING_STEPS = "reasoning_steps"
    const val TOOL_USE = "tool_use"
}
