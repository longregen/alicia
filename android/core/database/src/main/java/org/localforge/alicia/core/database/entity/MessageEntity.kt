package org.localforge.alicia.core.database.entity

import androidx.room.Entity
import androidx.room.ForeignKey
import androidx.room.Index
import androidx.room.PrimaryKey

/**
 * Room entity for storing messages in the local database.
 */
@Entity(
    tableName = "messages",
    foreignKeys = [
        ForeignKey(
            entity = ConversationEntity::class,
            parentColumns = ["id"],
            childColumns = ["conversationId"],
            onDelete = ForeignKey.CASCADE
        )
    ],
    indices = [
        Index(value = ["conversationId"]),
        Index(value = ["createdAt"]),
        Index(value = ["localId"]),
        Index(value = ["conversationId", "syncStatus"])
    ]
)
data class MessageEntity(
    @PrimaryKey
    val id: String,

    /**
     * ID of the conversation this message belongs to.
     */
    val conversationId: String,

    /**
     * Role of the message sender: "user" or "assistant".
     */
    val role: String,

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
     * Default is true since Alicia is primarily a voice-first assistant,
     * and most interactions are expected to be voice-based.
     */
    val isVoice: Boolean = true,

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
     * Current synchronization state: "pending", "synced", or "conflict".
     */
    val syncStatus: String = "pending",

    /**
     * Timestamp when the message was last synced with the server (milliseconds since epoch).
     */
    val syncedAt: Long? = null
)
