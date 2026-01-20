package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.Conversation
import org.localforge.alicia.core.domain.model.Message
import org.localforge.alicia.core.domain.model.TokenResponse
import kotlinx.coroutines.flow.Flow

/**
 * Repository interface for managing conversations and messages.
 * Provides abstraction over data sources (local database, remote API).
 */
interface ConversationRepository {

    // ========== Conversation Operations ==========

    /**
     * Get all conversations ordered by most recently updated.
     * @return Flow of conversation list that updates automatically.
     */
    fun getAllConversations(): Flow<List<Conversation>>

    /**
     * Get a specific conversation by ID.
     * @param id Conversation ID
     * @return Flow of conversation or null if not found.
     */
    fun getConversationById(id: String): Flow<Conversation?>

    /**
     * Create a new conversation.
     * @param title Optional title for the conversation
     * @return Result containing the created conversation or error
     */
    suspend fun createConversation(title: String? = null): Result<Conversation>

    /**
     * Update an existing conversation.
     * @param conversation Conversation to update
     * @return Result indicating success or failure
     */
    suspend fun updateConversation(conversation: Conversation): Result<Unit>

    /**
     * Update the title of a conversation.
     * @param conversationId Conversation ID
     * @param title New title
     * @return Result indicating success or failure
     */
    suspend fun updateConversationTitle(conversationId: String, title: String): Result<Unit>

    /**
     * Delete a conversation and all its messages.
     * @param conversationId Conversation ID to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteConversation(conversationId: String): Result<Unit>

    /**
     * Delete all conversations (clear history).
     * @return Result indicating success or failure
     */
    suspend fun deleteAllConversations(): Result<Unit>

    /**
     * Get the count of all conversations.
     * @return Total number of conversations
     */
    suspend fun getConversationCount(): Int

    /**
     * Get a LiveKit token for voice conversation.
     * @param conversationId Conversation ID
     * @return Result containing token response or error
     */
    suspend fun getConversationToken(conversationId: String): Result<TokenResponse>

    // ========== Message Operations ==========

    /**
     * Get all messages for a specific conversation.
     * @param conversationId Conversation ID
     * @return Flow of message list that updates automatically.
     */
    fun getMessagesForConversation(conversationId: String): Flow<List<Message>>

    /**
     * Get a specific message by ID.
     * Returns nullable Message; null indicates not found (not an error condition).
     * @param messageId Message ID
     * @return Message or null if not found
     */
    suspend fun getMessageById(messageId: String): Message?

    /**
     * Insert a new message.
     * @param message Message to insert
     * @return Result indicating success or failure
     */
    suspend fun insertMessage(message: Message): Result<Unit>

    /**
     * Insert multiple messages.
     * @param messages List of messages to insert
     * @return Result indicating success or failure
     */
    suspend fun insertMessages(messages: List<Message>): Result<Unit>

    /**
     * Update an existing message.
     * @param message Message to update
     * @return Result indicating success or failure
     */
    suspend fun updateMessage(message: Message): Result<Unit>

    /**
     * Delete a specific message.
     * @param messageId Message ID to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteMessage(messageId: String): Result<Unit>

    /**
     * Get the last N messages from a conversation.
     * Returns list directly; empty list for no messages (not an error).
     * @param conversationId Conversation ID
     * @param limit Number of messages to retrieve
     * @return List of most recent messages
     */
    suspend fun getLastMessages(conversationId: String, limit: Int): List<Message>

    /**
     * Get the count of messages in a conversation.
     * @param conversationId Conversation ID
     * @return Number of messages in the conversation
     */
    suspend fun getMessageCount(conversationId: String): Int

    /**
     * Send a text message to a conversation via the API.
     * @param conversationId Conversation ID
     * @param content Message content
     * @return Result containing the message response or error
     */
    suspend fun sendTextMessage(conversationId: String, content: String): Result<Message>

    // ========== Branch/Sibling Operations ==========

    /**
     * Get sibling messages for a specific message.
     * Siblings are messages that share the same parent (previous_id).
     * Used for branch navigation in the UI.
     *
     * @param messageId The message ID to get siblings for
     * @return Result containing list of sibling messages or error
     */
    suspend fun getMessageSiblings(messageId: String): Result<List<Message>>

    /**
     * Switch the conversation to a different branch by updating the tip message.
     * This changes which branch of the conversation is active.
     *
     * @param conversationId The conversation ID
     * @param tipMessageId The message ID to set as the new tip (last message in the branch)
     * @return Result indicating success or failure
     */
    suspend fun switchBranch(conversationId: String, tipMessageId: String): Result<Unit>

    // ========== Search Operations ==========

    /**
     * Search messages by content across all conversations.
     * @param query Search query
     * @return Flow of matching messages
     */
    fun searchMessages(query: String): Flow<List<Message>>

    /**
     * Search messages by content within a specific conversation.
     * @param conversationId Conversation ID
     * @param query Search query
     * @return Flow of matching messages
     */
    fun searchMessagesInConversation(conversationId: String, query: String): Flow<List<Message>>

    // ========== Sync Operations ==========

    /**
     * Get all unsynced messages that need to be uploaded to the server.
     * @return List of unsynced messages
     */
    suspend fun getUnsyncedMessages(): List<Message>

    /**
     * Mark a message as synced with the server.
     * @param messageId Message ID
     */
    suspend fun markMessageSynced(messageId: String)

    /**
     * Mark multiple messages as synced.
     * @param messageIds List of message IDs
     */
    suspend fun markMessagesSynced(messageIds: List<String>)

    /**
     * Get all conversations that need to be synced.
     * @return List of unsynced conversations
     */
    suspend fun getUnsyncedConversations(): List<Conversation>

    /**
     * Mark a conversation as synced with the server.
     * @param conversationId Conversation ID
     */
    suspend fun markConversationSynced(conversationId: String)

    /**
     * Sync local data with the server.
     * @return Result indicating success or failure
     */
    suspend fun syncWithServer(): Result<Unit>
}
