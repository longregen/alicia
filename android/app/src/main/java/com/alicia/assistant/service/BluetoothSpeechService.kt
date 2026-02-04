package com.alicia.assistant.service

import android.media.AudioFormat
import android.media.AudioRecord
import android.media.MediaRecorder
import android.util.Log
import org.vosk.Recognizer
import org.vosk.android.RecognitionListener
import java.nio.ByteBuffer
import java.nio.ByteOrder
import java.util.concurrent.locks.ReentrantLock
import kotlin.concurrent.withLock
import kotlin.math.ceil

/**
 * Custom speech service that supports Bluetooth headset microphones.
 *
 * Unlike Vosk's SpeechService, this implementation:
 * - Uses VOICE_COMMUNICATION audio source (works with Bluetooth SCO)
 * - Handles 8kHz Bluetooth audio resampled to 16kHz for Vosk
 * - Integrates with BluetoothAudioManager for device routing
 *
 * Audio is fed directly to Vosk's Recognizer via acceptWaveForm().
 */
class BluetoothSpeechService(
    private val recognizer: Recognizer,
    private val sampleRate: Float,
    private val bluetoothAudioManager: BluetoothAudioManager
) {
    companion object {
        private const val TAG = "BluetoothSpeechService"
        private const val BUFFER_SIZE_SECONDS = 0.4f
    }

    // Lock for synchronizing start/stop operations to prevent race conditions
    private val stateLock = ReentrantLock()

    private var recorder: AudioRecord? = null
    private var recognitionThread: Thread? = null
    @Volatile
    private var listener: RecognitionListener? = null

    @Volatile
    private var paused = false

    @Volatile
    private var running = false

    private var actualSampleRate: Int = sampleRate.toInt()
    private var needsResampling = false
    @Volatile
    private var streamResampler: AudioResampler.StreamResampler? = null

    /**
     * Start listening for speech.
     */
    fun startListening(listener: RecognitionListener): Boolean = stateLock.withLock {
        // Don't start if already running
        if (running) {
            Log.w(TAG, "startListening called while already running")
            return false
        }

        this.listener = listener

        // Enable Bluetooth audio if available
        val btEnabled = bluetoothAudioManager.enableBluetoothAudio()
        actualSampleRate = bluetoothAudioManager.getEffectiveSampleRate(sampleRate.toInt())
        needsResampling = actualSampleRate != sampleRate.toInt()

        if (needsResampling) {
            streamResampler = AudioResampler.StreamResampler(actualSampleRate, sampleRate.toInt())
        }

        Log.d(TAG, "Starting listening: bluetooth=$btEnabled, actualRate=$actualSampleRate, needsResampling=$needsResampling")

        // Use VOICE_COMMUNICATION for Bluetooth, VOICE_RECOGNITION otherwise
        val audioSource = if (btEnabled) {
            MediaRecorder.AudioSource.VOICE_COMMUNICATION
        } else {
            MediaRecorder.AudioSource.VOICE_RECOGNITION
        }

        val bufferSize = calculateBufferSize()

        try {
            recorder = AudioRecord(
                audioSource,
                actualSampleRate,
                AudioFormat.CHANNEL_IN_MONO,
                AudioFormat.ENCODING_PCM_16BIT,
                bufferSize
            )

            if (recorder?.state != AudioRecord.STATE_INITIALIZED) {
                Log.e(TAG, "AudioRecord failed to initialize")
                recorder?.release()
                recorder = null
                bluetoothAudioManager.disableBluetoothAudio()
                return false
            }

            recorder?.startRecording()
            running = true
            paused = false

            recognitionThread = Thread(RecognitionRunnable(), "BluetoothSpeechService")
            recognitionThread?.start()

            return true
        } catch (e: Exception) {
            Log.e(TAG, "Failed to start recording", e)
            recorder?.release()
            recorder = null
            bluetoothAudioManager.disableBluetoothAudio()
            return false
        }
    }

    /**
     * Stop listening and release resources.
     */
    fun stop() {
        // Capture local references under lock to prevent concurrent stop() calls
        // from both operating on the same resources
        val thread: Thread?
        val rec: AudioRecord?
        val listenerToNotify: RecognitionListener?

        stateLock.withLock {
            // Check if already stopped
            if (!running && recognitionThread == null && recorder == null) {
                Log.d(TAG, "stop() called but already stopped")
                return
            }

            // Set running to false first to signal thread to stop
            running = false

            // Capture local references before nulling
            thread = recognitionThread
            rec = recorder
            // Capture listener for final result delivery BEFORE nulling it
            listenerToNotify = listener

            // Null out fields - subsequent stop() calls will see nulls and return early
            recognitionThread = null
            recorder = null
            listener = null
            streamResampler = null
        }

        // Now outside the lock, perform cleanup operations that may block

        // Interrupt thread and wait for it to finish
        thread?.let {
            it.interrupt()
            try {
                it.join(1000) // Wait up to 1 second for thread to finish
            } catch (e: InterruptedException) {
                Log.w(TAG, "Interrupted while waiting for recognition thread", e)
                Thread.currentThread().interrupt()
            }
        }

        // Now safe to release recorder
        try {
            rec?.stop()
        } catch (e: Exception) {
            Log.w(TAG, "Error stopping recorder", e)
        }
        rec?.release()

        bluetoothAudioManager.disableBluetoothAudio()

        // Deliver final result to the captured listener reference
        // This ensures the final result is delivered even when stop() is called
        try {
            listenerToNotify?.onFinalResult(recognizer.finalResult)
        } catch (e: Exception) {
            Log.w(TAG, "Error delivering final result", e)
        }
    }

    /**
     * Pause recognition temporarily (e.g., during TTS playback).
     */
    fun setPause(paused: Boolean) {
        this.paused = paused
    }

    private fun calculateBufferSize(): Int {
        val frameSize = (actualSampleRate * BUFFER_SIZE_SECONDS).toInt()
        val minBufferSize = AudioRecord.getMinBufferSize(
            actualSampleRate,
            AudioFormat.CHANNEL_IN_MONO,
            AudioFormat.ENCODING_PCM_16BIT
        )
        // Enforce minimum of 1024 bytes and maximum of 64KB to prevent memory issues
        val calculatedSize = maxOf(frameSize * 2, minBufferSize)
        return calculatedSize.coerceIn(1024, 65536)
    }

    private inner class RecognitionRunnable : Runnable {
        override fun run() {
            val targetFrameSize = (sampleRate * BUFFER_SIZE_SECONDS).toInt()
            val inputFrameSize = if (needsResampling) {
                (actualSampleRate * BUFFER_SIZE_SECONDS).toInt()
            } else {
                targetFrameSize
            }

            val inputBuffer = ShortArray(inputFrameSize)
            // Calculate output buffer size based on actual resampling ratio
            // Use ceil() to ensure we always have enough space for any partial reads
            // The resampler calculates: outputLength = (input.size * ratio).toInt()
            // Using ceil ensures we handle any rounding up that might occur
            val maxOutputFrameSize = if (needsResampling) {
                val ratio = sampleRate.toDouble() / actualSampleRate
                ceil(inputFrameSize * ratio).toInt()
            } else {
                targetFrameSize
            }
            val byteBuffer = ByteBuffer.allocate(maxOutputFrameSize * 2).order(ByteOrder.LITTLE_ENDIAN)

            var consecutiveReadFailures = 0
            val maxConsecutiveFailures = 50 // ~2 seconds at 40ms buffer intervals
            val warnAfterFailures = 10

            while (running && !Thread.interrupted()) {
                // Capture listener reference at start of iteration for thread safety
                // This ensures consistent behavior within the iteration even if stop() is called
                val currentListener = listener

                try {
                    if (paused) {
                        Thread.sleep(100)
                        continue
                    }

                    val read = recorder?.read(inputBuffer, 0, inputFrameSize) ?: -1
                    if (read <= 0) {
                        consecutiveReadFailures++
                        if (consecutiveReadFailures == warnAfterFailures) {
                            Log.w(TAG, "AudioRecord read returning $read repeatedly ($consecutiveReadFailures failures)")
                        }
                        if (consecutiveReadFailures >= maxConsecutiveFailures) {
                            Log.e(TAG, "AudioRecord read failed $consecutiveReadFailures consecutive times, stopping recognition")
                            currentListener?.onError(Exception("AudioRecord read failed repeatedly (recorder may be in bad state)"))
                            break
                        }
                        continue
                    }
                    // Reset counter on successful read
                    consecutiveReadFailures = 0

                    // Resample if needed (8kHz -> 16kHz)
                    val frame = if (needsResampling && streamResampler != null) {
                        val inputSlice = if (read < inputFrameSize) {
                            inputBuffer.copyOf(read)
                        } else {
                            inputBuffer
                        }
                        streamResampler!!.resampleFrame(inputSlice)
                    } else {
                        if (read < inputFrameSize) inputBuffer.copyOf(read) else inputBuffer
                    }

                    // Convert to bytes for Vosk
                    byteBuffer.clear()
                    val frameSamples = minOf(frame.size, maxOutputFrameSize)
                    for (i in 0 until frameSamples) {
                        byteBuffer.putShort(frame[i])
                    }
                    val bytes = byteBuffer.array()
                    val byteLen = frameSamples * 2

                    // Feed to Vosk recognizer
                    val isFinal = recognizer.acceptWaveForm(bytes, byteLen)

                    if (isFinal) {
                        val result = recognizer.result
                        currentListener?.onResult(result)
                    } else {
                        val partial = recognizer.partialResult
                        currentListener?.onPartialResult(partial)
                    }

                } catch (e: InterruptedException) {
                    Thread.currentThread().interrupt()
                    break
                } catch (e: Exception) {
                    Log.e(TAG, "Error in recognition loop", e)
                    currentListener?.onError(e)
                    break
                }
            }

            // Note: Final result delivery is handled by stop() to ensure it's always
            // delivered even when stop() is called concurrently. The stop() method
            // captures the listener reference before nulling it and calls onFinalResult().
        }
    }
}
