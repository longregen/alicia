package com.alicia.assistant.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import com.alicia.assistant.service.VoiceAssistantService
import com.alicia.assistant.service.VpnManager
import com.alicia.assistant.storage.PreferencesManager
import com.alicia.assistant.telemetry.AliciaTelemetry
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class BootReceiver : BroadcastReceiver() {

    companion object {
        private const val TAG = "BootReceiver"
    }

    override fun onReceive(context: Context?, intent: Intent?) {
        if (context == null) return
        if (intent?.action == Intent.ACTION_BOOT_COMPLETED) {
            Log.i(TAG, "BootReceiver: device boot completed, starting wake word service")

            AliciaTelemetry.withSpan("app.boot_received") { span ->
                AliciaTelemetry.addSpanEvent(span, "service.auto_start")
                VoiceAssistantService.ensureRunning(context)
            }

            CoroutineScope(Dispatchers.IO).launch {
                val prefs = PreferencesManager(context)
                val vpnSettings = prefs.getVpnSettings()
                if (vpnSettings.autoConnect && vpnSettings.nodeRegistered) {
                    try {
                        if (android.net.VpnService.prepare(context) == null) {
                            Log.i(TAG, "BootReceiver: auto-connecting VPN")
                            VpnManager.init(context)
                            VpnManager.connect(context)
                        } else {
                            Log.i(TAG, "BootReceiver: VPN permission not granted, skipping auto-connect")
                        }
                    } catch (e: Exception) {
                        Log.w(TAG, "BootReceiver: VPN permission check failed, skipping auto-connect", e)
                    }
                }
            }
        }
    }
}
