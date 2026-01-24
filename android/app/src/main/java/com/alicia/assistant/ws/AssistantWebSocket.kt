package com.alicia.assistant.ws

import android.util.Log
import kotlinx.coroutines.*
import okhttp3.*
import okio.ByteString
import okio.ByteString.Companion.toByteString
import org.msgpack.core.MessagePack
import org.msgpack.value.Value
import org.msgpack.value.ValueFactory
import java.io.ByteArrayOutputStream
import java.util.concurrent.TimeUnit

class AssistantWebSocket(
    private val baseUrl: String,
    private val agentSecret: String,
    private val toolRegistry: ToolRegistry
) {
    companion object {
        private const val TAG = "AssistantWebSocket"

        // Message types matching shared/protocol/types.go
        private const val TYPE_TOOL_USE_REQUEST = 6
        private const val TYPE_TOOL_USE_RESULT = 7
        private const val TYPE_SUBSCRIBE = 40
        private const val TYPE_SUBSCRIBE_ACK = 42
        private const val TYPE_ASSISTANT_TOOLS_REGISTER = 70
        private const val TYPE_ASSISTANT_TOOLS_ACK = 71

        private const val RECONNECT_DELAY_MS = 5000L
        private const val MAX_RECONNECT_DELAY_MS = 60000L
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

    private fun closeCurrentConnection() {
        webSocket?.close(1000, "Replacing connection")
        webSocket = null
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
                reconnectDelay = RECONNECT_DELAY_MS
                reconnecting = false
                sendSubscribe(ws)
            }

            override fun onMessage(ws: WebSocket, bytes: ByteString) {
                handleMessage(bytes.toByteArray())
            }

            override fun onFailure(ws: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "WebSocket failure: ${t.message}")
                connected = false
                scheduleReconnect()
            }

            override fun onClosed(ws: WebSocket, code: Int, reason: String) {
                Log.i(TAG, "WebSocket closed: $reason")
                connected = false
                scheduleReconnect()
            }
        })
    }

    fun disconnect() {
        reconnecting = false
        reconnectJob?.cancel()
        webSocket?.close(1000, "Client closing")
        webSocket = null
        scope.cancel()
    }

    private fun sendSubscribe(ws: WebSocket) {
        val body = mapOf("assistantMode" to true)
        val envelope = buildEnvelope("", TYPE_SUBSCRIBE, body)
        if (!ws.send(envelope.toByteString())) {
            Log.w(TAG, "Failed to send subscribe message")
        }
    }

    private fun registerTools(ws: WebSocket) {
        val tools = toolRegistry.getAll().map { executor ->
            mapOf(
                "name" to executor.name,
                "description" to executor.description,
                "inputSchema" to executor.inputSchema
            )
        }
        val body = mapOf("tools" to tools)
        val envelope = buildEnvelope("", TYPE_ASSISTANT_TOOLS_REGISTER, body)
        if (!ws.send(envelope.toByteString())) {
            Log.w(TAG, "Failed to send tools registration")
            return
        }
        Log.i(TAG, "Registered ${tools.size} tools")
    }

    private fun handleMessage(data: ByteArray) {
        try {
            val envelope = decodeEnvelope(data) ?: return
            val type = (envelope["type"] as? Number)?.toInt() ?: return

            when (type) {
                TYPE_SUBSCRIBE_ACK -> {
                    val body = envelope["body"] as? Map<*, *> ?: return
                    val success = body["success"] as? Boolean ?: false
                    if (success) {
                        connected = true
                        Log.i(TAG, "Subscribed as assistant")
                        webSocket?.let { registerTools(it) }
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
                    val execution = body["execution"] as? String
                    if (execution == "client") {
                        handleToolRequest(envelope)
                    }
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
            val executor = toolRegistry.get(toolName)
            val result: Map<String, Any?>
            val success: Boolean
            val error: String?

            if (executor != null) {
                try {
                    val value = executor.execute(arguments)
                    result = value
                    success = true
                    error = null
                } catch (e: Exception) {
                    result = emptyMap()
                    success = false
                    error = e.message ?: "Unknown error"
                }
            } else {
                result = emptyMap()
                success = false
                error = "Unknown tool: $toolName"
            }

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

        val envelope = buildEnvelope(conversationId, TYPE_TOOL_USE_RESULT, body)
        val sent = webSocket?.send(envelope.toByteString()) ?: false
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

    // --- Msgpack Encoding/Decoding ---

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
