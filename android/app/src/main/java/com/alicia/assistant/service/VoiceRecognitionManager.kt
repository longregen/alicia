package com.alicia.assistant.service

import android.content.Context
import android.media.AudioFormat
import android.media.AudioRecord
import android.media.MediaRecorder
import android.util.Log
import com.alicia.assistant.telemetry.AliciaTelemetry
import com.google.gson.Gson
import io.opentelemetry.api.common.Attributes
import io.opentelemetry.api.trace.Span
import kotlinx.coroutines.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.Request
import okhttp3.RequestBody.Companion.asRequestBody
import com.alicia.assistant.model.ErrorReason
import com.alicia.assistant.model.RecognitionResult
import com.alicia.assistant.model.TimestampedWord
import com.alicia.assistant.model.VerboseTranscription
import java.io.ByteArrayOutputStream
import java.io.File
import java.nio.ByteBuffer
import java.nio.ByteOrder
import java.util.concurrent.atomic.AtomicBoolean

class VoiceRecognitionManager(
    private val context: Context,
    private val scope: CoroutineScope,
    private val bluetoothAudioManager: BluetoothAudioManager? = null
) {

    private var mediaRecorder: MediaRecorder? = null
    private var audioFile: File? = null
    private var activeCallback: ((RecognitionResult) -> Unit)? = null
    private val isRecording = AtomicBoolean(false)
    private val gson = Gson()

    private data class WhisperResponse(val text: String)

    private data class VerboseWhisperResponse(
        val text: String,
        val duration: Double,
        val words: List<WhisperWord>?
    )

    private data class WhisperWord(val word: String, val start: Double, val end: Double)

    private sealed class VadResult {
        data class Audio(val file: File) : VadResult()
        data object NoSpeech : VadResult()
        data object Error : VadResult()
    }

    private var vadJob: Job? = null

    companion object {
        private val WHISPER_URL = "${ApiClient.BASE_URL}/v1/audio/transcriptions"
        private const val TAG = "VoiceRecognition"
        private const val SPEECH_THRESHOLD = 0.5f
        private const val SILENCE_THRESHOLD = 0.3f
        private const val SILENCE_DURATION_MS = 1500L
        private const val MAX_RECORDING_MS = 30_000L
        private const val VAD_SAMPLE_RATE = SileroVadDetector.SAMPLE_RATE
        private const val VAD_FRAME_SIZE = SileroVadDetector.FRAME_SIZE
        private const val WAV_HEADER_SIZE = 44
    }

    fun startListening(onResult: (RecognitionResult) -> Unit) {
        if (!isRecording.compareAndSet(false, true)) {
            onResult(RecognitionResult.Error(ErrorReason.RECORDING_FAILED))
            return
        }
        activeCallback = onResult

        // Enable Bluetooth audio if available
        val btEnabled = bluetoothAudioManager?.enableBluetoothAudio() ?: false
        val audioSource = if (btEnabled) {
            MediaRecorder.AudioSource.VOICE_COMMUNICATION
        } else {
            MediaRecorder.AudioSource.MIC
        }
        Log.d(TAG, "startListening: bluetooth=$btEnabled, audioSource=$audioSource")

        val recorder = MediaRecorder(context)
        try {
            audioFile = File.createTempFile("voice_", ".m4a", context.cacheDir)

            recorder.apply {
                setAudioSource(audioSource)
                setOutputFormat(MediaRecorder.OutputFormat.MPEG_4)
                setAudioEncoder(MediaRecorder.AudioEncoder.AAC)
                setAudioSamplingRate(16000)
                setAudioChannels(1)
                setOutputFile(audioFile!!.absolutePath)
                prepare()
                start()
            }

            mediaRecorder = recorder
            Log.d(TAG, "Recording started")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to start recording", e)
            recorder.release()
            bluetoothAudioManager?.disableBluetoothAudio()
            isRecording.set(false)
            activeCallback = null
            onResult(RecognitionResult.Error(ErrorReason.RECORDING_FAILED, e))
        }
    }

    fun stopListening() {
        if (!isRecording.compareAndSet(true, false)) return

        val callback = activeCallback ?: return
        activeCallback = null

        try {
            mediaRecorder?.apply {
                stop()
                reset()
                release()
            }
            mediaRecorder = null
            Log.d(TAG, "Recording stopped")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to stop recording", e)
            try { mediaRecorder?.reset() } catch (_: Exception) {}
            mediaRecorder?.release()
            mediaRecorder = null
            bluetoothAudioManager?.disableBluetoothAudio()
            callback(RecognitionResult.Error(ErrorReason.RECORDING_STOPPED_EARLY, e))
            return
        } finally {
            bluetoothAudioManager?.disableBluetoothAudio()
        }

        val file = audioFile ?: return
        audioFile = null
        scope.launch { transcribe(file, callback) }
    }

    private suspend fun transcribe(file: File, onResult: (RecognitionResult) -> Unit) {
        val transcribeSpan = AliciaTelemetry.startSpan("asr.transcribe",
            Attributes.builder().put("asr.model", "whisper").build()
        )
        val startTimeMs = System.currentTimeMillis()
        try {
            val result = withContext(Dispatchers.IO) {
                val mimeType = if (file.extension == "wav") "audio/wav" else "audio/mp4"
                val requestBody = MultipartBody.Builder()
                    .setType(MultipartBody.FORM)
                    .addFormDataPart("model", "whisper")
                    .addFormDataPart(
                        "file", file.name,
                        file.asRequestBody(mimeType.toMediaType())
                    )
                    .build()

                val request = Request.Builder()
                    .url(WHISPER_URL)
                    .post(requestBody)
                    .build()

                ApiClient.client.newCall(request).execute().use { response ->
                    val body = response.body?.string()

                    if (response.isSuccessful && body != null) {
                        val whisperResponse = gson.fromJson(body, WhisperResponse::class.java)
                        whisperResponse.text.trim()
                    } else {
                        Log.e(TAG, "Whisper API error: ${response.code} $body")
                        null
                    }
                }
            }

            val durationMs = System.currentTimeMillis() - startTimeMs
            transcribeSpan.setAttribute("asr.duration_ms", durationMs)
            transcribeSpan.setAttribute("asr.text_length", (result?.length ?: 0).toLong())

            withContext(Dispatchers.Main) {
                when {
                    result == null -> onResult(RecognitionResult.Error(ErrorReason.SERVER_ERROR))
                    result.isNotBlank() -> {
                        Log.d(TAG, "Recognized: $result")
                        onResult(RecognitionResult.Success(result))
                    }
                    else -> onResult(RecognitionResult.Error(ErrorReason.NO_SPEECH_DETECTED))
                }
            }
            transcribeSpan.end()
        } catch (e: Exception) {
            Log.e(TAG, "Transcription failed", e)
            AliciaTelemetry.recordError(transcribeSpan, e)
            transcribeSpan.end()
            withContext(Dispatchers.Main) {
                onResult(RecognitionResult.Error(ErrorReason.NETWORK_ERROR, e))
            }
        } finally {
            file.delete()
        }
    }

    fun stopAndGetFile(): File? {
        if (!isRecording.compareAndSet(true, false)) return null
        activeCallback = null

        return try {
            mediaRecorder?.apply {
                stop()
                reset()
                release()
            }
            mediaRecorder = null
            val file = audioFile
            audioFile = null
            file
        } catch (e: Exception) {
            Log.e(TAG, "Failed to stop recording for file return", e)
            try { mediaRecorder?.reset() } catch (_: Exception) {}
            mediaRecorder?.release()
            mediaRecorder = null
            null
        } finally {
            bluetoothAudioManager?.disableBluetoothAudio()
        }
    }

    suspend fun transcribeVerbose(file: File): VerboseTranscription? = withContext(Dispatchers.IO) {
        val verboseSpan = AliciaTelemetry.startSpan("asr.transcribe_verbose",
            Attributes.builder().put("asr.model", "whisper").build()
        )
        val startTimeMs = System.currentTimeMillis()
        try {
            val requestBody = MultipartBody.Builder()
                .setType(MultipartBody.FORM)
                .addFormDataPart("model", "whisper")
                .addFormDataPart("response_format", "verbose_json")
                .addFormDataPart("timestamp_granularities[]", "word")
                .addFormDataPart(
                    "file", file.name,
                    file.asRequestBody("audio/mp4".toMediaType())
                )
                .build()

            val request = Request.Builder()
                .url(WHISPER_URL)
                .post(requestBody)
                .build()

            ApiClient.client.newCall(request).execute().use { response ->
                val body = response.body?.string()

                if (response.isSuccessful && body != null) {
                    val verbose = gson.fromJson(body, VerboseWhisperResponse::class.java)
                    val text = verbose.text.trim()
                    val durationMs = (verbose.duration * 1000).toInt()

                    val words = verbose.words?.map { w ->
                        TimestampedWord(
                            word = w.word,
                            start = w.start.toFloat(),
                            end = w.end.toFloat()
                        )
                    } ?: emptyList()

                    Log.d(TAG, "Verbose transcription: $text (${words.size} words, ${durationMs}ms)")
                    verboseSpan.setAttribute("asr.duration_ms", System.currentTimeMillis() - startTimeMs)
                    verboseSpan.setAttribute("asr.text_length", text.length.toLong())
                    verboseSpan.setAttribute("asr.word_count", words.size.toLong())
                    verboseSpan.end()
                    VerboseTranscription(text, words, durationMs)
                } else {
                    Log.e(TAG, "Whisper verbose API error: ${response.code} $body")
                    verboseSpan.end()
                    null
                }
            }
        } catch (e: Exception) {
            Log.e(TAG, "Verbose transcription failed", e)
            AliciaTelemetry.recordError(verboseSpan, e)
            verboseSpan.end()
            null
        }
    }

    fun startListeningWithVad(vadDetector: SileroVadDetector, onResult: (RecognitionResult) -> Unit) {
        if (!isRecording.compareAndSet(false, true)) {
            onResult(RecognitionResult.Error(ErrorReason.RECORDING_FAILED))
            return
        }
        activeCallback = onResult
        vadDetector.resetState()

        vadJob = scope.launch {
            val result = withContext(Dispatchers.IO) { recordWithVad(vadDetector) }
            isRecording.set(false)
            val callback = activeCallback ?: return@launch
            activeCallback = null

            when (result) {
                is VadResult.Audio -> transcribe(result.file, callback)
                is VadResult.NoSpeech -> callback(RecognitionResult.Error(ErrorReason.NO_SPEECH_DETECTED))
                is VadResult.Error -> callback(RecognitionResult.Error(ErrorReason.RECORDING_FAILED))
            }
        }
    }

    fun stopVadListeningEarly() {
        isRecording.set(false)
    }

    fun cancelVadListening() {
        isRecording.set(false)
        vadJob?.cancel()
        vadJob = null
        activeCallback = null
    }

    private fun recordWithVad(vadDetector: SileroVadDetector): VadResult {
        val recordingSpan = AliciaTelemetry.startSpan("voice.recording")

        // Enable Bluetooth audio if available
        val btEnabled = bluetoothAudioManager?.enableBluetoothAudio() ?: false
        val actualSampleRate = if (btEnabled) {
            bluetoothAudioManager?.getEffectiveSampleRate(VAD_SAMPLE_RATE) ?: VAD_SAMPLE_RATE
        } else {
            VAD_SAMPLE_RATE
        }
        val needsResampling = actualSampleRate != VAD_SAMPLE_RATE

        Log.d(TAG, "Recording with sampleRate=$actualSampleRate, bluetooth=$btEnabled, needsResampling=$needsResampling")
        recordingSpan.setAttribute("voice.bluetooth_enabled", btEnabled)
        recordingSpan.setAttribute("voice.sample_rate", actualSampleRate.toLong())

        // Use VOICE_COMMUNICATION for Bluetooth, MIC otherwise
        val audioSource = if (btEnabled) {
            MediaRecorder.AudioSource.VOICE_COMMUNICATION
        } else {
            MediaRecorder.AudioSource.MIC
        }

        // Calculate frame size for actual sample rate
        val actualFrameSize = if (needsResampling) {
            // For 8kHz->16kHz, we need half the frames to produce VAD_FRAME_SIZE after resampling
            VAD_FRAME_SIZE * actualSampleRate / VAD_SAMPLE_RATE
        } else {
            VAD_FRAME_SIZE
        }

        val bufferSize = maxOf(
            AudioRecord.getMinBufferSize(
                actualSampleRate,
                AudioFormat.CHANNEL_IN_MONO,
                AudioFormat.ENCODING_PCM_16BIT
            ),
            actualFrameSize * 2 * 4
        )

        val recorder = AudioRecord(
            audioSource,
            actualSampleRate,
            AudioFormat.CHANNEL_IN_MONO,
            AudioFormat.ENCODING_PCM_16BIT,
            bufferSize
        )

        if (recorder.state != AudioRecord.STATE_INITIALIZED) {
            Log.e(TAG, "AudioRecord failed to initialize")
            recorder.release()
            bluetoothAudioManager?.disableBluetoothAudio()
            AliciaTelemetry.addSpanEvent(recordingSpan, "vad.init_failed")
            recordingSpan.end()
            return VadResult.Error
        }

        recorder.startRecording()

        val pcmStream = ByteArrayOutputStream()
        val inputFrame = ShortArray(actualFrameSize)
        val pcmBuffer = ByteBuffer.allocate(VAD_FRAME_SIZE * 2).order(ByteOrder.LITTLE_ENDIAN)
        val floatFrame = FloatArray(VAD_FRAME_SIZE)
        val streamResampler = if (needsResampling) {
            AudioResampler.StreamResampler(actualSampleRate, VAD_SAMPLE_RATE)
        } else null
        var speechDetected = false
        var silenceStartMs = 0L
        val startTime = System.currentTimeMillis()

        try {
            while (isRecording.get()) {
                if (System.currentTimeMillis() - startTime > MAX_RECORDING_MS) {
                    AliciaTelemetry.addSpanEvent(recordingSpan, "vad.max_duration",
                        Attributes.builder()
                            .put("vad.max_duration_ms", MAX_RECORDING_MS.toLong())
                            .build()
                    )
                    break
                }

                val read = recorder.read(inputFrame, 0, actualFrameSize)
                if (read != actualFrameSize) continue

                // Resample if needed (8kHz -> 16kHz)
                val frame = if (streamResampler != null) {
                    streamResampler.resampleFrame(inputFrame)
                } else {
                    inputFrame
                }

                pcmBuffer.clear()
                for (sample in frame) pcmBuffer.putShort(sample)
                pcmStream.write(pcmBuffer.array(), 0, pcmBuffer.position())

                for (i in 0 until VAD_FRAME_SIZE) floatFrame[i] = frame[i] / 32768f
                val prob = vadDetector.isSpeech(floatFrame)

                if (!speechDetected) {
                    if (prob > SPEECH_THRESHOLD) {
                        speechDetected = true
                        silenceStartMs = 0L
                        AliciaTelemetry.addSpanEvent(recordingSpan, "vad.speech_start",
                            Attributes.builder()
                                .put("vad.probability", prob.toDouble())
                                .put("vad.elapsed_ms", (System.currentTimeMillis() - startTime))
                                .build()
                        )
                    }
                } else {
                    if (prob < SILENCE_THRESHOLD) {
                        if (silenceStartMs == 0L) {
                            silenceStartMs = System.currentTimeMillis()
                        } else if (System.currentTimeMillis() - silenceStartMs >= SILENCE_DURATION_MS) {
                            AliciaTelemetry.addSpanEvent(recordingSpan, "vad.silence_detected",
                                Attributes.builder()
                                    .put("vad.silence_duration_ms", SILENCE_DURATION_MS)
                                    .put("vad.total_duration_ms", (System.currentTimeMillis() - startTime))
                                    .build()
                            )
                            break
                        }
                    } else {
                        silenceStartMs = 0L
                    }
                }
            }
        } finally {
            recorder.stop()
            recorder.release()
            bluetoothAudioManager?.disableBluetoothAudio()
        }

        val pcmData = pcmStream.toByteArray()
        if (pcmData.isEmpty() || !speechDetected) {
            AliciaTelemetry.addSpanEvent(recordingSpan, "vad.no_speech")
            recordingSpan.end()
            return VadResult.NoSpeech
        }

        recordingSpan.setAttribute("voice.recording_duration_ms", System.currentTimeMillis() - startTime)
        recordingSpan.setAttribute("voice.pcm_bytes", pcmData.size.toLong())
        recordingSpan.end()
        return VadResult.Audio(writeWavFile(pcmData))
    }

    private fun writeWavFile(pcmData: ByteArray): File {
        val file = File.createTempFile("vad_", ".wav", context.cacheDir)
        file.outputStream().use { out ->
            val dataSize = pcmData.size
            val header = ByteBuffer.allocate(WAV_HEADER_SIZE).order(ByteOrder.LITTLE_ENDIAN).apply {
                put("RIFF".toByteArray())
                putInt(36 + dataSize)
                put("WAVE".toByteArray())
                put("fmt ".toByteArray())
                putInt(16)
                putShort(1)
                putShort(1)
                putInt(VAD_SAMPLE_RATE)
                putInt(VAD_SAMPLE_RATE * 2)
                putShort(2)
                putShort(16)
                put("data".toByteArray())
                putInt(dataSize)
            }
            out.write(header.array())
            out.write(pcmData)
        }
        return file
    }

    fun destroy() {
        cancelVadListening()
        try {
            mediaRecorder?.stop()
        } catch (_: Exception) {}
        mediaRecorder?.release()
        mediaRecorder = null
        activeCallback = null
        audioFile?.delete()
    }
}
