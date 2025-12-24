package org.localforge.alicia.service.voice

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import timber.log.Timber

/**
 * BroadcastReceiver for handling notification actions from external components.
 * This receiver is registered in AndroidManifest.xml and handles broadcasts from:
 * - System components (e.g., notification actions from notification shade)
 * - Other app components outside the VoiceService
 *
 * Note: VoiceService also has an internal BroadcastReceiver (notificationActionReceiver)
 * that is dynamically registered when the service starts. The internal receiver handles
 * actions while the service is running, while this standalone receiver can handle
 * actions even when the service is not running (e.g., starting the service from notification).
 *
 * Architecture:
 * - VoiceServiceReceiver (this class): Registered in manifest, creates/forwards intents to service
 * - VoiceService.notificationActionReceiver: Dynamically registered, directly calls service methods
 *
 * Both receivers handle the same actions but serve different lifecycle scopes.
 */
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
        Timber.i("Toggling mute")
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
