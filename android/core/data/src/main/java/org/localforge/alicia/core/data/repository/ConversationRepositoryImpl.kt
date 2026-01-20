package org.localforge.alicia.core.data.repository

import android.content.Context
import dagger.hilt.android.qualifiers.ApplicationContext
import org.localforge.alicia.core.common.DeviceId
import org.localforge.alicia.core.common.Logger
import org.localforge.alicia.core.common.parseTimestamp
import org.localforge.alicia.core.data.mapper.*
import org.localforge.alicia.core.database.dao.ConversationDao
import org.localforge.alicia.core.database.dao.MessageDao
import org.localforge.alicia.core.domain.model.Conversation
import org.localforge.alicia.core.domain.model.Message
import org.localforge.alicia.core.domain.model.MessageRole
import org.localforge.alicia.core.domain.model.SyncStatus
import org.localforge.alicia.core.domain.repository.ConversationRepository
import org.localforge.alicia.core.network.api.AliciaApiService
import org.localforge.alicia.core.network.model.SwitchBranchRequest
import org.localforge.alicia.core.network.model.SyncMessageItem
import org.localforge.alicia.core.network.model.SyncMessagesRequest
import org.localforge.alicia.core.network.model.UpdateConversationRequest
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import java.time.Instant
import java.util.UUID
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Implementation of ConversationRepository that combines local (Room) and remote (API) data sources.
 */
