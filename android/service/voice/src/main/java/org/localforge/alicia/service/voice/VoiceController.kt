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
import org.localforge.alicia.core.network.protocol.bodies.ConfigurationBody
import org.localforge.alicia.core.network.protocol.bodies.Features
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

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    suspend fun startWakeWordDetection() {
        updateState(VoiceState.ListeningForWakeWord)

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

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    suspend fun activate() {
        if (_currentState.value == VoiceState.Listening || _currentState.value == VoiceState.Processing) {
            Timber.w("Already active, ignoring activation request")
            return
        }

        handleWakeWordDetected()
    }

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

        clearProtocolMessages()

        updateState(VoiceState.ListeningForWakeWord)
        wakeWordDetector.resume()
    }

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

    fun sendStop() {
        val conversationId = currentConversationId.get() ?: run {
            Timber.w("Cannot send stop: no active conversation")
            return
        }

        try {
            val envelope = Envelope(
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
            throw e
        }
    }

    fun sendRegenerate(targetId: String) {
        val conversationId = currentConversationId.get() ?: run {
            Timber.w("Cannot send regenerate: no active conversation")
            return
        }

        try {
            val envelope = Envelope(
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
            throw e
        }
    }

    fun sendEdit(targetId: String, newContent: String) {
        val conversationId = currentConversationId.get() ?: run {
            Timber.w("Cannot send edit: no active conversation")
            return
        }

        try {
            val envelope = Envelope(
                stanzaId = generateClientStanzaId(),
                conversationId = conversationId,
                type = MessageType.CONTROL_VARIATION,
                body = org.localforge.alicia.core.network.protocol.bodies.ControlVariationBody(
                    conversationId = conversationId,
                    targetId = targetId,
                    mode = org.localforge.alicia.core.network.protocol.bodies.VariationType.EDIT,
                    newContent = newContent
                )
            )
            liveKitManager.sendData(envelope)
            Timber.i("Sent edit control message for target: $targetId")
        } catch (e: Exception) {
            Timber.e(e, "Failed to send edit message")
            throw e
        }
    }

    private fun sendInitialConfiguration(conversationId: String) {
        try {
            val envelope = Envelope(
                stanzaId = generateClientStanzaId(),
                conversationId = conversationId,
                type = MessageType.CONFIGURATION,
                body = ConfigurationBody(
                    conversationId = conversationId,
                    lastSequenceSeen = 0,
                    clientVersion = "android-1.0.0",
                    device = "android",
                    features = listOf(
                        Features.STREAMING,
                        Features.AUDIO_OUTPUT,
                        Features.PARTIAL_RESPONSES,
                        Features.REASONING_STEPS,
                        Features.TOOL_USE
                    )
                )
            )
            liveKitManager.sendData(envelope)
            Timber.i("Sent initial Configuration message")
        } catch (e: Exception) {
            Timber.e(e, "Failed to send initial configuration")
            throw e
        }
    }

    private fun sendConfigurationOnReconnect() {
        val conversationId = currentConversationId.get() ?: run {
            Timber.w("Cannot send configuration on reconnect: no active conversation")
            return
        }

        try {
            val lastSeenStanzaId = liveKitManager.getLastSeenStanzaId()
            val lastSequence = lastSeenStanzaId?.toIntOrNull()

            val envelope = Envelope(
                stanzaId = generateClientStanzaId(),
                conversationId = conversationId,
                type = MessageType.CONFIGURATION,
                body = ConfigurationBody(
                    conversationId = conversationId,
                    lastSequenceSeen = lastSequence,
                    clientVersion = "android-1.0.0",
                    device = "android",
                    features = listOf(
                        Features.STREAMING,
                        Features.AUDIO_OUTPUT,
                        Features.PARTIAL_RESPONSES,
                        Features.REASONING_STEPS,
                        Features.TOOL_USE
                    )
                )
            )
            liveKitManager.sendData(envelope)
            Timber.i("Sent Configuration on reconnect with lastSequenceSeen=$lastSequence")
        } catch (e: Exception) {
            Timber.e(e, "Failed to send configuration on reconnect")
            throw e
        }
    }

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

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    private fun handleWakeWordDetected() {
        Timber.i("Wake word detected!")

        updateState(VoiceState.Activated)
        wakeWordDetector.pause()

        conversationJob = controllerScope.launch {
            try {
                startVoiceConversation()
            } catch (e: Exception) {
                Timber.e(e, "Error in voice conversation")
                deactivate()
            }
        }
    }

    private fun handleSilenceDetected() {
        Timber.i("Silence detected")

        when (_currentState.value) {
            is VoiceState.Listening -> {
                updateState(VoiceState.Processing)
            }
            is VoiceState.Speaking -> {
                controllerScope.launch {
                    delay(END_CONVERSATION_TIMEOUT)
                    if (_currentState.value is VoiceState.Speaking) {
                        endConversation()
                    }
                }
            }
            else -> Unit
        }
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    private suspend fun startVoiceConversation() {
        updateState(VoiceState.Listening)

        liveKitManager.onDataReceived { envelope ->
            controllerScope.launch {
                handleProtocolMessage(envelope)
            }
        }

        liveKitManager.onReconnected {
            controllerScope.launch {
                sendConfigurationOnReconnect()
            }
        }

        liveKitManager.setAudioOutputEnabledCallback {
            kotlinx.coroutines.runBlocking {
                settingsRepository.audioOutputEnabled.first()
            }
        }

        try {
            val conversationId = currentConversationId.get() ?: run {
                val result = conversationRepository.createConversation(title = "Voice Conversation")
                if (result.isFailure) {
                    throw result.exceptionOrNull() ?: Exception("Failed to create conversation")
                }
                val conversation = result.getOrThrow()
                currentConversationId.set(conversation.id)
                conversation.id
            }

            val serverUrl = settingsRepository.serverUrl.first()

            val tokenResult = conversationRepository.getConversationToken(conversationId)
            if (tokenResult.isFailure) {
                throw tokenResult.exceptionOrNull() ?: Exception("Failed to get LiveKit token")
            }

            val tokenResponse = tokenResult.getOrThrow()
            val liveKitUrl = "wss://${serverUrl.removePrefix("http://").removePrefix("https://")}/livekit"

            Timber.i("Connecting to LiveKit: url=$liveKitUrl, room=${tokenResponse.roomName}")

            liveKitManager.connect(liveKitUrl, tokenResponse.token)

            sendInitialConfiguration(conversationId)

            startSilenceDetection()
        } catch (e: Exception) {
            Timber.e(e, "Failed to start voice conversation: ${e.message}")
            deactivate()
        }
    }

    private fun startSilenceDetection() {
        lastAudioTime.set(System.currentTimeMillis())

        silenceDetectionJob = controllerScope.launch {
            while (true) {
                delay(SILENCE_CHECK_INTERVAL)

                val silenceDuration = System.currentTimeMillis() - lastAudioTime.get()

                if (silenceDuration > SILENCE_THRESHOLD && _currentState.value == VoiceState.Listening) {
                    handleSilenceDetected()
                }
            }
        }
    }

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
                // Sync types (17-18)
                MessageType.SYNC_REQUEST -> handleSyncRequest(envelope)
                MessageType.SYNC_RESPONSE -> handleSyncResponse(envelope)
                // Feedback types (20-25)
                MessageType.FEEDBACK -> handleFeedback(envelope)
                MessageType.FEEDBACK_CONFIRMATION -> handleFeedbackConfirmation(envelope)
                MessageType.USER_NOTE -> handleUserNote(envelope)
                MessageType.NOTE_CONFIRMATION -> handleNoteConfirmation(envelope)
                MessageType.MEMORY_ACTION -> handleMemoryAction(envelope)
                MessageType.MEMORY_CONFIRMATION -> handleMemoryConfirmation(envelope)
                // Server info types (26-28)
                MessageType.SERVER_INFO -> handleServerInfo(envelope)
                MessageType.SESSION_STATS -> handleSessionStats(envelope)
                MessageType.CONVERSATION_UPDATE -> handleConversationUpdate(envelope)
                // Optimization types (29-32)
                MessageType.DIMENSION_PREFERENCE -> handleDimensionPreference(envelope)
                MessageType.ELITE_SELECT -> handleEliteSelect(envelope)
                MessageType.ELITE_OPTIONS -> handleEliteOptions(envelope)
                MessageType.OPTIMIZATION_PROGRESS -> handleOptimizationProgress(envelope)
                // Subscription types (40-43)
                MessageType.SUBSCRIBE -> handleSubscribe(envelope)
                MessageType.UNSUBSCRIBE -> handleUnsubscribe(envelope)
                MessageType.SUBSCRIBE_ACK -> handleSubscribeAck(envelope)
                MessageType.UNSUBSCRIBE_ACK -> handleUnsubscribeAck(envelope)
            }
        } catch (e: Exception) {
            Timber.e(e, "Error handling protocol message")
            throw e
        }
    }

    private fun handleTranscription(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val text = body["text"] as? String ?: ""
        val isFinal = body["final"] as? Boolean ?: false

        _currentTranscription.value = text

        if (isFinal) {
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

        val current = _streamingSentences.value.toMutableMap()
        current[sequence] = text
        _streamingSentences.value = current

        if (isFinal) {
            controllerScope.launch {
                delay(500)
                _streamingSentences.value = emptyMap()
                _isGenerating.value = false
            }
        }
    }

    private fun handleStartAnswer(_envelope: Envelope) {
        _streamingSentences.value = emptyMap()
        _isGenerating.value = true
        updateState(VoiceState.Processing)
    }

    private fun handleUserMessage(_envelope: Envelope) {
        Timber.d("User message received")
    }

    private fun handleAssistantMessage(_envelope: Envelope) {
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

        if (_reasoningSteps.value.none { it.id == step.id }) {
            _reasoningSteps.value = (_reasoningSteps.value + step).sortedBy { it.sequence }
        }
    }

    private fun handleToolUseRequest(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return

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

        if (_commentaries.value.none { it.id == commentary.id }) {
            _commentaries.value = _commentaries.value + commentary
        }
    }

    private fun handleControlStop(_envelope: Envelope) {
        Timber.d("Control stop received")
        _isGenerating.value = false
        updateState(VoiceState.Listening)
    }

    private fun handleControlVariation(_envelope: Envelope) {
        Timber.d("Control variation received")
        _isGenerating.value = true
    }

    private fun handleConfiguration(_envelope: Envelope) {
        Timber.d("Configuration received")
    }

    private fun handleAudioChunk(_envelope: Envelope) {
        Timber.d("Audio chunk received (handled by LiveKit)")
    }

    private fun handleSyncRequest(_envelope: Envelope) {
        Timber.d("Sync request received")
    }

    private fun handleSyncResponse(_envelope: Envelope) {
        Timber.d("Sync response received")
    }

    private fun handleFeedback(_envelope: Envelope) {
        Timber.d("Feedback received")
    }

    private fun handleFeedbackConfirmation(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val feedbackId = body["feedbackId"] as? String
        Timber.d("Feedback confirmation received for: $feedbackId")
    }

    private fun handleUserNote(_envelope: Envelope) {
        Timber.d("User note received")
    }

    private fun handleNoteConfirmation(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val noteId = body["noteId"] as? String
        val success = body["success"] as? Boolean ?: false
        Timber.d("Note confirmation received: noteId=$noteId, success=$success")
    }

    private fun handleMemoryAction(_envelope: Envelope) {
        Timber.d("Memory action received")
    }

    private fun handleMemoryConfirmation(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val memoryId = body["memoryId"] as? String
        val success = body["success"] as? Boolean ?: false
        Timber.d("Memory confirmation received: memoryId=$memoryId, success=$success")
    }

    private fun handleServerInfo(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        Timber.d("Server info received")
    }

    private fun handleSessionStats(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val messageCount = (body["messageCount"] as? Number)?.toInt() ?: 0
        val toolCallCount = (body["toolCallCount"] as? Number)?.toInt() ?: 0
        val memoriesUsed = (body["memoriesUsed"] as? Number)?.toInt() ?: 0
        Timber.d("Session stats: messages=$messageCount, tools=$toolCallCount, memories=$memoriesUsed")
    }

    private fun handleConversationUpdate(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val title = body["title"] as? String
        val status = body["status"] as? String
        Timber.d("Conversation update: title=$title, status=$status")
    }

    private fun handleDimensionPreference(_envelope: Envelope) {
        Timber.d("Dimension preference received")
    }

    private fun handleEliteSelect(_envelope: Envelope) {
        Timber.d("Elite select received")
    }

    private fun handleEliteOptions(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        @Suppress("UNCHECKED_CAST")
        val elites = body["elites"] as? List<*> ?: emptyList<Any>()
        Timber.d("Elite options received: ${elites.size} options")
    }

    private fun handleOptimizationProgress(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val iteration = (body["iteration"] as? Number)?.toInt() ?: 0
        val maxIterations = (body["maxIterations"] as? Number)?.toInt() ?: 0
        val status = body["status"] as? String
        Timber.d("Optimization progress: $iteration/$maxIterations, status=$status")
    }

    private fun handleSubscribe(_envelope: Envelope) {
        Timber.d("Subscribe request received")
    }

    private fun handleUnsubscribe(_envelope: Envelope) {
        Timber.d("Unsubscribe request received")
    }

    private fun handleSubscribeAck(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val success = body["success"] as? Boolean ?: false
        val missedMessages = (body["missedMessages"] as? Number)?.toInt()
        Timber.d("Subscribe ack: success=$success, missedMessages=$missedMessages")
    }

    private fun handleUnsubscribeAck(envelope: Envelope) {
        val body = envelope.body as? Map<*, *> ?: return
        val success = body["success"] as? Boolean ?: false
        Timber.d("Unsubscribe ack: success=$success")
    }

    private suspend fun endConversation() {
        Timber.i("Ending conversation")
        deactivate()
    }

    private fun updateState(newState: VoiceState) {
        Timber.d("State transition: ${_currentState.value::class.simpleName} -> ${newState::class.simpleName}")
        _currentState.value = newState
    }

    private suspend fun getConfiguredWakeWord(): WakeWordDetector.WakeWord {
        val wakeWordName = settingsRepository.wakeWord.first()
        return try {
            WakeWordDetector.WakeWord.valueOf(wakeWordName.uppercase())
        } catch (e: IllegalArgumentException) {
            Timber.w("Invalid wake word configured: $wakeWordName, using default ALICIA")
            WakeWordDetector.WakeWord.ALICIA
        }
    }

    private suspend fun getConfiguredSensitivity(): Float {
        val sensitivity = settingsRepository.wakeWordSensitivity.first()
        return sensitivity.coerceIn(0.0f, 1.0f)
    }

    private fun generateClientStanzaId(): Int {
        return clientStanzaId.incrementAndGet()
    }

    companion object {
        private const val SILENCE_THRESHOLD = 1500L
        private const val SILENCE_CHECK_INTERVAL = 200L
        private const val END_CONVERSATION_TIMEOUT = 3000L
    }
}

