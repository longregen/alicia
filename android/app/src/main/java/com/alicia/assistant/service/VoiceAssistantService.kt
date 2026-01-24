package com.alicia.assistant.service

import android.Manifest
import android.app.Notification
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.IBinder
import android.util.Log
import androidx.core.content.ContextCompat
import com.alicia.assistant.AliciaApplication
import com.alicia.assistant.MainActivity
import com.alicia.assistant.R
import com.alicia.assistant.model.VoskModelInfo
import com.alicia.assistant.storage.PreferencesManager
import com.google.gson.Gson
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import org.vosk.Model
import org.vosk.Recognizer
import org.vosk.android.RecognitionListener
import org.vosk.android.SpeechService
import java.io.File

class VoiceAssistantService : Service(), RecognitionListener {

    private var speechService: SpeechService? = null
    private var model: Model? = null
    @Volatile private var wakeWord: String = "alicia"
    private val serviceScope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private var consecutiveErrors = 0
    private lateinit var preferencesManager: PreferencesManager
    private var recognizer: Recognizer? = null
    private val gson = Gson()
    private val stateMutex = Mutex()
    @Volatile private var isRestarting = false

    private data class VoskHypothesis(val text: String?, val partial: String?)

    override fun onBind(intent: Intent?): IBinder? = null

    override fun attachBaseContext(newBase: Context) {
        super.attachBaseContext(newBase.createAttributionContext("wake_word"))
    }

