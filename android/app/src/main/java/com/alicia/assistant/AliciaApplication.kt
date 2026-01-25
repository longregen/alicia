package com.alicia.assistant

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.util.Log
import com.alicia.assistant.storage.NoteRepository
import com.alicia.assistant.storage.PreferencesManager
import com.alicia.assistant.telemetry.AliciaTelemetry
import com.alicia.assistant.telemetry.ActivityLifecycleTracer
import com.alicia.assistant.tools.*
import com.alicia.assistant.ws.AssistantWebSocket
import com.alicia.assistant.ws.ToolRegistry
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import java.io.File
import androidx.lifecycle.DefaultLifecycleObserver
import androidx.lifecycle.LifecycleOwner
import androidx.lifecycle.ProcessLifecycleOwner

class AliciaApplication : Application() {

    companion object {
        const val CHANNEL_ID = "alicia_service_channel"
        private const val TAG = "AliciaApplication"
        private const val BUNDLED_MODEL_ID = "small-en-us"
    }

    private val _modelReady = MutableStateFlow(false)
    val modelReady: StateFlow<Boolean> = _modelReady.asStateFlow()

    private val _extractionDone = MutableStateFlow(false)
    val extractionDone: StateFlow<Boolean> = _extractionDone.asStateFlow()

    private val applicationScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    lateinit var assistantWebSocket: AssistantWebSocket
        private set

    override fun onCreate() {
        super.onCreate()

        AliciaTelemetry.initialize(this)
        registerActivityLifecycleCallbacks(ActivityLifecycleTracer())

        AliciaTelemetry.withSpan("app.startup") { span ->
            createNotificationChannel()
            AliciaTelemetry.addSpanEvent(span, "notification_channel_created")

            extractBundledModelIfNeeded()
            AliciaTelemetry.addSpanEvent(span, "model_extraction_started")

            initAssistantWebSocket()
            AliciaTelemetry.addSpanEvent(span, "websocket_initialized")

            ProcessLifecycleOwner.get().lifecycle.addObserver(object : DefaultLifecycleObserver {
                override fun onStart(owner: LifecycleOwner) {
                    Log.i(TAG, "App in foreground, starting heartbeat")
                    assistantWebSocket.startHeartbeat()
                }

                override fun onStop(owner: LifecycleOwner) {
                    Log.i(TAG, "App in background, stopping heartbeat")
                    assistantWebSocket.stopHeartbeat()
                }
            })
            AliciaTelemetry.addSpanEvent(span, "lifecycle_observer_registered")

            applicationScope.launch {
                NoteRepository(this@AliciaApplication)
                    .migrateFromPreferences(PreferencesManager(this@AliciaApplication))
            }
            AliciaTelemetry.addSpanEvent(span, "migration_launched")
        }
    }

    private fun initAssistantWebSocket() {
        val toolRegistry = ToolRegistry().apply {
            register(GetTimeExecutor())
            register(GetDateExecutor())
            register(GetBatteryExecutor(this@AliciaApplication))
            register(GetLocationExecutor(this@AliciaApplication))
            register(ReadScreenExecutor())
            register(GetClipboardExecutor(this@AliciaApplication))
        }

        assistantWebSocket = AssistantWebSocket(
            baseUrl = "wss://alicia.hjkl.lol/api/v1/ws",
            agentSecret = "not-needed",
            toolRegistry = toolRegistry
        )
        assistantWebSocket.connect()
        Log.i(TAG, "Assistant WebSocket initialized with ${toolRegistry.getAll().size} tools")
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
            _extractionDone.value = true
            return
        }

        applicationScope.launch {
            try {
                modelDir.mkdirs()
                marker.createNewFile()
                copyAssetDir("vosk-models/$BUNDLED_MODEL_ID", modelDir)
                marker.delete()
                if (modelDir.listFiles()?.isNotEmpty() == true) {
                    _modelReady.value = true
                    Log.d(TAG, "Bundled model extracted")
                } else {
                    Log.w(TAG, "Bundled model not found in assets, must be downloaded via Model Manager")
                    modelDir.deleteRecursively()
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to extract bundled model", e)
                modelDir.deleteRecursively()
            } finally {
                _extractionDone.value = true
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
