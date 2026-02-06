package com.alicia.assistant.storage

import android.content.Context
import android.util.Log
import com.alicia.assistant.service.AliciaApiClient
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.IOException

class ConversationRepository(
    context: Context,
    private val apiClient: AliciaApiClient
) {
    companion object {
        private const val TAG = "ConversationRepo"
    }

    private val db = ConversationDatabase(context)

    suspend fun listConversations(): List<AliciaApiClient.Conversation> = withContext(Dispatchers.IO) {
        try {
            val conversations = apiClient.listConversations()
            db.cacheConversations(conversations)
            conversations
        } catch (e: IOException) {
            Log.w(TAG, "Failed to fetch conversations from server, using cache", e)
            db.getCachedConversations()
        }
    }

    suspend fun createConversation(title: String = "New Chat"): AliciaApiClient.Conversation = withContext(Dispatchers.IO) {
        val conversation = apiClient.createConversation(title)
        db.cacheConversation(conversation)
        conversation
    }

    suspend fun getMessages(conversationId: String): List<AliciaApiClient.Message> = withContext(Dispatchers.IO) {
        try {
            val messages = apiClient.getMessages(conversationId)
            db.cacheMessages(conversationId, messages)
            messages
        } catch (e: IOException) {
            Log.w(TAG, "Failed to fetch messages from server, using cache", e)
            db.getCachedMessages(conversationId)
        }
    }

    suspend fun sendMessage(
        conversationId: String,
        content: String,
        previousId: String? = null
    ): AliciaApiClient.SyncResponse = withContext(Dispatchers.IO) {
        val response = apiClient.sendMessageSync(conversationId, content, previousId)
        db.appendMessage(response.userMessage)
        db.appendMessage(response.assistantMessage)
        // Update cached conversation title if the server provided a new one
        response.conversationTitle?.let { newTitle ->
            db.updateConversationTitle(conversationId, newTitle)
        }
        response
    }

}
