package org.localforge.alicia.core.database.dao

import androidx.room.*
import org.localforge.alicia.core.database.entity.SyncQueueEntity
import kotlinx.coroutines.flow.Flow

@Dao
interface SyncQueueDao {

    @Query("SELECT * FROM sync_queue ORDER BY createdAt ASC")
    suspend fun getAll(): List<SyncQueueEntity>

    @Query("SELECT * FROM sync_queue ORDER BY createdAt ASC")
    fun getAllFlow(): Flow<List<SyncQueueEntity>>

    @Query("SELECT * FROM sync_queue WHERE conversationId = :conversationId ORDER BY createdAt ASC")
    suspend fun getForConversation(conversationId: String): List<SyncQueueEntity>

    @Query("SELECT * FROM sync_queue WHERE localId = :localId LIMIT 1")
    suspend fun getById(localId: String): SyncQueueEntity?

    @Query("SELECT COUNT(*) FROM sync_queue")
    suspend fun getCount(): Int

    @Query("SELECT COUNT(*) FROM sync_queue")
    fun getCountFlow(): Flow<Int>

    @Query("SELECT * FROM sync_queue WHERE retryCount < :maxRetries ORDER BY createdAt ASC")
    suspend fun getRetryable(maxRetries: Int = 3): List<SyncQueueEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(item: SyncQueueEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(items: List<SyncQueueEntity>)

    @Update
    suspend fun update(item: SyncQueueEntity)

    @Query("UPDATE sync_queue SET retryCount = retryCount + 1 WHERE localId = :localId")
    suspend fun incrementRetryCount(localId: String)

    @Query("DELETE FROM sync_queue WHERE localId = :localId")
    suspend fun delete(localId: String)

    @Query("DELETE FROM sync_queue WHERE localId IN (:localIds)")
    suspend fun deleteAll(localIds: List<String>)

    @Query("DELETE FROM sync_queue WHERE conversationId = :conversationId")
    suspend fun deleteForConversation(conversationId: String)

    @Query("DELETE FROM sync_queue")
    suspend fun clear()

    @Query("DELETE FROM sync_queue WHERE retryCount >= :maxRetries")
    suspend fun deleteFailedOperations(maxRetries: Int = 5)
}
