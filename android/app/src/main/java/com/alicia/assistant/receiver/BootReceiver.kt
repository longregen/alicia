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
        if (intent?.action != Intent.ACTION_BOOT_COMPLETED) return

        Log.i(TAG, "Boot completed, checking VPN auto-connect")

        AliciaTelemetry.withSpan("app.boot_received") { span ->
            AliciaTelemetry.addSpanEvent(span, "service.auto_start")
            VoiceAssistantService.ensureRunning(context)
        }

        val pendingResult = goAsync()
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val prefs = PreferencesManager(context)
                val vpnSettings = prefs.getVpnSettings()
                if (vpnSettings.autoConnect && vpnSettings.nodeRegistered) {
                    Log.i(TAG, "Auto-connecting VPN on boot")
                    VpnManager.init(context)
                    when (val result = VpnManager.connect(context)) {
                        is VpnManager.ConnectResult.Started ->
                            Log.i(TAG, "VPN auto-connect initiated")
                        is VpnManager.ConnectResult.NeedsPermission ->
                            Log.w(TAG, "VPN permission not granted, cannot auto-connect from boot")
                        is VpnManager.ConnectResult.Failed ->
                            Log.w(TAG, "VPN auto-connect failed: ${result.reason}")
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "VPN auto-connect error on boot", e)
            } finally {
                pendingResult.finish()
            }
        }
    }
}
