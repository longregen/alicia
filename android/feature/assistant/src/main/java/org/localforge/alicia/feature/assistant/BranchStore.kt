package org.localforge.alicia.feature.assistant

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import javax.inject.Inject
import javax.inject.Singleton

data class SiblingMessage(
    val id: String,
    val content: String,
    val createdAt: Long,
    val role: String,
    val sequenceNumber: Int
)

enum class BranchDirection {
    PREV,
    NEXT
}

data class MessageBranchState(
    val siblings: List<SiblingMessage> = emptyList(),
    val currentIndex: Int = 0,
    val isLoading: Boolean = false,
    val error: String? = null
) {
    val count: Int get() = siblings.size
    val currentSibling: SiblingMessage? get() = siblings.getOrNull(currentIndex)
    val hasPrevious: Boolean get() = currentIndex > 0
    val hasNext: Boolean get() = currentIndex < siblings.size - 1

    fun indexOfSibling(messageId: String): Int = siblings.indexOfFirst { it.id == messageId }
}

@Singleton
class BranchStore @Inject constructor() {

    private val _branchStates = MutableStateFlow<Map<String, MessageBranchState>>(emptyMap())
    val branchStates: StateFlow<Map<String, MessageBranchState>> = _branchStates.asStateFlow()

    fun updateSiblingsFromServer(
        messageId: String,
        siblings: List<SiblingMessage>,
        activeMessageId: String
    ) {
        _branchStates.update { currentMap ->
            val currentIndex = siblings.indexOfFirst { it.id == activeMessageId }.coerceAtLeast(0)
            currentMap + (messageId to MessageBranchState(
                siblings = siblings,
                currentIndex = currentIndex,
                isLoading = false,
                error = null
            ))
        }
    }

    fun setLoading(messageId: String, isLoading: Boolean) {
        _branchStates.update { currentMap ->
            val existingState = currentMap[messageId] ?: MessageBranchState()
            currentMap + (messageId to existingState.copy(isLoading = isLoading))
        }
    }

    fun setError(messageId: String, error: String?) {
        _branchStates.update { currentMap ->
            val existingState = currentMap[messageId] ?: MessageBranchState()
            currentMap + (messageId to existingState.copy(
                isLoading = false,
                error = error
            ))
        }
    }

    fun peekNavigationTarget(messageId: String, direction: BranchDirection): SiblingMessage? {
        val state = _branchStates.value[messageId] ?: return null
        if (state.siblings.size <= 1) return null

        val newIndex = when (direction) {
            BranchDirection.PREV -> state.currentIndex - 1
            BranchDirection.NEXT -> state.currentIndex + 1
        }

        return state.siblings.getOrNull(newIndex)
    }

    fun setActiveSibling(messageId: String, newActiveId: String) {
        _branchStates.update { currentMap ->
            val existingState = currentMap[messageId] ?: return@update currentMap
            val newIndex = existingState.indexOfSibling(newActiveId)
            if (newIndex >= 0) {
                currentMap + (messageId to existingState.copy(currentIndex = newIndex))
            } else {
                currentMap
            }
        }
    }

    fun getBranchCount(messageId: String): Int {
        return _branchStates.value[messageId]?.count ?: 0
    }

    fun getCurrentIndex(messageId: String): Int {
        return _branchStates.value[messageId]?.currentIndex ?: 0
    }

    fun getBranchState(messageId: String): MessageBranchState? {
        return _branchStates.value[messageId]
    }

    fun hasSiblings(messageId: String): Boolean {
        return _branchStates.value.containsKey(messageId)
    }

    fun hasMultipleBranches(messageId: String): Boolean {
        return (_branchStates.value[messageId]?.count ?: 0) > 1
    }

    fun clearAll() {
        _branchStates.value = emptyMap()
    }

    fun clearBranch(messageId: String) {
        _branchStates.update { it - messageId }
    }

}
