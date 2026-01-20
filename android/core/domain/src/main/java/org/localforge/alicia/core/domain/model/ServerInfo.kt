package org.localforge.alicia.core.domain.model

/**
 * Connection status for the server.
 */
enum class ConnectionStatus {
    CONNECTED,
    CONNECTING,
    DISCONNECTED,
    RECONNECTING;

    companion object {
        fun fromString(value: String): ConnectionStatus {
            return when (value.lowercase()) {
                "connected" -> CONNECTED
                "connecting" -> CONNECTING
                "disconnected" -> DISCONNECTED
                "reconnecting" -> RECONNECTING
                else -> DISCONNECTED
            }
        }
    }

    val displayName: String
        get() = when (this) {
            CONNECTED -> "Connected"
            CONNECTING -> "Connecting"
            DISCONNECTED -> "Disconnected"
            RECONNECTING -> "Reconnecting"
        }
}

/**
 * Connection quality based on latency.
 */
enum class ConnectionQuality {
    EXCELLENT,
    GOOD,
    FAIR,
    POOR;

    companion object {
        fun fromLatency(latency: Long, isConnected: Boolean): ConnectionQuality {
            if (!isConnected) return POOR
            return when {
                latency < 50 -> EXCELLENT
                latency < 100 -> GOOD
                latency < 200 -> FAIR
                else -> POOR
            }
        }
    }

    val displayName: String
        get() = name.lowercase().replaceFirstChar { it.uppercase() }
}

/**
 * Model information.
 */
data class ModelInfo(
    val name: String,
    val provider: String
)

/**
 * Session statistics.
 */
data class SessionStats(
    val messageCount: Int = 0,
    val toolCallCount: Int = 0,
    val memoriesUsed: Int = 0,
    val sessionDuration: Long = 0 // in seconds
) {
    val formattedDuration: String
        get() {
            if (sessionDuration < 60) {
                return "${sessionDuration}s"
            }
            val minutes = sessionDuration / 60
            val seconds = sessionDuration % 60
            if (minutes < 60) {
                return "${minutes}m ${seconds}s"
            }
            val hours = minutes / 60
            val remainingMinutes = minutes % 60
            return "${hours}h ${remainingMinutes}m"
        }
}

/**
 * Complete server info state.
 */
data class ServerInfo(
    val connectionStatus: ConnectionStatus = ConnectionStatus.DISCONNECTED,
    val latency: Long = 0,
    val modelInfo: ModelInfo? = null,
    val mcpServers: List<MCPServer> = emptyList(),
    val sessionStats: SessionStats = SessionStats()
) {
    val isConnected: Boolean
        get() = connectionStatus == ConnectionStatus.CONNECTED

    val isConnecting: Boolean
        get() = connectionStatus == ConnectionStatus.CONNECTING ||
                connectionStatus == ConnectionStatus.RECONNECTING

    val connectionQuality: ConnectionQuality
        get() = ConnectionQuality.fromLatency(latency, isConnected)

    val connectedMcpServers: List<MCPServer>
        get() = mcpServers.filter { it.status == MCPServerStatus.CONNECTED }

    val disconnectedMcpServers: List<MCPServer>
        get() = mcpServers.filter { it.status != MCPServerStatus.CONNECTED }

    val mcpServerSummary: String
        get() = "${connectedMcpServers.size}/${mcpServers.size}"
}
