package org.localforge.alicia.core.network.protocol.bodies

/**
 * Transcription (Type 9) represents speech-to-text output
 */
data class TranscriptionBody(
    val id: String,
    val previousId: String? = null,
    val conversationId: String,
    val text: String,
    val final: Boolean,
    val confidence: Float? = null,
    val language: String? = null
)
