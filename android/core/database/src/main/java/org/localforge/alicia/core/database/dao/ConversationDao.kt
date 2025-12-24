package org.localforge.alicia.core.database.dao

import androidx.room.*
import org.localforge.alicia.core.database.entity.ConversationEntity
import kotlinx.coroutines.flow.Flow

/**
 * Data Access Object for Conversation operations.
 *
 * Architecture Note:
 * - Use Flow return types for reactive queries that need to observe database changes
 *   (e.g., UI lists that should update automatically when data changes).
 * - Use suspend functions for one-time data access operations
 *   (e.g., checking existence, sync operations, mutations).
 */
@Dao
interface ConversationDao {

    /**
     * Get all conversations ordered by most recently updated first.
     * Excludes deleted conversations.
     * @return Flow of conversation list that updates automatically.
     */
    @Query("SELECT * FROM conversations WHERE isDeleted = 0 ORDER BY updatedAt DESC")
    fun getAllConversations(): Flow<List<ConversationEntity>>

    /**
     * Get a specific conversation by ID.
     * @param id Conversation ID
     * @return Flow of conversation or null if not found.
     */
    @Query("SELECT * FROM conversations WHERE id = :id AND isDeleted = 0 LIMIT 1")
    fun getConversationById(id: String): Flow<ConversationEntity?>

    /**
     * Get a specific conversation by ID (suspend version for direct access).
     * @param id Conversation ID
     * @return Conversation or null if not found.
     */
    @Query("SELECT * FROM conversations WHERE id = :id AND isDeleted = 0 LIMIT 1")
    suspend fun getConversationByIdSuspend(id: String): ConversationEntity?

    /**
     * Get all unsynced conversations that need to be uploaded to the server.
     * @return List of conversations where syncedAt is null or older than updatedAt.
     */
    @Query("SELECT * FROM conversations WHERE isDeleted = 0 AND (syncedAt IS NULL OR syncedAt < updatedAt)")
    suspend fun getUnsyncedConversations(): List<ConversationEntity>

    /**
     * Get all deleted conversations that need to be synced.
     * @return List of conversations marked as deleted.
     */
    @Query("SELECT * FROM conversations WHERE isDeleted = 1")
    suspend fun getDeletedConversations(): List<ConversationEntity>

    /**
     * Insert or replace a conversation.
     * @param conversation Conversation to insert/update
     */
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertConversation(conversation: ConversationEntity)

    /**
     * Insert or replace multiple conversations.
     * @param conversations List of conversations to insert/update
     */
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertConversations(conversations: List<ConversationEntity>)

    /**
     * Update an existing conversation.
     * @param conversation Conversation to update
     */
    @Update
    suspend fun updateConversation(conversation: ConversationEntity)

    /**
     * Mark a conversation as deleted (soft delete).
     * @param id Conversation ID to delete
     */
    @Query("UPDATE conversations SET isDeleted = 1, updatedAt = :timestamp WHERE id = :id")
    suspend fun markConversationDeleted(id: String, timestamp: Long = System.currentTimeMillis())

    /**
     * Permanently delete a conversation from the database.
     * @param id Conversation ID to delete
     */
    @Query("DELETE FROM conversations WHERE id = :id")
    suspend fun deleteConversation(id: String)

    /**
     * Delete all conversations (for clearing history).
     */
    @Query("DELETE FROM conversations")
    suspend fun deleteAllConversations()

    /**
     * Mark a conversation as synced with the server.
     * @param id Conversation ID
     * @param timestamp Sync timestamp
     */
    @Query("UPDATE conversations SET syncedAt = :timestamp WHERE id = :id")
    suspend fun markConversationSynced(id: String, timestamp: Long = System.currentTimeMillis())

    /**
     * Update the title of a conversation.
     * @param id Conversation ID
     * @param title New title
     */
    @Query("UPDATE conversations SET title = :title, updatedAt = :timestamp WHERE id = :id")
    suspend fun updateConversationTitle(id: String, title: String, timestamp: Long = System.currentTimeMillis())

    /**
     * Get the count of all conversations.
     * @return Total number of non-deleted conversations
     */
    @Query("SELECT COUNT(*) FROM conversations WHERE isDeleted = 0")
    suspend fun getConversationCount(): Int
}
