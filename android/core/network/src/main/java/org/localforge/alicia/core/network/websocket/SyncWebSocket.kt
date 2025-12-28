package org.localforge.alicia.core.network.websocket

import okhttp3.*
import okio.ByteString
import org.localforge.alicia.core.network.protocol.Envelope
import org.localforge.alicia.core.network.protocol.ProtocolHandler
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import timber.log.Timber
import javax.inject.Inject
import javax.inject.Singleton

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

    private var sessionId: String? = null

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

        _connectionState.value = WebSocketState.Connecting

        val request = Request.Builder()
            .url(url)
            .addHeader("Authorization", "Bearer $token")
            .build()

        webSocket = okHttpClient.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                Timber.d("WebSocket connection opened")
                sessionId = response.header("X-Session-ID") ?: generateSessionId()
                _connectionState.value = WebSocketState.Connected(sessionId!!)
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
                sessionId = null
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Timber.e(t, "WebSocket connection failed: ${response?.message}")
                _connectionState.value = WebSocketState.Error(t)
                sessionId = null
            }
        })
    }

    /**
     * Disconnect from the WebSocket server.
     * @param code Close code (default: 1000 for normal closure)
     * @param reason Close reason
     */
    fun disconnect(code: Int = 1000, reason: String? = null) {
        webSocket?.close(code, reason)
        webSocket = null
        _connectionState.value = WebSocketState.Disconnected
        sessionId = null
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
     * Get the current session ID.
     */
    fun getSessionId(): String? = sessionId

    private fun generateSessionId(): String {
        // Generate a simple session ID based on timestamp
        return "session_${System.currentTimeMillis()}_${(Math.random() * 10000).toInt()}"
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
     * @param sessionId Unique session identifier for this connection
     */
    data class Connected(val sessionId: String) : WebSocketState()

    /**
     * WebSocket encountered an error.
     * @param error The error that occurred
     */
    data class Error(val error: Throwable) : WebSocketState()
}
