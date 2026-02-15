package com.alicia.assistant.service

import android.animation.AnimatorSet
import android.animation.ObjectAnimator
import android.app.assist.AssistContent
import android.app.assist.AssistStructure
import android.content.Context
import android.graphics.Bitmap
import android.graphics.Color
import android.os.Bundle
import android.service.voice.VoiceInteractionSession
import android.util.Log
import com.alicia.assistant.telemetry.AliciaTelemetry
import com.alicia.assistant.tools.ReadScreenExecutor
import io.opentelemetry.api.common.Attributes
import io.opentelemetry.api.trace.Span
import android.view.View
import android.view.animation.AccelerateDecelerateInterpolator
import android.widget.ImageButton
import android.widget.TextView
import com.alicia.assistant.R
import com.alicia.assistant.model.RecognitionResult
import com.alicia.assistant.storage.PreferencesManager
import kotlinx.coroutines.*
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.UUID

class AliciaInteractionSession(context: Context) : VoiceInteractionSession(context) {

    companion object {
        private const val TAG = "AliciaSession"
        private const val ERROR_DISMISS_DELAY_MS = 1500L
        private const val RESPONSE_DISMISS_DELAY_MS = 2500L
        private const val WAVE_RING_1_DURATION_MS = 1000L
        private const val WAVE_RING_2_DURATION_MS = 1400L

        @Volatile
        var activeSession: AliciaInteractionSession? = null
    }

