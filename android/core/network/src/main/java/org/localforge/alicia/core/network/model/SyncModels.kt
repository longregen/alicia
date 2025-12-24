package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Request model for syncing messages with the server.
 */
@JsonClass(generateAdapter = true)
data class SyncMessagesRequest(
    @Json(name = "messages")
    val messages: List<SyncMessageItem>
)

/**
 * Individual message item for sync request.
 */
@JsonClass(generateAdapter = true)
data class SyncMessageItem(
    @Json(name = "local_id")
    val localId: String,

    @Json(name = "sequence_number")
    val sequenceNumber: Int? = null,

    @Json(name = "previous_id")
    val previousId: String? = null,

    @Json(name = "role")
    val role: String,

    @Json(name = "contents")
    val contents: String,

    @Json(name = "created_at")
    val createdAt: String,

    @Json(name = "updated_at")
    val updatedAt: String? = null
)

/**
 * Response model for sync operation.
 */
@JsonClass(generateAdapter = true)
data class SyncMessagesResponse(
    @Json(name = "synced_messages")
    val syncedMessages: List<SyncedMessageResult>,

    @Json(name = "synced_at")
    val syncedAt: String
)

/**
 * Result for a single synced message.
 */
@JsonClass(generateAdapter = true)
data class SyncedMessageResult(
    @Json(name = "local_id")
    val localId: String,

    @Json(name = "server_id")
    val serverId: String? = null,

    @Json(name = "status")
    val status: String,

    @Json(name = "message")
    val message: MessageResponse? = null,

    @Json(name = "conflict")
    val conflict: ConflictDetails? = null
)

/**
 * Details about a sync conflict.
 */
@JsonClass(generateAdapter = true)
data class ConflictDetails(
    @Json(name = "reason")
    val reason: String,

    @Json(name = "server_message")
    val serverMessage: MessageResponse? = null,

    @Json(name = "resolution")
    val resolution: String = RESOLUTION_MANUAL
) {
    companion object {
        const val RESOLUTION_MANUAL = "manual"
    }
}

/**
 * Response model for sync status.
 */
@JsonClass(generateAdapter = true)
data class SyncStatusResponse(
    @Json(name = "conversation_id")
    val conversationId: String,

    @Json(name = "pending_count")
    val pendingCount: Int,

    @Json(name = "synced_count")
    val syncedCount: Int,

    @Json(name = "conflict_count")
    val conflictCount: Int,

    @Json(name = "last_synced_at")
    val lastSyncedAt: String? = null
)
