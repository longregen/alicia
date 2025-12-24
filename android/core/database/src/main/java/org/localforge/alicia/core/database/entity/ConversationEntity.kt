package org.localforge.alicia.core.database.entity

import androidx.room.Entity
import androidx.room.PrimaryKey

/**
 * Room entity for storing conversations in the local database.
 */
@Entity(tableName = "conversations")
data class ConversationEntity(
    @PrimaryKey
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
    val isDeleted: Boolean = false
)
