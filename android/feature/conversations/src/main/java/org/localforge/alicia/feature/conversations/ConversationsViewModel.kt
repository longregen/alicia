package org.localforge.alicia.feature.conversations

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import org.localforge.alicia.core.data.sync.SyncManager
import org.localforge.alicia.core.domain.model.ConversationStatus
import org.localforge.alicia.core.domain.repository.ConversationRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class ConversationsViewModel @Inject constructor(
    private val conversationRepository: ConversationRepository,
    private val syncManager: SyncManager
) : ViewModel() {

    private val _conversations = MutableStateFlow<List<Conversation>>(emptyList())
    val conversations: StateFlow<List<Conversation>> = _conversations.asStateFlow()

    private val _errorState = MutableStateFlow<String?>(null)
    val errorState: StateFlow<String?> = _errorState.asStateFlow()

    init {
        loadConversations()
        syncManager.startPeriodicSync()
    }

    override fun onCleared() {
        super.onCleared()
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
                            messageCount = domainConv.messageCount,
                            isArchived = domainConv.status == ConversationStatus.ARCHIVED
                        )
                    }
                }
                .collect { conversationList ->
                    _conversations.value = conversationList
                }
        }
    }

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

    fun archiveConversation(id: String) {
        viewModelScope.launch {
            try {
                conversationRepository.archiveConversation(id)
                _errorState.value = null
            } catch (e: Exception) {
                _errorState.value = "Failed to archive conversation: ${e.message}"
            }
        }
    }

    fun unarchiveConversation(id: String) {
        viewModelScope.launch {
            try {
                conversationRepository.unarchiveConversation(id)
                _errorState.value = null
            } catch (e: Exception) {
                _errorState.value = "Failed to unarchive conversation: ${e.message}"
            }
        }
    }

    fun renameConversation(id: String, newTitle: String) {
        viewModelScope.launch {
            try {
                conversationRepository.updateConversationTitle(id, newTitle)
                _errorState.value = null
            } catch (e: Exception) {
                _errorState.value = "Failed to rename conversation: ${e.message}"
            }
        }
    }

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

    fun clearError() {
        _errorState.value = null
    }
}
