package org.localforge.alicia.core.network.websocket

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import org.localforge.alicia.core.network.protocol.Envelope
import org.localforge.alicia.core.network.protocol.MessageType
import org.localforge.alicia.core.network.protocol.bodies.SubscribeAckBody
import org.localforge.alicia.core.network.protocol.bodies.SubscribeBody
import org.localforge.alicia.core.network.protocol.bodies.UnsubscribeAckBody
import org.localforge.alicia.core.network.protocol.bodies.UnsubscribeBody
import timber.log.Timber
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicInteger
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class WebSocketSubscriptionManager @Inject constructor(
    private val syncWebSocket: SyncWebSocket
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val _activeSubscriptions = MutableStateFlow<Set<String>>(emptySet())
    val activeSubscriptions: StateFlow<Set<String>> = _activeSubscriptions.asStateFlow()

    private val pendingSubscriptions = ConcurrentHashMap<String, CompletableDeferred<SubscribeAckBody>>()
    private val pendingUnsubscriptions = ConcurrentHashMap<String, CompletableDeferred<UnsubscribeAckBody>>()

    private val messageHandlers = ConcurrentHashMap<String, MutableSet<(Envelope) -> Unit>>()
    private val syncHandlers = ConcurrentHashMap<String, MutableSet<() -> Unit>>()

    private val stanzaIdCounter = AtomicInteger(1)

    companion object {
        private const val SUBSCRIPTION_TIMEOUT_MS = 10_000L
    }

    init {
        scope.launch {
            syncWebSocket.incomingMessages.collect { envelope ->
                handleIncomingMessage(envelope)
            }
        }

        scope.launch {
            syncWebSocket.connectionState.collect { state ->
                if (state is WebSocketState.Connected) {
                    resubscribeAll()
                }
            }
        }
    }

    suspend fun subscribe(conversationId: String, fromSequence: Int? = null): SubscribeAckBody {
        if (conversationId in _activeSubscriptions.value) {
            Timber.d("Already subscribed to conversation: $conversationId")
            return SubscribeAckBody(conversationId, success = true)
        }

        val deferred = CompletableDeferred<SubscribeAckBody>()
        pendingSubscriptions[conversationId] = deferred

        try {
            val envelope = Envelope(
                stanzaId = stanzaIdCounter.getAndIncrement(),
                conversationId = conversationId,
                type = MessageType.SUBSCRIBE,
                body = SubscribeBody(conversationId, fromSequence)
            )

            val sent = syncWebSocket.send(envelope)
            if (!sent) {
                pendingSubscriptions.remove(conversationId)
                throw Exception("Failed to send subscribe message: WebSocket not connected")
            }

            return withTimeout(SUBSCRIPTION_TIMEOUT_MS) {
                deferred.await()
            }.also { ack ->
                if (ack.success) {
                    _activeSubscriptions.value = _activeSubscriptions.value + conversationId
                    Timber.d("Subscribed to conversation: $conversationId, missed=${ack.missedMessages}")
                } else {
                    Timber.w("Subscription failed for $conversationId: ${ack.error}")
                    throw Exception("Subscription failed: ${ack.error}")
                }
            }
        } finally {
            pendingSubscriptions.remove(conversationId)
        }
    }

    suspend fun unsubscribe(conversationId: String): UnsubscribeAckBody {
        if (conversationId !in _activeSubscriptions.value) {
            Timber.d("Not subscribed to conversation: $conversationId")
            return UnsubscribeAckBody(conversationId, success = true)
        }

        _activeSubscriptions.value = _activeSubscriptions.value - conversationId

        val deferred = CompletableDeferred<UnsubscribeAckBody>()
        pendingUnsubscriptions[conversationId] = deferred

        try {
            val envelope = Envelope(
                stanzaId = stanzaIdCounter.getAndIncrement(),
                conversationId = conversationId,
                type = MessageType.UNSUBSCRIBE,
                body = UnsubscribeBody(conversationId)
            )

            val sent = syncWebSocket.send(envelope)
            if (!sent) {
                pendingUnsubscriptions.remove(conversationId)
                return UnsubscribeAckBody(conversationId, success = true)
            }

            return withTimeout(SUBSCRIPTION_TIMEOUT_MS) {
                deferred.await()
            }.also { ack ->
                Timber.d("Unsubscribed from conversation: $conversationId, success=${ack.success}")
            }
        } catch (e: Exception) {
            Timber.w("Unsubscribe timeout for $conversationId, treating as success")
            return UnsubscribeAckBody(conversationId, success = true)
        } finally {
            pendingUnsubscriptions.remove(conversationId)
        }
    }

    fun registerMessageHandler(conversationId: String, handler: (Envelope) -> Unit): () -> Unit {
        val handlers = messageHandlers.getOrPut(conversationId) { mutableSetOf() }
        handlers.add(handler)
        Timber.d("Registered message handler for $conversationId, total=${handlers.size}")
        return {
            handlers.remove(handler)
            if (handlers.isEmpty()) {
                messageHandlers.remove(conversationId)
            }
        }
    }

    fun registerSyncHandler(conversationId: String, handler: () -> Unit): () -> Unit {
        val handlers = syncHandlers.getOrPut(conversationId) { mutableSetOf() }
        handlers.add(handler)
        return {
            handlers.remove(handler)
            if (handlers.isEmpty()) {
                syncHandlers.remove(conversationId)
            }
        }
    }

    fun notifySyncHandlers(conversationId: String) {
        syncHandlers[conversationId]?.forEach { handler ->
            try {
                handler()
            } catch (e: Exception) {
                Timber.e(e, "Sync handler error for $conversationId")
            }
        }
    }

    private fun handleIncomingMessage(envelope: Envelope) {
        when (envelope.type) {
            MessageType.SUBSCRIBE_ACK -> {
                val body = envelope.body as? SubscribeAckBody
                    ?: run {
                        Timber.w("Invalid SubscribeAck body")
                        return
                    }
                pendingSubscriptions[body.conversationId]?.complete(body)
            }

            MessageType.UNSUBSCRIBE_ACK -> {
                val body = envelope.body as? UnsubscribeAckBody
                    ?: run {
                        Timber.w("Invalid UnsubscribeAck body")
                        return
                    }
                pendingUnsubscriptions[body.conversationId]?.complete(body)
            }

            MessageType.SYNC_RESPONSE -> {
                notifySyncHandlers(envelope.conversationId)
            }

            else -> {
                messageHandlers[envelope.conversationId]?.forEach { handler ->
                    try {
                        handler(envelope)
                    } catch (e: Exception) {
                        Timber.e(e, "Message handler error for ${envelope.conversationId}")
                    }
                }
            }
        }
    }

    private suspend fun resubscribeAll() {
        val subscriptions = _activeSubscriptions.value.toList()
        Timber.d("Re-subscribing to ${subscriptions.size} conversations")

        subscriptions.forEach { conversationId ->
            try {
                _activeSubscriptions.value = _activeSubscriptions.value - conversationId
                subscribe(conversationId)
            } catch (e: Exception) {
                Timber.e(e, "Failed to re-subscribe to $conversationId")
            }
        }
    }

    fun isSubscribed(conversationId: String): Boolean {
        return conversationId in _activeSubscriptions.value
    }

    fun cleanup() {
        messageHandlers.clear()
        syncHandlers.clear()
        pendingSubscriptions.values.forEach { it.cancel() }
        pendingUnsubscriptions.values.forEach { it.cancel() }
        pendingSubscriptions.clear()
        pendingUnsubscriptions.clear()
        _activeSubscriptions.value = emptySet()
    }
}
