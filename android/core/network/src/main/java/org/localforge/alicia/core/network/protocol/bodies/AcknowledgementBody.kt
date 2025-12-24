package org.localforge.alicia.core.network.protocol.bodies

/**
 * Acknowledgement (Type 8) confirms receipt of a message
 */
data class AcknowledgementBody(
    val conversationId: String,
    val acknowledgedStanzaId: Int,
    val success: Boolean
)
