package org.localforge.alicia.core.domain.repository

import kotlinx.coroutines.flow.Flow
import org.localforge.alicia.core.domain.model.ServerInfo

/**
 * Repository interface for server information operations.
 */
interface ServerRepository {
    /**
     * Get server info as a flow for real-time updates.
     */
    fun getServerInfo(): Flow<ServerInfo>

    /**
     * Fetch the latest server info from the API.
     */
    suspend fun fetchServerInfo(): ServerInfo

    /**
     * Refresh server info.
     */
    suspend fun refreshServerInfo()
}
