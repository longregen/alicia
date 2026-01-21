package org.localforge.alicia.core.data.repository

import org.localforge.alicia.core.data.apiResult
import org.localforge.alicia.core.data.apiResultUnit
import org.localforge.alicia.core.data.mapBody
import org.localforge.alicia.core.data.mapper.toDomain
import org.localforge.alicia.core.data.mapper.toRequest
import org.localforge.alicia.core.domain.model.MCPServer
import org.localforge.alicia.core.domain.model.MCPServerConfig
import org.localforge.alicia.core.domain.model.MCPTool
import org.localforge.alicia.core.domain.repository.MCPRepository
import org.localforge.alicia.core.network.api.AliciaApiService
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Implementation of MCPRepository that fetches MCP servers and tools from the API.
 */
@Singleton
class MCPRepositoryImpl @Inject constructor(
    private val apiService: AliciaApiService
) : MCPRepository {

    override suspend fun getServers(): Result<List<MCPServer>> = apiResult {
        apiService.getMCPServers()
    }.mapBody { it.toDomain() }

    override suspend fun addServer(config: MCPServerConfig): Result<MCPServer> = apiResult {
        apiService.addMCPServer(config.toRequest())
    }.mapBody { it.toDomain() }

    override suspend fun deleteServer(name: String): Result<Unit> = apiResultUnit {
        apiService.deleteMCPServer(name)
    }

    override suspend fun getTools(): Result<List<MCPTool>> = apiResult {
        apiService.getMCPTools()
    }.mapBody { it.toDomain() }
}
