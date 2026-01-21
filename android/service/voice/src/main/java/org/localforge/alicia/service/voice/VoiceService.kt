package org.localforge.alicia.service.voice

import android.Manifest
import android.annotation.SuppressLint
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.os.IBinder
import android.os.VibratorManager
import androidx.annotation.RequiresPermission
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import org.localforge.alicia.core.common.vibrate
import timber.log.Timber
import javax.inject.Inject

@AndroidEntryPoint
class VoiceService : Service() {

    @Inject
    lateinit var voiceController: VoiceController

    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private val notificationManager by lazy {
        getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
    }

    private val _state = MutableStateFlow<VoiceState>(VoiceState.Idle)
    val state: StateFlow<VoiceState> = _state.asStateFlow()

    private val notificationActionReceiver = object : BroadcastReceiver() {
        @SuppressLint("MissingPermission")
        override fun onReceive(context: Context, intent: Intent) {
            when (intent.action) {
                ACTION_STOP -> {
                    stopService()
                }
                ACTION_MUTE -> {
                    toggleMute()
                }
                ACTION_ACTIVATE -> {
                    activateVoiceAssistant()
                }
            }
        }
    }

    @SuppressLint("MissingPermission")
    override fun onCreate() {
        super.onCreate()

        createNotificationChannel()

        val filter = IntentFilter().apply {
            addAction(ACTION_STOP)
            addAction(ACTION_MUTE)
            addAction(ACTION_ACTIVATE)
        }
        registerReceiver(notificationActionReceiver, filter, RECEIVER_NOT_EXPORTED)

        startForeground(NOTIFICATION_ID, createNotification(VoiceState.Idle))

        serviceScope.launch {
            voiceController.currentState.collect { newState ->
                _state.value = newState
                updateNotification(newState)
            }
        }

        // Permission must be requested by the UI before starting service
        if (ContextCompat.checkSelfPermission(
                this,
                Manifest.permission.RECORD_AUDIO
            ) == PackageManager.PERMISSION_GRANTED
        ) {
            startWakeWordDetection()
        }
    }

    @SuppressLint("MissingPermission")
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        intent?.let {
            when (it.action) {
                ACTION_ACTIVATE -> activateVoiceAssistant()
                ACTION_STOP -> stopService()
                ACTION_MUTE -> toggleMute()
            }
        }

        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onDestroy() {
        super.onDestroy()

        try {
            unregisterReceiver(notificationActionReceiver)
        } catch (_: IllegalArgumentException) {
            // Expected if receiver was never registered or already unregistered
        }

        serviceScope.launch {
            voiceController.shutdown()
        }
        serviceScope.cancel()
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    private fun activateVoiceAssistant() {
        if (ContextCompat.checkSelfPermission(
                this,
                Manifest.permission.RECORD_AUDIO
            ) != PackageManager.PERMISSION_GRANTED
        ) {
            return
        }

        performVibration(100)
        serviceScope.launch {
            voiceController.activate()
        }
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    private fun startWakeWordDetection() {
        serviceScope.launch {
            voiceController.startWakeWordDetection()
        }
    }

    private fun stopService() {
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    private var isMuted = false

    @SuppressLint("MissingPermission")
    private fun toggleMute() {
        isMuted = !isMuted
        serviceScope.launch {
            if (isMuted) {
                voiceController.mute()
            } else {
                voiceController.unmute()
            }
        }
        updateNotification(_state.value)
    }

    @RequiresPermission(Manifest.permission.VIBRATE)
    private fun performVibration(durationMs: Long) {
        try {
            vibrate(durationMs)
        } catch (_: SecurityException) {
            // Vibration is non-essential - continue without it
        }
    }

    private fun createNotificationChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID,
            "Voice Assistant Service",
            NotificationManager.IMPORTANCE_LOW
        ).apply {
            description = "Always-listening voice assistant"
            setShowBadge(false)
            lockscreenVisibility = Notification.VISIBILITY_PUBLIC
        }
        notificationManager.createNotificationChannel(channel)
    }

    private fun createNotification(state: VoiceState): Notification {
        val stopIntent = PendingIntent.getBroadcast(
            this,
            0,
            Intent(ACTION_STOP),
            PendingIntent.FLAG_IMMUTABLE
        )

        val muteIntent = PendingIntent.getBroadcast(
            this,
            1,
            Intent(ACTION_MUTE),
            PendingIntent.FLAG_IMMUTABLE
        )

        val activateIntent = PendingIntent.getBroadcast(
            this,
            2,
            Intent(ACTION_ACTIVATE),
            PendingIntent.FLAG_IMMUTABLE
        )

        val contentText = when (state) {
            is VoiceState.Idle -> "Initializing..."
            is VoiceState.ListeningForWakeWord -> "Listening for wake word"
            is VoiceState.Activated -> "Wake word detected!"
            is VoiceState.Listening -> "Listening..."
            is VoiceState.Processing -> "Processing..."
            is VoiceState.Speaking -> "Speaking..."
            is VoiceState.Connecting -> "Connecting..."
            is VoiceState.Disconnected -> "Disconnected"
            is VoiceState.Error -> "Error occurred"
        }

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("Alicia Voice Assistant")
            .setContentText(contentText)
            .setSmallIcon(android.R.drawable.ic_btn_speak_now)
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .addAction(
                android.R.drawable.ic_media_pause,
                if (isMuted) "Unmute" else "Mute",
                muteIntent
            )
            .addAction(
                android.R.drawable.ic_btn_speak_now,
                "Activate",
                activateIntent
            )
            .addAction(
                android.R.drawable.ic_delete,
                "Stop",
                stopIntent
            )
            .build()
    }

    private fun updateNotification(state: VoiceState) {
        if (ContextCompat.checkSelfPermission(
                this,
                Manifest.permission.POST_NOTIFICATIONS
            ) != PackageManager.PERMISSION_GRANTED
        ) {
            return
        }
        notificationManager.notify(NOTIFICATION_ID, createNotification(state))
    }

    companion object {
        private const val NOTIFICATION_ID = 1001
        private const val CHANNEL_ID = "voice_service_channel"

        const val ACTION_STOP = "org.localforge.alicia.service.voice.ACTION_STOP"
        const val ACTION_MUTE = "org.localforge.alicia.service.voice.ACTION_MUTE"
        const val ACTION_ACTIVATE = "org.localforge.alicia.service.voice.ACTION_ACTIVATE"
    }
}

sealed class VoiceState {
    object Idle : VoiceState()
    object ListeningForWakeWord : VoiceState()
    object Activated : VoiceState()
    object Listening : VoiceState()
    object Processing : VoiceState()
    object Speaking : VoiceState()
    object Error : VoiceState()
    object Connecting : VoiceState()
    object Disconnected : VoiceState()
}
