package org.localforge.alicia.core.network.protocol.bodies

/**
 * AudioChunk (Type 4) represents raw audio data segment
 */
data class AudioChunkBody(
    val conversationId: String,
    val format: String, // e.g., "audio/opus"
    val sequence: Int,
    val durationMs: Int,
    val trackSid: String? = null,
    val data: ByteArray? = null,
    val isLast: Boolean? = null,
    val timestamp: Long? = null
) {
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (javaClass != other?.javaClass) return false

        other as AudioChunkBody

        if (conversationId != other.conversationId) return false
        if (format != other.format) return false
        if (sequence != other.sequence) return false
        if (durationMs != other.durationMs) return false
        if (trackSid != other.trackSid) return false
        if (data != null) {
            if (other.data == null) return false
            if (!data.contentEquals(other.data)) return false
        } else if (other.data != null) return false
        if (isLast != other.isLast) return false
        if (timestamp != other.timestamp) return false

        return true
    }

    override fun hashCode(): Int {
        var result = conversationId.hashCode()
        result = 31 * result + format.hashCode()
        result = 31 * result + sequence
        result = 31 * result + durationMs
        result = 31 * result + (trackSid?.hashCode() ?: 0)
        result = 31 * result + (data?.contentHashCode() ?: 0)
        result = 31 * result + (isLast?.hashCode() ?: 0)
        result = 31 * result + (timestamp?.hashCode() ?: 0)
        return result
    }
}
