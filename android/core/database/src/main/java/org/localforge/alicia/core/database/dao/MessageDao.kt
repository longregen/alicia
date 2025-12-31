package org.localforge.alicia.core.database.dao

import androidx.room.*
import org.localforge.alicia.core.database.entity.MessageEntity
import kotlinx.coroutines.flow.Flow

/**
 * Data Access Object for Message operations.
 */
@Dao
interface MessageDao {

    /**
     * Get all messages for a specific conversation ordered by creation time.
     * @param conversationId Conversation ID
     * @return Flow of message list that updates automatically.
     */
    @Query("SELECT * FROM messages WHERE conversationId = :conversationId ORDER BY createdAt ASC")
    fun getMessagesForConversation(conversationId: String): Flow<List<MessageEntity>>

    /**
     * Get a specific message by ID.
     * @param id Message ID
     * @return Message or null if not found.
     */
    @Query("SELECT * FROM messages WHERE id = :id LIMIT 1")
    suspend fun getMessageById(id: String): MessageEntity?

    /**
     * Get all pending messages that need to be synced to the server.
     * @return List of messages with pending sync status.
     */
    @Query("SELECT * FROM messages WHERE syncStatus = 'pending' ORDER BY createdAt ASC")
    suspend fun getPendingMessages(): List<MessageEntity>

    /**
     * Get pending messages for a specific conversation.
     * @param conversationId Conversation ID
     * @return List of pending messages in this conversation.
     */
    @Query("SELECT * FROM messages WHERE conversationId = :conversationId AND syncStatus = 'pending' ORDER BY createdAt ASC")
    suspend fun getPendingMessagesForConversation(conversationId: String): List<MessageEntity>

    /**
     * Get a message by its local ID.
     * @param localId Local message identifier
     * @return Message or null if not found.
     */
    @Query("SELECT * FROM messages WHERE localId = :localId LIMIT 1")
    suspend fun getMessageByLocalId(localId: String): MessageEntity?

    /**
     * Get messages by sync status.
     * @param syncStatus Sync status to filter by
     * @return List of messages with the specified sync status.
     */
    @Query("SELECT * FROM messages WHERE syncStatus = :syncStatus ORDER BY createdAt ASC")
    suspend fun getMessagesBySyncStatus(syncStatus: String): List<MessageEntity>

    /**
     * Get messages by sync status for a specific conversation.
     * @param conversationId Conversation ID
     * @param syncStatus Sync status to filter by
     * @return List of messages with the specified sync status.
     */
    @Query("SELECT * FROM messages WHERE conversationId = :conversationId AND syncStatus = :syncStatus ORDER BY createdAt ASC")
    suspend fun getMessagesBySyncStatusForConversation(conversationId: String, syncStatus: String): List<MessageEntity>

    /**
     * Update the sync status and related fields for a message.
     * @param localId Local message ID
     * @param serverId Server-assigned ID
     * @param syncStatus New sync status
     * @param syncedAt Timestamp of sync
     */
    @Query("UPDATE messages SET serverId = :serverId, syncStatus = :syncStatus, syncedAt = :syncedAt WHERE localId = :localId")
    suspend fun updateSyncStatus(localId: String, serverId: String?, syncStatus: String, syncedAt: Long?)

    /**
     * Search messages by content.
     * @param query Search query (query will be wrapped with wildcards for partial matching)
     * @return Flow of matching messages ordered by most recent.
     */
    @Query("SELECT * FROM messages WHERE content LIKE '%' || :query || '%' ORDER BY createdAt DESC")
    fun searchMessages(query: String): Flow<List<MessageEntity>>

    /**
     * Search messages by content within a specific conversation.
     * @param conversationId Conversation ID
     * @param query Search query (query will be wrapped with wildcards for partial matching)
     * @return Flow of matching messages in this conversation.
     */
    @Query("SELECT * FROM messages WHERE conversationId = :conversationId AND content LIKE '%' || :query || '%' ORDER BY createdAt DESC")
    fun searchMessagesInConversation(conversationId: String, query: String): Flow<List<MessageEntity>>

    /**
     * Get the last N messages from a conversation.
     * @param conversationId Conversation ID
     * @param limit Number of messages to retrieve
     * @return List of most recent messages.
     */
    @Query("SELECT * FROM messages WHERE conversationId = :conversationId ORDER BY createdAt DESC LIMIT :limit")
    suspend fun getLastMessages(conversationId: String, limit: Int): List<MessageEntity>

    /**
     * Insert a new message.
     * @param message Message to insert
     */
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertMessage(message: MessageEntity)

    /**
     * Insert multiple messages.
     * @param messages List of messages to insert
     */
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertMessages(messages: List<MessageEntity>)

    /**
     * Update an existing message.
     * @param message Message to update
     */
    @Update
    suspend fun updateMessage(message: MessageEntity)

    /**
     * Delete a specific message.
     * @param messageId Message ID to delete
     */
    @Query("DELETE FROM messages WHERE id = :messageId")
    suspend fun deleteMessage(messageId: String)

    /**
     * Delete all messages in a conversation.
     * This method explicitly deletes messages without deleting the conversation itself.
     * Note: CASCADE delete on the foreign key will automatically delete messages when their
     * parent conversation is deleted, but this method is needed for selective operations
     * (e.g., clearing messages while keeping the conversation).
     * @param conversationId Conversation ID
     */
    @Query("DELETE FROM messages WHERE conversationId = :conversationId")
    suspend fun deleteMessagesForConversation(conversationId: String)

    /**
     * Delete all messages (for clearing history).
     */
    @Query("DELETE FROM messages")
    suspend fun deleteAllMessages()

    /**
     * Get the count of messages in a conversation.
     * @param conversationId Conversation ID
     * @return Number of messages in the conversation
     */
    @Query("SELECT COUNT(*) FROM messages WHERE conversationId = :conversationId")
    suspend fun getMessageCount(conversationId: String): Int

    /**
     * Get the count of all messages.
     * @return Total number of messages across all conversations
     */
    @Query("SELECT COUNT(*) FROM messages")
    suspend fun getTotalMessageCount(): Int

    /**
     * Get all voice messages with audio files.
     * @return Flow of voice messages that have audio paths.
     */
    @Query("SELECT * FROM messages WHERE isVoice = 1 AND audioPath IS NOT NULL ORDER BY createdAt DESC")
    fun getVoiceMessages(): Flow<List<MessageEntity>>
}
