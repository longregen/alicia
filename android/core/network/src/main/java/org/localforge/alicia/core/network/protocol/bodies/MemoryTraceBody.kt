package org.localforge.alicia.core.network.protocol.bodies

/**
 * MemoryTrace (Type 14) logs memory retrieval events
 */
data class MemoryTraceBody(
    val id: String,
    val messageId: String,
    val conversationId: String,
    val memoryId: String,
    val content: String,
    val relevance: Float
)
