package org.localforge.alicia.core.network.protocol

/**
 * Envelope wraps all protocol messages with common metadata for routing and ordering.
 * Envelopes are serialized using MessagePack and transmitted over LiveKit data channels.
 */
data class Envelope(
    /**
     * StanzaID identifies the message's position in the conversation sequence.
     * Client messages: positive, incrementing (1, 2, 3, ...)
     * Server messages: negative, decrementing (-1, -2, -3, ...)
     */
    val stanzaId: Int,

    /**
     * ConversationID is the unique identifier of the conversation.
     * Format: conv_{nanoid} - maps directly to LiveKit room name.
     */
    val conversationId: String,

    /**
     * Type is the numeric message type (1-16)
     */
    val type: MessageType,

    /**
     * Meta contains optional metadata including OpenTelemetry tracing fields
     */
    val meta: Map<String, Any>? = null,

    /**
     * Body contains the message-specific payload
     */
    val body: Any
) {
    companion object {
        // Common meta keys
        const val META_KEY_TIMESTAMP = "timestamp"
        const val META_KEY_CLIENT_VERSION = "clientVersion"
        const val META_KEY_TRACE_ID = "messaging.trace_id"
        const val META_KEY_SPAN_ID = "messaging.span_id"
    }

    /**
     * Add or update metadata
     */
    fun withMeta(key: String, value: Any): Envelope {
        val newMeta = (meta ?: emptyMap()).toMutableMap()
        newMeta[key] = value
        return copy(meta = newMeta)
    }

    /**
     * Add OpenTelemetry tracing fields
     */
    fun withTracing(traceId: String, spanId: String): Envelope {
        return withMeta(META_KEY_TRACE_ID, traceId)
            .withMeta(META_KEY_SPAN_ID, spanId)
    }
}
