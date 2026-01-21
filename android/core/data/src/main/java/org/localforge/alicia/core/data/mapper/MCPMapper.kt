package org.localforge.alicia.core.data.mapper

import org.localforge.alicia.core.domain.model.*
import org.localforge.alicia.core.network.model.*

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

fun List<MCPServerResponse>.toDomain(): List<MCPServer> {
    return map { it.toDomain() }
}

fun MCPServersResponse.toDomain(): List<MCPServer> {
    return servers.toDomain()
}

fun MCPToolResponse.toDomain(): MCPTool {
    return MCPTool(
        name = name,
        description = description,
        inputSchema = inputSchema
    )
}

fun List<MCPToolResponse>.toToolsDomain(): List<MCPTool> {
    return map { it.toDomain() }
}

fun MCPToolsResponse.toDomain(): List<MCPTool> {
    return tools.toToolsDomain()
}

fun MCPServerConfig.toRequest(): MCPServerConfigRequest {
    return MCPServerConfigRequest(
        name = name,
        transport = transport.toApiString(),
        command = command,
        args = args,
        env = env
    )
}
