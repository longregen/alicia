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
import java.io.ByteArrayOutputStream
import java.util.UUID
import java.util.concurrent.TimeUnit

/**
 * WebSocket client for MCP (Model Context Protocol) device tool handling.
 *
 * This WebSocket is used exclusively for:
 * - Registering device tools (battery, location, screen, clipboard) with the agent
 * - Receiving tool execution requests from the agent (execution: "client")
 * - Sending tool execution results back to the agent
 * - Maintaining connection health via heartbeat
 *
 * Message sending uses REST API via AliciaApiClient, not this WebSocket.
 */
class AssistantWebSocket(
    private val baseUrl: String,
    private val agentSecret: String,
    private val toolRegistry: ToolRegistry
) {
    companion object {
        private const val TAG = "AssistantWebSocket"

        private const val TYPE_TOOL_USE_REQUEST = 6
        private const val TYPE_TOOL_USE_RESULT = 7
        private const val TYPE_TITLE_UPDATE = 35
        private const val TYPE_SUBSCRIBE = 40
        private const val TYPE_SUBSCRIBE_ACK = 42
        private const val TYPE_ASSISTANT_TOOLS_REGISTER = 70
        private const val TYPE_ASSISTANT_TOOLS_ACK = 71
        private const val TYPE_ASSISTANT_HEARTBEAT = 72
        private const val HEARTBEAT_INTERVAL_MS = 15_000L

        private const val RECONNECT_DELAY_MS = 5000L
        private const val MAX_RECONNECT_DELAY_MS = 60000L
    }

    /**
     * Listener for title updates received via WebSocket.
     */
    interface TitleUpdateListener {
        fun onTitleUpdate(conversationId: String, title: String)
    }

    private var titleUpdateListener: TitleUpdateListener? = null

    fun setTitleUpdateListener(listener: TitleUpdateListener?) {
        titleUpdateListener = listener
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

    // Reflects transport connection and subscription acknowledgment, not tool registration readiness
    fun isConnected(): Boolean = connected

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
                TYPE_TOOL_USE_REQUEST -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    if (body["execution"] == "client") {
                        val toolName = body["toolName"] as? String ?: "unknown"
                        Log.i(TAG, "Received tool request: $toolName")
                        handleToolRequest(envelope)
                    }
                }
                TYPE_TOOL_USE_RESULT -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val success = body["success"] as? Boolean ?: false
                    Log.d(TAG, "Tool result acknowledged: success=$success")
                }
                TYPE_TITLE_UPDATE -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val title = body["title"] as? String ?: return
                    val convId = body["conversationId"] as? String ?: conversationId
                    Log.i(TAG, "Received title update for $convId: $title")
                    titleUpdateListener?.onTitleUpdate(convId, title)
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

        val arguments = (body["arguments"] as? Map<String, Any>) ?: emptyMap()

        scope.launch {
            val toolSpan = AliciaTelemetry.startSpan("websocket.tool_execution", Attributes.builder()
                .put("tool.name", toolName)
                .build())
            val executor = toolRegistry.get(toolName)
            val (result, success, error) = if (executor != null) {
                try {
                    Triple(executor.execute(arguments), true, null)
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
            "id" to UUID.randomUUID().toString(),
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
