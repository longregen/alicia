package org.localforge.alicia.service.voice

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.pm.PackageManager
import android.media.AudioAttributes
import android.media.AudioFormat
import android.media.AudioManager as SystemAudioManager
import android.media.AudioRecord
import android.media.AudioTrack
import android.media.MediaRecorder
import android.media.audiofx.AcousticEchoCanceler
import android.media.audiofx.NoiseSuppressor
import androidx.annotation.RequiresPermission
import timber.log.Timber
import androidx.core.content.ContextCompat
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.nio.ByteBuffer
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicReference
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AudioManager @Inject constructor(
    @ApplicationContext private val context: Context
) {
    private val systemAudioManager = context.getSystemService(Context.AUDIO_SERVICE) as SystemAudioManager

    private var audioRecord: AudioRecord? = null
    private var audioTrack: AudioTrack? = null

    private var echoCanceler: AcousticEchoCanceler? = null
    private var noiseSuppressor: NoiseSuppressor? = null

    private val isCapturing = AtomicBoolean(false)
    private var captureThread: Thread? = null
    private val onAudioDataCallback = AtomicReference<((ByteArray) -> Unit)?>(null)

    private var hasAudioFocus = false
    private val audioFocusRequest by lazy { createAudioFocusRequest() }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    suspend fun startCapture(onAudioData: (ByteArray) -> Unit) = withContext(Dispatchers.IO) {
        if (isCapturing.get()) {
            Timber.w("Audio capture already running")
            return@withContext
        }

        if (!checkMicrophonePermission()) {
            throw SecurityException("Microphone permission not granted")
        }

        onAudioDataCallback.set(onAudioData)

        try {
            requestAudioFocus()
            initializeAudioRecord()
            enableAudioEffects()
            startRecording()

            isCapturing.set(true)
            Timber.i("Audio capture started")
        } catch (e: Exception) {
            Timber.e(e, "Failed to start audio capture")
            stopCapture()
            throw e
        }
    }

    fun stopCapture() {
        if (!isCapturing.get()) {
            return
        }

        isCapturing.set(false)

        captureThread?.interrupt()
        captureThread?.join(1000)
        captureThread = null

        disableAudioEffects()
        releaseAudioRecord()
        abandonAudioFocus()

        onAudioDataCallback.set(null)

        Timber.i("Audio capture stopped")
    }

    suspend fun playAudio(audioData: ByteArray) = withContext(Dispatchers.IO) {
        if (audioTrack == null) {
            initializeAudioTrack()
        }

        audioTrack?.let { track ->
            if (track.state == AudioTrack.STATE_INITIALIZED) {
                track.play()
                val written = track.write(audioData, 0, audioData.size)

                if (written < 0) {
                    throw IllegalStateException("Error writing to AudioTrack: $written")
                }
            }
        }
    }

    fun stopPlayback() {
        audioTrack?.stop()
        audioTrack?.flush()
        audioTrack?.release()
        audioTrack = null
        Timber.i("Audio playback stopped")
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    fun pauseCapture() {
        audioRecord?.stop()
        Timber.i("Audio capture paused")
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    fun resumeCapture() {
        audioRecord?.startRecording()
        Timber.i("Audio capture resumed")
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    private fun initializeAudioRecord() {
        val bufferSize = AudioRecord.getMinBufferSize(
            SAMPLE_RATE,
            CHANNEL_CONFIG,
            AUDIO_FORMAT
        ).coerceAtLeast(BUFFER_SIZE_BYTES)

        audioRecord = AudioRecord(
            MediaRecorder.AudioSource.VOICE_COMMUNICATION,
            SAMPLE_RATE,
            CHANNEL_CONFIG,
            AUDIO_FORMAT,
            bufferSize
        )

        if (audioRecord?.state != AudioRecord.STATE_INITIALIZED) {
            throw IllegalStateException("Failed to initialize AudioRecord")
        }
    }

    private fun initializeAudioTrack() {
        val bufferSize = AudioTrack.getMinBufferSize(
            PLAYBACK_SAMPLE_RATE,
            AudioFormat.CHANNEL_OUT_MONO,
            AUDIO_FORMAT
        ).coerceAtLeast(BUFFER_SIZE_BYTES)

        val audioAttributes = AudioAttributes.Builder()
            .setUsage(AudioAttributes.USAGE_ASSISTANT)
            .setContentType(AudioAttributes.CONTENT_TYPE_SPEECH)
            .build()

        val audioFormat = AudioFormat.Builder()
            .setSampleRate(PLAYBACK_SAMPLE_RATE)
            .setEncoding(AUDIO_FORMAT)
            .setChannelMask(AudioFormat.CHANNEL_OUT_MONO)
            .build()

        audioTrack = AudioTrack.Builder()
            .setAudioAttributes(audioAttributes)
            .setAudioFormat(audioFormat)
            .setBufferSizeInBytes(bufferSize)
            .setTransferMode(AudioTrack.MODE_STREAM)
            .build()

        if (audioTrack?.state != AudioTrack.STATE_INITIALIZED) {
            throw IllegalStateException("Failed to initialize AudioTrack")
        }
    }

    private fun enableAudioEffects() {
        val audioSessionId = audioRecord?.audioSessionId ?: return

        if (AcousticEchoCanceler.isAvailable()) {
            echoCanceler = AcousticEchoCanceler.create(audioSessionId)?.apply {
                enabled = true
                Timber.i("Acoustic echo canceler enabled")
            }
        }

        if (NoiseSuppressor.isAvailable()) {
            noiseSuppressor = NoiseSuppressor.create(audioSessionId)?.apply {
                enabled = true
                Timber.i("Noise suppressor enabled")
            }
        }
    }

    private fun disableAudioEffects() {
        echoCanceler?.release()
        echoCanceler = null

        noiseSuppressor?.release()
        noiseSuppressor = null

        Timber.i("Audio effects disabled")
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    private fun startRecording() {
        audioRecord?.startRecording()

        captureThread = Thread {
            processAudioCapture()
        }.apply {
            name = "AudioManager-Capture"
            priority = Thread.MAX_PRIORITY
            start()
        }
    }

    private fun processAudioCapture() {
        val shortBuffer = ShortArray(BUFFER_SIZE_BYTES / 2)

        while (isCapturing.get() && !Thread.currentThread().isInterrupted) {
            try {
                val readResult = audioRecord?.read(shortBuffer, 0, shortBuffer.size) ?: -1

                if (readResult < 0) {
                    Timber.e("AudioRecord read error: $readResult")
                    break
                }

                if (readResult > 0) {
                    // Convert short array to byte array
                    val byteBuffer = ByteBuffer.allocate(readResult * 2)
                    for (i in 0 until readResult) {
                        byteBuffer.putShort(shortBuffer[i])
                    }

                    val audioData = byteBuffer.array()
                    onAudioDataCallback.get()?.invoke(audioData)
                }
            } catch (e: InterruptedException) {
                Timber.d("Audio capture thread interrupted")
                break
            } catch (e: Exception) {
                Timber.e(e, "Error in audio capture loop")
                break
            }
        }
    }

    private fun releaseAudioRecord() {
        audioRecord?.stop()
        audioRecord?.release()
        audioRecord = null
    }

    private fun requestAudioFocus() {
        val result = systemAudioManager.requestAudioFocus(audioFocusRequest)
        hasAudioFocus = result == SystemAudioManager.AUDIOFOCUS_REQUEST_GRANTED
        Timber.i("Audio focus request result: ${if (hasAudioFocus) "granted" else "denied"}")
    }

    private fun abandonAudioFocus() {
        if (!hasAudioFocus) {
            return
        }

        systemAudioManager.abandonAudioFocusRequest(audioFocusRequest)
        hasAudioFocus = false
        Timber.i("Audio focus abandoned")
    }

    @SuppressLint("MissingPermission")
    private fun createAudioFocusRequest(): android.media.AudioFocusRequest {
        val audioAttributes = AudioAttributes.Builder()
            .setUsage(AudioAttributes.USAGE_ASSISTANT)
            .setContentType(AudioAttributes.CONTENT_TYPE_SPEECH)
            .build()

        return android.media.AudioFocusRequest.Builder(SystemAudioManager.AUDIOFOCUS_GAIN_TRANSIENT_EXCLUSIVE)
            .setAudioAttributes(audioAttributes)
            .setAcceptsDelayedFocusGain(false)
            .setWillPauseWhenDucked(true)
            .setOnAudioFocusChangeListener { focusChange ->
                Timber.d("Audio focus changed: $focusChange")
                when (focusChange) {
                    SystemAudioManager.AUDIOFOCUS_LOSS,
                    SystemAudioManager.AUDIOFOCUS_LOSS_TRANSIENT -> {
                        pauseCapture()
                    }
                    SystemAudioManager.AUDIOFOCUS_GAIN -> {
                        resumeCapture()
                    }
                }
            }
            .build()
    }

    private fun checkMicrophonePermission(): Boolean {
        return ContextCompat.checkSelfPermission(
            context,
            Manifest.permission.RECORD_AUDIO
        ) == PackageManager.PERMISSION_GRANTED
    }

    companion object {
        private const val SAMPLE_RATE = 16000
        private const val PLAYBACK_SAMPLE_RATE = 24000
        private const val CHANNEL_CONFIG = AudioFormat.CHANNEL_IN_MONO
        private const val AUDIO_FORMAT = AudioFormat.ENCODING_PCM_16BIT
        private const val BUFFER_SIZE_BYTES = 3200
    }
}
