package org.localforge.alicia.feature.assistant

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Represents a sibling message from the backend.
 * A sibling is a message that shares the same parent (previous_id).
 */
data class SiblingMessage(
    val id: String,
    val content: String,
    val createdAt: Long,
    val role: String,
    val sequenceNumber: Int
)

/**
 * Branch navigation direction
 */
enum class BranchDirection {
    PREV,
    NEXT
}

/**
 * State holder for a single message's siblings/branches.
 * Unlike the web frontend's local-only branchStore, this syncs with the backend.
 *
 * @property siblings List of sibling messages from backend (messages sharing same parent)
 * @property currentIndex Index of the currently active/displayed sibling
 * @property isLoading Whether siblings are being fetched from server
 * @property error Error message if fetch failed
 */
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

    /**
     * Find the index of a sibling by message ID.
     */
    fun indexOfSibling(messageId: String): Int = siblings.indexOfFirst { it.id == messageId }
}

/**
 * Store for managing message branches/siblings.
 *
 * Unlike the old local-only implementation, this store:
 * 1. Stores sibling message data fetched from backend (message IDs, not just content)
 * 2. Tracks which sibling is currently active
 * 3. Does NOT store local-only branches - all branching goes through the server
 *
 * The flow is:
 * 1. When a message is displayed, fetch siblings from GET /messages/{id}/siblings
 * 2. When user navigates branches, call PUT /conversations/{id}/switch-branch
 * 3. Server updates conversation tip, returns new message chain
 * 4. UI reloads the conversation with the new active branch
 *
 * Matches web frontend behavior in useConversations.ts and ChatBubble.tsx
 */
@Singleton
class BranchStore @Inject constructor() {

    // Map of messageId -> branch state (siblings from server)
    private val _branchStates = MutableStateFlow<Map<String, MessageBranchState>>(emptyMap())
    val branchStates: StateFlow<Map<String, MessageBranchState>> = _branchStates.asStateFlow()

    /**
     * Update siblings for a message from server response.
     * Called after fetching siblings from GET /messages/{id}/siblings
     *
     * @param messageId The message ID whose siblings were fetched
     * @param siblings List of sibling messages from server
     * @param activeMessageId The currently active/displayed message ID (to set currentIndex)
     */
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

    /**
     * Mark a message as loading siblings.
     * Called before making API request.
     */
    fun setLoading(messageId: String, isLoading: Boolean) {
        _branchStates.update { currentMap ->
            val existingState = currentMap[messageId] ?: MessageBranchState()
            currentMap + (messageId to existingState.copy(isLoading = isLoading))
        }
    }

    /**
     * Set error state for a message's siblings fetch.
     */
    fun setError(messageId: String, error: String?) {
        _branchStates.update { currentMap ->
            val existingState = currentMap[messageId] ?: MessageBranchState()
            currentMap + (messageId to existingState.copy(
                isLoading = false,
                error = error
            ))
        }
    }

    /**
     * Get the sibling that would be navigated to without changing state.
     * Used to determine the target message ID for switch-branch API call.
     *
     * @param messageId The current message ID
     * @param direction Navigation direction (PREV or NEXT)
     * @return The target sibling message, or null if navigation not possible
     */
    fun peekNavigationTarget(messageId: String, direction: BranchDirection): SiblingMessage? {
        val state = _branchStates.value[messageId] ?: return null
        if (state.siblings.size <= 1) return null

        val newIndex = when (direction) {
            BranchDirection.PREV -> state.currentIndex - 1
            BranchDirection.NEXT -> state.currentIndex + 1
        }

        return state.siblings.getOrNull(newIndex)
    }

    /**
     * Update the current index after successful branch switch.
     * Called after PUT /conversations/{id}/switch-branch succeeds.
     *
     * @param messageId The message ID
     * @param newActiveId The ID of the newly active sibling
     */
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

    /**
     * Get the branch count for a message.
     * @param messageId The message ID
     * @return The number of siblings (branches)
     */
    fun getBranchCount(messageId: String): Int {
        return _branchStates.value[messageId]?.count ?: 0
    }

    /**
     * Get the current branch index for a message.
     * @param messageId The message ID
     * @return The current index (0-based)
     */
    fun getCurrentIndex(messageId: String): Int {
        return _branchStates.value[messageId]?.currentIndex ?: 0
    }

    /**
     * Get the branch state for a message.
     * @param messageId The message ID
     * @return The branch state, or null if not initialized
     */
    fun getBranchState(messageId: String): MessageBranchState? {
        return _branchStates.value[messageId]
    }

    /**
     * Check if siblings have been fetched for a message.
     */
    fun hasSiblings(messageId: String): Boolean {
        return _branchStates.value.containsKey(messageId)
    }

    /**
     * Check if a message has multiple siblings (branches).
     */
    fun hasMultipleBranches(messageId: String): Boolean {
        return (_branchStates.value[messageId]?.count ?: 0) > 1
    }

    /**
     * Clear all branch state.
     * Called when starting a new conversation.
     */
    fun clearAll() {
        _branchStates.value = emptyMap()
    }

    /**
     * Clear branch state for a specific message.
     * @param messageId The message ID
     */
    fun clearBranch(messageId: String) {
        _branchStates.update { it - messageId }
    }

}