@Singleton
class ConversationRepositoryImpl @Inject constructor(
    @ApplicationContext private val context: Context,
    private val conversationDao: ConversationDao,
    private val messageDao: MessageDao,
    private val apiService: AliciaApiService
) : ConversationRepository {

    private val logger = Logger.forTag("ConversationRepository")

    // ========== Conversation Operations ==========

    override fun getAllConversations(): Flow<List<Conversation>> {
        return conversationDao.getAllConversations()
            .map { entities -> entities.toDomain() }
    }

    override fun getConversationById(id: String): Flow<Conversation?> {
        return conversationDao.getAllConversations()
            .map { entities -> entities.find { it.id == id }?.toDomain() }
    }

    override suspend fun createConversation(title: String?): Result<Conversation> {
        return try {
            // Try to create on server first
            val request = org.localforge.alicia.core.network.model.CreateConversationRequest(
                title = title ?: "New Conversation"
            )
            val response = apiService.createConversation(request)

            if (response.isSuccessful && response.body() != null) {
                val conversation = response.body()!!.toDomain().copy(title = title)

                // Save to local database
                conversationDao.insertConversation(conversation.toEntity())

                Result.success(conversation)
            } else {
                // Fallback: create locally only
                val localConversation = Conversation(
                    id = UUID.randomUUID().toString(),
                    title = title,
                    createdAt = System.currentTimeMillis(),
                    updatedAt = System.currentTimeMillis(),
                    syncedAt = null
                )

                conversationDao.insertConversation(localConversation.toEntity())

                Result.success(localConversation)
            }
        } catch (e: Exception) {
            // Network error: create locally
            logger.w("Failed to create conversation on server, falling back to local creation", e)

            val localConversation = Conversation(
                id = UUID.randomUUID().toString(),
                title = title,
                createdAt = System.currentTimeMillis(),
                updatedAt = System.currentTimeMillis(),
                syncedAt = null
            )

            conversationDao.insertConversation(localConversation.toEntity())

            Result.success(localConversation)
        }
    }

    override suspend fun updateConversation(conversation: Conversation): Result<Unit> {
        return try {
            val updatedConversation = conversation.copy(
                updatedAt = System.currentTimeMillis()
            )
            conversationDao.updateConversation(updatedConversation.toEntity())
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun updateConversationTitle(conversationId: String, title: String): Result<Unit> {
        return try {
            val conversation = conversationDao.getConversationByIdSuspend(conversationId)
                ?: return Result.failure(IllegalArgumentException("Conversation not found"))

            // Sync to server first (matching web frontend behavior)
            try {
                apiService.updateConversation(
                    conversationId,
                    UpdateConversationRequest(title = title)
                )
            } catch (e: Exception) {
                // Network errors are expected in offline-first mode; log and proceed with local update
                logger.w("Failed to update conversation $conversationId on server, proceeding with local update", e)
            }

            // Update local database
            val updated = conversation.copy(
                title = title,
                updatedAt = System.currentTimeMillis()
            )

            conversationDao.updateConversation(updated)
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun deleteConversation(conversationId: String): Result<Unit> {
        return try {
            // Delete from server
            try {
                apiService.deleteConversation(conversationId)
            } catch (e: Exception) {
                // Network errors are expected in offline-first mode; log and proceed with local deletion
                logger.w("Failed to delete conversation $conversationId from server, proceeding with local deletion", e)
            }

            // Delete from local database (cascade will delete messages too)
            conversationDao.deleteConversation(conversationId)

            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun deleteAllConversations(): Result<Unit> {
        return try {
            conversationDao.deleteAllConversations()
            messageDao.deleteAllMessages()
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getConversationCount(): Int {
        return try {
            conversationDao.getConversationCount()
        } catch (e: Exception) {
            0
        }
    }

    override suspend fun getConversationToken(conversationId: String): Result<org.localforge.alicia.core.domain.model.TokenResponse> {
        return try {
            // Use persistent device ID instead of random timestamp
            val participantId = DeviceId.getParticipantId(context)

            val request = org.localforge.alicia.core.network.model.GenerateTokenRequest(
                participantId = participantId,
                participantName = "Android User"
            )
            val response = apiService.getConversationToken(conversationId, request)

            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!.toDomain())
            } else {
                Result.failure(Exception("Failed to get token: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ========== Message Operations ==========

    override fun getMessagesForConversation(conversationId: String): Flow<List<Message>> {
        return messageDao.getMessagesForConversation(conversationId)
            .map { entities -> entities.toDomain() }
    }

    override suspend fun getMessageById(messageId: String): Message? {
        return try {
            messageDao.getMessageById(messageId)?.toDomain()
        } catch (e: Exception) {
            null
        }
    }

    override suspend fun insertMessage(message: Message): Result<Unit> {
        return try {
            messageDao.insertMessage(message.toEntity())

            // Update conversation timestamp
            val conversation = conversationDao.getConversationByIdSuspend(message.conversationId)
            if (conversation != null) {
                conversationDao.updateConversation(
                    conversation.copy(updatedAt = System.currentTimeMillis())
                )
            }

            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun insertMessages(messages: List<Message>): Result<Unit> {
        return try {
            val entities = messages.map { it.toEntity() }
            messageDao.insertMessages(entities)
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun updateMessage(message: Message): Result<Unit> {
        return try {
            messageDao.updateMessage(message.toEntity())
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun deleteMessage(messageId: String): Result<Unit> {
        return try {
            messageDao.deleteMessage(messageId)
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getLastMessages(conversationId: String, limit: Int): List<Message> {
        return try {
            messageDao.getLastMessages(conversationId, limit).toDomain()
        } catch (e: Exception) {
            emptyList()
        }
    }

    override suspend fun getMessageCount(conversationId: String): Int {
        return try {
            messageDao.getMessageCount(conversationId)
        } catch (e: Exception) {
            0
        }
    }

    override suspend fun sendTextMessage(conversationId: String, content: String): Result<Message> {
        return try {
            // Send message to server
            val request = org.localforge.alicia.core.network.model.SendMessageRequest(contents = content)
            val response = apiService.sendMessage(conversationId, request)

            if (response.isSuccessful && response.body() != null) {
                val messageResponse = response.body()!!
                val message = messageResponse.toDomain()

                // Save to local database
                messageDao.insertMessage(message.toEntity())

                // Update conversation timestamp
                val conversation = conversationDao.getConversationByIdSuspend(conversationId)
                if (conversation != null) {
                    conversationDao.updateConversation(
                        conversation.copy(updatedAt = System.currentTimeMillis())
                    )
                }

                Result.success(message)
            } else {
                Result.failure(Exception("Failed to send message: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ========== Branch/Sibling Operations ==========

    override suspend fun getMessageSiblings(messageId: String): Result<List<Message>> {
        return try {
            val response = apiService.getMessageSiblings(messageId)

            if (response.isSuccessful && response.body() != null) {
                val siblings = response.body()!!.messages.map { it.toDomain() }
                Result.success(siblings)
            } else {
                Result.failure(Exception("Failed to get siblings: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            logger.e("Failed to get siblings for message $messageId", e)
            Result.failure(e)
        }
    }

    override suspend fun switchBranch(conversationId: String, tipMessageId: String): Result<Unit> {
        return try {
            val request = SwitchBranchRequest(tipMessageId = tipMessageId)
            val response = apiService.switchBranch(conversationId, request)

            if (response.isSuccessful) {
                logger.i("Successfully switched branch for conversation $conversationId to tip $tipMessageId")
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to switch branch: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            logger.e("Failed to switch branch for conversation $conversationId", e)
            Result.failure(e)
        }
    }

    // ========== Search Operations ==========

    /**
     * Search messages across all conversations using database query. See MessageDao.searchMessages() for search implementation details.
     */
    override fun searchMessages(query: String): Flow<List<Message>> {
        return messageDao.searchMessages(query)
            .map { entities -> entities.toDomain() }
    }

    override fun searchMessagesInConversation(
        conversationId: String,
        query: String
    ): Flow<List<Message>> {
        // Client-side filter: searches messages by substring match (case-insensitive). Unlike searchMessages(), this filters in-memory rather than using a database query.
        return messageDao.getMessagesForConversation(conversationId)
            .map { entities ->
                entities.filter { it.content.contains(query, ignoreCase = true) }
                    .toDomain()
            }
    }

    // ========== Sync Operations ==========

    override suspend fun getUnsyncedMessages(): List<Message> {
        return try {
            messageDao.getPendingMessages().toDomain()
        } catch (e: Exception) {
            emptyList()
        }
    }

    override suspend fun markMessageSynced(messageId: String) {
        try {
            val message = messageDao.getMessageById(messageId)
            if (message != null) {
                messageDao.updateSyncStatus(
                    localId = message.localId ?: messageId,
                    serverId = messageId,
                    syncStatus = SyncStatus.SYNCED.value,
                    syncedAt = System.currentTimeMillis()
                )
            }
        } catch (e: Exception) {
            logger.e("Failed to mark message $messageId as synced", e)
        }
    }

    override suspend fun markMessagesSynced(messageIds: List<String>) {
        try {
            val syncedAt = System.currentTimeMillis()
            messageIds.forEach { messageId ->
                val message = messageDao.getMessageById(messageId)
                if (message != null) {
                    messageDao.updateSyncStatus(
                        localId = message.localId ?: messageId,
                        serverId = messageId,
                        syncStatus = SyncStatus.SYNCED.value,
                        syncedAt = syncedAt
                    )
                }
            }
        } catch (e: Exception) {
            logger.e("Failed to mark messages as synced: $messageIds", e)
        }
    }

    override suspend fun getUnsyncedConversations(): List<Conversation> {
        return try {
            conversationDao.getUnsyncedConversations().toDomain()
        } catch (e: Exception) {
            emptyList()
        }
    }

    override suspend fun markConversationSynced(conversationId: String) {
        try {
            val conversation = conversationDao.getConversationByIdSuspend(conversationId)
            if (conversation != null) {
                conversationDao.updateConversation(
                    conversation.copy(syncedAt = System.currentTimeMillis())
                )
            }
        } catch (e: Exception) {
            logger.e("Failed to mark conversation $conversationId as synced", e)
        }
    }

    override suspend fun syncWithServer(): Result<Unit> {
        return try {
            // Fetch conversations from server
            val conversationsResponse = apiService.getConversations()

            if (conversationsResponse.isSuccessful && conversationsResponse.body() != null) {
                val conversationListResponse = conversationsResponse.body()!!
                val conversations = conversationListResponse.conversations

                // Save to local database
                val conversationEntities = conversations.map { it.toEntity() }
                conversationDao.insertConversations(conversationEntities)

                // Sync messages for each conversation using bidirectional sync protocol
                for (conversation in conversations) {
                    syncConversationMessages(conversation.id)
                }
            }

            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Sync messages for a specific conversation using bidirectional sync protocol.
     * Matches the web frontend implementation in useSync.ts.
     */
    private suspend fun syncConversationMessages(conversationId: String): Result<Unit> {
        return try {
            // Step 1: Get all local messages for the conversation
            val localMessages = messageDao.getMessagesForConversation(conversationId)
                .map { entities -> entities.toDomain() }
                .first() // Get current state for sync request

            // Step 2: Build sync request with local message state
            val syncItems = localMessages.map { message ->
                SyncMessageItem(
                    localId = message.localId ?: message.id,
                    sequenceNumber = message.sequenceNumber,
                    previousId = message.previousId,
                    role = message.role.value,
                    contents = message.content,
                    createdAt = Instant.ofEpochMilli(message.createdAt).toString(),
                    updatedAt = message.updatedAt?.let { Instant.ofEpochMilli(it).toString() }
                )
            }

            val syncRequest = SyncMessagesRequest(messages = syncItems)

            // Step 3: Call sync API endpoint (POST)
            val syncResponse = apiService.syncMessages(conversationId, syncRequest)

            if (syncResponse.isSuccessful && syncResponse.body() != null) {
                val result = syncResponse.body()!!
                val syncedAt = parseTimestamp(result.syncedAt)

                // Step 4: Process SyncResponse and update message sync status
                result.syncedMessages.forEach { syncedMessage ->
                    when (syncedMessage.status) {
                        "synced" -> {
                            // Message was synced successfully
                            val message = syncedMessage.message
                            if (message != null) {
                                // Insert or update the server message
                                val serverMessage = message.toDomain()
                                messageDao.insertMessage(serverMessage.toEntity())
                            }

                            // Update sync status for local message
                            messageDao.updateSyncStatus(
                                localId = syncedMessage.localId,
                                serverId = syncedMessage.serverId,
                                syncStatus = SyncStatus.SYNCED.value,
                                syncedAt = syncedAt
                            )
                        }
                        "conflict" -> {
                            // Handle conflicts: use server version as authoritative
                            val conflict = syncedMessage.conflict
                            if (conflict != null) {
                                val serverMessage = conflict.serverMessage
                                if (serverMessage != null) {
                                    val domainMessage = serverMessage.toDomain()
                                    messageDao.insertMessage(domainMessage.toEntity())
                                }

                                // Mark local message as conflict
                                messageDao.updateSyncStatus(
                                    localId = syncedMessage.localId,
                                    serverId = null,
                                    syncStatus = SyncStatus.CONFLICT.value,
                                    syncedAt = syncedAt
                                )

                                // Log conflict for debugging
                                logger.w("Sync conflict for message ${syncedMessage.localId}: ${conflict.reason}")
                            }
                        }
                        "local-only" -> {
                            // Message exists only locally, keep pending status
                            // No action needed - will be synced in next round
                            logger.d("Message ${syncedMessage.localId} exists only locally, keeping pending status for next sync")
                        }
                    }
                }

                // Update conversation sync timestamp
                markConversationSynced(conversationId)
            }

            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
