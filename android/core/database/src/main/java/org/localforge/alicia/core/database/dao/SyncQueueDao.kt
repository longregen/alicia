package org.localforge.alicia.core.database.dao

import androidx.room.*
import org.localforge.alicia.core.database.entity.SyncQueueEntity
import kotlinx.coroutines.flow.Flow

/**
 * Data Access Object for sync queue operations.
 */
@Dao
interface SyncQueueDao {

    /**
     * Get all pending sync operations ordered by creation time.
     * @return List of sync operations to be processed.
     */
    @Query("SELECT * FROM sync_queue ORDER BY createdAt ASC")
    suspend fun getAll(): List<SyncQueueEntity>

    /**
     * Get all pending sync operations as a Flow for reactive updates.
     * @return Flow of sync operations that updates automatically.
     */
    @Query("SELECT * FROM sync_queue ORDER BY createdAt ASC")
    fun getAllFlow(): Flow<List<SyncQueueEntity>>

    /**
     * Get pending operations for a specific conversation.
     * @param conversationId Conversation ID
     * @return List of pending operations for this conversation.
     */
    @Query("SELECT * FROM sync_queue WHERE conversationId = :conversationId ORDER BY createdAt ASC")
    suspend fun getForConversation(conversationId: String): List<SyncQueueEntity>

    /**
     * Get a specific sync operation by local ID.
     * @param localId Local operation identifier
     * @return Sync operation or null if not found.
     */
    @Query("SELECT * FROM sync_queue WHERE localId = :localId LIMIT 1")
    suspend fun getById(localId: String): SyncQueueEntity?

    /**
     * Get the count of pending sync operations.
     * @return Number of operations in the queue.
     */
    @Query("SELECT COUNT(*) FROM sync_queue")
    suspend fun getCount(): Int

    /**
     * Get the count of pending sync operations as a Flow.
     * @return Flow of pending operation count.
     */
    @Query("SELECT COUNT(*) FROM sync_queue")
    fun getCountFlow(): Flow<Int>

    /**
     * Get operations that have failed and need retry.
     * @param maxRetries Maximum retry count to consider
     * @return List of operations that failed but can be retried.
     */
    @Query("SELECT * FROM sync_queue WHERE retryCount < :maxRetries ORDER BY createdAt ASC")
    suspend fun getRetryable(maxRetries: Int = 3): List<SyncQueueEntity>

    /**
     * Insert a new sync operation into the queue.
     * @param item Sync operation to insert
     */
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(item: SyncQueueEntity)

    /**
     * Insert multiple sync operations into the queue.
     * @param items List of sync operations to insert
     */
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(items: List<SyncQueueEntity>)

    /**
     * Update an existing sync operation (e.g., to increment retry count).
     * @param item Sync operation to update
     */
    @Update
    suspend fun update(item: SyncQueueEntity)

    /**
     * Increment the retry count for a specific operation.
     * @param localId Local operation identifier
     */
    @Query("UPDATE sync_queue SET retryCount = retryCount + 1 WHERE localId = :localId")
    suspend fun incrementRetryCount(localId: String)

    /**
     * Delete a specific sync operation (after successful sync).
     * @param localId Local operation identifier
     */
    @Query("DELETE FROM sync_queue WHERE localId = :localId")
    suspend fun delete(localId: String)

    /**
     * Delete multiple sync operations.
     * @param localIds List of local operation identifiers
     */
    @Query("DELETE FROM sync_queue WHERE localId IN (:localIds)")
    suspend fun deleteAll(localIds: List<String>)

    /**
     * Delete all sync operations for a conversation.
     * @param conversationId Conversation ID
     */
    @Query("DELETE FROM sync_queue WHERE conversationId = :conversationId")
    suspend fun deleteForConversation(conversationId: String)

    /**
     * Clear all sync operations (for testing or reset).
     */
    @Query("DELETE FROM sync_queue")
    suspend fun clear()

    /**
     * Delete operations that have exceeded the maximum retry count.
     * @param maxRetries Maximum retry count before deletion
     */
    @Query("DELETE FROM sync_queue WHERE retryCount >= :maxRetries")
    suspend fun deleteFailedOperations(maxRetries: Int = 5)
}
