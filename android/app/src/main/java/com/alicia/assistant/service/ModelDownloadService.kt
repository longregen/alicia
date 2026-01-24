package com.alicia.assistant.service

import android.app.Notification
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.IBinder
import android.util.Log
import com.alicia.assistant.AliciaApplication
import com.alicia.assistant.R
import com.alicia.assistant.model.VoskModelInfo
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import java.io.File
import java.io.FileOutputStream
import java.net.HttpURLConnection
import java.net.URL
import java.util.zip.ZipInputStream

class ModelDownloadService : Service() {

    companion object {
        private const val TAG = "ModelDownloadService"
        private const val NOTIFICATION_ID = 2001
        private const val EXTRA_MODEL_ID = "model_id"
        private const val CONNECT_TIMEOUT_MS = 15_000
        private const val READ_TIMEOUT_MS = 30_000

        private val _downloadState = MutableStateFlow<Map<String, Int>>(emptyMap())
        val downloadState: StateFlow<Map<String, Int>> = _downloadState

        fun start(context: Context, modelId: String) {
            val intent = Intent(context, ModelDownloadService::class.java)
            intent.putExtra(EXTRA_MODEL_ID, modelId)
            context.startForegroundService(intent)
        }

        fun isDownloading(modelId: String): Boolean = _downloadState.value.containsKey(modelId)
    }

    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private val activeJobs = mutableMapOf<String, Job>()

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val modelId = intent?.getStringExtra(EXTRA_MODEL_ID)
        if (modelId == null) {
            stopSelfIfIdle()
            return START_NOT_STICKY
        }

        if (activeJobs.containsKey(modelId)) return START_NOT_STICKY

        startForeground(NOTIFICATION_ID, buildNotification("Downloading model..."))

        val modelInfo = VoskModelInfo.fromId(modelId)
        val job = scope.launch {
            downloadAndExtract(modelInfo)
            withContext(Dispatchers.Main) {
                activeJobs.remove(modelId)
                stopSelfIfIdle()
            }
        }
        activeJobs[modelId] = job

        return START_NOT_STICKY
    }

    private suspend fun downloadAndExtract(modelInfo: VoskModelInfo) {
        val modelDir = File(filesDir, "vosk-models/${modelInfo.id}")
        val zipFile = File(cacheDir, "${modelInfo.id}.zip")

        try {
            if (File(modelDir, ".extracting").exists()) return
            updateProgress(modelInfo.id, 0)
            modelDir.mkdirs()

            val connection = URL(modelInfo.url).openConnection() as HttpURLConnection
            connection.connectTimeout = CONNECT_TIMEOUT_MS
            connection.readTimeout = READ_TIMEOUT_MS
            connection.connect()
            val totalSize = connection.contentLength.toLong()

            connection.inputStream.use { inputStream ->
                FileOutputStream(zipFile).use { output ->
                    val buffer = ByteArray(8192)
                    var downloaded = 0L
                    var bytesRead: Int
                    while (inputStream.read(buffer).also { bytesRead = it } != -1) {
                        yield()
                        output.write(buffer, 0, bytesRead)
                        downloaded += bytesRead
                        if (totalSize > 0) {
                            val pct = (downloaded * 100 / totalSize).toInt()
                            updateProgress(modelInfo.id, pct)
                        }
                    }
                }
            }

            updateProgress(modelInfo.id, -1)

            ZipInputStream(zipFile.inputStream()).use { zip ->
                var entry = zip.nextEntry
                while (entry != null) {
                    yield()
                    val stripped = entry.name.substringAfter('/')
                    if (stripped.isEmpty()) {
                        zip.closeEntry()
                        entry = zip.nextEntry
                        continue
                    }
                    val outFile = File(modelDir, stripped)
                    if (!outFile.canonicalPath.startsWith(modelDir.canonicalPath)) {
                        throw SecurityException("Zip entry escapes target: ${entry.name}")
                    }
                    if (entry.isDirectory) {
                        outFile.mkdirs()
                    } else {
                        outFile.parentFile?.mkdirs()
                        FileOutputStream(outFile).use { out -> zip.copyTo(out) }
                    }
                    zip.closeEntry()
                    entry = zip.nextEntry
                }
            }
            zipFile.delete()

            removeProgress(modelInfo.id)
            Log.d(TAG, "Model ${modelInfo.id} download complete")
        } catch (e: CancellationException) {
            modelDir.deleteRecursively()
            zipFile.delete()
            removeProgress(modelInfo.id)
            throw e
        } catch (e: Exception) {
            Log.e(TAG, "Download failed for ${modelInfo.id}", e)
            modelDir.deleteRecursively()
            zipFile.delete()
            removeProgress(modelInfo.id)
        }
    }

    private fun updateProgress(modelId: String, progress: Int) {
        _downloadState.update { it + (modelId to progress) }
        val text = if (progress == -1) "Extracting $modelId..." else "Downloading $modelId: $progress%"
        updateNotification(text)
    }

    private fun removeProgress(modelId: String) {
        _downloadState.update { it - modelId }
    }

    private fun updateNotification(text: String) {
        getSystemService(android.app.NotificationManager::class.java)
            .notify(NOTIFICATION_ID, buildNotification(text))
    }

    private fun buildNotification(text: String): Notification {
        return Notification.Builder(this, AliciaApplication.CHANNEL_ID)
            .setContentTitle("Alicia")
            .setContentText(text)
            .setSmallIcon(R.drawable.ic_microphone)
            .setOngoing(true)
            .build()
    }

    private fun stopSelfIfIdle() {
        if (activeJobs.isEmpty()) {
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
        }
    }

    override fun onDestroy() {
        scope.cancel()
        super.onDestroy()
    }
}
