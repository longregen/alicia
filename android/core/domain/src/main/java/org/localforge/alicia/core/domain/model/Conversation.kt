package org.localforge.alicia.core.domain.model

import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

enum class ConversationStatus(val value: String) {
    ACTIVE("active"),
    ARCHIVED("archived");

    companion object {
        fun fromString(value: String?): ConversationStatus {
            return entries.find { it.value == value } ?: ACTIVE
        }
    }
}

data class Conversation(
    val id: String,
    val title: String? = null,
    val status: ConversationStatus = ConversationStatus.ACTIVE,
    val createdAt: Long,
    val updatedAt: Long,
    val syncedAt: Long? = null,
    val isDeleted: Boolean = false,
    val lastMessagePreview: String? = null,
    val messageCount: Int = 0
) {
    val displayTitle: String
        get() = title ?: "Conversation ${formatTimestamp(createdAt)}"

    val needsSync: Boolean
        get() = syncedAt == null || syncedAt < updatedAt

    val isRecent: Boolean
        get() = System.currentTimeMillis() - updatedAt < 60 * 60 * 1000

    companion object {
        private val dateFormatter = DateTimeFormatter.ofPattern("MMM dd, yyyy")
            .withZone(ZoneId.systemDefault())

        private fun formatTimestamp(timestamp: Long): String {
            return dateFormatter.format(Instant.ofEpochMilli(timestamp))
        }
    }
}
