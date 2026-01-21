package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.MCPServer
import org.localforge.alicia.core.domain.model.MCPServerConfig
import org.localforge.alicia.core.domain.model.MCPTool

interface MCPRepository {
    suspend fun getServers(): Result<List<MCPServer>>

    suspend fun addServer(config: MCPServerConfig): Result<MCPServer>

    suspend fun deleteServer(name: String): Result<Unit>

    suspend fun getTools(): Result<List<MCPTool>>
}
