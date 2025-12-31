package org.localforge.alicia.feature.conversations

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import org.localforge.alicia.core.data.sync.SyncManager
import org.localforge.alicia.core.domain.repository.ConversationRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * ViewModel for managing the conversations list screen.
 *
 * This ViewModel handles loading, displaying, and managing user conversations,
 * as well as coordinating synchronization with the backend server.
 * It observes the local database for conversation changes and manages synchronization
 * with the backend server via the SyncManager.
 *
 * @property conversationRepository Repository for accessing conversation data
 * @property syncManager Manager for handling data synchronization with the backend
 */
@HiltViewModel
class ConversationsViewModel @Inject constructor(
    private val conversationRepository: ConversationRepository,
    private val syncManager: SyncManager
) : ViewModel() {

    private val _conversations = MutableStateFlow<List<Conversation>>(emptyList())

    /**
     * StateFlow of all conversations. Sorting is handled by the repository layer.
     * This list is automatically updated when the local database changes.
     */
    val conversations: StateFlow<List<Conversation>> = _conversations.asStateFlow()

    private val _errorState = MutableStateFlow<String?>(null)

    /**
     * StateFlow containing error messages from failed operations.
     * Null if no error has occurred.
     */
    val errorState: StateFlow<String?> = _errorState.asStateFlow()

    init {
        loadConversations()
        // Start periodic sync when ViewModel is created
        syncManager.startPeriodicSync()
    }

    override fun onCleared() {
        super.onCleared()
        // Stop periodic sync when ViewModel is cleared
        syncManager.stopPeriodicSync()
    }

    private fun loadConversations() {
        viewModelScope.launch {
            conversationRepository.getAllConversations()
                .map { domainConversations ->
                    domainConversations.map { domainConv ->
                        Conversation(
                            id = domainConv.id,
                            title = domainConv.title,
                            lastMessage = domainConv.lastMessagePreview,
                            timestamp = domainConv.updatedAt,
                            messageCount = domainConv.messageCount
                        )
                    }
                }
                .collect { conversationList ->
                    _conversations.value = conversationList
                }
        }
    }

    /**
     * Deletes a specific conversation and all its messages.
     * Attempts server deletion first, then deletes locally regardless of server result.
     *
     * @param id The unique identifier of the conversation to delete
     */
    fun deleteConversation(id: String) {
        viewModelScope.launch {
            try {
                conversationRepository.deleteConversation(id)
                _errorState.value = null
            } catch (e: Exception) {
                _errorState.value = "Failed to delete conversation: ${e.message}"
            }
        }
    }

    /**
     * Deletes all conversations and their messages from the local database.
     * This operation cannot be undone. Note: Does not sync deletions to the server.
     */
    fun clearAllConversations() {
        viewModelScope.launch {
            try {
                conversationRepository.deleteAllConversations()
                _errorState.value = null
            } catch (e: Exception) {
                _errorState.value = "Failed to clear conversations: ${e.message}"
            }
        }
    }

    /**
     * Triggers an immediate synchronization with the backend server.
     * This bypasses the normal periodic sync schedule and attempts to sync now.
     */
    fun syncNow() {
        viewModelScope.launch {
            try {
                syncManager.syncNow()
                _errorState.value = null
            } catch (e: Exception) {
                _errorState.value = "Failed to sync: ${e.message}"
            }
        }
    }

    /**
     * Clears the current error state.
     */
    fun clearError() {
        _errorState.value = null
    }
}
