package org.localforge.alicia.core.database.dao

import androidx.room.*
import org.localforge.alicia.core.database.entity.MessageEntity
import kotlinx.coroutines.flow.Flow

@Dao
interface MessageDao {

    @Query("SELECT * FROM messages WHERE conversationId = :conversationId ORDER BY createdAt ASC")
    fun getMessagesForConversation(conversationId: String): Flow<List<MessageEntity>>

    @Query("SELECT * FROM messages WHERE id = :id LIMIT 1")
    suspend fun getMessageById(id: String): MessageEntity?

    @Query("SELECT * FROM messages WHERE syncStatus = 'pending' ORDER BY createdAt ASC")
    suspend fun getPendingMessages(): List<MessageEntity>

    @Query("SELECT * FROM messages WHERE conversationId = :conversationId AND syncStatus = 'pending' ORDER BY createdAt ASC")
    suspend fun getPendingMessagesForConversation(conversationId: String): List<MessageEntity>

    @Query("SELECT * FROM messages WHERE localId = :localId LIMIT 1")
    suspend fun getMessageByLocalId(localId: String): MessageEntity?

    @Query("SELECT * FROM messages WHERE syncStatus = :syncStatus ORDER BY createdAt ASC")
    suspend fun getMessagesBySyncStatus(syncStatus: String): List<MessageEntity>

    @Query("SELECT * FROM messages WHERE conversationId = :conversationId AND syncStatus = :syncStatus ORDER BY createdAt ASC")
    suspend fun getMessagesBySyncStatusForConversation(conversationId: String, syncStatus: String): List<MessageEntity>

    @Query("UPDATE messages SET serverId = :serverId, syncStatus = :syncStatus, syncedAt = :syncedAt WHERE localId = :localId")
    suspend fun updateSyncStatus(localId: String, serverId: String?, syncStatus: String, syncedAt: Long?)

    @Query("SELECT * FROM messages WHERE content LIKE '%' || :query || '%' ORDER BY createdAt DESC")
    fun searchMessages(query: String): Flow<List<MessageEntity>>

    @Query("SELECT * FROM messages WHERE conversationId = :conversationId AND content LIKE '%' || :query || '%' ORDER BY createdAt DESC")
    fun searchMessagesInConversation(conversationId: String, query: String): Flow<List<MessageEntity>>

    @Query("SELECT * FROM messages WHERE conversationId = :conversationId ORDER BY createdAt DESC LIMIT :limit")
    suspend fun getLastMessages(conversationId: String, limit: Int): List<MessageEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertMessage(message: MessageEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertMessages(messages: List<MessageEntity>)

    @Update
    suspend fun updateMessage(message: MessageEntity)

    @Query("DELETE FROM messages WHERE id = :messageId")
    suspend fun deleteMessage(messageId: String)

    // CASCADE on foreign key auto-deletes messages when conversation is deleted;
    // this method is for clearing messages while keeping the conversation.
    @Query("DELETE FROM messages WHERE conversationId = :conversationId")
    suspend fun deleteMessagesForConversation(conversationId: String)

    @Query("DELETE FROM messages")
    suspend fun deleteAllMessages()

    @Query("SELECT COUNT(*) FROM messages WHERE conversationId = :conversationId")
    suspend fun getMessageCount(conversationId: String): Int

    @Query("SELECT COUNT(*) FROM messages")
    suspend fun getTotalMessageCount(): Int

    @Query("SELECT * FROM messages WHERE isVoice = 1 AND audioPath IS NOT NULL ORDER BY createdAt DESC")
    fun getVoiceMessages(): Flow<List<MessageEntity>>
}
