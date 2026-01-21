package org.localforge.alicia.core.domain.repository

import kotlinx.coroutines.flow.Flow
import org.localforge.alicia.core.domain.model.ServerInfo

interface ServerRepository {
    fun getServerInfo(): Flow<ServerInfo>

    suspend fun fetchServerInfo(): ServerInfo

    suspend fun refreshServerInfo()
}