    private val sessionScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var bluetoothAudioManager: BluetoothAudioManager
    private lateinit var voiceRecognitionManager: VoiceRecognitionManager
    private lateinit var vadDetector: SileroVadDetector
    private lateinit var ttsManager: TtsManager
    private lateinit var preferencesManager: PreferencesManager
    private val apiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)
    private val screenContextManager = ScreenContextManager()
    private var screenContext: String? = null
    private var ocrJob: Job? = null

    private var statusText: TextView? = null
    private var transcribedText: TextView? = null
    private var responseText: TextView? = null
    private var micButton: ImageButton? = null
    private var waveRing1: View? = null
    private var waveRing2: View? = null
    private var overlayRoot: View? = null

    internal var isListening = false
    private var isProcessing = false
    private var processingJob: Job? = null
    private var waveAnimator: AnimatorSet? = null
    private var sessionSpan: Span? = null

    override fun onCreate() {
        super.onCreate()
        bluetoothAudioManager = BluetoothAudioManager(context)
        voiceRecognitionManager = VoiceRecognitionManager(context, sessionScope, bluetoothAudioManager)
        vadDetector = SileroVadDetector.create(context)
        ttsManager = TtsManager(context, sessionScope)
        preferencesManager = PreferencesManager(context)
    }

    override fun onCreateContentView(): View {
        val view = layoutInflater.inflate(R.layout.voice_session_overlay, null)
        statusText = view.findViewById(R.id.statusText)
        transcribedText = view.findViewById(R.id.transcribedText)
        responseText = view.findViewById(R.id.responseText)
        micButton = view.findViewById(R.id.micButton)
        waveRing1 = view.findViewById(R.id.waveRing1)
        waveRing2 = view.findViewById(R.id.waveRing2)
        overlayRoot = view.findViewById(R.id.overlayRoot)

        micButton?.setOnClickListener {
            if (isListening) {
                stopListening()
            } else if (!isProcessing) {
                finish()
            }
        }

        overlayRoot?.setOnClickListener {
            if (!isProcessing) finish()
        }

        return view
    }

    override fun onShow(args: Bundle?, showFlags: Int) {
        super.onShow(args, showFlags)
        activeSession = this
        val voiceSessionId = UUID.randomUUID().toString()
        sessionSpan = AliciaTelemetry.startSpan("voice.interaction_session",
            Attributes.builder()
                .put("voice_session.id", voiceSessionId)
                .build()
        )
        screenContext = null
        startListening()
    }

    @Suppress("DEPRECATION")
    override fun onHandleAssist(
        data: Bundle?,
        structure: AssistStructure?,
        content: AssistContent?
    ) {
        val parts = mutableListOf<String>()
        val structureText = screenContextManager.extractFromStructure(structure)
        if (structureText.isNotBlank()) parts.add(structureText)
        val contentText = screenContextManager.extractFromContent(content)
        if (contentText.isNotBlank()) parts.add(contentText)
        if (parts.isNotEmpty()) {
            screenContext = parts.joinToString("\n")
            ReadScreenExecutor.updateScreenContent(screenContext!!)
            Log.d(TAG, "Captured screen context (${screenContext!!.length} chars)")
            sessionSpan?.let {
                AliciaTelemetry.addSpanEvent(it, "screen_context.captured",
                    Attributes.builder()
                        .put("screen_context.length", screenContext!!.length.toLong())
                        .build()
                )
            }
        }
    }

    override fun onHandleScreenshot(screenshot: Bitmap?) {
        if (screenshot == null) return
        ocrJob = sessionScope.launch {
            val ocrText = withContext(Dispatchers.IO) {
                screenContextManager.extractFromScreenshot(screenshot)
            }
            if (ocrText.isNotBlank()) {
                screenContext = buildString {
                    screenContext?.let { append(it).append("\n") }
                    append(ocrText)
                }
                ReadScreenExecutor.updateScreenContent(screenContext!!)
                Log.d(TAG, "Added OCR context (${ocrText.length} chars)")
            }
        }
    }

    private fun startListening() {
        isListening = true
        statusText?.text = context.getString(R.string.listening)
        transcribedText?.visibility = View.GONE
        responseText?.visibility = View.GONE

        micButton?.setBackgroundResource(R.drawable.mic_button_recording)
        micButton?.setColorFilter(Color.WHITE)
        startWaveAnimation()

        voiceRecognitionManager.startListeningWithVad(vadDetector) { result ->
            sessionScope.launch(Dispatchers.Main) {
                isListening = false
                isProcessing = true
                stopWaveAnimation()
                micButton?.isEnabled = false
                micButton?.setBackgroundResource(R.drawable.mic_button_bg)
                micButton?.clearColorFilter()
                micButton?.alpha = 0.5f
                statusText?.text = context.getString(R.string.processing)

                when (result) {
                    is RecognitionResult.Success -> processInput(result.text)
                    is RecognitionResult.Error -> {
                        Log.e(TAG, "Recognition error: ${result.reason}")
                        statusText?.text = context.getString(R.string.recognition_error)
                        delay(ERROR_DISMISS_DELAY_MS)
                        finish()
                    }
                }
            }
        }
    }

    internal fun stopListening() {
        isListening = false
        isProcessing = true
        stopWaveAnimation()
        micButton?.isEnabled = false
        micButton?.setBackgroundResource(R.drawable.mic_button_bg)
        micButton?.clearColorFilter()
        micButton?.alpha = 0.5f
        statusText?.text = context.getString(R.string.processing)
        voiceRecognitionManager.stopVadListeningEarly()
    }

    private fun processInput(text: String) {
        isProcessing = true
        transcribedText?.text = "\"$text\""
        transcribedText?.visibility = View.VISIBLE
        statusText?.text = context.getString(R.string.processing)

        sessionSpan?.let {
            AliciaTelemetry.addSpanEvent(it, "voice.transcription_complete",
                Attributes.builder()
                    .put("voice.text", text)
                    .build()
            )
        }

        processingJob = sessionScope.launch {
            ocrJob?.join()
            val capturedContext = screenContext
            sessionSpan?.let {
                AliciaTelemetry.addSpanEvent(it, "voice.processing_start")
            }

            val response = try {
                withContext(Dispatchers.IO) {
                    val dateFormat = SimpleDateFormat("MMM d, h:mm a", Locale.getDefault())
                    val title = "Voice ${dateFormat.format(Date())}"
                    val conversation = apiClient.createConversation(title)
                    val content = if (capturedContext != null) {
                        "[Screen content]\n$capturedContext\n[End screen content]\n\nUser: $text"
                    } else {
                        text
                    }
                    apiClient.sendMessageSync(conversation.id, content).assistantMessage.content
                }
            } catch (e: Exception) {
                Log.e(TAG, "Chat failed", e)
                "Sorry, I couldn't get a response right now."
            }

            sessionSpan?.let {
                AliciaTelemetry.addSpanEvent(it, "voice.response_received",
                    Attributes.builder()
                        .put("voice.response_length", response.length.toLong())
                        .build()
                )
            }

            responseText?.text = response
            responseText?.visibility = View.VISIBLE
            statusText?.visibility = View.GONE

            micButton?.isEnabled = true
            micButton?.alpha = 1f
            isProcessing = false

            val settings = withContext(Dispatchers.IO) {
                preferencesManager.getSettings()
            }

            if (settings.voiceFeedbackEnabled) {
                ttsManager.speak(response, settings.ttsSpeed) {
                    finish()
                }
            } else {
                delay(RESPONSE_DISMISS_DELAY_MS)
                finish()
            }
        }
    }

    private fun startWaveAnimation() {
        waveRing1?.visibility = View.VISIBLE
        waveRing2?.visibility = View.VISIBLE

        val pulse1 = ObjectAnimator.ofFloat(waveRing1, View.SCALE_X, 0.7f, 1f).apply {
            repeatCount = ObjectAnimator.INFINITE
            repeatMode = ObjectAnimator.REVERSE
            duration = WAVE_RING_1_DURATION_MS
        }
        val pulse1y = ObjectAnimator.ofFloat(waveRing1, View.SCALE_Y, 0.7f, 1f).apply {
            repeatCount = ObjectAnimator.INFINITE
            repeatMode = ObjectAnimator.REVERSE
            duration = WAVE_RING_1_DURATION_MS
        }
        val pulse2 = ObjectAnimator.ofFloat(waveRing2, View.SCALE_X, 0.8f, 1f).apply {
            repeatCount = ObjectAnimator.INFINITE
            repeatMode = ObjectAnimator.REVERSE
            duration = WAVE_RING_2_DURATION_MS
        }
        val pulse2y = ObjectAnimator.ofFloat(waveRing2, View.SCALE_Y, 0.8f, 1f).apply {
            repeatCount = ObjectAnimator.INFINITE
            repeatMode = ObjectAnimator.REVERSE
            duration = WAVE_RING_2_DURATION_MS
        }

        waveAnimator = AnimatorSet().apply {
            interpolator = AccelerateDecelerateInterpolator()
            playTogether(pulse1, pulse1y, pulse2, pulse2y)
            start()
        }
    }

    private fun stopWaveAnimation() {
        waveAnimator?.cancel()
        waveAnimator = null
        waveRing1?.visibility = View.GONE
        waveRing2?.visibility = View.GONE
    }

    override fun onHide() {
        super.onHide()
        activeSession = null
        processingJob?.cancel()
        stopWaveAnimation()
        if (isListening) {
            voiceRecognitionManager.cancelVadListening()
            isListening = false
        }
        ttsManager.stopPlayback()
        sessionSpan?.end()
        sessionSpan = null
    }

    override fun onDestroy() {
        sessionSpan?.end()
        sessionSpan = null
        voiceRecognitionManager.destroy()
        bluetoothAudioManager.release()
        vadDetector.close()
        ttsManager.destroy()
        screenContextManager.release()
        sessionScope.cancel()
        super.onDestroy()
    }
}
