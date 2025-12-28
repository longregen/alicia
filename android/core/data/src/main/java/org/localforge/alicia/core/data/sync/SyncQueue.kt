package org.localforge.alicia.core.data.sync

import org.localforge.alicia.core.database.dao.SyncQueueDao
import org.localforge.alicia.core.database.entity.MessageEntity
import org.localforge.alicia.core.database.entity.SyncQueueEntity
import org.localforge.alicia.core.network.protocol.Envelope
import org.localforge.alicia.core.network.protocol.MessageType
import org.localforge.alicia.core.network.protocol.ProtocolHandler
import org.localforge.alicia.core.network.protocol.bodies.UserMessageBody
import kotlinx.coroutines.flow.Flow
import timber.log.Timber
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Manages the queue of messages pending synchronization with the server.
 * Provides reliable message delivery by persisting messages until they're confirmed synced.
 */
@Singleton
class SyncQueue @Inject constructor(
    private val syncQueueDao: SyncQueueDao,
    private val protocolHandler: ProtocolHandler
) {
    /**
     * Enqueue a message for synchronization.
     * @param message Message entity to be synced
     * @param stanzaId Stanza ID for the protocol envelope
     */
    suspend fun enqueue(message: MessageEntity, stanzaId: Int) {
        try {
            // Create envelope from message
            val envelope = createEnvelopeFromMessage(message, stanzaId)

            // Encode envelope to bytes
            val data = protocolHandler.encode(envelope)

            // Create sync queue entity
            val queueEntity = SyncQueueEntity(
                localId = message.localId ?: message.id,
                type = "message",
                data = data,
                conversationId = message.conversationId,
                retryCount = 0,
                createdAt = System.currentTimeMillis()
            )

            // Insert into queue
            syncQueueDao.insert(queueEntity)
            Timber.d("Enqueued message for sync: localId=${queueEntity.localId}")
        } catch (e: Exception) {
            Timber.e(e, "Failed to enqueue message for sync")
            throw e
        }
    }

    /**
     * Enqueue a raw envelope for synchronization.
     * @param envelope Protocol envelope to be synced
     * @param localId Local identifier for tracking
     */
    suspend fun enqueueEnvelope(envelope: Envelope, localId: String) {
        try {
            val data = protocolHandler.encode(envelope)
            val queueEntity = SyncQueueEntity(
                localId = localId,
                type = envelope.type.name.lowercase(),
                data = data,
                conversationId = envelope.conversationId,
                retryCount = 0,
                createdAt = System.currentTimeMillis()
            )

            syncQueueDao.insert(queueEntity)
            Timber.d("Enqueued envelope for sync: localId=$localId, type=${envelope.type}")
        } catch (e: Exception) {
            Timber.e(e, "Failed to enqueue envelope for sync")
            throw e
        }
    }

    /**
     * Get all pending sync operations.
     * @return List of pending sync operations
     */
    suspend fun getPending(): List<SyncQueueEntity> {
        return syncQueueDao.getAll()
    }

    /**
     * Get pending sync operations for a specific conversation.
     * @param conversationId Conversation ID
     * @return List of pending sync operations for this conversation
     */
    suspend fun getPendingForConversation(conversationId: String): List<SyncQueueEntity> {
        return syncQueueDao.getForConversation(conversationId)
    }

    /**
     * Get the count of pending sync operations.
     * @return Flow of pending operation count
     */
    fun getPendingCountFlow(): Flow<Int> {
        return syncQueueDao.getCountFlow()
    }

    /**
     * Mark a sync operation as successfully synced and remove from queue.
     * @param localId Local identifier of the synced operation
     */
    suspend fun markSynced(localId: String) {
        try {
            syncQueueDao.delete(localId)
            Timber.d("Marked as synced and removed from queue: localId=$localId")
        } catch (e: Exception) {
            Timber.e(e, "Failed to mark as synced: localId=$localId")
            throw e
        }
    }

    /**
     * Mark a sync operation as successfully synced with server ID.
     * @param localId Local identifier of the synced operation
     * @param serverId Server-assigned identifier
     */
    suspend fun markSynced(localId: String, serverId: String) {
        try {
            syncQueueDao.delete(localId)
            Timber.d("Marked as synced: localId=$localId, serverId=$serverId")
        } catch (e: Exception) {
            Timber.e(e, "Failed to mark as synced: localId=$localId")
            throw e
        }
    }

    /**
     * Increment the retry count for a failed sync operation.
     * @param localId Local identifier of the operation
     */
    suspend fun incrementRetryCount(localId: String) {
        try {
            syncQueueDao.incrementRetryCount(localId)
            Timber.d("Incremented retry count for: localId=$localId")
        } catch (e: Exception) {
            Timber.e(e, "Failed to increment retry count: localId=$localId")
        }
    }

    /**
     * Get operations that can be retried (haven't exceeded max retries).
     * @param maxRetries Maximum retry count
     * @return List of retryable operations
     */
    suspend fun getRetryable(maxRetries: Int = 3): List<SyncQueueEntity> {
        return syncQueueDao.getRetryable(maxRetries)
    }

    /**
     * Clear all sync operations for a conversation.
     * @param conversationId Conversation ID
     */
    suspend fun clearForConversation(conversationId: String) {
        syncQueueDao.deleteForConversation(conversationId)
        Timber.d("Cleared sync queue for conversation: $conversationId")
    }

    /**
     * Clear all sync operations.
     */
    suspend fun clearAll() {
        syncQueueDao.clear()
        Timber.d("Cleared entire sync queue")
    }

    /**
     * Delete operations that have failed too many times.
     * @param maxRetries Maximum retry count before deletion
     */
    suspend fun deleteFailedOperations(maxRetries: Int = 5) {
        syncQueueDao.deleteFailedOperations(maxRetries)
        Timber.d("Deleted failed operations with retries >= $maxRetries")
    }

    private fun createEnvelopeFromMessage(message: MessageEntity, stanzaId: Int): Envelope {
        // Create appropriate body based on message role
        val body = when (message.role) {
            "user" -> UserMessageBody(
                id = message.serverId ?: message.id,
                previousId = message.previousId,
                conversationId = message.conversationId,
                content = message.content,
                timestamp = message.createdAt
            )
            else -> throw IllegalArgumentException("Cannot create envelope for role: ${message.role}")
        }

        // Create envelope
        return Envelope(
            stanzaId = stanzaId,
            conversationId = message.conversationId,
            type = MessageType.USER_MESSAGE,
            meta = mapOf(
                "timestamp" to message.createdAt,
                "localId" to (message.localId ?: message.id)
            ),
            body = body
        )
    }

    /**
     * Decode a sync queue entity back to an envelope.
     * @param queueEntity Sync queue entity to decode
     * @return Decoded envelope
     */
    fun decodeEnvelope(queueEntity: SyncQueueEntity): Envelope {
        return protocolHandler.decode(queueEntity.data)
    }
}
