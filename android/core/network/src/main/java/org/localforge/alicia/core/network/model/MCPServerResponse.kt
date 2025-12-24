package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Network model for MCP server configuration request.
 */
@JsonClass(generateAdapter = true)
data class MCPServerConfigRequest(
    @Json(name = "name")
    val name: String,

    @Json(name = "transport")
    val transport: String,

    @Json(name = "command")
    val command: String,

    @Json(name = "args")
    val args: List<String> = emptyList(),

    @Json(name = "env")
    val env: Map<String, String>? = null
)

/**
 * Network model for MCP server response.
 */
@JsonClass(generateAdapter = true)
data class MCPServerResponse(
    @Json(name = "name")
    val name: String,

    @Json(name = "transport")
    val transport: String,

    @Json(name = "command")
    val command: String,

    @Json(name = "args")
    val args: List<String> = emptyList(),

    @Json(name = "env")
    val env: Map<String, String>? = null,

    @Json(name = "status")
    val status: String,

    @Json(name = "tools")
    val tools: List<String> = emptyList(),

    @Json(name = "error")
    val error: String? = null
)

/**
 * Network model for MCP servers list response.
 */
@JsonClass(generateAdapter = true)
data class MCPServersResponse(
    @Json(name = "servers")
    val servers: List<MCPServerResponse>
)

/**
 * Network model for MCP tool response.
 */
@JsonClass(generateAdapter = true)
data class MCPToolResponse(
    @Json(name = "name")
    val name: String,

    @Json(name = "description")
    val description: String? = null,

    @Json(name = "inputSchema")
    val inputSchema: Map<String, Any>? = null
)

/**
 * Network model for MCP tools list response.
 */
@JsonClass(generateAdapter = true)
data class MCPToolsResponse(
    @Json(name = "tools")
    val tools: List<MCPToolResponse>
)
