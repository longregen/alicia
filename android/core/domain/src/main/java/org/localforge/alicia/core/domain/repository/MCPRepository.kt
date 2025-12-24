package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.MCPServer
import org.localforge.alicia.core.domain.model.MCPServerConfig
import org.localforge.alicia.core.domain.model.MCPTool

/**
 * Repository interface for managing MCP servers and tools.
 */
interface MCPRepository {
    /**
     * Get all configured MCP servers.
     * @return Result containing list of MCP servers.
     */
    suspend fun getServers(): Result<List<MCPServer>>

    /**
     * Add a new MCP server.
     * @param config Server configuration to add.
     * @return Result containing the newly created server.
     */
    suspend fun addServer(config: MCPServerConfig): Result<MCPServer>

    /**
     * Delete an MCP server by name.
     * @param name Name of the server to delete.
     * @return Result indicating success or failure.
     */
    suspend fun deleteServer(name: String): Result<Unit>

    /**
     * Get all available MCP tools from all configured servers.
     * @return Result containing list of MCP tools.
     */
    suspend fun getTools(): Result<List<MCPTool>>
}
