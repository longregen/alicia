package org.localforge.alicia.core.domain.repository

import kotlinx.coroutines.flow.Flow
import org.localforge.alicia.core.domain.model.Memory
import org.localforge.alicia.core.domain.model.MemoryCategory

interface MemoryRepository {
    fun getAllMemories(): Flow<List<Memory>>

    fun getArchivedMemories(): Flow<List<Memory>>

    suspend fun getMemory(id: String): Memory?

    fun searchMemories(query: String): Flow<List<Memory>>

    fun getMemoriesByCategory(category: MemoryCategory): Flow<List<Memory>>

    fun getPinnedMemories(): Flow<List<Memory>>

    suspend fun createMemory(content: String, category: MemoryCategory): Memory

    suspend fun updateMemory(id: String, content: String, category: MemoryCategory)

    suspend fun pinMemory(id: String, pinned: Boolean)

    suspend fun archiveMemory(id: String)

    suspend fun unarchiveMemory(id: String)

    suspend fun deleteMemory(id: String)

    suspend fun addTags(memoryId: String, tags: List<String>): Result<Memory>

    suspend fun removeTag(memoryId: String, tag: String): Result<Memory>

    suspend fun getMemoriesByTags(tags: List<String>): Result<List<Memory>>

    suspend fun setImportance(memoryId: String, importance: Double): Result<Memory>

    suspend fun searchMemoriesOnServer(query: String, limit: Int = 10): Result<List<Memory>>
}
