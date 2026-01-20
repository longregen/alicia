package org.localforge.alicia.ui

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import org.localforge.alicia.core.domain.model.Conversation
import org.localforge.alicia.core.domain.repository.ConversationRepository
import javax.inject.Inject

/**
 * App-level ViewModel for managing sidebar state and global navigation.
 *
 * Manages:
 * - Conversations list for sidebar
 * - Selected conversation tracking
 * - Connection status
 * - Conversation operations (rename, archive, delete)
 */
@HiltViewModel
class AliciaAppViewModel @Inject constructor(
    private val conversationRepository: ConversationRepository
) : ViewModel() {

    private val _conversations = MutableStateFlow<List<Conversation>>(emptyList())
    val conversations: StateFlow<List<Conversation>> = _conversations.asStateFlow()

    private val _selectedConversationId = MutableStateFlow<String?>(null)
    val selectedConversationId: StateFlow<String?> = _selectedConversationId.asStateFlow()

    private val _isConnected = MutableStateFlow(false)
    val isConnected: StateFlow<Boolean> = _isConnected.asStateFlow()

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    init {
        loadConversations()
        observeConnectionStatus()
    }

    private fun loadConversations() {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                conversationRepository.getAllConversations()
                    .collect { conversations ->
                        _conversations.value = conversations.sortedByDescending { it.updatedAt }
                        _isLoading.value = false
                    }
            } catch (e: Exception) {
                _isLoading.value = false
                // Log error but don't crash
            }
        }
    }

    private fun observeConnectionStatus() {
        // TODO: Implement connection status observation from WebSocket/network
        // For now, assume connected
        _isConnected.value = true
    }

    fun setSelectedConversation(conversationId: String?) {
        _selectedConversationId.value = conversationId
    }

    fun renameConversation(id: String, newTitle: String) {
        viewModelScope.launch {
            try {
                conversationRepository.updateConversationTitle(id, newTitle)
            } catch (e: Exception) {
                // Handle error - maybe show a toast
            }
        }
    }

    fun archiveConversation(id: String) {
        viewModelScope.launch {
            try {
                conversationRepository.archiveConversation(id)
            } catch (e: Exception) {
                // Handle error
            }
        }
    }

    fun unarchiveConversation(id: String) {
        viewModelScope.launch {
            try {
                conversationRepository.unarchiveConversation(id)
            } catch (e: Exception) {
                // Handle error
            }
        }
    }

    fun deleteConversation(id: String) {
        viewModelScope.launch {
            try {
                conversationRepository.deleteConversation(id)
                // Clear selection if this was the selected conversation
                if (_selectedConversationId.value == id) {
                    _selectedConversationId.value = null
                }
            } catch (e: Exception) {
                // Handle error
            }
        }
    }

    fun refresh() {
        loadConversations()
    }
}
