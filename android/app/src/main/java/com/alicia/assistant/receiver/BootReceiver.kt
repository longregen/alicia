package com.alicia.assistant.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import com.alicia.assistant.service.VoiceAssistantService

class BootReceiver : BroadcastReceiver() {

    companion object {
        private const val TAG = "WakeWordService"
    }

    override fun onReceive(context: Context?, intent: Intent?) {
        if (context == null) return
        if (intent?.action == Intent.ACTION_BOOT_COMPLETED) {
            Log.i(TAG, "BootReceiver: device boot completed, starting wake word service")
            VoiceAssistantService.ensureRunning(context)
        }
    }
}
