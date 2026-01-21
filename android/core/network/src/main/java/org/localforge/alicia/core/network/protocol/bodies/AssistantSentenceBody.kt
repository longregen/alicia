package org.localforge.alicia.core.network.protocol.bodies

data class AssistantSentenceBody(
    val id: String? = null,
    val previousId: String,
    val conversationId: String,
    val sequence: Int,
    val text: String,
    val isFinal: Boolean? = null,
    val audio: ByteArray? = null
) {
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (javaClass != other?.javaClass) return false

        other as AssistantSentenceBody

        if (id != other.id) return false
        if (previousId != other.previousId) return false
        if (conversationId != other.conversationId) return false
        if (sequence != other.sequence) return false
        if (text != other.text) return false
        if (isFinal != other.isFinal) return false
        if (audio != null) {
            if (other.audio == null) return false
            if (!audio.contentEquals(other.audio)) return false
        } else if (other.audio != null) return false

        return true
    }

    override fun hashCode(): Int {
        var result = id?.hashCode() ?: 0
        result = 31 * result + previousId.hashCode()
        result = 31 * result + conversationId.hashCode()
        result = 31 * result + sequence
        result = 31 * result + text.hashCode()
        result = 31 * result + (isFinal?.hashCode() ?: 0)
        result = 31 * result + (audio?.contentHashCode() ?: 0)
        return result
    }
}
