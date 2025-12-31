package org.localforge.alicia.service.voice

import android.Manifest
import android.content.Context
import androidx.annotation.RequiresPermission
import timber.log.Timber
import dagger.hilt.android.qualifiers.ApplicationContext
import org.localforge.alicia.core.domain.model.*
import org.localforge.alicia.core.domain.repository.ConversationRepository
import org.localforge.alicia.core.domain.repository.SettingsRepository
import org.localforge.alicia.core.network.LiveKitManager
import org.localforge.alicia.core.network.protocol.Envelope
import org.localforge.alicia.core.network.protocol.MessageType
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicLong
import java.util.concurrent.atomic.AtomicReference
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Central coordinator for the voice assistant.
 * Manages the lifecycle and orchestrates interactions between:
 * - Wake word detection
 * - Audio capture/playback
 * - LiveKit connection
 * - State management
 */
@Singleton
class VoiceController @Inject constructor(
    @ApplicationContext context: Context,
    private val wakeWordDetector: WakeWordDetector,
    private val audioManager: AudioManager,
    private val liveKitManager: LiveKitManager,
    private val conversationRepository: ConversationRepository,
    private val settingsRepository: SettingsRepository
) {
    private val controllerScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    private val _currentState = MutableStateFlow<VoiceState>(VoiceState.Idle)
    val currentState: StateFlow<VoiceState> = _currentState.asStateFlow()

    // Protocol message state
    private val _currentTranscription = MutableStateFlow<String>("")
    val currentTranscription: StateFlow<String> = _currentTranscription.asStateFlow()

    private val _streamingSentences = MutableStateFlow<Map<Int, String>>(emptyMap())
    val streamingSentences: StateFlow<Map<Int, String>> = _streamingSentences.asStateFlow()

    private val _reasoningSteps = MutableStateFlow<List<ReasoningStep>>(emptyList())
    val reasoningSteps: StateFlow<List<ReasoningStep>> = _reasoningSteps.asStateFlow()

    private val _toolUsages = MutableStateFlow<List<ToolUsage>>(emptyList())
    val toolUsages: StateFlow<List<ToolUsage>> = _toolUsages.asStateFlow()

    private val _errors = MutableStateFlow<List<ErrorMessage>>(emptyList())
    val errors: StateFlow<List<ErrorMessage>> = _errors.asStateFlow()

    private val _memoryTraces = MutableStateFlow<List<MemoryTrace>>(emptyList())
    val memoryTraces: StateFlow<List<MemoryTrace>> = _memoryTraces.asStateFlow()

    private val _commentaries = MutableStateFlow<List<Commentary>>(emptyList())
    val commentaries: StateFlow<List<Commentary>> = _commentaries.asStateFlow()

    private val _isGenerating = MutableStateFlow<Boolean>(false)
    val isGenerating: StateFlow<Boolean> = _isGenerating.asStateFlow()

    private var conversationJob: Job? = null
    private var silenceDetectionJob: Job? = null

    private val isMuted = AtomicBoolean(false)
    private val lastAudioTime = AtomicLong(0L)
    private val currentConversationId = AtomicReference<String?>(null)
    private val clientStanzaId = AtomicInteger(0)

    /**
     * Start wake word detection.
     */
    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    suspend fun startWakeWordDetection() {
        updateState(VoiceState.ListeningForWakeWord)

        // Get configured wake word from preferences (default to ALICIA)
        val wakeWord = getConfiguredWakeWord()
        val sensitivity = getConfiguredSensitivity()

        wakeWordDetector.start(
            wakeWord = wakeWord,
            sensitivity = sensitivity,
            onDetected = {
                handleWakeWordDetected()
            }
        )
    }

    /**
     * Activate the voice assistant (manually or via wake word).
     * Permission is validated by VoiceService before calling this method.
     */
    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    suspend fun activate() {
        if (_currentState.value == VoiceState.Listening || _currentState.value == VoiceState.Processing) {
            Timber.w("Already active, ignoring activation request")
            return
        }

        handleWakeWordDetected()
    }

    /**
     * Deactivate the voice assistant and return to wake word listening.
     */
    suspend fun deactivate() {
        Timber.i("Deactivating voice assistant")

        conversationJob?.cancel()
        conversationJob = null

        silenceDetectionJob?.cancel()
        silenceDetectionJob = null

        audioManager.stopCapture()
        audioManager.stopPlayback()
        liveKitManager.disconnect()

        currentConversationId.set(null)

        // Clear protocol message state
        clearProtocolMessages()

        updateState(VoiceState.ListeningForWakeWord)
        wakeWordDetector.resume()
    }

    /**
     * Clear all protocol message state.
     */
    private fun clearProtocolMessages() {
        _currentTranscription.value = ""
        _streamingSentences.value = emptyMap()
        _reasoningSteps.value = emptyList()
        _toolUsages.value = emptyList()
        _errors.value = emptyList()
        _memoryTraces.value = emptyList()
        _commentaries.value = emptyList()
        _isGenerating.value = false
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    fun mute() {
        isMuted.set(true)
        audioManager.pauseCapture()
        Timber.i("Microphone muted")
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    fun unmute() {
        isMuted.set(false)
        audioManager.resumeCapture()
        Timber.i("Microphone unmuted")
    }

    /**
     * Send a stop control message to halt current generation.
     */
    fun sendStop() {
        val conversationId = currentConversationId.get() ?: run {
            Timber.w("Cannot send stop: no active conversation")
            return
        }

        try {
            val envelope = Envelope.create(
                stanzaId = generateClientStanzaId(),
                conversationId = conversationId,
                type = MessageType.CONTROL_STOP,
                body = org.localforge.alicia.core.network.protocol.bodies.ControlStopBody(
                    conversationId = conversationId,
                    targetId = null,
                    reason = "User requested stop",
                    stopType = org.localforge.alicia.core.network.protocol.bodies.StopType.ALL
                )
            )
            liveKitManager.sendData(envelope)
            Timber.i("Sent stop control message")
        } catch (e: Exception) {
            Timber.e(e, "Failed to send stop message")
        }
    }

    /**
     * Send a regenerate control message to request a variation of the last assistant message.
     * @param targetId ID of the message to regenerate
     */
    fun sendRegenerate(targetId: String) {
        val conversationId = currentConversationId.get() ?: run {
            Timber.w("Cannot send regenerate: no active conversation")
            return
        }

        try {
            val envelope = Envelope.create(
                stanzaId = generateClientStanzaId(),
                conversationId = conversationId,
                type = MessageType.CONTROL_VARIATION,
                body = org.localforge.alicia.core.network.protocol.bodies.ControlVariationBody(
                    conversationId = conversationId,
                    targetId = targetId,
                    mode = org.localforge.alicia.core.network.protocol.bodies.VariationType.REGENERATE,
                    newContent = null
                )
            )
            liveKitManager.sendData(envelope)
            Timber.i("Sent regenerate control message for target: $targetId")
        } catch (e: Exception) {
            Timber.e(e, "Failed to send regenerate message")
        }
    }

    /**
     * Shutdown the voice controller and release all resources.
     */
    suspend fun shutdown() {
        Timber.i("Shutting down voice controller")

        wakeWordDetector.stop()
        audioManager.stopCapture()
        audioManager.stopPlayback()
        liveKitManager.disconnect()

        conversationJob?.cancel()
        silenceDetectionJob?.cancel()

        updateState(VoiceState.Idle)
    }

    /**
     * Handle wake word detection event.
     * Permission is already validated by the wake word detector or activate() caller.
     */
    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    private fun handleWakeWordDetected() {
        Timber.i("Wake word detected!")

        updateState(VoiceState.Activated)
        wakeWordDetector.pause()

        // Start voice conversation
        conversationJob = controllerScope.launch {
            try {
                startVoiceConversation()
            } catch (e: Exception) {
                Timber.e(e, "Error in voice conversation")
                deactivate()
            }
        }
    }

    /**
     * Handle silence detection (user stopped speaking).
     */
    private fun handleSilenceDetected() {
        Timber.i("Silence detected")

        when (_currentState.value) {
            is VoiceState.Listening -> {
                // User finished speaking, wait for response
                updateState(VoiceState.Processing)
            }
            is VoiceState.Speaking -> {
                // Assistant finished speaking, wait for user
                // If silence continues, end conversation
                controllerScope.launch {
                    delay(END_CONVERSATION_TIMEOUT)
                    if (_currentState.value is VoiceState.Speaking) {
                        endConversation()
                    }
                }
            }
            else -> Unit // Ignore silence in other states
        }
    }

    /**
     * Start a voice conversation via LiveKit.
     * Permission is already validated by the wake word detector or activate() caller.
     */
    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    private suspend fun startVoiceConversation() {
        updateState(VoiceState.Listening)

        // Set up callbacks before connecting
        // Handle protocol messages
        liveKitManager.onDataReceived { envelope ->
            controllerScope.launch {
                handleProtocolMessage(envelope)
            }
        }

        try {
            // Get or create a conversation for this voice session
            val conversationId = currentConversationId.get() ?: run {
                val result = conversationRepository.createConversation(title = "Voice Conversation")
                if (result.isFailure) {
                    throw result.exceptionOrNull() ?: Exception("Failed to create conversation")
                }
                val conversation = result.getOrThrow()
                currentConversationId.set(conversation.id)
                conversation.id
            }

            // Get server URL from settings
            val serverUrl = settingsRepository.serverUrl.first()

            // Get LiveKit token from backend API
            val tokenResult = conversationRepository.getConversationToken(conversationId)
            if (tokenResult.isFailure) {
                throw tokenResult.exceptionOrNull() ?: Exception("Failed to get LiveKit token")
            }

            val tokenResponse = tokenResult.getOrThrow()
            val liveKitUrl = "wss://${serverUrl.removePrefix("http://").removePrefix("https://")}/livekit"

            Timber.i("Connecting to LiveKit: url=$liveKitUrl, room=${tokenResponse.roomName}")

            // Connect to LiveKit with real token
            liveKitManager.connect(liveKitUrl, tokenResponse.token, tokenResponse.roomName)

            // LiveKit automatically handles audio capture and publishing
            // Start silence detection
            startSilenceDetection()
        } catch (e: Exception) {
            Timber.e(e, "Failed to start voice conversation: ${e.message}")
            deactivate()
        }
    }

    /**
     * Start silence detection to determine when user stops speaking.
     */
    private fun startSilenceDetection() {
        lastAudioTime.set(System.currentTimeMillis())

        silenceDetectionJob = controllerScope.launch {
            // Infinite loop exits via coroutine cancellation when silenceDetectionJob is cancelled
            // (in deactivate() or shutdown() methods)
            while (true) {
                delay(SILENCE_CHECK_INTERVAL)

                val silenceDuration = System.currentTimeMillis() - lastAudioTime.get()

                if (silenceDuration > SILENCE_THRESHOLD && _currentState.value == VoiceState.Listening) {
                    handleSilenceDetected()
                }
            }
        }
    }

    /**
     * Handle protocol messages from LiveKit data channel.
     */
    private fun handleProtocolMessage(envelope: Envelope) {
        try {
            Timber.d("Received protocol message: type=${envelope.type}")

            when (envelope.type) {
                MessageType.TRANSCRIPTION -> handleTranscription(envelope)
                MessageType.ASSISTANT_SENTENCE -> handleAssistantSentence(envelope)
                MessageType.START_ANSWER -> handleStartAnswer(envelope)
                MessageType.USER_MESSAGE -> handleUserMessage(envelope)
                MessageType.ASSISTANT_MESSAGE -> handleAssistantMessage(envelope)
                MessageType.ERROR_MESSAGE -> handleErrorMessage(envelope)
                MessageType.REASONING_STEP -> handleReasoningStep(envelope)
                MessageType.TOOL_USE_REQUEST -> handleToolUseRequest(envelope)
                MessageType.TOOL_USE_RESULT -> handleToolUseResult(envelope)
                MessageType.ACKNOWLEDGEMENT -> handleAcknowledgement(envelope)
                MessageType.MEMORY_TRACE -> handleMemoryTrace(envelope)
                MessageType.COMMENTARY -> handleCommentary(envelope)
                MessageType.CONTROL_STOP -> handleControlStop(envelope)
                MessageType.CONTROL_VARIATION -> handleControlVariation(envelope)
                MessageType.CONFIGURATION -> handleConfiguration(envelope)
                MessageType.AUDIO_CHUNK -> handleAudioChunk(envelope)
                else -> Timber.d("Unknown message type: ${envelope.type}")
            }
        } catch (e: Exception) {
            Timber.e(e, "Error handling protocol message")
        }
    }

    private fun handleTranscription(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val text = body["text"] as? String ?: ""
        val isFinal = body["final"] as? Boolean ?: false

        _currentTranscription.value = text

        if (isFinal) {
            // Clear transcription after a delay
            controllerScope.launch {
                delay(1000)
                _currentTranscription.value = ""
            }
        }
    }

    private fun handleAssistantSentence(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val sequence = (body["sequence"] as? Number)?.toInt() ?: 0
        val text = body["text"] as? String ?: ""
        val isFinal = body["isFinal"] as? Boolean ?: false

        // Update streaming sentences map
        val current = _streamingSentences.value.toMutableMap()
        current[sequence] = text
        _streamingSentences.value = current

        if (isFinal) {
            // Clear streaming sentences and stop generating when complete
            controllerScope.launch {
                delay(500)
                _streamingSentences.value = emptyMap()
                _isGenerating.value = false
            }
        }
    }

    private fun handleStartAnswer(_envelope: Envelope) {
        // Clear streaming sentences when new answer starts
        _streamingSentences.value = emptyMap()
        _isGenerating.value = true
        updateState(VoiceState.Processing)
    }

    private fun handleUserMessage(_envelope: Envelope) {
        // User messages are typically already displayed via transcription
        Timber.d("User message received")
    }

    private fun handleAssistantMessage(_envelope: Envelope) {
        // Non-streaming assistant message
        Timber.d("Assistant message received")
    }

    private fun handleErrorMessage(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return

        val error = ErrorMessage(
            id = body["id"] as? String ?: "",
            conversationId = envelope.conversationId,
            code = (body["code"] as? Number)?.toInt() ?: 0,
            message = body["message"] as? String ?: "Unknown error",
            severity = Severity.fromInt((body["severity"] as? Number)?.toInt() ?: 0),
            recoverable = body["recoverable"] as? Boolean ?: true,
            originatingId = body["originatingId"] as? String
        )

        _errors.value = _errors.value + error
        Timber.e("Protocol error: ${error.message}")
    }

    private fun handleReasoningStep(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return

        val step = ReasoningStep(
            id = body["id"] as? String ?: "",
            messageId = body["messageId"] as? String ?: "",
            conversationId = envelope.conversationId,
            sequence = (body["sequence"] as? Number)?.toInt() ?: 0,
            content = body["content"] as? String ?: ""
        )

        // Add step if not already present
        if (_reasoningSteps.value.none { it.id == step.id }) {
            _reasoningSteps.value = (_reasoningSteps.value + step).sortedBy { it.sequence }
        }
    }

    private fun handleToolUseRequest(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return

        // Safe cast with type checking - Map<*, *> can contain any key-value pairs
        // We validate each parameter individually with safe casts, filtering out null values
        val parameters: Map<String, Any> = when (val params = body["parameters"]) {
            is Map<*, *> -> params.mapNotNull { (k, v) ->
                val key = k as? String ?: return@mapNotNull null
                v?.let { key to it }
            }.toMap()
            else -> emptyMap()
        }

        val request = ToolUseRequest(
            id = body["id"] as? String ?: "",
            messageId = body["messageId"] as? String ?: "",
            conversationId = envelope.conversationId,
            toolName = body["toolName"] as? String ?: "",
            parameters = parameters,
            execution = body["execution"] as? String ?: "server",
            timeoutMs = (body["timeoutMs"] as? Number)?.toInt()
        )

        _toolUsages.value = _toolUsages.value + ToolUsage(request, null)
    }

    private fun handleToolUseResult(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return

        val result = ToolUseResult(
            id = body["id"] as? String ?: "",
            requestId = body["requestId"] as? String ?: "",
            conversationId = envelope.conversationId,
            success = body["success"] as? Boolean ?: false,
            result = body["result"],
            errorCode = body["errorCode"] as? String,
            errorMessage = body["errorMessage"] as? String
        )

        // Update corresponding tool usage
        _toolUsages.value = _toolUsages.value.map { usage ->
            if (usage.request.id == result.requestId) {
                usage.copy(result = result)
            } else {
                usage
            }
        }
    }

    private fun handleAcknowledgement(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val success = body["success"] as? Boolean ?: false
        Timber.d("Acknowledgement received: success=$success")
    }

    private fun handleMemoryTrace(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return

        val trace = MemoryTrace(
            id = body["id"] as? String ?: "",
            messageId = body["messageId"] as? String ?: "",
            conversationId = envelope.conversationId,
            memoryId = body["memoryId"] as? String ?: "",
            content = body["content"] as? String ?: "",
            relevance = (body["relevance"] as? Number)?.toDouble() ?: 0.0
        )

        // Add trace if not already present
        if (_memoryTraces.value.none { it.id == trace.id }) {
            _memoryTraces.value = _memoryTraces.value + trace
        }
    }

    private fun handleCommentary(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return

        val commentary = Commentary(
            id = body["id"] as? String ?: "",
            messageId = body["messageId"] as? String ?: "",
            conversationId = envelope.conversationId,
            content = body["content"] as? String ?: "",
            commentaryType = body["commentaryType"] as? String
        )

        // Add commentary if not already present
        if (_commentaries.value.none { it.id == commentary.id }) {
            _commentaries.value = _commentaries.value + commentary
        }
    }

    private fun handleControlStop(_envelope: Envelope) {
        Timber.d("Control stop received")
        // Stop current processing/generation
        _isGenerating.value = false
        updateState(VoiceState.Listening)
    }

    private fun handleControlVariation(_envelope: Envelope) {
        Timber.d("Control variation received")
        // Handle regeneration/variation requests
        _isGenerating.value = true
    }

    private fun handleConfiguration(_envelope: Envelope) {
        Timber.d("Configuration received")
        // Handle configuration updates
    }

    private fun handleAudioChunk(_envelope: Envelope) {
        // Audio chunks are typically handled via LiveKit audio tracks
        Timber.d("Audio chunk received (handled by LiveKit)")
    }

    /**
     * End the current conversation and return to wake word listening.
     */
    private suspend fun endConversation() {
        Timber.i("Ending conversation")
        deactivate()
    }

    private fun updateState(newState: VoiceState) {
        Timber.d("State transition: ${_currentState.value::class.simpleName} -> ${newState::class.simpleName}")
        _currentState.value = newState
    }

    /**
     * Get configured wake word from user preferences.
     * Reads from SettingsRepository and maps string to WakeWord enum.
     * Falls back to ALICIA if configured value is invalid.
     */
    private suspend fun getConfiguredWakeWord(): WakeWordDetector.WakeWord {
        val wakeWordName = settingsRepository.wakeWord.first()
        return try {
            WakeWordDetector.WakeWord.valueOf(wakeWordName.uppercase())
        } catch (e: IllegalArgumentException) {
            Timber.w("Invalid wake word configured: $wakeWordName, using default ALICIA")
            WakeWordDetector.WakeWord.ALICIA
        }
    }

    /**
     * Get configured sensitivity from user preferences.
     * Reads from SettingsRepository with range validation (0.0-1.0).
     * Falls back to 0.5f if configured value is out of range.
     */
    private suspend fun getConfiguredSensitivity(): Float {
        val sensitivity = settingsRepository.wakeWordSensitivity.first()
        return sensitivity.coerceIn(0.0f, 1.0f)
    }

    /**
     * Generate incrementing client stanza IDs (positive integers).
     */
    private fun generateClientStanzaId(): Int {
        return clientStanzaId.incrementAndGet()
    }

    companion object {
        // Silence detection configuration
        private const val SILENCE_THRESHOLD = 1500L // 1.5 seconds of silence
        private const val SILENCE_CHECK_INTERVAL = 200L // Check every 200ms
        private const val END_CONVERSATION_TIMEOUT = 3000L // End conversation after 3s of silence
    }
}

