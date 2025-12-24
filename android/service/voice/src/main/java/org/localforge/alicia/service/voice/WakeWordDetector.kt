package org.localforge.alicia.service.voice

import ai.picovoice.porcupine.Porcupine
import ai.picovoice.porcupine.PorcupineException
import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.media.AudioFormat
import android.media.AudioRecord
import android.media.MediaRecorder
import androidx.annotation.RequiresPermission
import timber.log.Timber
import androidx.core.content.ContextCompat
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import java.util.concurrent.atomic.AtomicBoolean
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Wake word detector using Porcupine for on-device wake word detection.
 * Supports multiple built-in and custom wake words.
 */
@Singleton
class WakeWordDetector @Inject constructor(
    @ApplicationContext private val context: Context
) {
    private var porcupine: Porcupine? = null
    private var audioRecord: AudioRecord? = null
    private val isListening = AtomicBoolean(false)
    private val isPaused = AtomicBoolean(false)
    private var recordingThread: Thread? = null

    private var currentWakeWord: WakeWord? = null
    private var currentSensitivity: Float = DEFAULT_SENSITIVITY
    private var onDetectedCallback: (() -> Unit)? = null

    /**
     * Supported wake words.
     * ALICIA and HEY_ALICIA are custom-trained models.
     * JARVIS and COMPUTER are built-in Porcupine models.
     */
    enum class WakeWord(val displayName: String, val modelFileName: String) {
        ALICIA("Alicia", "alicia_android.ppn"),
        HEY_ALICIA("Hey Alicia", "hey_alicia_android.ppn"),
        JARVIS("Jarvis", "jarvis_android.ppn"),
        COMPUTER("Computer", "computer_android.ppn")
    }

    /**
     * Start wake word detection.
     *
     * @param wakeWord The wake word to detect
     * @param sensitivity Detection sensitivity (0.0 to 1.0). Higher = more sensitive but more false positives.
     * @param onDetected Callback when wake word is detected
     */
    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    suspend fun start(
        wakeWord: WakeWord,
        sensitivity: Float = DEFAULT_SENSITIVITY,
        onDetected: () -> Unit
    ) = withContext(Dispatchers.IO) {
        if (isListening.get()) {
            Timber.w("Wake word detector already running")
            return@withContext
        }

        if (!checkMicrophonePermission()) {
            throw SecurityException("Microphone permission not granted")
        }

        currentWakeWord = wakeWord
        currentSensitivity = sensitivity.coerceIn(0f, 1f)
        onDetectedCallback = onDetected

        try {
            initializePorcupine(wakeWord, currentSensitivity)
            startAudioCapture()
            isListening.set(true)
            isPaused.set(false)
            Timber.i("Wake word detection started for: ${wakeWord.displayName}")
        } catch (e: Exception) {
            Timber.e(e, "Failed to start wake word detection")
            cleanup()
            throw e
        }
    }

    /**
     * Stop wake word detection and release resources.
     */
    fun stop() {
        if (!isListening.get()) {
            return
        }

        isListening.set(false)
        isPaused.set(false)

        recordingThread?.interrupt()
        recordingThread?.join(1000)
        recordingThread = null

        cleanup()
        Timber.i("Wake word detection stopped")
    }

    /**
     * Pause wake word detection (keeps resources allocated).
     * Useful when user is actively speaking to the assistant.
     */
    fun pause() {
        if (!isListening.get()) {
            Timber.w("Cannot pause - not currently listening")
            return
        }

        isPaused.set(true)
        Timber.i("Wake word detection paused")
    }

    /**
     * Resume wake word detection after pause.
     */
    fun resume() {
        if (!isListening.get()) {
            Timber.w("Cannot resume - not currently listening")
            return
        }

        isPaused.set(false)
        Timber.i("Wake word detection resumed")
    }

    /**
     * Update detection sensitivity.
     *
     * @param sensitivity New sensitivity value (0.0 to 1.0)
     */
    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    suspend fun setSensitivity(sensitivity: Float) {
        currentSensitivity = sensitivity.coerceIn(0f, 1f)

        // If currently running, restart with new sensitivity
        if (isListening.get()) {
            val wakeWord = currentWakeWord ?: return
            val callback = onDetectedCallback ?: return
            stop()
            start(wakeWord, currentSensitivity, callback)
        }
    }

    private fun initializePorcupine(wakeWord: WakeWord, sensitivity: Float) {
        val accessKey = getAccessKey()
        val modelPath = getModelPath(wakeWord)

        porcupine = Porcupine.Builder()
            .setAccessKey(accessKey)
            .setKeywordPath(modelPath)
            .setSensitivity(sensitivity)
            .build(context)
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    private fun startAudioCapture() {
        val porcupineInstance = porcupine ?: throw IllegalStateException("Porcupine not initialized")

        val frameLength = porcupineInstance.frameLength
        val sampleRate = porcupineInstance.sampleRate
        val bufferSize = frameLength * 2

        audioRecord = AudioRecord(
            MediaRecorder.AudioSource.VOICE_RECOGNITION,
            sampleRate,
            AudioFormat.CHANNEL_IN_MONO,
            AudioFormat.ENCODING_PCM_16BIT,
            bufferSize
        )

        if (audioRecord?.state != AudioRecord.STATE_INITIALIZED) {
            throw IllegalStateException("Failed to initialize AudioRecord")
        }

        audioRecord?.startRecording()

        recordingThread = Thread {
            processAudioStream(porcupineInstance, frameLength)
        }.apply {
            name = "WakeWordDetector-AudioCapture"
            priority = Thread.MAX_PRIORITY
            start()
        }
    }

    private fun processAudioStream(porcupine: Porcupine, frameLength: Int) {
        val buffer = ShortArray(frameLength)

        while (isListening.get() && !Thread.currentThread().isInterrupted) {
            try {
                // Skip processing if paused
                if (isPaused.get()) {
                    Thread.sleep(100)
                    continue
                }

                val readResult = audioRecord?.read(buffer, 0, frameLength) ?: -1

                if (readResult < 0) {
                    Timber.e("AudioRecord read error: $readResult")
                    break
                }

                val keywordIndex = porcupine.process(buffer)

                if (keywordIndex >= 0) {
                    Timber.i("Wake word detected!")
                    onDetectedCallback?.invoke()

                    // Automatically pause after detection
                    pause()
                }
            } catch (e: PorcupineException) {
                Timber.e(e, "Porcupine processing error")
                break
            } catch (e: InterruptedException) {
                Timber.d("Audio processing thread interrupted")
                break
            } catch (e: Exception) {
                Timber.e(e, "Unexpected error in audio processing")
                break
            }
        }
    }

    private fun cleanup() {
        try {
            audioRecord?.stop()
            audioRecord?.release()
            audioRecord = null
        } catch (e: Exception) {
            Timber.e(e, "Error releasing AudioRecord")
        }

        try {
            porcupine?.delete()
            porcupine = null
        } catch (e: Exception) {
            Timber.e(e, "Error releasing Porcupine")
        }

        onDetectedCallback = null
    }

    private fun checkMicrophonePermission(): Boolean {
        return ContextCompat.checkSelfPermission(
            context,
            Manifest.permission.RECORD_AUDIO
        ) == PackageManager.PERMISSION_GRANTED
    }

    /**
     * Get Porcupine access key from BuildConfig.
     * The access key is configured via build.gradle.kts:
     * buildConfigField("String", "PORCUPINE_ACCESS_KEY", "\"your-key-here\"")
     *
     * @throws IllegalStateException if the access key is not configured
     */
    private fun getAccessKey(): String {
        // BuildConfig.PORCUPINE_ACCESS_KEY is generated at compile time by Android Gradle Plugin
        // based on the buildConfigField defined in build.gradle.kts
        return BuildConfig.PORCUPINE_ACCESS_KEY.ifEmpty {
            throw IllegalStateException(
                "Porcupine access key not configured. " +
                    "Please set PORCUPINE_ACCESS_KEY in build.gradle.kts buildConfigField."
            )
        }
    }

    private fun getModelPath(wakeWord: WakeWord): String {
        // Check if custom model exists in assets
        val modelPath = "models/${wakeWord.modelFileName}"

        return try {
            // Verify the model exists in assets
            context.assets.open(modelPath).use { }

            // Copy to internal storage for Porcupine to access
            val outputFile = File(context.filesDir, wakeWord.modelFileName)
            if (!outputFile.exists()) {
                context.assets.open(modelPath).use { input ->
                    outputFile.outputStream().use { output ->
                        input.copyTo(output)
                    }
                }
            }

            outputFile.absolutePath
        } catch (e: Exception) {
            Timber.w(e, "Custom model not found, using built-in: ${wakeWord.modelFileName}")

            // Fall back to built-in models if custom not available
            when (wakeWord) {
                WakeWord.JARVIS, WakeWord.COMPUTER -> {
                    // These are built-in Porcupine keywords
                    wakeWord.modelFileName
                }
                else -> {
                    throw IllegalArgumentException(
                        "Custom wake word model not found: ${wakeWord.modelFileName}. " +
                            "Please add model to assets/models/"
                    )
                }
            }
        }
    }

    companion object {
        private const val DEFAULT_SENSITIVITY = 0.5f
    }
}
