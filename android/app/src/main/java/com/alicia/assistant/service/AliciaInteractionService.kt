package com.alicia.assistant.service

import android.content.Intent
import android.os.Build
import android.os.Bundle
import android.service.voice.VoiceInteractionService
import android.support.v4.media.session.MediaSessionCompat
import android.support.v4.media.session.PlaybackStateCompat
import android.util.Log
import android.view.KeyEvent
import com.alicia.assistant.model.AppSettings
import com.alicia.assistant.model.RecognitionResult
import com.alicia.assistant.storage.PreferencesManager
import com.alicia.assistant.telemetry.AliciaTelemetry
import io.opentelemetry.api.common.Attributes
import io.opentelemetry.api.trace.Span
import kotlinx.coroutines.*
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

class AliciaInteractionService : VoiceInteractionService() {

    companion object {
        private const val TAG = "AliciaVIS"
        private const val SHOW_WITH_ASSIST = 1
        private const val SHOW_WITH_SCREENSHOT = 2
        private const val BUTTON_DEBOUNCE_MS = 500L
        private const val BEEP_SETTLE_DELAY_MS = 200L

        private var instance: AliciaInteractionService? = null

        fun triggerAssistSession() {
            instance?.triggerSession() ?: Log.w(TAG, "Service not active, cannot trigger session")
        }
    }

    private enum class ToggleTalkState {
        IDLE,
        RECORDING,
        PROCESSING,
        SPEAKING
    }

    private var mediaSession: MediaSessionCompat? = null

    // Toggle-to-talk state
    private var toggleTalkState = ToggleTalkState.IDLE
    private var lastButtonPressMs = 0L
    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    // Lazy-initialized components for toggle-to-talk
    private var bluetoothAudioManager: BluetoothAudioManager? = null
    private var voiceRecognitionManager: VoiceRecognitionManager? = null
    private var ttsManager: TtsManager? = null
    private var audioFeedback: AudioFeedbackManager? = null
    private var preferencesManager: PreferencesManager? = null
    private var apiClient: AliciaApiClient? = null
    private var processingJob: Job? = null
    private var recordingJob: Job? = null
    private var toggleTalkSpan: Span? = null

    override fun onReady() {
        super.onReady()
        instance = this
        setupMediaSession()
        Log.d(TAG, "VoiceInteractionService ready")
    }

