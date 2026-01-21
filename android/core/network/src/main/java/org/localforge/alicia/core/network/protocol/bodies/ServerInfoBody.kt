package org.localforge.alicia.core.network.protocol.bodies

enum class ProtocolConnectionStatus(val value: String) {
    CONNECTED("connected"),
    CONNECTING("connecting"),
    DISCONNECTED("disconnected"),
    RECONNECTING("reconnecting");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

data class ConnectionInfoBody(
    val status: ProtocolConnectionStatus,
    val latency: Long
)

data class ModelInfoBody(
    val name: String,
    val provider: String
)

enum class ProtocolMCPServerStatus(val value: String) {
    CONNECTED("connected"),
    DISCONNECTED("disconnected"),
    ERROR("error");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

data class MCPServerInfoBody(
    val name: String,
    val status: ProtocolMCPServerStatus
)

data class ServerInfoBody(
    val connection: ConnectionInfoBody,
    val model: ModelInfoBody,
    val mcpServers: List<MCPServerInfoBody>
)

data class SessionStatsBody(
    val messageCount: Int,
    val toolCallCount: Int,
    val memoriesUsed: Int,
    val sessionDuration: Long
)

data class ConversationUpdateBody(
    val conversationId: String,
    val title: String? = null,
    val status: String? = null,
    val updatedAt: String
)
