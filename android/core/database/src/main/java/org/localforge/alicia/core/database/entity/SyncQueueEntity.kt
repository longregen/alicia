package org.localforge.alicia.core.database.entity

import androidx.room.Entity
import androidx.room.Index
import androidx.room.PrimaryKey

/**
 * Room entity for storing pending sync operations.
 * This queue ensures messages are reliably synced to the server even when offline.
 */
@Entity(
    tableName = "sync_queue",
    indices = [
        Index(value = ["conversationId"]),
        Index(value = ["createdAt"])
    ]
)
data class SyncQueueEntity(
    /**
     * Local identifier for this sync operation (matches message localId).
     */
    @PrimaryKey
    val localId: String,

    /**
     * Type of sync operation (e.g., "message", "control", etc.).
     */
    val type: String,

    /**
     * Serialized data payload (MessagePack encoded envelope).
     */
    val data: ByteArray,

    /**
     * ID of the conversation this operation belongs to.
     */
    val conversationId: String,

    /**
     * Number of retry attempts made for this operation.
     */
    val retryCount: Int = 0,

    /**
     * Timestamp when this operation was created (milliseconds since epoch).
     */
    val createdAt: Long = System.currentTimeMillis()
) {
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (javaClass != other?.javaClass) return false

        other as SyncQueueEntity

        if (localId != other.localId) return false
        if (type != other.type) return false
        if (!data.contentEquals(other.data)) return false
        if (conversationId != other.conversationId) return false
        if (retryCount != other.retryCount) return false
        if (createdAt != other.createdAt) return false

        return true
    }

    override fun hashCode(): Int {
        var result = localId.hashCode()
        result = 31 * result + type.hashCode()
        result = 31 * result + data.contentHashCode()
        result = 31 * result + conversationId.hashCode()
        result = 31 * result + retryCount
        result = 31 * result + createdAt.hashCode()
        return result
    }
}
