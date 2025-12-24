package org.localforge.alicia.service.voice

import timber.log.Timber
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlin.math.abs
import kotlin.math.sqrt

/**
 * Detects silence in audio streams to determine when user has stopped speaking.
 * Uses RMS (Root Mean Square) energy calculation to detect audio activity.
 */
class SilenceDetector(
    private val silenceThresholdMs: Long = DEFAULT_SILENCE_THRESHOLD_MS,
    private val energyThreshold: Float = DEFAULT_ENERGY_THRESHOLD,
    private val onSilenceDetected: () -> Unit
) {
    private val scope = CoroutineScope(Dispatchers.Default)
    private var monitorJob: Job? = null

    private var lastAudioActivityTime = System.currentTimeMillis()
    private var isCurrentlySilent = false

    /**
     * Process audio data and detect silence.
     *
     * @param audioData PCM audio data (16-bit samples)
     */
    fun processAudio(audioData: ByteArray) {
        val energy = calculateRMSEnergy(audioData)

        val hasAudioActivity = energy > energyThreshold

        if (hasAudioActivity) {
            lastAudioActivityTime = System.currentTimeMillis()
            isCurrentlySilent = false
        } else {
            // Check if silence duration exceeds threshold
            val silenceDuration = System.currentTimeMillis() - lastAudioActivityTime

            if (silenceDuration > silenceThresholdMs && !isCurrentlySilent) {
                isCurrentlySilent = true
                onSilenceDetected()
                Timber.d("Silence detected after ${silenceDuration}ms")
            }
        }
    }

    /**
     * Start monitoring for silence.
     */
    fun start() {
        stop() // Stop any existing monitoring

        lastAudioActivityTime = System.currentTimeMillis()
        isCurrentlySilent = false

        monitorJob = scope.launch {
            while (isActive) {
                val silenceDuration = System.currentTimeMillis() - lastAudioActivityTime

                if (silenceDuration > silenceThresholdMs && !isCurrentlySilent) {
                    isCurrentlySilent = true
                    onSilenceDetected()
                    Timber.d("Silence detected after ${silenceDuration}ms")
                }
                delay(CHECK_INTERVAL_MS)
            }
        }

        Timber.d("Silence detection started")
    }

    /**
     * Stop monitoring for silence.
     */
    fun stop() {
        monitorJob?.cancel()
        monitorJob = null
        isCurrentlySilent = false
        Timber.d("Silence detection stopped")
    }

    /**
     * Reset the silence timer.
     * Call this when you know there's audio activity.
     */
    fun reset() {
        lastAudioActivityTime = System.currentTimeMillis()
        isCurrentlySilent = false
    }


    /**
     * Calculate RMS (Root Mean Square) energy of audio samples.
     * This represents the "loudness" or energy level of the audio.
     *
     * @param audioData PCM audio data as byte array (16-bit samples)
     * @return RMS energy value
     */
    private fun calculateRMSEnergy(audioData: ByteArray): Float {
        if (audioData.isEmpty()) {
            return 0f
        }

        var sum = 0.0
        val sampleCount = audioData.size / 2 // 16-bit = 2 bytes per sample

        // Convert byte array to 16-bit samples and calculate sum of squares
        for (i in 0 until sampleCount) {
            val sample = ((audioData[i * 2 + 1].toInt() shl 8) or (audioData[i * 2].toInt() and 0xFF)).toShort()
            sum += (sample * sample).toDouble()
        }

        // Calculate RMS
        val meanSquare = sum / sampleCount
        val rms = sqrt(meanSquare).toFloat()

        return rms
    }

    companion object {
        // Default silence threshold: 1.5 seconds
        private const val DEFAULT_SILENCE_THRESHOLD_MS = 1500L

        // Default energy threshold (tune this based on testing)
        // Higher values = less sensitive (requires louder audio to be considered "non-silent")
        private const val DEFAULT_ENERGY_THRESHOLD = 500f

        // How often to check for silence
        private const val CHECK_INTERVAL_MS = 200L
    }
}
