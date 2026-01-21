package org.localforge.alicia.core.network.protocol

data class Envelope(
    val stanzaId: Int,
    val conversationId: String,
    val type: MessageType,
    val meta: Map<String, Any>? = null,
    val body: Any
) {
    companion object {
        const val META_KEY_TIMESTAMP = "timestamp"
        const val META_KEY_CLIENT_VERSION = "clientVersion"
        const val META_KEY_TRACE_ID = "messaging.trace_id"
        const val META_KEY_SPAN_ID = "messaging.span_id"
    }

    fun withMeta(key: String, value: Any): Envelope {
        val newMeta = (meta ?: emptyMap()).toMutableMap()
        newMeta[key] = value
        return copy(meta = newMeta)
    }

    fun withTracing(traceId: String, spanId: String): Envelope {
        return withMeta(META_KEY_TRACE_ID, traceId)
            .withMeta(META_KEY_SPAN_ID, spanId)
    }
}
