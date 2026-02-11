package com.alicia.assistant

import android.content.res.ColorStateList
import android.os.Bundle
import android.util.TypedValue
import android.view.View
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.model.TailnetPeer
import com.alicia.assistant.service.VpnManager
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.card.MaterialCardView
import com.google.android.material.color.MaterialColors
import com.google.android.material.progressindicator.LinearProgressIndicator
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.temporal.ChronoUnit

class NetworkStatusActivity : ComponentActivity() {

    private lateinit var loadingProgress: LinearProgressIndicator
    private lateinit var emptyState: TextView
    private lateinit var selfSectionHeader: TextView
    private lateinit var selfContainer: LinearLayout
    private lateinit var peersSectionHeader: TextView
    private lateinit var peersContainer: LinearLayout

    private var refreshJob: Job? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_network_status)

        val toolbar = findViewById<MaterialToolbar>(R.id.toolbar)
        toolbar.setNavigationOnClickListener { finish() }

        loadingProgress = findViewById(R.id.loadingProgress)
        emptyState = findViewById(R.id.emptyState)
        selfSectionHeader = findViewById(R.id.selfSectionHeader)
        selfContainer = findViewById(R.id.selfContainer)
        peersSectionHeader = findViewById(R.id.peersSectionHeader)
        peersContainer = findViewById(R.id.peersContainer)

        loadPeers()
    }

    override fun onResume() {
        super.onResume()
        startAutoRefresh()
    }

    override fun onPause() {
        super.onPause()
        refreshJob?.cancel()
        refreshJob = null
    }

    private fun startAutoRefresh() {
        refreshJob?.cancel()
        refreshJob = lifecycleScope.launch {
            while (isActive) {
                delay(10_000)
                loadPeers()
            }
        }
    }

    private fun loadPeers() {
        lifecycleScope.launch {
            loadingProgress.visibility = View.VISIBLE
            val peers = VpnManager.getTailnetPeers()
            loadingProgress.visibility = View.GONE

            if (peers.isEmpty()) {
                emptyState.visibility = View.VISIBLE
                selfSectionHeader.visibility = View.GONE
                selfContainer.removeAllViews()
                peersSectionHeader.visibility = View.GONE
                peersContainer.removeAllViews()
                return@launch
            }

            emptyState.visibility = View.GONE

            val selfPeers = peers.filter { it.isSelf }
            val otherPeers = peers.filter { !it.isSelf }

            selfContainer.removeAllViews()
            if (selfPeers.isNotEmpty()) {
                selfSectionHeader.visibility = View.VISIBLE
                for (peer in selfPeers) {
                    selfContainer.addView(createPeerCard(peer))
                }
            } else {
                selfSectionHeader.visibility = View.GONE
            }

            peersContainer.removeAllViews()
            if (otherPeers.isNotEmpty()) {
                peersSectionHeader.visibility = View.VISIBLE
                for (peer in otherPeers) {
                    peersContainer.addView(createPeerCard(peer))
                }
            } else {
                peersSectionHeader.visibility = View.GONE
            }
        }
    }

    private fun createPeerCard(peer: TailnetPeer): MaterialCardView {
        val card = MaterialCardView(this).apply {
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply {
                bottomMargin = dpToPx(8)
            }
            radius = dpToPx(16).toFloat()
            cardElevation = 0f
        }

        val content = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dpToPx(16), dpToPx(16), dpToPx(16), dpToPx(16))
        }

        val topRow = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = android.view.Gravity.CENTER_VERTICAL
        }

        val statusDot = ImageView(this).apply {
            layoutParams = LinearLayout.LayoutParams(dpToPx(8), dpToPx(8)).apply {
                marginEnd = dpToPx(8)
            }
            setImageResource(
                if (peer.online) R.drawable.status_dot_online
                else R.drawable.status_dot_offline
            )
        }
        topRow.addView(statusDot)

        val hostName = TextView(this).apply {
            text = peer.hostName.ifEmpty { peer.dnsName.trimEnd('.') }
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
            setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurface))
        }
        topRow.addView(hostName)

        if (peer.os.isNotEmpty()) {
            val osLabel = TextView(this).apply {
                text = " (${peer.os})"
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurfaceVariant))
            }
            topRow.addView(osLabel)
        }

        content.addView(topRow)

        if (peer.dnsName.isNotEmpty() && peer.hostName.isNotEmpty()) {
            val dnsView = TextView(this).apply {
                text = peer.dnsName.trimEnd('.')
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurfaceVariant))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply {
                    topMargin = dpToPx(2)
                }
            }
            content.addView(dnsView)
        }

        if (peer.tailscaleIPs.isNotEmpty()) {
            val ipsView = TextView(this).apply {
                text = peer.tailscaleIPs.joinToString(", ")
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurfaceVariant))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply {
                    topMargin = dpToPx(4)
                }
            }
            content.addView(ipsView)
        }

        if (peer.curAddr.isNotEmpty()) {
            val connView = TextView(this).apply {
                text = getString(R.string.network_connection_direct, peer.curAddr)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurfaceVariant))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply {
                    topMargin = dpToPx(4)
                }
            }
            content.addView(connView)
        } else if (peer.relay.isNotEmpty()) {
            val relayView = TextView(this).apply {
                text = getString(R.string.network_connection_relay, peer.relay)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurfaceVariant))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply {
                    topMargin = dpToPx(4)
                }
            }
            content.addView(relayView)
        }

        if (peer.lastHandshake.isNotEmpty() && !peer.isSelf) {
            val relativeTime = formatRelativeTime(peer.lastHandshake)
            if (relativeTime.isNotEmpty()) {
                val handshakeView = TextView(this).apply {
                    text = getString(R.string.network_last_seen, relativeTime)
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                    setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurfaceVariant))
                    layoutParams = LinearLayout.LayoutParams(
                        LinearLayout.LayoutParams.MATCH_PARENT,
                        LinearLayout.LayoutParams.WRAP_CONTENT
                    ).apply {
                        topMargin = dpToPx(4)
                    }
                }
                content.addView(handshakeView)
            }
        }

        if (peer.rxBytes > 0 || peer.txBytes > 0) {
            val trafficView = TextView(this).apply {
                text = getString(R.string.network_traffic, formatBytes(peer.rxBytes), formatBytes(peer.txBytes))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurfaceVariant))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply {
                    topMargin = dpToPx(4)
                }
            }
            content.addView(trafficView)
        }

        card.addView(content)
        return card
    }

    private fun formatRelativeTime(isoTimestamp: String): String {
        return try {
            val instant = Instant.parse(isoTimestamp)
            val now = Instant.now()
            val seconds = ChronoUnit.SECONDS.between(instant, now)
            when {
                seconds < 0 -> ""
                seconds < 60 -> "just now"
                seconds < 3600 -> "${seconds / 60}m ago"
                seconds < 86400 -> "${seconds / 3600}h ago"
                else -> "${seconds / 86400}d ago"
            }
        } catch (e: Exception) {
            ""
        }
    }

    private fun formatBytes(bytes: Long): String {
        return when {
            bytes < 1024 -> "$bytes B"
            bytes < 1024 * 1024 -> "${bytes / 1024} KB"
            bytes < 1024 * 1024 * 1024 -> String.format("%.1f MB", bytes / (1024.0 * 1024))
            else -> String.format("%.1f GB", bytes / (1024.0 * 1024 * 1024))
        }
    }

    private fun dpToPx(dp: Int): Int {
        return TypedValue.applyDimension(
            TypedValue.COMPLEX_UNIT_DIP, dp.toFloat(), resources.displayMetrics
        ).toInt()
    }
}
