package org.localforge.alicia.core.network.websocket

import okhttp3.*
import okio.ByteString
import org.localforge.alicia.core.network.protocol.Envelope
import org.localforge.alicia.core.network.protocol.ProtocolHandler
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import timber.log.Timber
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.math.min
import kotlin.math.pow

/**
 * WebSocket connection for real-time message synchronization.
 * Manages the lifecycle of the WebSocket connection and handles incoming/outgoing protocol messages.
 */
@Singleton
class SyncWebSocket @Inject constructor(
    private val okHttpClient: OkHttpClient,
    private val protocolHandler: ProtocolHandler
) {
    private var webSocket: WebSocket? = null
    private val _connectionState = MutableStateFlow<WebSocketState>(WebSocketState.Disconnected)
    val connectionState: StateFlow<WebSocketState> = _connectionState

    private val _incomingMessages = MutableSharedFlow<Envelope>(replay = 0, extraBufferCapacity = 64)
    val incomingMessages: SharedFlow<Envelope> = _incomingMessages

    // Reconnection logic
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var reconnectJob: Job? = null
    private var reconnectAttempts = 0
    private var isIntentionalDisconnect = false
    private var lastUrl: String? = null
    private var lastToken: String? = null

    companion object {
        private const val INITIAL_RECONNECT_DELAY_MS = 1000L
        private const val MAX_RECONNECT_DELAY_MS = 30000L
    }

    /**
     * Connect to the WebSocket server.
     * @param url WebSocket server URL (e.g., "wss://api.example.com/sync")
     * @param token Authentication token
     */
    fun connect(url: String, token: String) {
        if (_connectionState.value is WebSocketState.Connected ||
            _connectionState.value is WebSocketState.Connecting) {
            Timber.w("Already connected or connecting to WebSocket")
            return
        }

        // Cancel any pending reconnection attempt
        reconnectJob?.cancel()
        reconnectJob = null

        // Store connection parameters for reconnection
        lastUrl = url
        lastToken = token
        isIntentionalDisconnect = false

        _connectionState.value = WebSocketState.Connecting

        val request = Request.Builder()
            .url(url)
            .addHeader("Authorization", "Bearer $token")
            .build()

        webSocket = okHttpClient.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                Timber.d("WebSocket connection opened")
                _connectionState.value = WebSocketState.Connected
                reconnectAttempts = 0
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                Timber.w("Received unexpected text message (expected binary): $text")
            }

            override fun onMessage(webSocket: WebSocket, bytes: ByteString) {
                try {
                    val envelope = protocolHandler.decode(bytes.toByteArray())
                    Timber.d("Received message: type=${envelope.type}, stanzaId=${envelope.stanzaId}")
                    _incomingMessages.tryEmit(envelope)
                } catch (e: Exception) {
                    Timber.e(e, "Failed to decode incoming message")
                    _connectionState.value = WebSocketState.Error(e)
                }
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                Timber.d("WebSocket closing: code=$code, reason=$reason")
                webSocket.close(1000, null)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                Timber.d("WebSocket closed: code=$code, reason=$reason")
                _connectionState.value = WebSocketState.Disconnected

                // Schedule reconnection if not intentionally disconnected
                if (!isIntentionalDisconnect && lastUrl != null && lastToken != null) {
                    scheduleReconnect()
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Timber.e(t, "WebSocket connection failed: ${response?.message}")
                _connectionState.value = WebSocketState.Error(t)

                // Schedule reconnection if not intentionally disconnected
                if (!isIntentionalDisconnect && lastUrl != null && lastToken != null) {
                    scheduleReconnect()
                }
            }
        })
    }

    /**
     * Disconnect from the WebSocket server.
     * This will prevent automatic reconnection.
     * @param code Close code (default: 1000 for normal closure)
     * @param reason Close reason
     */
    fun disconnect(code: Int = 1000, reason: String? = null) {
        // Mark as intentional disconnect to prevent reconnection
        isIntentionalDisconnect = true

        // Cancel any pending reconnection attempt
        reconnectJob?.cancel()
        reconnectJob = null

        webSocket?.close(code, reason)
        webSocket = null
        _connectionState.value = WebSocketState.Disconnected
    }

    /**
     * Send an envelope through the WebSocket.
     * @param envelope Protocol envelope to send
     * @return true if the message was queued for sending, false otherwise
     */
    fun send(envelope: Envelope): Boolean {
        val ws = webSocket
        if (ws == null || _connectionState.value !is WebSocketState.Connected) {
            Timber.w("Cannot send message: WebSocket not connected")
            return false
        }

        return try {
            val bytes = protocolHandler.encode(envelope)
            val sent = ws.send(ByteString.of(*bytes))
            if (sent) {
                Timber.d("Sent message: type=${envelope.type}, stanzaId=${envelope.stanzaId}")
            } else {
                Timber.w("Failed to queue message for sending (buffer full?)")
            }
            sent
        } catch (e: Exception) {
            Timber.e(e, "Failed to encode and send message")
            false
        }
    }

    /**
     * Check if the WebSocket is currently connected.
     */
    fun isConnected(): Boolean = _connectionState.value is WebSocketState.Connected

    /**
     * Schedule automatic reconnection with exponential backoff.
     * Matches the frontend implementation behavior.
     */
    private fun scheduleReconnect() {
        // Cancel any existing reconnection job
        reconnectJob?.cancel()

        // Calculate delay with exponential backoff: min(1000 * 2^attempts, 30000)
        val delay = min(
            INITIAL_RECONNECT_DELAY_MS * 2.0.pow(reconnectAttempts.toDouble()).toLong(),
            MAX_RECONNECT_DELAY_MS
        )

        Timber.d("Scheduling reconnection attempt ${reconnectAttempts + 1} in ${delay}ms")
        reconnectAttempts++

        reconnectJob = scope.launch {
            delay(delay)
            if (!isIntentionalDisconnect && lastUrl != null && lastToken != null) {
                Timber.d("Attempting to reconnect to WebSocket")
                connect(lastUrl!!, lastToken!!)
            }
        }
    }

    /**
     * Clean up resources. Should be called when the component is destroyed.
     */
    fun cleanup() {
        isIntentionalDisconnect = true
        reconnectJob?.cancel()
        webSocket?.close(1000, "Cleanup")
        webSocket = null
        scope.cancel()
    }
}

/**
 * Represents the current state of the WebSocket connection.
 */
sealed class WebSocketState {
    /**
     * WebSocket is disconnected.
     */
    object Disconnected : WebSocketState()

    /**
     * WebSocket is attempting to connect.
     */
    object Connecting : WebSocketState()

    /**
     * WebSocket is connected and ready for communication.
     */
    object Connected : WebSocketState()

    /**
     * WebSocket encountered an error.
     * @param error The error that occurred
     */
    data class Error(val error: Throwable) : WebSocketState()
}
