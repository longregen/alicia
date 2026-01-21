package org.localforge.alicia.core.database.entity

import androidx.room.Entity
import androidx.room.ForeignKey
import androidx.room.Index
import androidx.room.PrimaryKey

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
    val conversationId: String,
    val role: String,
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
    val syncStatus: String = "pending",
    val syncedAt: Long? = null
)
