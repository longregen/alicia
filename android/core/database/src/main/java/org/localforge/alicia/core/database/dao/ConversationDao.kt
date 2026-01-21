package org.localforge.alicia.core.database.dao

import androidx.room.*
import org.localforge.alicia.core.database.entity.ConversationEntity
import kotlinx.coroutines.flow.Flow

@Dao
interface ConversationDao {

    @Query("SELECT * FROM conversations WHERE isDeleted = 0 ORDER BY updatedAt DESC")
    fun getAllConversations(): Flow<List<ConversationEntity>>

    @Query("SELECT * FROM conversations WHERE id = :id AND isDeleted = 0 LIMIT 1")
    fun getConversationById(id: String): Flow<ConversationEntity?>

    @Query("SELECT * FROM conversations WHERE id = :id AND isDeleted = 0 LIMIT 1")
    suspend fun getConversationByIdSuspend(id: String): ConversationEntity?

    @Query("SELECT * FROM conversations WHERE isDeleted = 0 AND (syncedAt IS NULL OR syncedAt < updatedAt)")
    suspend fun getUnsyncedConversations(): List<ConversationEntity>

    @Query("SELECT * FROM conversations WHERE isDeleted = 1")
    suspend fun getDeletedConversations(): List<ConversationEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertConversation(conversation: ConversationEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertConversations(conversations: List<ConversationEntity>)

    @Update
    suspend fun updateConversation(conversation: ConversationEntity)

    @Query("UPDATE conversations SET isDeleted = 1, updatedAt = :timestamp WHERE id = :id")
    suspend fun markConversationDeleted(id: String, timestamp: Long = System.currentTimeMillis())

    @Query("DELETE FROM conversations WHERE id = :id")
    suspend fun deleteConversation(id: String)

    @Query("DELETE FROM conversations")
    suspend fun deleteAllConversations()

    @Query("UPDATE conversations SET syncedAt = :timestamp WHERE id = :id")
    suspend fun markConversationSynced(id: String, timestamp: Long = System.currentTimeMillis())

    @Query("UPDATE conversations SET title = :title, updatedAt = :timestamp WHERE id = :id")
    suspend fun updateConversationTitle(id: String, title: String, timestamp: Long = System.currentTimeMillis())

    @Query("SELECT COUNT(*) FROM conversations WHERE isDeleted = 0")
    suspend fun getConversationCount(): Int
}
