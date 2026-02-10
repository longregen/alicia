package com.alicia.assistant.service

import android.content.Context
import android.media.MediaPlayer
import android.util.Log
import com.alicia.assistant.telemetry.AliciaTelemetry
import com.google.gson.Gson
import io.opentelemetry.api.common.Attributes
import io.opentelemetry.api.trace.Span
import kotlinx.coroutines.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.File

class TtsManager(private val context: Context, private val scope: CoroutineScope) {

    private var mediaPlayer: MediaPlayer? = null
    // Accessed from coroutine threads (speak/IO), Main thread (playAudio callbacks), and callers of stopPlayback()
    @Volatile
    private var currentTtsSpan: Span? = null
    private val gson = Gson()

    private data class TtsRequest(
        val model: String,
        val input: String,
        val voice: String,
        val speed: Double
    )

    companion object {
        private val TTS_URL: String
            get() = "${ApiClient.BASE_URL}/v1/audio/speech"
        private const val TAG = "TtsManager"
    }

    init {
        scope.launch(Dispatchers.IO) { cleanupOldTtsFiles() }
    }

    private fun cleanupOldTtsFiles() {
        context.cacheDir.listFiles()?.forEach { file ->
            if (file.name.startsWith("tts_") && file.name.endsWith(".mp3")) {
                file.delete()
            }
        }
    }

    fun speak(text: String, speed: Float = 1.5f, onDone: (() -> Unit)? = null) {
        scope.launch {
            val ttsSpan = AliciaTelemetry.startSpan("tts.synthesize",
                Attributes.builder()
                    .put("tts.model", "kokoro")
                    .put("tts.text_length", text.length.toLong())
                    .put("tts.voice", "af_heart")
                    .put("tts.speed", speed.toDouble())
                    .build()
            )
            currentTtsSpan = ttsSpan
            try {
                val tempFile = withContext(Dispatchers.IO) {
                    val ttsRequest = TtsRequest(
                        model = "kokoro",
                        input = text,
                        voice = "af_heart",
                        speed = speed.toDouble()
                    )

                    val requestBody = gson.toJson(ttsRequest)
                        .toRequestBody("application/json".toMediaType())

                    val request = Request.Builder()
                        .url(TTS_URL)
                        .post(requestBody)
                        .build()

                    val apiStartMs = System.currentTimeMillis()
                    ApiClient.client.newCall(request).execute().use { response ->
                        if (response.isSuccessful) {
                            val audioData = response.body?.bytes() ?: return@withContext null
                            AliciaTelemetry.addSpanEvent(ttsSpan, "tts.api_complete",
                                Attributes.builder()
                                    .put("tts.audio_bytes", audioData.size.toLong())
                                    .put("tts.api_duration_ms", System.currentTimeMillis() - apiStartMs)
                                    .build()
                            )
                            val file = File.createTempFile("tts_", ".mp3", context.cacheDir)
                            file.writeBytes(audioData)
                            file
                        } else {
                            Log.e(TAG, "TTS API error: ${response.code} ${response.body?.string()}")
                            null
                        }
                    }
                }

                withContext(Dispatchers.Main) {
                    if (tempFile != null) {
                        playAudio(tempFile, onDone)
                    } else {
                        ttsSpan.end()
                        currentTtsSpan = null
                        onDone?.invoke()
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "TTS failed", e)
                AliciaTelemetry.recordError(ttsSpan, e)
                ttsSpan.end()
                currentTtsSpan = null
                withContext(Dispatchers.Main) { onDone?.invoke() }
            }
        }
    }

    private fun playAudio(file: File, onDone: (() -> Unit)?) {
        stopPlayback()
        VoiceAssistantService.pauseDetection()
        val player = MediaPlayer()
        val ttsSpan = currentTtsSpan
        try {
            player.setDataSource(file.absolutePath)
            player.setOnCompletionListener {
                ttsSpan?.let { span ->
                    AliciaTelemetry.addSpanEvent(span, "tts.playback_complete")
                    span.end()
                    currentTtsSpan = null
                }
                VoiceAssistantService.resumeDetection()
                file.delete()
                onDone?.invoke()
            }
            player.setOnErrorListener { _, _, _ ->
                ttsSpan?.let { span ->
                    AliciaTelemetry.addSpanEvent(span, "tts.playback_error")
                    span.end()
                    currentTtsSpan = null
                }
                VoiceAssistantService.resumeDetection()
                file.delete()
                onDone?.invoke()
                true
            }
            player.prepare()
            player.start()
            ttsSpan?.let { AliciaTelemetry.addSpanEvent(it, "tts.playback_start") }
            mediaPlayer = player
        } catch (e: Exception) {
            ttsSpan?.let { span ->
                AliciaTelemetry.recordError(span, e)
                span.end()
                currentTtsSpan = null
            }
            VoiceAssistantService.resumeDetection()
            player.release()
            file.delete()
            onDone?.invoke()
        }
    }

    fun stopPlayback() {
        currentTtsSpan?.end()
        currentTtsSpan = null
        try {
            mediaPlayer?.apply {
                if (isPlaying) stop()
                release()
            }
        } catch (e: IllegalStateException) { Log.w(TAG, "MediaPlayer cleanup failed", e) }
        mediaPlayer = null
        VoiceAssistantService.resumeDetection()
    }

    fun destroy() {
        stopPlayback()
    }
}
