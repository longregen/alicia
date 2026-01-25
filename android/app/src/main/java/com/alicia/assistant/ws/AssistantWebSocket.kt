package com.alicia.assistant.ws

import android.util.Log
import com.alicia.assistant.telemetry.AliciaTelemetry
import io.opentelemetry.api.common.Attributes
import io.opentelemetry.api.trace.Span
import kotlinx.coroutines.*
import okhttp3.*
import okio.ByteString
import okio.ByteString.Companion.toByteString
import org.msgpack.core.MessagePack
import org.msgpack.value.Value
import org.msgpack.value.ValueFactory
import java.io.ByteArrayOutputStream
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit

interface MessageListener {
    fun onUserMessage(conversationId: String, messageId: String, content: String, previousId: String?)
    fun onAssistantMessage(conversationId: String, messageId: String, content: String, previousId: String?)
    fun onThinkingUpdate(conversationId: String, messageId: String, content: String, progress: Float)
    fun onToolUseStarted(conversationId: String, toolName: String, arguments: Map<String, Any?>)
    fun onToolUseCompleted(conversationId: String, toolName: String, success: Boolean)
    fun onError(conversationId: String, error: String)
}

class AssistantWebSocket(
    private val baseUrl: String,
    private val agentSecret: String,
    private val toolRegistry: ToolRegistry
) {
    companion object {
        private const val TAG = "AssistantWebSocket"

        private const val TYPE_USER_MESSAGE = 2
        private const val TYPE_ASSISTANT_MSG = 3
        private const val TYPE_TOOL_USE_REQUEST = 6
        private const val TYPE_TOOL_USE_RESULT = 7
        private const val TYPE_THINKING_SUMMARY = 34
        private const val TYPE_SUBSCRIBE = 40
        private const val TYPE_UNSUBSCRIBE = 41
        private const val TYPE_SUBSCRIBE_ACK = 42
        private const val TYPE_ASSISTANT_TOOLS_REGISTER = 70
        private const val TYPE_ASSISTANT_TOOLS_ACK = 71
        private const val TYPE_ASSISTANT_HEARTBEAT = 72
        private const val HEARTBEAT_INTERVAL_MS = 15_000L

        private const val RECONNECT_DELAY_MS = 5000L
        private const val MAX_RECONNECT_DELAY_MS = 60000L
    }

    // Listeners keyed by conversation ID to support multiple concurrent conversations
    private val conversationListeners = ConcurrentHashMap<String, MessageListener>()

    /**
     * Register a listener for a specific conversation.
     * Multiple conversations can have different listeners simultaneously.
     */
    fun addMessageListener(conversationId: String, listener: MessageListener) {
        conversationListeners[conversationId] = listener
    }

    /**
     * Remove listener for a specific conversation.
     */
    fun removeMessageListener(conversationId: String) {
        conversationListeners.remove(conversationId)
    }

    @Deprecated("Use addMessageListener(conversationId, listener) instead", ReplaceWith("addMessageListener(conversationId, listener)"))
    fun setMessageListener(listener: MessageListener?) {
        // Legacy support: sets a global listener for empty conversation ID
        if (listener != null) {
            conversationListeners[""] = listener
        } else {
            conversationListeners.remove("")
        }
    }

    private fun getListener(conversationId: String): MessageListener? {
        return conversationListeners[conversationId] ?: conversationListeners[""]
    }

    private val client: OkHttpClient = OkHttpClient.Builder()
        .readTimeout(0, TimeUnit.MILLISECONDS)
        .build()

    private var webSocket: WebSocket? = null
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private var reconnectJob: Job? = null
    private var reconnectDelay = RECONNECT_DELAY_MS
    @Volatile
    private var connected = false
    @Volatile
    private var reconnecting = false
    private var heartbeatJob: Job? = null
    @Volatile
    private var heartbeatEnabled = false
    @Volatile
    private var lastSendTime = 0L
    @Volatile
    private var sessionSpan: Span? = null
    private val subscribedConversations = mutableSetOf<String>()

    private fun closeCurrentConnection() {
        webSocket?.close(1000, "Replacing connection")
        webSocket = null
    }

    private fun sendRaw(data: ByteArray): Boolean {
        val sent = webSocket?.send(data.toByteString()) ?: false
        if (sent) lastSendTime = System.currentTimeMillis()
        return sent
    }

    fun connect() {
        closeCurrentConnection()

        val request = Request.Builder()
            .url(baseUrl)
            .apply {
                if (agentSecret.isNotEmpty()) {
                    addHeader("Authorization", "Bearer $agentSecret")
                }
            }
            .build()

        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(ws: WebSocket, response: Response) {
                Log.i(TAG, "WebSocket connected")
                sessionSpan = AliciaTelemetry.startSpan("websocket.session", Attributes.builder()
                    .put("ws.url", baseUrl)
                    .build())
                reconnectDelay = RECONNECT_DELAY_MS
                reconnecting = false
                sendSubscribe(ws)
            }

            override fun onMessage(ws: WebSocket, bytes: ByteString) {
                handleMessage(bytes.toByteArray())
            }

            override fun onFailure(ws: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "WebSocket failure: ${t.message}")
                sessionSpan?.let { span ->
                    AliciaTelemetry.addSpanEvent(span, "websocket.failure", Attributes.builder()
                        .put("error.message", t.message ?: "unknown")
                        .build())
                    AliciaTelemetry.recordError(span, t)
                    span.end()
                    sessionSpan = null
                }
                connected = false
                scheduleReconnect()
            }

            override fun onClosed(ws: WebSocket, code: Int, reason: String) {
                Log.i(TAG, "WebSocket closed: $reason")
                sessionSpan?.let { span ->
                    AliciaTelemetry.addSpanEvent(span, "websocket.closed", Attributes.builder()
                        .put("ws.close_code", code.toLong())
                        .put("ws.close_reason", reason)
                        .build())
                    span.end()
                    sessionSpan = null
                }
                connected = false
                scheduleReconnect()
            }
        })
    }

    fun disconnect() {
        stopHeartbeat()
        reconnecting = false
        reconnectJob?.cancel()
        subscribedConversations.clear()
        webSocket?.close(1000, "Client closing")
        webSocket = null
        scope.cancel()
    }

    fun startHeartbeat() {
        heartbeatEnabled = true
        heartbeatJob?.cancel()
        heartbeatJob = scope.launch {
            while (isActive && heartbeatEnabled) {
                sendHeartbeat()
                delay(HEARTBEAT_INTERVAL_MS)
            }
        }
        Log.i(TAG, "Heartbeat started (interval=${HEARTBEAT_INTERVAL_MS}ms)")
    }

    fun stopHeartbeat() {
        heartbeatEnabled = false
        heartbeatJob?.cancel()
        heartbeatJob = null
        Log.i(TAG, "Heartbeat stopped")
    }

    fun isConnected(): Boolean = connected

    /**
     * Send a user message to the server for processing by the agent.
     * The response will be delivered via the MessageListener callbacks.
     */
    fun sendUserMessage(conversationId: String, content: String, previousId: String? = null): Boolean {
        if (!connected) {
            Log.w(TAG, "Cannot send message: not connected")
            return false
        }

        val body = mutableMapOf<String, Any?>(
            "id" to java.util.UUID.randomUUID().toString(),
            "conversationId" to conversationId,
            "content" to content
        )
        if (previousId != null) {
            body["previousId"] = previousId
        }
        AliciaTelemetry.injectTraceContext(body)

        val envelope = buildEnvelope(conversationId, TYPE_USER_MESSAGE, body)
        val sent = sendRaw(envelope)
        if (sent) {
            Log.i(TAG, "Sent user message to conversation $conversationId")
        } else {
            Log.w(TAG, "Failed to send user message")
        }
        return sent
    }

    private fun sendHeartbeat() {
        if (!connected) return
        if (System.currentTimeMillis() - lastSendTime < HEARTBEAT_INTERVAL_MS) return
        val envelope = buildEnvelope("", TYPE_ASSISTANT_HEARTBEAT, emptyMap<String, Any>())
        sendRaw(envelope)
    }

    private fun sendSubscribe(ws: WebSocket) {
        val parentSpan = sessionSpan
        val subscribeSpan = if (parentSpan != null) {
            AliciaTelemetry.startChildSpan("websocket.subscribe", parentSpan)
        } else {
            AliciaTelemetry.startSpan("websocket.subscribe")
        }
        val body = mutableMapOf<String, Any?>("assistantMode" to true)
        AliciaTelemetry.injectTraceContext(body, subscribeSpan)
        val envelope = buildEnvelope("", TYPE_SUBSCRIBE, body)
        if (!sendRaw(envelope)) {
            Log.w(TAG, "Failed to send subscribe message")
        }
        subscribeSpan.end()
    }

    /**
     * Subscribe to a conversation to receive messages and updates.
     * Must be called before sending messages to that conversation.
     */
    fun subscribeToConversation(conversationId: String): Boolean {
        if (!connected) {
            Log.w(TAG, "Cannot subscribe: not connected")
            return false
        }
        if (conversationId in subscribedConversations) {
            return true // Already subscribed
        }

        val body = mutableMapOf<String, Any?>("conversationId" to conversationId)
        AliciaTelemetry.injectTraceContext(body)
        val envelope = buildEnvelope(conversationId, TYPE_SUBSCRIBE, body)
        val sent = sendRaw(envelope)
        if (sent) {
            subscribedConversations.add(conversationId)
            Log.i(TAG, "Subscribed to conversation $conversationId")
        } else {
            Log.w(TAG, "Failed to subscribe to conversation $conversationId")
        }
        return sent
    }

    /**
     * Unsubscribe from a conversation.
     */
    fun unsubscribeFromConversation(conversationId: String): Boolean {
        if (!connected) return false
        subscribedConversations.remove(conversationId)

        val body = mapOf("conversationId" to conversationId)
        val envelope = buildEnvelope(conversationId, TYPE_UNSUBSCRIBE, body)
        return sendRaw(envelope)
    }

    private fun registerTools(ws: WebSocket) {
        val tools = toolRegistry.getAll().map { executor ->
            mapOf(
                "name" to executor.name,
                "description" to executor.description,
                "inputSchema" to executor.inputSchema
            )
        }
        val parentSpan = sessionSpan
        val registerSpan = if (parentSpan != null) {
            AliciaTelemetry.startChildSpan("websocket.tools_register", parentSpan, Attributes.builder()
                .put("ws.tool_count", tools.size.toLong())
                .build())
        } else {
            AliciaTelemetry.startSpan("websocket.tools_register", Attributes.builder()
                .put("ws.tool_count", tools.size.toLong())
                .build())
        }
        val body = mapOf("tools" to tools)
        val envelope = buildEnvelope("", TYPE_ASSISTANT_TOOLS_REGISTER, body)
        if (!sendRaw(envelope)) {
            Log.w(TAG, "Failed to send tools registration")
            registerSpan.end()
            return
        }
        Log.i(TAG, "Registered ${tools.size} tools")
        registerSpan.end()
    }

    private fun handleMessage(data: ByteArray) {
        try {
            val envelope = decodeEnvelope(data) ?: return
            val type = (envelope["type"] as? Number)?.toInt() ?: return
            val conversationId = envelope["conversationId"] as? String ?: ""

            when (type) {
                TYPE_SUBSCRIBE_ACK -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val success = body["success"] as? Boolean ?: false
                    if (success) {
                        connected = true
                        Log.i(TAG, "Subscribed as assistant")
                        webSocket?.let { registerTools(it) }
                        if (heartbeatEnabled) {
                            heartbeatJob?.cancel()
                            startHeartbeat()
                        }
                    } else {
                        val error = body["error"] as? String ?: "unknown"
                        Log.e(TAG, "Subscribe failed: $error")
                    }
                }
                TYPE_ASSISTANT_TOOLS_ACK -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val count = (body["toolCount"] as? Number)?.toInt() ?: 0
                    Log.i(TAG, "Tools acknowledged: $count")
                }
                TYPE_USER_MESSAGE -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val messageId = body["id"] as? String ?: return
                    val content = body["content"] as? String ?: ""
                    val previousId = body["previousId"] as? String
                    Log.i(TAG, "Received user message confirmation: $messageId")
                    getListener(conversationId)?.onUserMessage(conversationId, messageId, content, previousId)
                }
                TYPE_ASSISTANT_MSG -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val messageId = body["id"] as? String ?: return
                    val content = body["content"] as? String ?: ""
                    val previousId = body["previousId"] as? String
                    Log.i(TAG, "Received assistant message: $messageId")
                    getListener(conversationId)?.onAssistantMessage(conversationId, messageId, content, previousId)
                }
                TYPE_THINKING_SUMMARY -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val messageId = body["messageId"] as? String ?: return
                    val content = body["content"] as? String ?: ""
                    val progress = (body["progress"] as? Number)?.toFloat() ?: 0f
                    getListener(conversationId)?.onThinkingUpdate(conversationId, messageId, content, progress)
                }
                TYPE_TOOL_USE_REQUEST -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val execution = body["execution"] as? String
                    val toolName = body["toolName"] as? String ?: "unknown"
                    @Suppress("UNCHECKED_CAST")
                    val arguments = (body["arguments"] as? Map<String, Any?>) ?: emptyMap()

                    getListener(conversationId)?.onToolUseStarted(conversationId, toolName, arguments)

                    if (execution == "client") {
                        handleToolRequest(envelope)
                    }
                }
                TYPE_TOOL_USE_RESULT -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val success = body["success"] as? Boolean ?: false
                    getListener(conversationId)?.onToolUseCompleted(conversationId, "", success)
                }
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error handling message", e)
        }
    }

    private fun handleToolRequest(envelope: Map<String, Any?>) {
        val body = envelope["body"] as? Map<*, *> ?: return
        val requestId = body["id"] as? String ?: return
        val toolName = body["toolName"] as? String ?: return
        val conversationId = envelope["conversationId"] as? String ?: ""

        @Suppress("UNCHECKED_CAST")
        val arguments = (body["arguments"] as? Map<String, Any>) ?: emptyMap()

        scope.launch {
            val toolSpan = AliciaTelemetry.startSpan("websocket.tool_execution", Attributes.builder()
                .put("tool.name", toolName)
                .build())
            val executor = toolRegistry.get(toolName)

            val (result, success, error) = if (executor != null) {
                try {
                    val value = executor.execute(arguments)
                    Triple(value, true, null)
                } catch (e: Exception) {
                    AliciaTelemetry.recordError(toolSpan, e)
                    Triple(emptyMap<String, Any?>(), false, e.message ?: "Unknown error")
                }
            } else {
                Triple(emptyMap<String, Any?>(), false, "Unknown tool: $toolName")
            }

            toolSpan.setAttribute("tool.status", if (success) "success" else "error")
            toolSpan.end()
            sendToolResult(requestId, conversationId, success, result, error)
        }
    }

    private fun sendToolResult(
        requestId: String,
        conversationId: String,
        success: Boolean,
        result: Map<String, Any?>,
        error: String?
    ) {
        val body = mutableMapOf<String, Any?>(
            "id" to java.util.UUID.randomUUID().toString(),
            "requestId" to requestId,
            "conversationId" to conversationId,
            "success" to success
        )
        if (success) {
            body["result"] = result
        } else {
            body["error"] = error
        }
        AliciaTelemetry.injectTraceContext(body)

        val envelope = buildEnvelope(conversationId, TYPE_TOOL_USE_RESULT, body)
        val sent = sendRaw(envelope)
        if (sent) {
            Log.i(TAG, "Sent tool result for request $requestId (success=$success)")
        } else {
            Log.w(TAG, "Failed to send tool result for request $requestId")
        }
    }

    private fun scheduleReconnect() {
        if (reconnecting) return
        reconnecting = true
        closeCurrentConnection()
        reconnectJob?.cancel()
        reconnectJob = scope.launch {
            delay(reconnectDelay)
            reconnectDelay = (reconnectDelay * 2).coerceAtMost(MAX_RECONNECT_DELAY_MS)
            Log.i(TAG, "Reconnecting...")
            connect()
        }
    }

    private fun buildEnvelope(conversationId: String, type: Int, body: Any?): ByteArray {
        val out = ByteArrayOutputStream()
        val packer = MessagePack.newDefaultPacker(out)

        val map = mutableMapOf<String, Any?>()
        if (conversationId.isNotEmpty()) {
            map["conversationId"] = conversationId
        }
        map["type"] = type
        map["body"] = body

        packValue(packer, map)
        packer.flush()
        return out.toByteArray()
    }

    private fun packValue(packer: org.msgpack.core.MessagePacker, value: Any?) {
        when (value) {
            null -> packer.packNil()
            is Boolean -> packer.packBoolean(value)
            is Int -> packer.packInt(value)
            is Long -> packer.packLong(value)
            is Float -> packer.packFloat(value)
            is Double -> packer.packDouble(value)
            is String -> packer.packString(value)
            is Map<*, *> -> {
                packer.packMapHeader(value.size)
                for ((k, v) in value) {
                    packer.packString(k.toString())
                    packValue(packer, v)
                }
            }
            is List<*> -> {
                packer.packArrayHeader(value.size)
                for (item in value) {
                    packValue(packer, item)
                }
            }
            is ByteArray -> {
                packer.packBinaryHeader(value.size)
                packer.writePayload(value)
            }
            else -> packer.packString(value.toString())
        }
    }

    private fun decodeEnvelope(data: ByteArray): Map<String, Any?>? {
        return try {
            val unpacker = MessagePack.newDefaultUnpacker(data)
            val value = unpacker.unpackValue()
            valueToObject(value) as? Map<String, Any?>
        } catch (e: Exception) {
            Log.e(TAG, "Failed to decode msgpack envelope", e)
            null
        }
    }

    private fun valueToObject(value: Value): Any? {
        return when {
            value.isNilValue -> null
            value.isBooleanValue -> value.asBooleanValue().boolean
            value.isIntegerValue -> {
                val iv = value.asIntegerValue()
                if (iv.isInIntRange) iv.toInt() else iv.toLong()
            }
            value.isFloatValue -> value.asFloatValue().toDouble()
            value.isStringValue -> value.asStringValue().asString()
            value.isBinaryValue -> value.asBinaryValue().asByteArray()
            value.isArrayValue -> {
                value.asArrayValue().list().map { valueToObject(it) }
            }
            value.isMapValue -> {
                val map = mutableMapOf<String, Any?>()
                for ((k, v) in value.asMapValue().map()) {
                    val key = if (k.isStringValue) k.asStringValue().asString() else k.toString()
                    map[key] = valueToObject(v)
                }
                map
            }
            else -> value.toString()
        }
    }
}
