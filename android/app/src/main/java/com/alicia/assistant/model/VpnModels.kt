package com.alicia.assistant.model

enum class VpnStatus { DISCONNECTED, CONNECTING, CONNECTED, PENDING_APPROVAL, ERROR }

data class VpnState(
    val status: VpnStatus = VpnStatus.DISCONNECTED,
    val exitNode: ExitNode? = null,
    val ipAddress: String? = null,
    val since: Long? = null
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
