package org.localforge.alicia.core.domain.model

import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/**
 * Domain model for a conversation with the voice assistant.
 */
data class Conversation(
    /**
     * Unique identifier for the conversation.
     */
    val id: String,

    /**
     * Optional title for the conversation.
     * Can be generated from first message or user-provided.
     */
    val title: String? = null,

    /**
     * Timestamp when the conversation was created (milliseconds since epoch).
     */
    val createdAt: Long,

    /**
     * Timestamp when the conversation was last updated (milliseconds since epoch).
     */
    val updatedAt: Long,

    /**
     * Timestamp when the conversation was last synced with the server.
     * Null if never synced.
     */
    val syncedAt: Long? = null,

    /**
     * Whether this conversation has been deleted locally but not yet synced.
     */
    val isDeleted: Boolean = false,

    /**
     * Optional preview of the last message in the conversation.
     * This is not stored in the database entity but can be added when loading.
     */
    val lastMessagePreview: String? = null,

    /**
     * Number of messages in this conversation.
     * This is not stored in the database entity but can be added when loading.
     */
    val messageCount: Int = 0
) {
    /**
     * Display title for the conversation.
     * Returns the title if set, otherwise a default based on creation time.
     */
    val displayTitle: String
        get() = title ?: "Conversation ${formatTimestamp(createdAt)}"

    /**
     * Check if the conversation needs to be synced.
     */
    val needsSync: Boolean
        get() = syncedAt == null || syncedAt < updatedAt

    /**
     * Check if the conversation was updated recently (within last hour).
     */
    val isRecent: Boolean
        get() = System.currentTimeMillis() - updatedAt < 60 * 60 * 1000

    companion object {
        /**
         * Date formatter for conversation timestamps.
         */
        private val dateFormatter = DateTimeFormatter.ofPattern("MMM dd, yyyy")
            .withZone(ZoneId.systemDefault())

        /**
         * Format timestamp to a readable date string.
         */
        private fun formatTimestamp(timestamp: Long): String {
            return dateFormatter.format(Instant.ofEpochMilli(timestamp))
        }
    }
}