    override fun onCreate() {
        super.onCreate()
        instance = this
        running = true
        Log.d(TAG, "onCreate: VoiceAssistantService created")
        preferencesManager = PreferencesManager(this)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        Log.i(TAG, "onStartCommand: starting wake word service (startId=$startId)")
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.RECORD_AUDIO)
            != PackageManager.PERMISSION_GRANTED
        ) {
            Log.e(TAG, "onStartCommand: RECORD_AUDIO not granted, cannot start foreground service")
            stopSelf()
            return START_NOT_STICKY
        }
        startForeground(NOTIFICATION_ID, buildNotification())
        serviceScope.launch {
            stateMutex.withLock {
                shutdownNativeResources()
            }
            initWakeWordDetection()
        }
        return START_STICKY
    }

    private fun buildNotification(): Notification {
        val launchIntent = Intent(this, MainActivity::class.java)
        val pendingIntent = PendingIntent.getActivity(
            this, 0, launchIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        return Notification.Builder(this, AliciaApplication.CHANNEL_ID)
            .setContentTitle(getString(R.string.service_notification_title))
            .setContentText(getString(R.string.service_notification_desc))
            .setSmallIcon(android.R.drawable.ic_btn_speak_now)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .build()
    }

    /**
     * Shuts down native Vosk resources in the correct order, waiting for the audio thread
     * to fully exit before releasing the recognizer and model. Must be called within [stateMutex].
     */
    private suspend fun shutdownNativeResources() {
        val hadSpeechService = speechService != null
        withContext(Dispatchers.Main) {
            speechService?.stop()
            speechService = null
        }
        if (hadSpeechService) {
            // The Vosk SpeechService audio thread may still be executing a recognition
            // callback when stop() returns. Allow it time to fully exit before we
            // release the native recognizer/model to avoid use-after-free.
            delay(NATIVE_SHUTDOWN_DELAY_MS)
        }
        recognizer?.close()
        recognizer = null
        model?.close()
        model = null
    }

    private suspend fun initWakeWordDetection() = stateMutex.withLock {
        Log.d(TAG, "initWakeWordDetection: beginning initialization")
        try {
            val settings = preferencesManager.getSettings()
            if (!settings.wakeWordEnabled) {
                Log.i(TAG, "initWakeWordDetection: wake word disabled, stopping service")
                withContext(Dispatchers.Main) { stopSelf() }
                return@withLock
            }
            wakeWord = settings.wakeWord.lowercase().trim()
            val modelInfo = VoskModelInfo.fromId(settings.voskModelId)
            val modelDir = File(filesDir, "vosk-models/${modelInfo.id}")
            Log.d(TAG, "initWakeWordDetection: wakeWord='$wakeWord', model='${modelInfo.id}'")

            if (modelInfo == VoskModelInfo.SMALL_EN_US) {
                Log.d(TAG, "initWakeWordDetection: waiting for bundled model extraction")
                val app = application as AliciaApplication
                withTimeout(30_000) { app.modelReady.first { it } }
            }

            if (!modelDir.exists() || modelDir.listFiles()?.isEmpty() != false) {
                Log.e(TAG, "initWakeWordDetection: model not downloaded: ${modelInfo.id}")
                stopSelf()
                return@withLock
            }

            model?.close()
            model = Model(modelDir.absolutePath)
            val grammar = "[\"${wakeWord}\", \"[unk]\"]"
            Log.d(TAG, "initWakeWordDetection: recognizer grammar=$grammar")
            recognizer = Recognizer(model, SAMPLE_RATE, grammar)

            withContext(Dispatchers.Main) {
                speechService = SpeechService(recognizer, SAMPLE_RATE)
                speechService?.startListening(this@VoiceAssistantService)
            }

            consecutiveErrors = 0
            Log.i(TAG, "initWakeWordDetection: ACTIVE — listening for '$wakeWord' (model=${modelInfo.id})")
        } catch (e: Exception) {
            Log.e(TAG, "initWakeWordDetection: FAILED", e)
            stopSelf()
        }
    }

    override fun onPartialResult(hypothesis: String?) {
        Log.v(TAG, "onPartialResult: $hypothesis")
        checkForWakeWord(hypothesis)
    }

    override fun onResult(hypothesis: String?) {
        Log.d(TAG, "onResult: $hypothesis")
        checkForWakeWord(hypothesis)
    }

    override fun onFinalResult(hypothesis: String?) {
        Log.d(TAG, "onFinalResult: $hypothesis")
        checkForWakeWord(hypothesis)
    }

    override fun onError(exception: Exception?) {
        Log.e(TAG, "onError: recognition error (consecutiveErrors=$consecutiveErrors)", exception)
        restartRecognition()
    }

    override fun onTimeout() {
        Log.w(TAG, "onTimeout: recognition timed out, restarting")
        restartRecognition()
    }

    private fun restartRecognition() {
        serviceScope.launch {
            val shouldRestart = stateMutex.withLock {
                consecutiveErrors++
                Log.w(TAG, "restartRecognition: attempt $consecutiveErrors/$MAX_RETRIES")
                if (consecutiveErrors >= MAX_RETRIES) {
                    Log.e(TAG, "restartRecognition: max retries reached, stopping service")
                    withContext(Dispatchers.Main) { stopSelf() }
                    return@withLock false
                }
                shutdownNativeResources()
                true
            }
            if (shouldRestart) {
                delay(ERROR_RESTART_DELAY_MS)
                initWakeWordDetection()
            }
        }
    }

    private fun checkForWakeWord(hypothesisStr: String?) {
        if (hypothesisStr == null) return
        try {
            val hypothesis = gson.fromJson(hypothesisStr, VoskHypothesis::class.java)
            val text = hypothesis.text ?: hypothesis.partial ?: ""
            if (text.isNotEmpty()) {
                Log.d(TAG, "checkForWakeWord: heard text='$text', looking for '$wakeWord'")
            }
            if (text.contains(wakeWord, ignoreCase = true)) {
                Log.i(TAG, "checkForWakeWord: *** WAKE WORD DETECTED *** text='$text'")
                onWakeWordDetected()
            }
        } catch (e: Exception) {
            Log.w(TAG, "checkForWakeWord: failed to parse hypothesis: $hypothesisStr", e)
        }
    }

    private fun onWakeWordDetected() {
        if (isRestarting) return
        isRestarting = true
        Log.i(TAG, "onWakeWordDetected: triggering assist session")
        serviceScope.launch {
            stateMutex.withLock {
                shutdownNativeResources()
                withContext(Dispatchers.Main) {
                    AliciaInteractionService.triggerAssistSession()
                }
            }
            delay(RESTART_DELAY_MS)
            isRestarting = false
            initWakeWordDetection()
        }
    }

    override fun onDestroy() {
        Log.i(TAG, "onDestroy: VoiceAssistantService shutting down")
        instance = null
        running = false
        super.onDestroy()
        serviceScope.cancel()
        speechService?.stop()
        speechService = null
        recognizer?.close()
        recognizer = null
        model?.close()
        model = null
        stopForeground(STOP_FOREGROUND_REMOVE)
    }

    companion object {
        private const val TAG = "WakeWordService"
        private const val NOTIFICATION_ID = 1001
        private const val SAMPLE_RATE = 16000.0f
        private const val MAX_RETRIES = 5
        private const val RESTART_DELAY_MS = 3000L
        private const val ERROR_RESTART_DELAY_MS = 1000L
        private const val NATIVE_SHUTDOWN_DELAY_MS = 200L

        @Volatile private var instance: VoiceAssistantService? = null
        @Volatile private var running = false
        @Volatile private var lastStartAttempt = 0L
        private const val START_DEBOUNCE_MS = 5000L

        fun ensureRunning(context: Context) {
            if (running) return
            val now = System.currentTimeMillis()
            if (now - lastStartAttempt < START_DEBOUNCE_MS) {
                Log.d(TAG, "ensureRunning: debounced (too soon since last attempt)")
                return
            }
            lastStartAttempt = now
            Log.d(TAG, "ensureRunning: starting wake word service")
            try {
                context.startForegroundService(Intent(context, VoiceAssistantService::class.java))
            } catch (e: Exception) {
                Log.e(TAG, "ensureRunning: failed to start service", e)
            }
        }

        fun pauseDetection() {
            instance?.speechService?.setPause(true)
        }

        fun resumeDetection() {
            instance?.speechService?.setPause(false)
        }

        fun stop(context: Context) {
            Log.d(TAG, "stop: stopping wake word service")
            context.stopService(Intent(context, VoiceAssistantService::class.java))
        }
    }
}
