package com.alicia.assistant.service

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.ParcelFileDescriptor
import android.util.Log
import androidx.core.app.NotificationCompat
import com.alicia.assistant.R
import com.alicia.assistant.VpnSettingsActivity
import com.alicia.assistant.model.VpnState
import com.alicia.assistant.model.VpnStatus

class AliciaVpnService : VpnService() {

    companion object {
        private const val TAG = "AliciaVpnService"
        const val ACTION_START_VPN = "com.alicia.assistant.START_VPN"
        const val ACTION_STOP_VPN = "com.alicia.assistant.STOP_VPN"
        const val CHANNEL_ID = "vpn_service"
        private const val NOTIFICATION_ID = 2
    }

    private var vpnInterface: ParcelFileDescriptor? = null

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "VPN service created")
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START_VPN -> startVpn()
            ACTION_STOP_VPN -> stopVpn()
        }
        return START_STICKY
    }

    private fun startVpn() {
        Log.i(TAG, "Starting VPN tunnel")
        startForeground(NOTIFICATION_ID, buildNotification("Connecting..."))

        try {
            val builder = Builder()
                .setSession("Alicia VPN")
                .setMtu(1280)

            // Default Tailscale CGNAT configuration
            // Will be managed by libtailscale when IPNService interface is implemented
            builder.addAddress("100.64.0.1", 32)
            builder.addRoute("0.0.0.0", 0)
            builder.addDnsServer("100.100.100.100")

            vpnInterface = builder.establish()
            if (vpnInterface != null) {
                Log.i(TAG, "VPN tunnel established")
                updateNotification("Connected")
                VpnManager.updateState(VpnState(status = VpnStatus.CONNECTED, since = System.currentTimeMillis()))
            } else {
                Log.e(TAG, "Failed to establish VPN tunnel")
                updateNotification("Connection failed")
                VpnManager.updateState(VpnState(status = VpnStatus.ERROR))
                stopSelf()
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to start VPN", e)
            updateNotification("Error: ${e.message}")
            VpnManager.updateState(VpnState(status = VpnStatus.ERROR))
            stopSelf()
        }
    }

    private fun stopVpn() {
        Log.i(TAG, "Stopping VPN tunnel")
        try {
            vpnInterface?.close()
            vpnInterface = null
        } catch (e: Exception) {
            Log.e(TAG, "Error closing VPN interface", e)
        }
        VpnManager.updateState(VpnState(status = VpnStatus.DISCONNECTED))
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    override fun onDestroy() {
        super.onDestroy()
        try {
            vpnInterface?.close()
            vpnInterface = null
        } catch (e: Exception) {
            Log.e(TAG, "Error closing VPN interface on destroy", e)
        }
        Log.i(TAG, "VPN service destroyed")
    }

    override fun onRevoke() {
        Log.i(TAG, "VPN permission revoked")
        VpnManager.updateState(VpnState(status = VpnStatus.DISCONNECTED))
        stopVpn()
    }

    private fun buildNotification(statusText: String): android.app.Notification {
        val pendingIntent = PendingIntent.getActivity(
            this, 0,
            Intent(this, VpnSettingsActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("Alicia VPN")
            .setContentText(statusText)
            .setSmallIcon(R.drawable.ic_vpn_shield)
            .setOngoing(true)
            .setContentIntent(pendingIntent)
            .setForegroundServiceBehavior(NotificationCompat.FOREGROUND_SERVICE_IMMEDIATE)
            .build()
    }

    private fun updateNotification(statusText: String) {
        val notification = buildNotification(statusText)
        val notificationManager = getSystemService(NotificationManager::class.java)
        notificationManager.notify(NOTIFICATION_ID, notification)
    }
}
