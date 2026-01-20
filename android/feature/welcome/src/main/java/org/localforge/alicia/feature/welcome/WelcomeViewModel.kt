package org.localforge.alicia.feature.welcome

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import org.localforge.alicia.core.domain.model.Conversation
import org.localforge.alicia.core.domain.repository.ConversationRepository
import javax.inject.Inject

/**
 * ViewModel for the WelcomeScreen.
 *
 * Manages:
 * - Loading recent conversations
 * - Loading state
 */
@HiltViewModel
class WelcomeViewModel @Inject constructor(
    private val conversationRepository: ConversationRepository
) : ViewModel() {

    private val _recentConversations = MutableStateFlow<List<Conversation>>(emptyList())
    val recentConversations: StateFlow<List<Conversation>> = _recentConversations.asStateFlow()

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    init {
        loadRecentConversations()
    }

    private fun loadRecentConversations() {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                conversationRepository.getAllConversations()
                    .map { conversations ->
                        // Filter active conversations and sort by updated date
                        conversations
                            .filter { !it.isDeleted }
                            .sortedByDescending { it.updatedAt }
                            .take(5)
                    }
                    .collect { conversations ->
                        _recentConversations.value = conversations
                        _isLoading.value = false
                    }
            } catch (e: Exception) {
                _isLoading.value = false
                // Log error but don't crash - show empty list
                _recentConversations.value = emptyList()
            }
        }
    }

    fun refresh() {
        loadRecentConversations()
    }
}
