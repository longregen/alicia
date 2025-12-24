package org.localforge.alicia.core.data.mapper

import org.localforge.alicia.core.domain.model.*
import org.localforge.alicia.core.network.model.*

/**
 * Mapper for converting between MCP domain models and network responses.
 */

/**
 * Convert MCPServerResponse to MCPServer domain model.
 */
fun MCPServerResponse.toDomain(): MCPServer {
    return MCPServer(
        name = name,
        transport = MCPTransport.fromString(transport),
        command = command,
        args = args,
        env = env,
        status = MCPServerStatus.fromString(status),
        tools = tools,
        error = error
    )
}

/**
 * Convert list of MCPServerResponses to list of MCPServer domain models.
 */
fun List<MCPServerResponse>.toDomain(): List<MCPServer> {
    return map { it.toDomain() }
}

/**
 * Convert MCPServersResponse to list of MCPServer domain models.
 */
fun MCPServersResponse.toDomain(): List<MCPServer> {
    return servers.toDomain()
}

/**
 * Convert MCPToolResponse to MCPTool domain model.
 */
fun MCPToolResponse.toDomain(): MCPTool {
    return MCPTool(
        name = name,
        description = description,
        inputSchema = inputSchema
    )
}

/**
 * Convert list of MCPToolResponses to list of MCPTool domain models.
 */
fun List<MCPToolResponse>.toToolsDomain(): List<MCPTool> {
    return map { it.toDomain() }
}

/**
 * Convert MCPToolsResponse to list of MCPTool domain models.
 */
fun MCPToolsResponse.toDomain(): List<MCPTool> {
    return tools.toToolsDomain()
}

/**
 * Convert MCPServerConfig domain model to MCPServerConfigRequest network model.
 */
fun MCPServerConfig.toRequest(): MCPServerConfigRequest {
    return MCPServerConfigRequest(
        name = name,
        transport = transport.toApiString(),
        command = command,
        args = args,
        env = env
    )
}
