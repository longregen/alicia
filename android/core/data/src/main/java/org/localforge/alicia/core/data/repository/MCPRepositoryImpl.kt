package org.localforge.alicia.core.data.repository

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

    override suspend fun getServers(): Result<List<MCPServer>> {
        return try {
            val response = apiService.getMCPServers()

            if (response.isSuccessful && response.body() != null) {
                val servers = response.body()!!.toDomain()
                Result.success(servers)
            } else {
                Result.failure(
                    Exception("Failed to fetch MCP servers: ${response.code()} ${response.message()}")
                )
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun addServer(config: MCPServerConfig): Result<MCPServer> {
        return try {
            val request = config.toRequest()
            val response = apiService.addMCPServer(request)

            if (response.isSuccessful && response.body() != null) {
                val server = response.body()!!.toDomain()
                Result.success(server)
            } else {
                Result.failure(
                    Exception("Failed to add MCP server: ${response.code()} ${response.message()}")
                )
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun deleteServer(name: String): Result<Unit> {
        return try {
            val response = apiService.deleteMCPServer(name)

            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(
                    Exception("Failed to delete MCP server: ${response.code()} ${response.message()}")
                )
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getTools(): Result<List<MCPTool>> {
        return try {
            val response = apiService.getMCPTools()

            if (response.isSuccessful && response.body() != null) {
                val tools = response.body()!!.toDomain()
                Result.success(tools)
            } else {
                Result.failure(
                    Exception("Failed to fetch MCP tools: ${response.code()} ${response.message()}")
                )
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
