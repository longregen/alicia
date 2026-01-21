package org.localforge.alicia.service.voice

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import timber.log.Timber

class VoiceServiceReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        Timber.d("Received action: ${intent.action}")

        when (intent.action) {
            VoiceService.ACTION_STOP -> {
                handleStopAction(context)
            }
            VoiceService.ACTION_MUTE -> {
                handleMuteAction(context)
            }
            VoiceService.ACTION_ACTIVATE -> {
                handleActivateAction(context)
            }
        }
    }

    private fun handleStopAction(context: Context) {
        Timber.i("Stopping voice service")
        val intent = Intent(context, VoiceService::class.java).apply {
            action = VoiceService.ACTION_STOP
        }
        context.startService(intent)
    }

    private fun handleMuteAction(context: Context) {
        Timber.i("Request mute toggle from service")
        val intent = Intent(context, VoiceService::class.java).apply {
            action = VoiceService.ACTION_MUTE
        }
        context.startService(intent)
    }

    private fun handleActivateAction(context: Context) {
        Timber.i("Activating voice assistant")
        val intent = Intent(context, VoiceService::class.java).apply {
            action = VoiceService.ACTION_ACTIVATE
        }
        context.startService(intent)
    }
}
