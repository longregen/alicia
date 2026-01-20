package org.localforge.alicia.core.network.protocol.bodies

/**
 * Connection status
 */
enum class ProtocolConnectionStatus(val value: String) {
    CONNECTED("connected"),
    CONNECTING("connecting"),
    DISCONNECTED("disconnected"),
    RECONNECTING("reconnecting");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * Connection info
 */
data class ConnectionInfoBody(
    val status: ProtocolConnectionStatus,
    val latency: Long
)

/**
 * Model info
 */
data class ModelInfoBody(
    val name: String,
    val provider: String
)

/**
 * MCP server status
 */
enum class ProtocolMCPServerStatus(val value: String) {
    CONNECTED("connected"),
    DISCONNECTED("disconnected"),
    ERROR("error");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * MCP server info
 */
data class MCPServerInfoBody(
    val name: String,
    val status: ProtocolMCPServerStatus
)

/**
 * Server info body (Type 26)
 */
data class ServerInfoBody(
    val connection: ConnectionInfoBody,
    val model: ModelInfoBody,
    val mcpServers: List<MCPServerInfoBody>
)

/**
 * Session stats body (Type 27)
 */
data class SessionStatsBody(
    val messageCount: Int,
    val toolCallCount: Int,
    val memoriesUsed: Int,
    val sessionDuration: Long
)

/**
 * Conversation update body (Type 28)
 */
data class ConversationUpdateBody(
    val conversationId: String,
    val title: String? = null,
    val status: String? = null,
    val updatedAt: String
)
