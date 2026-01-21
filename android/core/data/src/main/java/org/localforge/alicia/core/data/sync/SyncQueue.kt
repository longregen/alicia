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

@Singleton
class SyncQueue @Inject constructor(
    private val syncQueueDao: SyncQueueDao,
    private val protocolHandler: ProtocolHandler
) {
    suspend fun enqueue(message: MessageEntity, stanzaId: Int) {
        try {
            val envelope = createEnvelopeFromMessage(message, stanzaId)
            val data = protocolHandler.encode(envelope)
            val queueEntity = SyncQueueEntity(
                localId = message.localId ?: message.id,
                type = "message",
                data = data,
                conversationId = message.conversationId,
                retryCount = 0,
                createdAt = System.currentTimeMillis()
            )
            syncQueueDao.insert(queueEntity)
            Timber.d("Enqueued message for sync: localId=${queueEntity.localId}")
        } catch (e: Exception) {
            Timber.e(e, "Failed to enqueue message for sync")
            throw e
        }
    }

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

    suspend fun getPending(): List<SyncQueueEntity> {
        return syncQueueDao.getAll()
    }

    suspend fun getPendingForConversation(conversationId: String): List<SyncQueueEntity> {
        return syncQueueDao.getForConversation(conversationId)
    }

    fun getPendingCountFlow(): Flow<Int> {
        return syncQueueDao.getCountFlow()
    }

    suspend fun markSynced(localId: String) {
        try {
            syncQueueDao.delete(localId)
            Timber.d("Marked as synced and removed from queue: localId=$localId")
        } catch (e: Exception) {
            Timber.e(e, "Failed to mark as synced: localId=$localId")
            throw e
        }
    }

    suspend fun markSynced(localId: String, serverId: String) {
        try {
            syncQueueDao.delete(localId)
            Timber.d("Marked as synced: localId=$localId, serverId=$serverId")
        } catch (e: Exception) {
            Timber.e(e, "Failed to mark as synced: localId=$localId")
            throw e
        }
    }

    suspend fun incrementRetryCount(localId: String) {
        try {
            syncQueueDao.incrementRetryCount(localId)
            Timber.d("Incremented retry count for: localId=$localId")
        } catch (e: Exception) {
            Timber.e(e, "Failed to increment retry count: localId=$localId")
            throw e
        }
    }

    suspend fun getRetryable(maxRetries: Int = 3): List<SyncQueueEntity> {
        return syncQueueDao.getRetryable(maxRetries)
    }

    suspend fun clearForConversation(conversationId: String) {
        syncQueueDao.deleteForConversation(conversationId)
        Timber.d("Cleared sync queue for conversation: $conversationId")
    }

    suspend fun clearAll() {
        syncQueueDao.clear()
        Timber.d("Cleared entire sync queue")
    }

    suspend fun deleteFailedOperations(maxRetries: Int = 5) {
        syncQueueDao.deleteFailedOperations(maxRetries)
        Timber.d("Deleted failed operations with retries >= $maxRetries")
    }

    // Only user messages can be synced upstream. Assistant/system messages are server-generated
    // and pushed to clients, so they never need upstream sync.
    private fun createEnvelopeFromMessage(message: MessageEntity, stanzaId: Int): Envelope {
        val body = when (message.role) {
            "user" -> UserMessageBody(
                id = message.serverId ?: message.id,
                previousId = message.previousId,
                conversationId = message.conversationId,
                content = message.content,
                timestamp = message.createdAt
            )
            else -> throw IllegalArgumentException(
                "Only user messages can be enqueued for sync. " +
                "Assistant and system messages are server-generated. " +
                "Received role: ${message.role}"
            )
        }

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

    fun decodeEnvelope(queueEntity: SyncQueueEntity): Envelope {
        return protocolHandler.decode(queueEntity.data)
    }
}
