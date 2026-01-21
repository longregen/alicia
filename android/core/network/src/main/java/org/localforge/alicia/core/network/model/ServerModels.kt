package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class ServerInfoResponse(
    @Json(name = "connection")
    val connection: ConnectionInfo,

    @Json(name = "model")
    val model: ModelInfo,

    @Json(name = "mcpServers")
    val mcpServers: List<MCPServerInfo> = emptyList()
)

@JsonClass(generateAdapter = true)
data class ConnectionInfo(
    @Json(name = "status")
    val status: String,

    @Json(name = "latency")
    val latency: Long
)

@JsonClass(generateAdapter = true)
data class ModelInfo(
    @Json(name = "name")
    val name: String,

    @Json(name = "provider")
    val provider: String
)

@JsonClass(generateAdapter = true)
data class MCPServerInfo(
    @Json(name = "name")
    val name: String,

    @Json(name = "status")
    val status: String
)

/**
 * Response model for session stats
 */
@JsonClass(generateAdapter = true)
data class SessionStatsResponse(
    @Json(name = "messageCount")
    val messageCount: Int,

    @Json(name = "toolCallCount")
    val toolCallCount: Int,

    @Json(name = "memoriesUsed")
    val memoriesUsed: Int,

    @Json(name = "sessionDuration")
    val sessionDuration: Long,

    @Json(name = "conversationId")
    val conversationId: String? = null
)

@JsonClass(generateAdapter = true)
data class PublicConfigResponse(
    @Json(name = "livekit_url")
    val livekitUrl: String? = null,

    @Json(name = "tts_enabled")
    val ttsEnabled: Boolean = false,

    @Json(name = "asr_enabled")
    val asrEnabled: Boolean = false,

    @Json(name = "tts")
    val tts: TTSConfig? = null
)

@JsonClass(generateAdapter = true)
data class TTSConfig(
    @Json(name = "endpoint")
    val endpoint: String,

    @Json(name = "model")
    val model: String,

    @Json(name = "default_voice")
    val defaultVoice: String,

    @Json(name = "default_speed")
    val defaultSpeed: Double,

    @Json(name = "speed_min")
    val speedMin: Double,

    @Json(name = "speed_max")
    val speedMax: Double,

    @Json(name = "speed_step")
    val speedStep: Double,

    @Json(name = "voices")
    val voices: List<TTSVoice> = emptyList()
)

@JsonClass(generateAdapter = true)
data class TTSVoice(
    @Json(name = "id")
    val id: String,

    @Json(name = "name")
    val name: String,

    @Json(name = "category")
    val category: String
)

typealias AddTagsRequest = AddMemoryTagsRequest
