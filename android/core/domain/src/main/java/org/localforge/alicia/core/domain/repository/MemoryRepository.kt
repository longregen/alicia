package org.localforge.alicia.core.domain.repository

import kotlinx.coroutines.flow.Flow
import org.localforge.alicia.core.domain.model.Memory
import org.localforge.alicia.core.domain.model.MemoryCategory

/**
 * Repository interface for memory management operations.
 */
interface MemoryRepository {
    /**
     * Get all memories (non-archived)
     */
    fun getAllMemories(): Flow<List<Memory>>

    /**
     * Get archived memories
     */
    fun getArchivedMemories(): Flow<List<Memory>>

    /**
     * Get a specific memory by ID
     */
    suspend fun getMemory(id: String): Memory?

    /**
     * Search memories by content
     */
    fun searchMemories(query: String): Flow<List<Memory>>

    /**
     * Get memories by category
     */
    fun getMemoriesByCategory(category: MemoryCategory): Flow<List<Memory>>

    /**
     * Get pinned memories
     */
    fun getPinnedMemories(): Flow<List<Memory>>

    /**
     * Create a new memory
     */
    suspend fun createMemory(content: String, category: MemoryCategory): Memory

    /**
     * Update an existing memory
     */
    suspend fun updateMemory(id: String, content: String, category: MemoryCategory)

    /**
     * Pin or unpin a memory
     */
    suspend fun pinMemory(id: String, pinned: Boolean)

    /**
     * Archive a memory
     */
    suspend fun archiveMemory(id: String)

    /**
     * Unarchive a memory
     */
    suspend fun unarchiveMemory(id: String)

    /**
     * Delete a memory
     */
    suspend fun deleteMemory(id: String)

    // ========== Tag Operations (matching web frontend) ==========

    /**
     * Add tags to a memory
     */
    suspend fun addTags(memoryId: String, tags: List<String>): Result<Memory>

    /**
     * Remove a tag from a memory
     */
    suspend fun removeTag(memoryId: String, tag: String): Result<Memory>

    /**
     * Get memories by tags
     */
    suspend fun getMemoriesByTags(tags: List<String>): Result<List<Memory>>

    // ========== Importance Operations (matching web frontend) ==========

    /**
     * Set memory importance (0.0 - 1.0)
     */
    suspend fun setImportance(memoryId: String, importance: Double): Result<Memory>

    // ========== Server-side Search (matching web frontend) ==========

    /**
     * Search memories on server with semantic search
     */
    suspend fun searchMemoriesOnServer(query: String, limit: Int = 10): Result<List<Memory>>
}
