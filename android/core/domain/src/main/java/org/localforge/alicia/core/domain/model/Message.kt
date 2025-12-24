package org.localforge.alicia.core.domain.model

/**
 * Domain model for a message in a conversation.
 */
data class Message(
    /**
     * Unique identifier for the message.
     */
    val id: String,

    /**
     * ID of the conversation this message belongs to.
     */
    val conversationId: String,

    /**
     * Role of the message sender.
     */
    val role: MessageRole,

    /**
     * Text content of the message.
     */
    val content: String,

    /**
     * Timestamp when the message was created (milliseconds since epoch).
     */
    val createdAt: Long,

    /**
     * Timestamp when the message was last updated (milliseconds since epoch).
     */
    val updatedAt: Long? = null,

    /**
     * Sequence number for message ordering (used in sync protocol).
     */
    val sequenceNumber: Int? = null,

    /**
     * ID of the previous message in the conversation (used in sync protocol).
     */
    val previousId: String? = null,

    /**
     * Whether this message was generated via voice (true) or text (false).
     */
    val isVoice: Boolean = false,

    /**
     * Optional audio file path for voice messages.
     */
    val audioPath: String? = null,

    /**
     * Duration of the audio in milliseconds (for voice messages).
     */
    val audioDurationMs: Long? = null,

    /**
     * Client-generated local identifier for offline messages.
     */
    val localId: String? = null,

    /**
     * Server-assigned canonical identifier (assigned during sync).
     */
    val serverId: String? = null,

    /**
     * Current synchronization state.
     */
    val syncStatus: SyncStatus = SyncStatus.SYNCED,

    /**
     * Timestamp when the message was last synced with the server (milliseconds since epoch).
     */
    val syncedAt: Long? = null
) {
    /**
     * Check if this is a user message.
     */
    val isUserMessage: Boolean
        get() = role == MessageRole.USER

    /**
     * Check if this is an assistant message.
     */
    val isAssistantMessage: Boolean
        get() = role == MessageRole.ASSISTANT

    /**
     * Check if this message has audio.
     */
    val hasAudio: Boolean
        get() = audioPath != null && audioDurationMs != null
}

/**
 * Represents the role of a message sender.
 */
enum class MessageRole(val value: String) {
    USER("user"),
    ASSISTANT("assistant"),
    SYSTEM("system");

    companion object {
        /**
         * Convert a string to MessageRole.
         */
        fun fromString(value: String): MessageRole {
            return entries.find { it.value == value } ?: USER
        }
    }
}

/**
 * Represents the synchronization status of a message.
 */
enum class SyncStatus(val value: String) {
    /**
     * Message exists locally but hasn't been synced to server.
     */
    PENDING("pending"),

    /**
     * Message has been successfully synchronized with the server.
     */
    SYNCED("synced"),

    /**
     * A conflict was detected between local and server versions.
     */
    CONFLICT("conflict");

    companion object {
        /**
         * Convert a string to SyncStatus.
         */
        fun fromString(value: String): SyncStatus {
            return entries.find { it.value == value } ?: SYNCED
        }
    }
}
