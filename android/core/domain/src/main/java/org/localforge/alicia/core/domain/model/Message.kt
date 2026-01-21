package org.localforge.alicia.core.domain.model

data class Message(
    val id: String,
    val conversationId: String,
    val role: MessageRole,
    val content: String,
    val createdAt: Long,
    val updatedAt: Long? = null,
    val sequenceNumber: Int? = null,
    val previousId: String? = null,
    val isVoice: Boolean = false,
    val audioPath: String? = null,
    val audioDurationMs: Long? = null,
    val localId: String? = null,
    val serverId: String? = null,
    val syncStatus: SyncStatus = SyncStatus.SYNCED,
    val syncedAt: Long? = null
) {
    val isUserMessage: Boolean
        get() = role == MessageRole.USER

    val isAssistantMessage: Boolean
        get() = role == MessageRole.ASSISTANT

    val hasAudio: Boolean
        get() = audioPath != null && audioDurationMs != null
}

enum class MessageRole(val value: String) {
    USER("user"),
    ASSISTANT("assistant"),
    SYSTEM("system");

    companion object {
        fun fromString(value: String): MessageRole {
            return entries.find { it.value == value } ?: USER
        }
    }
}

enum class SyncStatus(val value: String) {
    PENDING("pending"),
    SYNCED("synced"),
    CONFLICT("conflict");

    companion object {
        fun fromString(value: String): SyncStatus {
            return entries.find { it.value == value } ?: SYNCED
        }
    }
}
