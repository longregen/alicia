package com.alicia.assistant.model

enum class VpnStatus { DISCONNECTED, CONNECTING, CONNECTED, PENDING_APPROVAL, IN_USE_OTHER_USER, ERROR }

data class VpnState(
    val status: VpnStatus = VpnStatus.DISCONNECTED,
    val exitNode: ExitNode? = null,
    val ipAddress: String? = null,
    val since: Long? = null,
    val healthWarning: String? = null
)

data class ExitNode(
    val id: String,
    val name: String,
    val location: String,
    val online: Boolean,
    val countryCode: String
)

data class VpnSettings(
    val autoConnect: Boolean = true,
    val selectedExitNodeId: String? = null,
    val headscaleUrl: String = "",
    val authKey: String = "",
    val nodeRegistered: Boolean = false
)

data class TailnetPeer(
    val id: String,
    val hostName: String,
    val dnsName: String,
    val tailscaleIPs: List<String>,
    val online: Boolean,
    val active: Boolean,
    val curAddr: String,
    val relay: String,
    val rxBytes: Long,
    val txBytes: Long,
    val lastHandshake: String,
    val isSelf: Boolean,
    val os: String,
    val exitNodeOption: Boolean
)
