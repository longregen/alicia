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

/**
 * Manages WebSocket subscriptions for conversations, matching the web frontend's
 * subscription pattern (Subscribe, Unsubscribe, SubscribeAck, UnsubscribeAck).
 *
 * This component handles:
 * - Subscribing/unsubscribing to conversations
 * - Tracking active subscriptions
 * - Pending subscription promises with timeouts
 * - Auto-resubscription on reconnection
 */
@Singleton
class WebSocketSubscriptionManager @Inject constructor(
    private val syncWebSocket: SyncWebSocket
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    // Active subscriptions
    private val _activeSubscriptions = MutableStateFlow<Set<String>>(emptySet())
    val activeSubscriptions: StateFlow<Set<String>> = _activeSubscriptions.asStateFlow()

    // Pending subscription promises
    private val pendingSubscriptions = ConcurrentHashMap<String, CompletableDeferred<SubscribeAckBody>>()
    private val pendingUnsubscriptions = ConcurrentHashMap<String, CompletableDeferred<UnsubscribeAckBody>>()

    // Message handlers per conversation
    private val messageHandlers = ConcurrentHashMap<String, MutableSet<(Envelope) -> Unit>>()
    private val syncHandlers = ConcurrentHashMap<String, MutableSet<() -> Unit>>()

    // Stanza ID counter (positive for client-initiated)
    private val stanzaIdCounter = AtomicInteger(1)

    companion object {
        private const val SUBSCRIPTION_TIMEOUT_MS = 10_000L
    }

    init {
        // Observe incoming messages and route appropriately
        scope.launch {
            syncWebSocket.incomingMessages.collect { envelope ->
                handleIncomingMessage(envelope)
            }
        }

        // Auto-resubscribe on reconnection
        scope.launch {
            syncWebSocket.connectionState.collect { state ->
                if (state is WebSocketState.Connected) {
                    resubscribeAll()
                }
            }
        }
    }

    /**
     * Subscribe to a conversation.
     *
     * @param conversationId The conversation to subscribe to
     * @param fromSequence Optional sequence number to receive missed messages from
     * @return SubscribeAckBody on success
     * @throws Exception on timeout or failure
     */
    suspend fun subscribe(conversationId: String, fromSequence: Int? = null): SubscribeAckBody {
        // Check if already subscribed
        if (conversationId in _activeSubscriptions.value) {
            Timber.d("Already subscribed to conversation: $conversationId")
            return SubscribeAckBody(conversationId, success = true)
        }

        // Create pending subscription
        val deferred = CompletableDeferred<SubscribeAckBody>()
        pendingSubscriptions[conversationId] = deferred

        try {
            // Send subscribe envelope
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

            // Wait for acknowledgement with timeout
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

    /**
     * Unsubscribe from a conversation.
     *
     * @param conversationId The conversation to unsubscribe from
     * @return UnsubscribeAckBody on success
     */
    suspend fun unsubscribe(conversationId: String): UnsubscribeAckBody {
        // Check if not subscribed
        if (conversationId !in _activeSubscriptions.value) {
            Timber.d("Not subscribed to conversation: $conversationId")
            return UnsubscribeAckBody(conversationId, success = true)
        }

        // Optimistically remove from active subscriptions (matching web frontend behavior)
        _activeSubscriptions.value = _activeSubscriptions.value - conversationId

        // Create pending unsubscription
        val deferred = CompletableDeferred<UnsubscribeAckBody>()
        pendingUnsubscriptions[conversationId] = deferred

        try {
            // Send unsubscribe envelope
            val envelope = Envelope(
                stanzaId = stanzaIdCounter.getAndIncrement(),
                conversationId = conversationId,
                type = MessageType.UNSUBSCRIBE,
                body = UnsubscribeBody(conversationId)
            )

            val sent = syncWebSocket.send(envelope)
            if (!sent) {
                pendingUnsubscriptions.remove(conversationId)
                // Already removed from active subscriptions, consider it unsubscribed
                return UnsubscribeAckBody(conversationId, success = true)
            }

            // Wait for acknowledgement with timeout (shorter timeout for unsubscribe)
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

    /**
     * Register a message handler for a conversation.
     *
     * @param conversationId The conversation to handle messages for
     * @param handler The handler function
     * @return Unregister function
     */
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

    /**
     * Register a sync handler for a conversation.
     *
     * @param conversationId The conversation to handle sync events for
     * @param handler The handler function
     * @return Unregister function
     */
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

    /**
     * Notify sync handlers for a conversation.
     */
    fun notifySyncHandlers(conversationId: String) {
        syncHandlers[conversationId]?.forEach { handler ->
            try {
                handler()
            } catch (e: Exception) {
                Timber.e(e, "Sync handler error for $conversationId")
            }
        }
    }

    /**
     * Handle incoming WebSocket messages.
     */
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
                // Notify sync handlers
                notifySyncHandlers(envelope.conversationId)
            }

            else -> {
                // Route to message handlers
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

    /**
     * Re-subscribe to all previously active subscriptions.
     * Called on reconnection.
     */
    private suspend fun resubscribeAll() {
        val subscriptions = _activeSubscriptions.value.toList()
        Timber.d("Re-subscribing to ${subscriptions.size} conversations")

        subscriptions.forEach { conversationId ->
            try {
                // Temporarily remove from active to allow re-subscription
                _activeSubscriptions.value = _activeSubscriptions.value - conversationId
                subscribe(conversationId)
            } catch (e: Exception) {
                Timber.e(e, "Failed to re-subscribe to $conversationId")
            }
        }
    }

    /**
     * Check if subscribed to a conversation.
     */
    fun isSubscribed(conversationId: String): Boolean {
        return conversationId in _activeSubscriptions.value
    }

    /**
     * Clean up resources.
     */
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
