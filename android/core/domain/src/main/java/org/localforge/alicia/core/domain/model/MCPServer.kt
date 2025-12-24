package org.localforge.alicia.core.domain.model

import org.localforge.alicia.core.common.Logger

/**
 * Domain model representing MCP transport types.
 */
enum class MCPTransport {
    STDIO,
    SSE;

    companion object {
        private val logger = Logger.forTag("MCPTransport")

        fun fromString(value: String): MCPTransport {
            return when (value.lowercase()) {
                "stdio" -> STDIO
                "sse" -> SSE
                else -> {
                    logger.w("Unknown MCPTransport value: $value, defaulting to STDIO")
                    STDIO
                }
            }
        }
    }

    fun toApiString(): String {
        return when (this) {
            STDIO -> "stdio"
            SSE -> "sse"
        }
    }
}

/**
 * Domain model representing MCP server status.
 */
enum class MCPServerStatus {
    CONNECTED,
    DISCONNECTED,
    ERROR;

    companion object {
        private val logger = Logger.forTag("MCPServerStatus")

        fun fromString(value: String): MCPServerStatus {
            return when (value.lowercase()) {
                "connected" -> CONNECTED
                "disconnected" -> DISCONNECTED
                "error" -> ERROR
                else -> {
                    logger.w("Unknown MCPServerStatus value: $value, defaulting to DISCONNECTED")
                    DISCONNECTED
                }
            }
        }
    }
}

/**
 * Domain model for MCP server configuration.
 */
data class MCPServerConfig(
    val name: String,
    val transport: MCPTransport,
    val command: String,
    val args: List<String> = emptyList(),
    val env: Map<String, String>? = null
)

/**
 * Domain model representing an MCP server with status.
 */
data class MCPServer(
    val name: String,
    val transport: MCPTransport,
    val command: String,
    val args: List<String> = emptyList(),
    val env: Map<String, String>? = null,
    val status: MCPServerStatus = MCPServerStatus.DISCONNECTED,
    val tools: List<String> = emptyList(),
    val error: String? = null
)

/**
 * Domain model representing an MCP tool.
 */
data class MCPTool(
    val name: String,
    val description: String? = null,
    val inputSchema: Map<String, Any>? = null
)
