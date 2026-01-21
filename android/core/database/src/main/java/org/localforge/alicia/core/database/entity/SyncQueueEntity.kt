package org.localforge.alicia.core.database.entity

import androidx.room.Entity
import androidx.room.Index
import androidx.room.PrimaryKey

@Entity(
    tableName = "sync_queue",
    indices = [
        Index(value = ["conversationId"]),
        Index(value = ["createdAt"])
    ]
)
data class SyncQueueEntity(
    @PrimaryKey
    val localId: String,
    val type: String,
    val data: ByteArray,
    val conversationId: String,
    val retryCount: Int = 0,
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