    private fun setupMediaSession() {
        mediaSession = MediaSessionCompat(this, "AliciaAssistant").apply {
            setCallback(object : MediaSessionCompat.Callback() {
                override fun onMediaButtonEvent(mediaButtonEvent: Intent): Boolean {
                    val keyEvent = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                        mediaButtonEvent.getParcelableExtra(Intent.EXTRA_KEY_EVENT, KeyEvent::class.java)
                    } else {
                        @Suppress("DEPRECATION")
                        mediaButtonEvent.getParcelableExtra(Intent.EXTRA_KEY_EVENT)
                    } ?: return super.onMediaButtonEvent(mediaButtonEvent)

                    if (keyEvent.action == KeyEvent.ACTION_UP) {
                        when (keyEvent.keyCode) {
                            KeyEvent.KEYCODE_HEADSETHOOK,
                            KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE -> {
                                handleButtonPress()
                                return true
                            }
                        }
                    }
                    return super.onMediaButtonEvent(mediaButtonEvent)
                }
            })

            val state = PlaybackStateCompat.Builder()
                .setActions(PlaybackStateCompat.ACTION_PLAY_PAUSE)
                .setState(PlaybackStateCompat.STATE_STOPPED, 0, 0f)
                .build()
            setPlaybackState(state)
            isActive = true
        }
    }

    private fun handleButtonPress() {
        // Debounce rapid presses
        val now = System.currentTimeMillis()
        if (now - lastButtonPressMs < BUTTON_DEBOUNCE_MS) {
            Log.d(TAG, "Button press debounced")
            return
        }
        lastButtonPressMs = now

        // If the overlay session is active, delegate the button press to it
        AliciaInteractionSession.activeSession?.let { session ->
            if (session.isListening) {
                Log.d(TAG, "Overlay active and listening, delegating stop to session")
                session.stopListening()
            }
            return
        }

        when (toggleTalkState) {
            ToggleTalkState.IDLE -> startToggleRecording()
            ToggleTalkState.RECORDING -> stopToggleRecording()
            ToggleTalkState.PROCESSING -> {
                Log.d(TAG, "Button press ignored during processing")
            }
            ToggleTalkState.SPEAKING -> {
                // Interrupt TTS and start a new recording
                Log.i(TAG, "Interrupting TTS, starting new recording")
                ttsManager?.stopPlayback()
                toggleTalkSpan?.end()
                toggleTalkSpan = null
                startToggleRecording()
            }
        }
    }

    private fun ensureComponentsInitialized() {
        if (bluetoothAudioManager == null) {
            bluetoothAudioManager = BluetoothAudioManager(this)
        }
        if (voiceRecognitionManager == null) {
            voiceRecognitionManager = VoiceRecognitionManager(this, serviceScope, bluetoothAudioManager)
        }
        if (ttsManager == null) {
            ttsManager = TtsManager(this, serviceScope)
        }
        if (audioFeedback == null) {
            audioFeedback = AudioFeedbackManager()
        }
        if (preferencesManager == null) {
            preferencesManager = PreferencesManager(this)
        }
        if (apiClient == null) {
            apiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)
        }
    }

    private fun startToggleRecording() {
        Log.i(TAG, "Toggle-to-talk: starting recording")
        ensureComponentsInitialized()

        toggleTalkSpan = AliciaTelemetry.startSpan("voice.toggle_to_talk",
            Attributes.builder()
                .put("voice.trigger", "bluetooth_button")
                .build()
        )

        VoiceAssistantService.pauseDetection()
        toggleTalkState = ToggleTalkState.RECORDING
        audioFeedback?.playStartListening()

        // Small delay after beep to avoid recording the beep itself
        recordingJob = serviceScope.launch {
            delay(BEEP_SETTLE_DELAY_MS)

            voiceRecognitionManager?.startListening { result ->
                serviceScope.launch(Dispatchers.Main) {
                    handleRecognitionResult(result)
                }
            }
        }
    }

    private fun stopToggleRecording() {
        Log.i(TAG, "Toggle-to-talk: stopping recording")
        toggleTalkState = ToggleTalkState.PROCESSING
        audioFeedback?.playStopListening()

        toggleTalkSpan?.let {
            AliciaTelemetry.addSpanEvent(it, "voice.recording_stopped")
        }

        voiceRecognitionManager?.stopListening()
    }

    private fun handleRecognitionResult(result: RecognitionResult) {
        when (result) {
            is RecognitionResult.Success -> {
                Log.i(TAG, "Toggle-to-talk: transcribed: ${result.text}")
                toggleTalkSpan?.let {
                    AliciaTelemetry.addSpanEvent(it, "voice.transcription_complete",
                        Attributes.builder()
                            .put("voice.text", result.text)
                            .build()
                    )
                }
                processToggleTalkInput(result.text)
            }
            is RecognitionResult.Error -> {
                Log.e(TAG, "Toggle-to-talk: recognition error: ${result.reason}")
                audioFeedback?.playError()
                resetToIdle()
            }
        }
    }

    private fun processToggleTalkInput(text: String) {
        processingJob = serviceScope.launch {
            val response = try {
                withContext(Dispatchers.IO) {
                    val dateFormat = SimpleDateFormat("MMM d, h:mm a", Locale.getDefault())
                    val title = "Voice ${dateFormat.format(Date())}"
                    val client = apiClient ?: throw IllegalStateException("API client not initialized")
                    val conversation = client.createConversation(title)
                    client.sendMessageSync(conversation.id, text).assistantMessage.content
                }
            } catch (e: Exception) {
                Log.e(TAG, "Toggle-to-talk: API call failed", e)
                toggleTalkSpan?.let { AliciaTelemetry.recordError(it, e) }
                audioFeedback?.playError()
                resetToIdle()
                return@launch
            }

            toggleTalkSpan?.let {
                AliciaTelemetry.addSpanEvent(it, "voice.response_received",
                    Attributes.builder()
                        .put("voice.response_length", response.length.toLong())
                        .build()
                )
            }

            val settings = withContext(Dispatchers.IO) {
                preferencesManager?.getSettings() ?: AppSettings()
            }

            toggleTalkState = ToggleTalkState.SPEAKING
            ttsManager?.speak(response, settings.ttsSpeed) {
                resetToIdle()
            } ?: resetToIdle()
        }
    }

    private fun resetToIdle() {
        toggleTalkState = ToggleTalkState.IDLE
        processingJob?.cancel()
        processingJob = null
        recordingJob?.cancel()
        recordingJob = null
        toggleTalkSpan?.end()
        toggleTalkSpan = null
        VoiceAssistantService.resumeDetection()
    }

    private fun triggerSession() {
        Log.d(TAG, "Triggering voice interaction session")
        showSession(Bundle(), SHOW_WITH_ASSIST or SHOW_WITH_SCREENSHOT)
    }

    override fun onShutdown() {
        instance = null
        processingJob?.cancel()
        recordingJob?.cancel()
        toggleTalkSpan?.end()
        toggleTalkSpan = null
        mediaSession?.apply {
            isActive = false
            release()
        }
        mediaSession = null
        voiceRecognitionManager?.destroy()
        bluetoothAudioManager?.release()
        ttsManager?.destroy()
        audioFeedback?.release()
        serviceScope.cancel()
        super.onShutdown()
    }
}
