package com.alicia.assistant.service

import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.system.OsConstants
import android.util.Log
import androidx.core.app.NotificationCompat
import com.alicia.assistant.R
import com.alicia.assistant.VpnSettingsActivity
import com.alicia.assistant.model.VpnState
import com.alicia.assistant.model.VpnStatus
import libtailscale.Libtailscale
import java.util.UUID
import java.util.concurrent.atomic.AtomicBoolean

class AliciaVpnService : VpnService(), libtailscale.IPNService {

    companion object {
        private const val TAG = "AliciaVpnService"
        const val ACTION_START_VPN = "com.alicia.assistant.START_VPN"
        const val ACTION_STOP_VPN = "com.alicia.assistant.STOP_VPN"
        const val CHANNEL_ID = "vpn_service"
        private const val NOTIFICATION_ID = 2
    }

    private val serviceId = UUID.randomUUID().toString()
    private val closed = AtomicBoolean(false)

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "VPN service created")
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START_VPN -> startVpn()
            ACTION_STOP_VPN -> {
                stopVpn()
                return START_NOT_STICKY
            }
            // Always-On VPN or OOM restart (null intent): start tunnel if registered
            else -> startVpn()
        }
        return START_STICKY
    }

    private fun startVpn() {
        Log.i(TAG, "Starting VPN tunnel")
        startForeground(NOTIFICATION_ID, buildNotification("Connecting..."))
        try {
            Libtailscale.requestVPN(this)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to request VPN from libtailscale", e)
            updateNotification("Engine unavailable")
            VpnManager.updateState(VpnState(status = VpnStatus.ERROR))
            stopSelf()
        }
    }

    private fun stopVpn() {
        Log.i(TAG, "Stopping VPN tunnel")
        close()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    override fun onRevoke() {
        Log.i(TAG, "VPN permission revoked")
        stopVpn()
    }

    override fun onDestroy() {
        super.onDestroy()
        Log.i(TAG, "VPN service destroyed")
    }

    override fun id(): String = serviceId

    override fun newBuilder(): libtailscale.VPNServiceBuilder {
        val b = Builder()
            .setSession("Alicia VPN")
            .setConfigureIntent(
                PendingIntent.getActivity(
                    this, 0,
                    Intent(this, VpnSettingsActivity::class.java),
                    PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
                )
            )
            .allowFamily(OsConstants.AF_INET)
            .allowFamily(OsConstants.AF_INET6)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            b.setMetered(false)
        }
        b.setUnderlyingNetworks(null)
        return VPNServiceBuilderAdapter(b)
    }

    override fun close() {
        if (!closed.compareAndSet(false, true)) return
        try {
            Libtailscale.serviceDisconnect(this)
        } catch (e: Exception) {
            Log.w(TAG, "serviceDisconnect failed", e)
        }
        VpnManager.updateState(VpnState(status = VpnStatus.DISCONNECTED))
    }

    override fun disconnectVPN() {
        stopSelf()
    }

    override fun updateVpnStatus(active: Boolean) {
        if (active) {
            updateNotification("Connected")
        } else {
            updateNotification("Disconnected")
        }
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
        val notificationManager = getSystemService(NotificationManager::class.java)
        notificationManager.notify(NOTIFICATION_ID, buildNotification(statusText))
    }
}

/**
 * Wraps Android's [VpnService.Builder] as a [libtailscale.VPNServiceBuilder].
 * The Go side calls these methods to configure the tunnel with addresses, routes, and DNS.
 */
class VPNServiceBuilderAdapter(
    private val builder: VpnService.Builder
) : libtailscale.VPNServiceBuilder {

    override fun addAddress(addr: String, prefix: Int) {
        builder.addAddress(addr, prefix)
    }

    override fun addDNSServer(server: String) {
        builder.addDnsServer(server)
    }

    override fun addSearchDomain(domain: String) {
        builder.addSearchDomain(domain)
    }

    override fun addRoute(addr: String, prefix: Int) {
        builder.addRoute(addr, prefix)
    }

    override fun excludeRoute(addr: String, prefix: Int) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            builder.excludeRoute(
                android.net.IpPrefix(java.net.InetAddress.getByName(addr), prefix)
            )
        }
    }

    override fun setMTU(mtu: Int) {
        builder.setMtu(mtu)
    }

    override fun establish(): libtailscale.ParcelFileDescriptor? {
        return builder.establish()?.let { ParcelFileDescriptorAdapter(it) }
    }
}

/**
 * Wraps Android's [android.os.ParcelFileDescriptor] for the Go side.
 */
class ParcelFileDescriptorAdapter(
    private val fd: android.os.ParcelFileDescriptor
) : libtailscale.ParcelFileDescriptor {

    override fun detach(): Int = fd.detachFd()
}

/**
 * Wraps a Java [java.io.InputStream] as a [libtailscale.InputStream] for Go interop.
 */
class InputStreamAdapter(
    private val stream: java.io.InputStream
) : libtailscale.InputStream {

    override fun read(): ByteArray? {
        val buf = ByteArray(4096)
        val n = stream.read(buf)
        if (n == -1) return null
        return buf.sliceArray(0 until n)
    }

    override fun close() {
        stream.close()
    }
}
