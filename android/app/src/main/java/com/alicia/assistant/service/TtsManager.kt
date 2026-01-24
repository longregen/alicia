package com.alicia.assistant.service

import android.content.Context
import android.media.MediaPlayer
import android.util.Log
import com.google.gson.Gson
import kotlinx.coroutines.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.File

class TtsManager(private val context: Context, private val scope: CoroutineScope) {

    private var mediaPlayer: MediaPlayer? = null
    private val gson = Gson()

    private data class TtsRequest(
        val model: String,
        val input: String,
        val voice: String,
        val speed: Double
    )

    companion object {
        private val TTS_URL = "${ApiClient.BASE_URL}/v1/audio/speech"
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

                    ApiClient.client.newCall(request).execute().use { response ->
                        if (response.isSuccessful) {
                            val audioData = response.body?.bytes() ?: return@withContext null
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
                        onDone?.invoke()
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "TTS failed", e)
                withContext(Dispatchers.Main) { onDone?.invoke() }
            }
        }
    }

    private fun playAudio(file: File, onDone: (() -> Unit)?) {
        stopPlayback()
        VoiceAssistantService.pauseDetection()
        val player = MediaPlayer()
        try {
            player.setDataSource(file.absolutePath)
            player.setOnCompletionListener {
                VoiceAssistantService.resumeDetection()
                file.delete()
                onDone?.invoke()
            }
            player.setOnErrorListener { _, _, _ ->
                VoiceAssistantService.resumeDetection()
                file.delete()
                onDone?.invoke()
                true
            }
            player.prepare()
            player.start()
            mediaPlayer = player
        } catch (e: Exception) {
            VoiceAssistantService.resumeDetection()
            player.release()
            file.delete()
            onDone?.invoke()
        }
    }

    fun stopPlayback() {
        try {
            mediaPlayer?.apply {
                if (isPlaying) stop()
                release()
            }
        } catch (_: IllegalStateException) {}
        mediaPlayer = null
        VoiceAssistantService.resumeDetection()
    }

    fun destroy() {
        stopPlayback()
    }
}
