package org.localforge.alicia.feature.assistant

import android.annotation.SuppressLint
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import org.localforge.alicia.core.domain.model.MessageRole
import org.localforge.alicia.core.domain.repository.ConversationRepository
import org.localforge.alicia.core.domain.repository.NotesRepository
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
    private val votingRepository: VotingRepository,
    private val notesRepository: NotesRepository
) : ViewModel() {

    private val _messages = MutableStateFlow<List<org.localforge.alicia.core.domain.model.Message>>(emptyList())
    val messages: StateFlow<List<org.localforge.alicia.core.domain.model.Message>> = _messages.asStateFlow()

    val branchStates: StateFlow<Map<String, MessageBranchState>> = branchStore.branchStates

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

    val errors: StateFlow<List<org.localforge.alicia.core.domain.model.ErrorMessage>> = voiceController.errors
    val reasoningSteps: StateFlow<List<ReasoningStep>> = voiceController.reasoningSteps
    val toolUsages: StateFlow<List<ToolUsage>> = voiceController.toolUsages
    val memoryTraces: StateFlow<List<MemoryTrace>> = voiceController.memoryTraces
    val commentaries: StateFlow<List<Commentary>> = voiceController.commentaries

    private var currentConversationId: String? = null

    private val fetchedSiblingsFor = mutableSetOf<String>()

    init {
        loadCurrentConversation()
    }

    @OptIn(kotlinx.coroutines.ExperimentalCoroutinesApi::class)
    private fun loadCurrentConversation() {
        viewModelScope.launch {
            conversationRepository.getAllConversations()
                .flatMapLatest { conversations ->
                    if (conversations.isEmpty()) {
                        val result = conversationRepository.createConversation()
                        result.onSuccess { conversation ->
                            currentConversationId = conversation.id
                        }
                        emptyFlow()
                    } else {
                        val recentConversation = conversations.first()
                        currentConversationId = recentConversation.id
                        conversationRepository.getMessagesForConversation(recentConversation.id)
                    }
                }
                .collect { domainMessages ->
                    _messages.value = domainMessages
                    fetchSiblingsForMessages(domainMessages)
                }
        }
    }

    fun loadSpecificConversation(conversationId: String) {
        currentConversationId = conversationId
        branchStore.clearAll()
        fetchedSiblingsFor.clear()

        viewModelScope.launch {
            conversationRepository.getMessagesForConversation(conversationId)
                .collect { domainMessages ->
                    _messages.value = domainMessages
                    fetchSiblingsForMessages(domainMessages)
                }
        }
    }

    private fun fetchSiblingsForMessages(messages: List<org.localforge.alicia.core.domain.model.Message>) {
        viewModelScope.launch {
            for (message in messages) {
                if (fetchedSiblingsFor.contains(message.id)) continue
                fetchedSiblingsFor.add(message.id)
                fetchSiblings(message.id)
            }
        }
    }

    fun fetchSiblings(messageId: String) {
        viewModelScope.launch {
            branchStore.setLoading(messageId, true)

            val result = conversationRepository.getMessageSiblings(messageId)

            result.onSuccess { siblings ->
                val siblingMessages = siblings.map { msg ->
                    SiblingMessage(
                        id = msg.id,
                        content = msg.content,
                        createdAt = msg.createdAt,
                        role = msg.role.value,
                        sequenceNumber = msg.sequenceNumber ?: 0
                    )
                }

                if (siblingMessages.isNotEmpty()) {
                    branchStore.updateSiblingsFromServer(
                        messageId = messageId,
                        siblings = siblingMessages,
                        activeMessageId = messageId
                    )
                }
            }.onFailure { error ->
                branchStore.setError(messageId, error.message)
            }
        }
    }

    fun navigateBranch(messageId: String, direction: BranchDirection) {
        val conversationId = currentConversationId ?: return
        val targetSibling = branchStore.peekNavigationTarget(messageId, direction) ?: return

        viewModelScope.launch {
            val result = conversationRepository.switchBranch(conversationId, targetSibling.id)

            result.onSuccess {
                branchStore.setActiveSibling(messageId, targetSibling.id)
                reloadMessages(conversationId)
            }.onFailure { error ->
                _textMessageError.value = "Failed to switch branch: ${error.message}"
            }
        }
    }

    private suspend fun reloadMessages(conversationId: String) {
        try {
            val domainMessages = conversationRepository.getMessagesForConversation(conversationId)
                .first()
            _messages.value = domainMessages
        } catch (e: Exception) {
            _textMessageError.value = "Failed to reload messages: ${e.message}"
        }
    }

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
                else -> { }
            }
        }
    }

    fun startNewConversation() {
        viewModelScope.launch {
            val result = conversationRepository.createConversation()
            result.onSuccess { conversation ->
                currentConversationId = conversation.id
                _messages.value = emptyList()
                branchStore.clearAll()
                fetchedSiblingsFor.clear()
            }
        }
    }

    @SuppressLint("MissingPermission")
    fun toggleInputMode() {
        viewModelScope.launch {
            _inputMode.value = when (_inputMode.value) {
                InputMode.Voice -> {
                    voiceController.deactivate()
                    InputMode.Text
                }
                InputMode.Text -> {
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

    fun sendTextMessage() {
        val content = _textInput.value.trim()
        if (content.isEmpty() || currentConversationId == null) {
            return
        }

        viewModelScope.launch {
            _isSendingMessage.value = true

            try {
                val userMessage = org.localforge.alicia.core.domain.model.Message(
                    id = UUID.randomUUID().toString(),
                    conversationId = currentConversationId!!,
                    role = MessageRole.USER,
                    content = content,
                    createdAt = System.currentTimeMillis(),
                    isVoice = false
                )
                conversationRepository.insertMessage(userMessage)

                _textInput.value = ""

                val result = conversationRepository.sendTextMessage(currentConversationId!!, content)

                result.onSuccess { _ ->
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

    fun stopGeneration() {
        voiceController.sendStop()
    }

    fun regenerateResponse() {
        val lastAssistantMessage = _messages.value
            .filter { it.role == MessageRole.ASSISTANT }
            .maxByOrNull { it.createdAt }

        lastAssistantMessage?.let { message ->
            voiceController.sendRegenerate(message.id)
        }
    }

    fun editMessage(messageId: String, newContent: String) {
        voiceController.sendEdit(messageId, newContent)
        // Re-fetch siblings after edit to include the new branch
        fetchedSiblingsFor.remove(messageId)
    }

    fun voteOnMessage(messageId: String, isUpvote: Boolean) {
        viewModelScope.launch {
            try {
                val vote = if (isUpvote) Vote.UP else Vote.DOWN
                votingRepository.voteOnMessage(messageId, vote)
            } catch (e: Exception) {
                _textMessageError.value = "Failed to vote: ${e.message}"
            }
        }
    }

    fun voteOnToolUse(toolUseId: String, isUpvote: Boolean) {
        viewModelScope.launch {
            try {
                val vote = if (isUpvote) Vote.UP else Vote.DOWN
                votingRepository.voteOnToolUse(toolUseId, vote)
            } catch (e: Exception) {
                _textMessageError.value = "Failed to vote: ${e.message}"
            }
        }
    }

    private val _messageNotes = MutableStateFlow<Map<String, List<Note>>>(emptyMap())
    val messageNotes: StateFlow<Map<String, List<Note>>> = _messageNotes.asStateFlow()

    private val _notesLoading = MutableStateFlow<Set<String>>(emptySet())
    val notesLoading: StateFlow<Set<String>> = _notesLoading.asStateFlow()

    private val _notesError = MutableStateFlow<String?>(null)
    val notesError: StateFlow<String?> = _notesError.asStateFlow()

    private val _showNotesForMessage = MutableStateFlow<String?>(null)
    val showNotesForMessage: StateFlow<String?> = _showNotesForMessage.asStateFlow()

    fun openNotesForMessage(messageId: String) {
        _showNotesForMessage.value = messageId
        loadNotesForMessage(messageId)
    }

    fun closeNotes() {
        _showNotesForMessage.value = null
    }

    fun loadNotesForMessage(messageId: String) {
        viewModelScope.launch {
            _notesLoading.value = _notesLoading.value + messageId
            _notesError.value = null

            notesRepository.getMessageNotes(messageId)
                .onSuccess { notes ->
                    _messageNotes.value = _messageNotes.value + (messageId to notes)
                }
                .onFailure { e ->
                    _notesError.value = e.message ?: "Failed to load notes"
                }

            _notesLoading.value = _notesLoading.value - messageId
        }
    }

    fun addMessageNote(messageId: String, content: String, category: NoteCategory) {
        viewModelScope.launch {
            _notesLoading.value = _notesLoading.value + messageId
            _notesError.value = null

            notesRepository.createMessageNote(messageId, content, category)
                .onSuccess { note ->
                    // Add the new note to the local state
                    val currentNotes = _messageNotes.value[messageId] ?: emptyList()
                    _messageNotes.value = _messageNotes.value + (messageId to (currentNotes + note))
                }
                .onFailure { e ->
                    _notesError.value = e.message ?: "Failed to add note"
                }

            _notesLoading.value = _notesLoading.value - messageId
        }
    }

    fun updateNote(noteId: String, messageId: String, content: String) {
        viewModelScope.launch {
            _notesLoading.value = _notesLoading.value + messageId
            _notesError.value = null

            notesRepository.updateNote(noteId, content)
                .onSuccess { updatedNote ->
                    // Update the note in local state
                    val currentNotes = _messageNotes.value[messageId] ?: emptyList()
                    val updatedNotes = currentNotes.map { if (it.id == noteId) updatedNote else it }
                    _messageNotes.value = _messageNotes.value + (messageId to updatedNotes)
                }
                .onFailure { e ->
                    _notesError.value = e.message ?: "Failed to update note"
                }

            _notesLoading.value = _notesLoading.value - messageId
        }
    }

    fun deleteNote(noteId: String, messageId: String) {
        viewModelScope.launch {
            _notesLoading.value = _notesLoading.value + messageId
            _notesError.value = null

            notesRepository.deleteNote(noteId)
                .onSuccess {
                    // Remove the note from local state
                    val currentNotes = _messageNotes.value[messageId] ?: emptyList()
                    val updatedNotes = currentNotes.filter { it.id != noteId }
                    _messageNotes.value = _messageNotes.value + (messageId to updatedNotes)
                }
                .onFailure { e ->
                    _notesError.value = e.message ?: "Failed to delete note"
                }

            _notesLoading.value = _notesLoading.value - messageId
        }
    }

    fun clearNotesError() {
        _notesError.value = null
    }

    override fun onCleared() {
        super.onCleared()
        viewModelScope.launch {
            voiceController.shutdown()
        }
    }
}
