package com.alicia.assistant

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.util.Log
import com.alicia.assistant.storage.NoteRepository
import com.alicia.assistant.storage.PreferencesManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import java.io.File

class AliciaApplication : Application() {

    companion object {
        const val CHANNEL_ID = "alicia_service_channel"
        private const val TAG = "AliciaApplication"
        private const val BUNDLED_MODEL_ID = "small-en-us"
    }

    private val _modelReady = MutableStateFlow(false)
    val modelReady: StateFlow<Boolean> = _modelReady.asStateFlow()

    private val applicationScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
        extractBundledModelIfNeeded()
        applicationScope.launch {
            NoteRepository(this@AliciaApplication)
                .migrateFromPreferences(PreferencesManager(this@AliciaApplication))
        }
    }

    private fun createNotificationChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID,
            "Alicia Voice Assistant",
            NotificationManager.IMPORTANCE_LOW
        ).apply {
            description = "Wake word detection service"
        }

        val notificationManager = getSystemService(NotificationManager::class.java)
        notificationManager.createNotificationChannel(channel)
    }

    private fun extractBundledModelIfNeeded() {
        val modelDir = File(filesDir, "vosk-models/$BUNDLED_MODEL_ID")
        val marker = File(modelDir, ".extracting")

        if (marker.exists()) {
            modelDir.deleteRecursively()
        }
        if (modelDir.exists() && modelDir.listFiles()?.isNotEmpty() == true) {
            _modelReady.value = true
            return
        }

        applicationScope.launch {
            try {
                modelDir.mkdirs()
                marker.createNewFile()
                copyAssetDir("vosk-models/$BUNDLED_MODEL_ID", modelDir)
                marker.delete()
                _modelReady.value = true
                Log.d(TAG, "Bundled model extracted")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to extract bundled model", e)
                modelDir.deleteRecursively()
            }
        }
    }

    private fun copyAssetDir(assetPath: String, targetDir: File) {
        val entries = assets.list(assetPath) ?: return
        targetDir.mkdirs()
        for (entry in entries) {
            val childAsset = "$assetPath/$entry"
            val childTarget = File(targetDir, entry)
            val subEntries = assets.list(childAsset)
            if (subEntries != null && subEntries.isNotEmpty()) {
                copyAssetDir(childAsset, childTarget)
            } else {
                assets.open(childAsset).use { input ->
                    childTarget.outputStream().use { output ->
                        input.copyTo(output)
                    }
                }
            }
        }
    }
}
