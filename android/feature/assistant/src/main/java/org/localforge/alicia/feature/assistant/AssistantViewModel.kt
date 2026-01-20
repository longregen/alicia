package org.localforge.alicia.feature.assistant

import android.annotation.SuppressLint
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import org.localforge.alicia.core.domain.model.MessageRole
import org.localforge.alicia.core.domain.repository.ConversationRepository
import org.localforge.alicia.core.domain.repository.VotingRepository
import org.localforge.alicia.core.domain.model.*
import org.localforge.alicia.service.voice.VoiceController
import org.localforge.alicia.service.voice.VoiceState as ServiceVoiceState
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.launch
import java.util.UUID
import javax.inject.Inject

enum class InputMode {
    Voice,
    Text
}

@HiltViewModel
class AssistantViewModel @Inject constructor(
    private val conversationRepository: ConversationRepository,
    private val voiceController: VoiceController,
    private val branchStore: BranchStore,
    private val votingRepository: VotingRepository
) : ViewModel() {

    private val _messages = MutableStateFlow<List<org.localforge.alicia.core.domain.model.Message>>(emptyList())
    val messages: StateFlow<List<org.localforge.alicia.core.domain.model.Message>> = _messages.asStateFlow()

    // Expose branch states for UI
    val branchStates: StateFlow<Map<String, MessageBranchState>> = branchStore.branchStates

    // Map VoiceController's service VoiceState to domain VoiceState
    val voiceState: StateFlow<VoiceState> = voiceController.currentState.map { serviceState ->
        when (serviceState) {
            is ServiceVoiceState.Idle -> VoiceState.IDLE
            is ServiceVoiceState.ListeningForWakeWord -> VoiceState.LISTENING_FOR_WAKE_WORD
            is ServiceVoiceState.Activated -> VoiceState.ACTIVATED
            is ServiceVoiceState.Listening -> VoiceState.LISTENING
            is ServiceVoiceState.Processing -> VoiceState.PROCESSING
            is ServiceVoiceState.Speaking -> VoiceState.SPEAKING
            is ServiceVoiceState.Error -> VoiceState.ERROR
            is ServiceVoiceState.Connecting -> VoiceState.CONNECTING
            is ServiceVoiceState.Disconnected -> VoiceState.DISCONNECTED
        }
    }.stateIn(viewModelScope, SharingStarted.Eagerly, VoiceState.IDLE)
    val currentTranscription: StateFlow<String> = voiceController.currentTranscription
    val streamingSentences: StateFlow<Map<Int, String>> = voiceController.streamingSentences
    val isGenerating: StateFlow<Boolean> = voiceController.isGenerating

    private val _inputMode = MutableStateFlow(InputMode.Text)
    val inputMode: StateFlow<InputMode> = _inputMode.asStateFlow()

    private val _textInput = MutableStateFlow("")
    val textInput: StateFlow<String> = _textInput.asStateFlow()

    private val _isSendingMessage = MutableStateFlow(false)
    val isSendingMessage: StateFlow<Boolean> = _isSendingMessage.asStateFlow()

    private val _textMessageError = MutableStateFlow<String?>(null)
    val textMessageError: StateFlow<String?> = _textMessageError.asStateFlow()

    // Expose VoiceController's protocol message states directly
    val errors: StateFlow<List<org.localforge.alicia.core.domain.model.ErrorMessage>> = voiceController.errors
    val reasoningSteps: StateFlow<List<ReasoningStep>> = voiceController.reasoningSteps
    val toolUsages: StateFlow<List<ToolUsage>> = voiceController.toolUsages
    val memoryTraces: StateFlow<List<MemoryTrace>> = voiceController.memoryTraces
    val commentaries: StateFlow<List<Commentary>> = voiceController.commentaries

    private var currentConversationId: String? = null

    // Track which messages we've already fetched siblings for
    private val fetchedSiblingsFor = mutableSetOf<String>()

    init {
        // Load initial conversation
        loadCurrentConversation()
    }

    @OptIn(kotlinx.coroutines.ExperimentalCoroutinesApi::class)
    private fun loadCurrentConversation() {
        viewModelScope.launch {
            conversationRepository.getAllConversations()
                .flatMapLatest { conversations ->
                    if (conversations.isEmpty()) {
                        // Create a new conversation
                        val result = conversationRepository.createConversation()
                        result.onSuccess { conversation ->
                            currentConversationId = conversation.id
                        }
                        emptyFlow()
                    } else {
                        // Use the most recent conversation and load its messages
                        val recentConversation = conversations.first()
                        currentConversationId = recentConversation.id
                        conversationRepository.getMessagesForConversation(recentConversation.id)
                    }
                }
                .collect { domainMessages ->
                    _messages.value = domainMessages
                    // Fetch siblings for messages that might have branches
                    fetchSiblingsForMessages(domainMessages)
                }
        }
    }

    fun loadSpecificConversation(conversationId: String) {
        currentConversationId = conversationId
        // Clear previous siblings data when switching conversations
        branchStore.clearAll()
        fetchedSiblingsFor.clear()

        viewModelScope.launch {
            // Load messages for this specific conversation
            conversationRepository.getMessagesForConversation(conversationId)
                .collect { domainMessages ->
                    _messages.value = domainMessages
                    // Fetch siblings for messages that might have branches
                    fetchSiblingsForMessages(domainMessages)
                }
        }
    }

    /**
     * Fetch siblings for messages that might have branches.
     * This is called when messages are loaded to populate branch navigation.
     * Only fetches for messages we haven't already fetched siblings for.
     */
    private fun fetchSiblingsForMessages(messages: List<org.localforge.alicia.core.domain.model.Message>) {
        viewModelScope.launch {
            for (message in messages) {
                // Skip if we've already fetched siblings for this message
                if (fetchedSiblingsFor.contains(message.id)) continue

                // Mark as fetched to avoid duplicate requests
                fetchedSiblingsFor.add(message.id)

                // Fetch siblings from backend
                fetchSiblings(message.id)
            }
        }
    }

    /**
     * Fetch siblings for a specific message from the backend.
     * Updates the BranchStore with the results.
     */
    fun fetchSiblings(messageId: String) {
        viewModelScope.launch {
            branchStore.setLoading(messageId, true)

            val result = conversationRepository.getMessageSiblings(messageId)

            result.onSuccess { siblings ->
                // Convert domain messages to SiblingMessage
                val siblingMessages = siblings.map { msg ->
                    SiblingMessage(
                        id = msg.id,
                        content = msg.content,
                        createdAt = msg.createdAt,
                        role = msg.role.value,
                        sequenceNumber = msg.sequenceNumber ?: 0
                    )
                }

                // Only update store if there are siblings (including the message itself)
                if (siblingMessages.isNotEmpty()) {
                    branchStore.updateSiblingsFromServer(
                        messageId = messageId,
                        siblings = siblingMessages,
                        activeMessageId = messageId // The current message is active
                    )
                }
            }.onFailure { error ->
                android.util.Log.w("AssistantViewModel", "Failed to fetch siblings for $messageId: ${error.message}")
                branchStore.setError(messageId, error.message)
            }
        }
    }

    /**
     * Navigate to a different branch (sibling message).
     * This calls the backend to switch the conversation's active branch,
     * then reloads the messages.
     *
     * @param messageId The current message ID
     * @param direction Navigation direction (PREV or NEXT)
     */
    fun navigateBranch(messageId: String, direction: BranchDirection) {
        val conversationId = currentConversationId ?: return

        // Get the target sibling without changing local state yet
        val targetSibling = branchStore.peekNavigationTarget(messageId, direction) ?: return

        viewModelScope.launch {
            android.util.Log.i("AssistantViewModel", "Switching branch from $messageId to ${targetSibling.id}")

            // Call backend to switch branch
            val result = conversationRepository.switchBranch(conversationId, targetSibling.id)

            result.onSuccess {
                // Update local state to reflect the change
                branchStore.setActiveSibling(messageId, targetSibling.id)

                // Reload messages to get the new chain
                // The Flow collection in loadCurrentConversation/loadSpecificConversation
                // will automatically update when the backend returns new data
                reloadMessages(conversationId)

                android.util.Log.i("AssistantViewModel", "Branch switch successful, reloading messages")
            }.onFailure { error ->
                android.util.Log.e("AssistantViewModel", "Failed to switch branch: ${error.message}")
                _textMessageError.value = "Failed to switch branch: ${error.message}"
            }
        }
    }

    /**
     * Reload messages for the current conversation.
     * Called after branch switching to get the updated message chain.
     * Uses first() to get a single snapshot rather than continuous subscription.
     */
    private suspend fun reloadMessages(conversationId: String) {
        try {
            // Get the latest messages snapshot (one-shot, not continuous subscription)
            val domainMessages = conversationRepository.getMessagesForConversation(conversationId)
                .first()
            _messages.value = domainMessages
            // Note: We don't re-fetch siblings here as the existing data is still valid
            // The sibling list doesn't change, only which one is active
        } catch (e: Exception) {
            android.util.Log.e("AssistantViewModel", "Failed to reload messages: ${e.message}")
        }
    }

    /**
     * Toggle voice activation based on current state.
     * - IDLE or LISTENING_FOR_WAKE_WORD: Activates listening
     * - LISTENING: Deactivates
     * - Other states (Processing, Speaking, Activated): No action (ignored)
     */
    @SuppressLint("MissingPermission")
    fun toggleListening() {
        viewModelScope.launch {
            when (voiceController.currentState.value) {
                ServiceVoiceState.Idle, ServiceVoiceState.ListeningForWakeWord -> {
                    voiceController.activate()
                }
                ServiceVoiceState.Listening -> {
                    voiceController.deactivate()
                }
                else -> {
                    // Do nothing during processing or speaking
                }
            }
        }
    }

    /**
     * Creates a new conversation and clears the current message history.
     */
    fun startNewConversation() {
        viewModelScope.launch {
            val result = conversationRepository.createConversation()
            result.onSuccess { conversation ->
                currentConversationId = conversation.id
                _messages.value = emptyList()
                // Clear branch state for new conversation
                branchStore.clearAll()
                fetchedSiblingsFor.clear()
            }
        }
    }

    /**
     * Toggles between voice and text input modes.
     * When switching to text mode, deactivates voice controller.
     * When switching to voice mode, starts wake word detection.
     */
    @SuppressLint("MissingPermission")
    fun toggleInputMode() {
        viewModelScope.launch {
            _inputMode.value = when (_inputMode.value) {
                InputMode.Voice -> {
                    // Switching to text mode - deactivate voice
                    voiceController.deactivate()
                    InputMode.Text
                }
                InputMode.Text -> {
                    // Switching to voice mode - start wake word detection
                    voiceController.startWakeWordDetection()
                    InputMode.Voice
                }
            }
        }
    }

    fun updateTextInput(text: String) {
        _textInput.value = text
    }

    fun clearTextMessageError() {
        _textMessageError.value = null
    }

    /**
     * Sends the current text input as a user message to the conversation.
     * Creates a local message immediately and sends it to the server for processing.
     * The assistant's response is automatically saved by the repository.
     */
    fun sendTextMessage() {
        val content = _textInput.value.trim()
        if (content.isEmpty() || currentConversationId == null) {
            return
        }

        viewModelScope.launch {
            _isSendingMessage.value = true

            try {
                // Create and save user message locally first
                val userMessage = org.localforge.alicia.core.domain.model.Message(
                    id = UUID.randomUUID().toString(),
                    conversationId = currentConversationId!!,
                    role = MessageRole.USER,
                    content = content,
                    createdAt = System.currentTimeMillis(),
                    isVoice = false
                )
                conversationRepository.insertMessage(userMessage)

                // Clear input
                _textInput.value = ""

                // Send to server and get response
                val result = conversationRepository.sendTextMessage(currentConversationId!!, content)

                result.onSuccess { _ ->
                    // The assistant's response message is already saved by sendTextMessage
                    _textMessageError.value = null
                }.onFailure { exception ->
                    _textMessageError.value = exception.message ?: "Failed to send message"
                }
            } finally {
                _isSendingMessage.value = false
            }
        }
    }

    @SuppressLint("MissingPermission")
    fun muteVoice() {
        voiceController.mute()
    }

    @SuppressLint("MissingPermission")
    fun unmuteVoice() {
        voiceController.unmute()
    }

    /**
     * Stops the current assistant response generation.
     */
    fun stopGeneration() {
        voiceController.sendStop()
    }

    /**
     * Requests regeneration of the last assistant message.
     * Finds the most recent assistant message and sends a regenerate request for it.
     */
    fun regenerateResponse() {
        // Find the last assistant message
        val lastAssistantMessage = _messages.value
            .filter { it.role == MessageRole.ASSISTANT }
            .maxByOrNull { it.createdAt }

        lastAssistantMessage?.let { message ->
            voiceController.sendRegenerate(message.id)
        }
    }

    /**
     * Edits a user message and triggers a new assistant response.
     * The server will create a new branch with the edited message.
     * @param messageId The ID of the message to edit
     * @param newContent The new content for the message
     */
    fun editMessage(messageId: String, newContent: String) {
        // Send edit to server - server will create the branch
        voiceController.sendEdit(messageId, newContent)

        // After edit, we'll need to re-fetch siblings to include the new branch
        // Remove from fetched set so it gets re-fetched
        fetchedSiblingsFor.remove(messageId)
    }

    /**
     * @deprecated No longer used - siblings are fetched from server automatically.
     * Initialize branches for a message.
     */
    @Deprecated("Siblings are fetched from server automatically")
    fun initializeBranch(messageId: String, content: String) {
        // No-op - siblings are fetched from server, not initialized locally
    }

    /**
     * Submits feedback (vote) for an assistant message.
     * Uses the VotingRepository to send the vote to the backend.
     * @param messageId The ID of the message to vote on
     * @param isUpvote True for thumbs up, false for thumbs down
     */
    fun voteOnMessage(messageId: String, isUpvote: Boolean) {
        viewModelScope.launch {
            try {
                val vote = if (isUpvote) Vote.UP else Vote.DOWN
                votingRepository.voteOnMessage(messageId, vote)
                android.util.Log.i("AssistantViewModel", "Vote submitted for message $messageId: ${vote.value}")
            } catch (e: Exception) {
                android.util.Log.e("AssistantViewModel", "Failed to vote on message: ${e.message}")
            }
        }
    }

    /**
     * Submits feedback (vote) for a tool use.
     * @param toolUseId The ID of the tool use to vote on
     * @param isUpvote True for thumbs up, false for thumbs down
     */
    fun voteOnToolUse(toolUseId: String, isUpvote: Boolean) {
        viewModelScope.launch {
            try {
                val vote = if (isUpvote) Vote.UP else Vote.DOWN
                votingRepository.voteOnToolUse(toolUseId, vote)
                android.util.Log.i("AssistantViewModel", "Vote submitted for tool use $toolUseId: ${vote.value}")
            } catch (e: Exception) {
                android.util.Log.e("AssistantViewModel", "Failed to vote on tool use: ${e.message}")
            }
        }
    }

    override fun onCleared() {
        super.onCleared()
        viewModelScope.launch {
            voiceController.shutdown()
        }
    }
}
