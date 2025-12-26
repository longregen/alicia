package org.localforge.alicia.feature.assistant

import android.annotation.SuppressLint
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import org.localforge.alicia.core.domain.model.MessageRole
import org.localforge.alicia.core.domain.repository.ConversationRepository
import org.localforge.alicia.core.domain.model.*
import org.localforge.alicia.service.voice.VoiceController
import org.localforge.alicia.service.voice.VoiceState as ServiceVoiceState
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
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
    private val voiceController: VoiceController
) : ViewModel() {

    private val _messages = MutableStateFlow<List<org.localforge.alicia.core.domain.model.Message>>(emptyList())
    val messages: StateFlow<List<org.localforge.alicia.core.domain.model.Message>> = _messages.asStateFlow()

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
                }
        }
    }

    fun loadSpecificConversation(conversationId: String) {
        currentConversationId = conversationId

        viewModelScope.launch {
            // Load messages for this specific conversation
            conversationRepository.getMessagesForConversation(conversationId)
                .collect { domainMessages ->
                    _messages.value = domainMessages
                }
        }
    }

    /**
     * Toggles between listening and idle states for voice input.
     * Activates voice listening when idle or waiting for wake word, deactivates when already listening.
     * Does nothing during processing or speaking states.
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

    override fun onCleared() {
        super.onCleared()
        viewModelScope.launch {
            voiceController.shutdown()
        }
    }
}
