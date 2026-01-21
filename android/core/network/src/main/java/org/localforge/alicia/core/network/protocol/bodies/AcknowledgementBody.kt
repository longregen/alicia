package org.localforge.alicia.core.network.protocol.bodies

data class AcknowledgementBody(
    val conversationId: String,
    val acknowledgedStanzaId: Int,
    val success: Boolean
)
