package org.localforge.alicia.core.network.protocol.bodies

data class MemoryTraceBody(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val memoryId: String,
    val content: String,
    val relevance: Float
)
