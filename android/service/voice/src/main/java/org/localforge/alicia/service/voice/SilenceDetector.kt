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

class SilenceDetector(
    private val silenceThresholdMs: Long = DEFAULT_SILENCE_THRESHOLD_MS,
    private val energyThreshold: Float = DEFAULT_ENERGY_THRESHOLD,
    private val onSilenceDetected: () -> Unit
) {
    private val scope = CoroutineScope(Dispatchers.Default)
    private var monitorJob: Job? = null

    private var lastAudioActivityTime = System.currentTimeMillis()
    private var isCurrentlySilent = false

    fun processAudio(audioData: ByteArray) {
        val energy = calculateRMSEnergy(audioData)

        val hasAudioActivity = energy > energyThreshold

        if (hasAudioActivity) {
            lastAudioActivityTime = System.currentTimeMillis()
            isCurrentlySilent = false
        } else {
            val silenceDuration = System.currentTimeMillis() - lastAudioActivityTime

            if (silenceDuration > silenceThresholdMs && !isCurrentlySilent) {
                isCurrentlySilent = true
                onSilenceDetected()
                Timber.d("Silence detected after ${silenceDuration}ms")
            }
        }
    }

    fun start() {
        stop()

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

    fun stop() {
        monitorJob?.cancel()
        monitorJob = null
        isCurrentlySilent = false
        Timber.d("Silence detection stopped")
    }

    fun reset() {
        lastAudioActivityTime = System.currentTimeMillis()
        isCurrentlySilent = false
    }

    private fun calculateRMSEnergy(audioData: ByteArray): Float {
        if (audioData.isEmpty()) {
            return 0f
        }

        var sum = 0.0
        val sampleCount = audioData.size / 2

        for (i in 0 until sampleCount) {
            val sample = ((audioData[i * 2 + 1].toInt() shl 8) or (audioData[i * 2].toInt() and 0xFF)).toShort()
            sum += (sample * sample).toDouble()
        }

        val meanSquare = sum / sampleCount
        val rms = sqrt(meanSquare).toFloat()

        return rms
    }

    companion object {
        private const val DEFAULT_SILENCE_THRESHOLD_MS = 1500L
        private const val DEFAULT_ENERGY_THRESHOLD = 500f
        private const val CHECK_INTERVAL_MS = 200L
    }
}
