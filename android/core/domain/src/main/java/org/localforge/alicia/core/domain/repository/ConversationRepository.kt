package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.Conversation
import org.localforge.alicia.core.domain.model.Message
import org.localforge.alicia.core.domain.model.TokenResponse
import kotlinx.coroutines.flow.Flow

interface ConversationRepository {

    fun getAllConversations(): Flow<List<Conversation>>

    fun getConversationById(id: String): Flow<Conversation?>

    suspend fun createConversation(title: String? = null): Result<Conversation>

    suspend fun updateConversation(conversation: Conversation): Result<Unit>

    suspend fun updateConversationTitle(conversationId: String, title: String): Result<Unit>

    suspend fun archiveConversation(conversationId: String): Result<Unit>

    suspend fun unarchiveConversation(conversationId: String): Result<Unit>

    suspend fun deleteConversation(conversationId: String): Result<Unit>

    suspend fun deleteAllConversations(): Result<Unit>

    suspend fun getConversationCount(): Int

    suspend fun getConversationToken(conversationId: String): Result<TokenResponse>

    fun getMessagesForConversation(conversationId: String): Flow<List<Message>>

    suspend fun getMessageById(messageId: String): Message?

    suspend fun insertMessage(message: Message): Result<Unit>

    suspend fun insertMessages(messages: List<Message>): Result<Unit>

    suspend fun updateMessage(message: Message): Result<Unit>

    suspend fun deleteMessage(messageId: String): Result<Unit>

    suspend fun getLastMessages(conversationId: String, limit: Int): List<Message>

    suspend fun getMessageCount(conversationId: String): Int

    suspend fun sendTextMessage(conversationId: String, content: String): Result<Message>

    suspend fun getMessageSiblings(messageId: String): Result<List<Message>>

    suspend fun switchBranch(conversationId: String, tipMessageId: String): Result<Unit>

    fun searchMessages(query: String): Flow<List<Message>>

    fun searchMessagesInConversation(conversationId: String, query: String): Flow<List<Message>>

    suspend fun getUnsyncedMessages(): List<Message>

    suspend fun markMessageSynced(messageId: String)

    suspend fun markMessagesSynced(messageIds: List<String>)

    suspend fun getUnsyncedConversations(): List<Conversation>

    suspend fun markConversationSynced(conversationId: String)

    suspend fun syncWithServer(): Result<Unit>
}
