package org.localforge.alicia.core.data.sync

import org.localforge.alicia.core.database.dao.MessageDao
import org.localforge.alicia.core.network.protocol.Envelope
import org.localforge.alicia.core.network.protocol.MessageType
import org.localforge.alicia.core.network.protocol.ProtocolHandler
import org.localforge.alicia.core.network.protocol.bodies.AcknowledgementBody
import org.localforge.alicia.core.network.websocket.SyncWebSocket
import org.localforge.alicia.core.network.websocket.WebSocketState
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import timber.log.Timber
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeoutException
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class SyncEngine @Inject constructor(
    private val syncWebSocket: SyncWebSocket,
    private val syncQueue: SyncQueue,
    private val messageDao: MessageDao,
    private val protocolHandler: ProtocolHandler
) {
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private var syncJob: Job? = null
    private var messageProcessingJob: Job? = null
    private var currentConversationId: String? = null
    private var stanzaIdCounter = 0

    private val _syncState = MutableStateFlow<SyncState>(SyncState.Idle)
    val syncState: StateFlow<SyncState> = _syncState

    // Track pending acknowledgements: stanzaId -> AckResult
    private val pendingAcks = ConcurrentHashMap<Int, CompletableDeferred<AckResult>>()

    companion object {
        private const val ACK_TIMEOUT_MS = 5000L
    }

    suspend fun startSync(conversationId: String, serverUrl: String, token: String) {
        if (_syncState.value is SyncState.Syncing) {
            Timber.w("Sync already in progress")
            return
        }

        currentConversationId = conversationId
        _syncState.value = SyncState.Syncing(conversationId)

        // Connect to WebSocket
        syncWebSocket.connect(serverUrl, token)

        // Start processing incoming messages
        startMessageProcessing()

        // Start syncing pending messages
        startPendingSync()

        Timber.d("Started sync for conversation: $conversationId")
    }

    suspend fun stopSync() {
        Timber.d("Stopping sync")

        syncJob?.cancel()
        messageProcessingJob?.cancel()
        syncWebSocket.disconnect()

        // Clean up any pending acknowledgements
        pendingAcks.forEach { (stanzaId, deferred) ->
            deferred.completeExceptionally(
                IllegalStateException("Sync stopped while waiting for acknowledgement")
            )
        }
        pendingAcks.clear()

        currentConversationId = null
        _syncState.value = SyncState.Idle
    }

    suspend fun syncNow(): Result<SyncResult> {
        return try {
            if (!syncWebSocket.isConnected()) {
                return Result.failure(IllegalStateException("WebSocket not connected"))
            }

            val conversationId = currentConversationId
                ?: return Result.failure(IllegalStateException("No active conversation"))

            val pending = syncQueue.getPendingForConversation(conversationId)
            var successCount = 0
            var failureCount = 0

            for (queueEntity in pending) {
                try {
                    val envelope = syncQueue.decodeEnvelope(queueEntity)
                    val sent = syncWebSocket.send(envelope)

                    if (sent) {
                        // Wait for acknowledgement from server with timeout
                        val ackResult = waitForAcknowledgement(
                            stanzaId = envelope.stanzaId,
                            localId = queueEntity.localId
                        )

                        if (ackResult.success) {
                            syncQueue.markSynced(queueEntity.localId)
                            successCount++
                        } else {
                            Timber.w("Received negative acknowledgement for ${queueEntity.localId}: ${ackResult.error}")
                            syncQueue.incrementRetryCount(queueEntity.localId)
                            failureCount++
                        }
                    } else {
                        syncQueue.incrementRetryCount(queueEntity.localId)
                        failureCount++
                    }
                } catch (e: TimeoutException) {
                    Timber.e("Timeout waiting for acknowledgement: ${queueEntity.localId}")
                    syncQueue.incrementRetryCount(queueEntity.localId)
                    failureCount++
                } catch (e: Exception) {
                    Timber.e(e, "Failed to sync message: ${queueEntity.localId}")
                    syncQueue.incrementRetryCount(queueEntity.localId)
                    failureCount++
                }
            }

            Result.success(SyncResult(successCount, failureCount))
        } catch (e: Exception) {
            Timber.e(e, "Sync failed")
            Result.failure(e)
        }
    }

    fun getPendingCount(): Flow<Int> {
        return syncQueue.getPendingCountFlow()
    }

    fun getSyncStateFlow(): StateFlow<SyncState> = syncState

    private fun startMessageProcessing() {
        messageProcessingJob?.cancel()
        messageProcessingJob = scope.launch {
            syncWebSocket.incomingMessages.collect { envelope ->
                processIncomingMessage(envelope)
            }
        }
    }

    private fun startPendingSync() {
        syncJob?.cancel()
        syncJob = scope.launch {
            syncWebSocket.connectionState.collect { state ->
                when (state) {
                    is WebSocketState.Connected -> {
                        Timber.d("WebSocket connected, syncing pending messages")
                        syncPendingMessages()
                    }
                    is WebSocketState.Error -> {
                        Timber.e(state.error, "WebSocket error")
                        _syncState.value = SyncState.Error(state.error)
                    }
                    is WebSocketState.Disconnected -> {
                        Timber.d("WebSocket disconnected")
                        _syncState.value = SyncState.Idle
                    }
                    else -> {}
                }
            }
        }
    }

    private suspend fun syncPendingMessages() {
        try {
            val conversationId = currentConversationId ?: return
            val pending = syncQueue.getRetryable(maxRetries = 3)

            Timber.d("Syncing ${pending.size} pending messages")

            for (queueEntity in pending) {
                if (queueEntity.conversationId != conversationId) {
                    continue
                }

                try {
                    val envelope = syncQueue.decodeEnvelope(queueEntity)
                    val sent = syncWebSocket.send(envelope)

                    if (!sent) {
                        syncQueue.incrementRetryCount(queueEntity.localId)
                        Timber.w("Failed to send message: ${queueEntity.localId}")
                    }

                    delay(50)
                } catch (e: Exception) {
                    Timber.e(e, "Error syncing message: ${queueEntity.localId}")
                    syncQueue.incrementRetryCount(queueEntity.localId)
                }
            }
        } catch (e: Exception) {
            Timber.e(e, "Error in syncPendingMessages")
        }
    }

    private suspend fun processIncomingMessage(envelope: Envelope) {
        try {
            Timber.d("Processing incoming message: type=${envelope.type}, stanzaId=${envelope.stanzaId}")

            when (envelope.type) {
                MessageType.ACKNOWLEDGEMENT -> {
                    handleAcknowledgement(envelope)
                }
                MessageType.ASSISTANT_MESSAGE -> {
                    Timber.d("Received ASSISTANT_MESSAGE: conversationId=${envelope.conversationId}")
                }
                MessageType.ASSISTANT_SENTENCE -> {
                    Timber.v("Received ASSISTANT_SENTENCE: conversationId=${envelope.conversationId}")
                }
                MessageType.START_ANSWER -> {
                    Timber.v("Received START_ANSWER: conversationId=${envelope.conversationId}")
                }
                MessageType.ERROR_MESSAGE -> {
                    Timber.e("Received error message from server")
                }
                else -> {
                    Timber.d("Received message of type: ${envelope.type}")
                }
            }
        } catch (e: Exception) {
            Timber.e(e, "Error processing incoming message")
        }
    }

    private suspend fun handleAcknowledgement(envelope: Envelope) {
        try {
            val body = envelope.body as? AcknowledgementBody
            if (body == null) {
                Timber.w("Invalid acknowledgement body")
                return
            }

            val stanzaId = body.acknowledgedStanzaId
            val deferred = pendingAcks.remove(stanzaId)

            if (deferred != null) {
                if (body.success) {
                    deferred.complete(AckResult(success = true, error = null))
                    Timber.d("Acknowledgement received for stanzaId=$stanzaId")
                } else {
                    deferred.complete(AckResult(success = false, error = "Server rejected message"))
                    Timber.w("Received negative acknowledgement for stanzaId=$stanzaId")
                }
            } else {
                if (body.success) {
                    val localId = envelope.meta?.get("localId") as? String
                    if (localId != null) {
                        syncQueue.markSynced(localId)

                        // Update message sync status in database
                        messageDao.updateSyncStatus(
                            localId = localId,
                            serverId = null,
                            syncStatus = "synced",
                            syncedAt = System.currentTimeMillis()
                        )

                        Timber.d("Message acknowledged and synced: localId=$localId")
                    } else {
                        Timber.w("Acknowledgement missing localId in meta")
                    }
                } else {
                    Timber.w("Received negative acknowledgement for stanzaId: $stanzaId (no pending wait)")
                }
            }
        } catch (e: Exception) {
            Timber.e(e, "Error handling acknowledgement")
        }
    }

    private suspend fun waitForAcknowledgement(stanzaId: Int, localId: String): AckResult {
        val deferred = CompletableDeferred<AckResult>()
        pendingAcks[stanzaId] = deferred

        return try {
            withTimeout(ACK_TIMEOUT_MS) {
                deferred.await()
            }
        } catch (e: kotlinx.coroutines.TimeoutCancellationException) {
            pendingAcks.remove(stanzaId)
            Timber.w("Timeout waiting for acknowledgement: stanzaId=$stanzaId, localId=$localId")
            throw TimeoutException("Acknowledgement timeout for stanzaId=$stanzaId")
        }
    }

    fun getNextStanzaId(): Int {
        return ++stanzaIdCounter
    }

    fun shutdown() {
        scope.cancel()
        runBlocking {
            stopSync()
        }
    }
}

sealed class SyncState {
    object Idle : SyncState()
    data class Syncing(val conversationId: String) : SyncState()
    data class Error(val error: Throwable) : SyncState()
}

data class SyncResult(
    val successCount: Int,
    val failureCount: Int
) {
    val totalCount: Int get() = successCount + failureCount
    val isFullSuccess: Boolean get() = failureCount == 0
}

private data class AckResult(
    val success: Boolean,
    val error: String?
)
