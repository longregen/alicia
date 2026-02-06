package com.alicia.assistant.service

import android.content.Context
import android.content.Intent
import android.net.VpnService
import android.util.Log
import com.alicia.assistant.model.ExitNode
import com.alicia.assistant.model.VpnSettings
import com.alicia.assistant.model.VpnState
import com.alicia.assistant.model.VpnStatus
import com.alicia.assistant.storage.PreferencesManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import libtailscale.Libtailscale
import org.json.JSONObject

object VpnManager {
    private const val TAG = "VpnManager"
    private const val API_TIMEOUT_MS = 30_000L

    sealed class ConnectResult {
        data object Started : ConnectResult()
        data class NeedsPermission(val intent: Intent) : ConnectResult()
        data class Failed(val reason: String) : ConnectResult()
    }

    private val supervisorJob = SupervisorJob()
    private val scope = CoroutineScope(supervisorJob + Dispatchers.IO)

    private val _state = MutableStateFlow(VpnState())
    val state: StateFlow<VpnState> = _state.asStateFlow()

    private var app: libtailscale.Application? = null
    private var notificationManager: libtailscale.NotificationManager? = null
    private var isInitialized = false
    private lateinit var appContext: Context
    private val prefs: PreferencesManager by lazy { PreferencesManager(appContext) }
    private var connectJob: Job? = null

    fun init(context: Context) {
        if (isInitialized) return
        isInitialized = true
        appContext = context.applicationContext
        Log.i(TAG, "Initializing VPN manager")

        try {
            val filesDir = appContext.filesDir.absolutePath
            val directFileRoot = appContext.getDir("tailscale", Context.MODE_PRIVATE).absolutePath
            val appCtx = AppContextImpl(appContext)
            app = Libtailscale.start(filesDir, directFileRoot, false, appCtx)
            Log.i(TAG, "libtailscale backend started")
            startNotificationWatcher()
        } catch (e: UnsupportedOperationException) {
            Log.w(TAG, "libtailscale not available (stubs only) - VPN features disabled")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to initialize libtailscale", e)
            _state.value = VpnState(status = VpnStatus.ERROR)
        }

        scope.launch {
            val settings = prefs.getVpnSettings()
            if (settings.nodeRegistered) {
                Log.i(TAG, "Node previously registered with Headscale")
            }
        }
    }

    private fun startNotificationWatcher() {
        val tsApp = app ?: return
        // Bitmask: InitialState(2) | Prefs(4) | Netmap(8) = 14
        val mask = 2L or 4L or 8L
        try {
            notificationManager = tsApp.watchNotifications(mask) { notification ->
                try {
                    val json = JSONObject(String(notification))
                    json.optInt("State", -1).takeIf { it >= 0 }?.let { stateInt ->
                        val newStatus = when (stateInt) {
                            0 -> VpnStatus.DISCONNECTED  // NoState
                            1 -> VpnStatus.DISCONNECTED  // InUseOtherUser
                            2 -> VpnStatus.DISCONNECTED  // NeedsLogin
                            3 -> VpnStatus.DISCONNECTED  // NeedsMachineAuth
                            4 -> VpnStatus.DISCONNECTED  // Stopped
                            5 -> VpnStatus.CONNECTING     // Starting
                            6 -> VpnStatus.CONNECTED      // Running
                            else -> null
                        }
                        if (newStatus != null && newStatus != _state.value.status) {
                            _state.value = _state.value.copy(status = newStatus)
                            if (newStatus == VpnStatus.CONNECTED) {
                                scope.launch { refreshConnectionInfo() }
                            }
                        }
                    }
                } catch (e: Exception) {
                    Log.w(TAG, "Failed to parse notification", e)
                }
            }
            Log.i(TAG, "Notification watcher started")
        } catch (e: Exception) {
            Log.w(TAG, "Failed to start notification watcher", e)
        }
    }

    private suspend fun refreshConnectionInfo() {
        val status = getBackendStatus() ?: return
        val selfNode = status.optJSONObject("Self")
        val ipAddress = selfNode?.optJSONArray("TailscaleIPs")?.optString(0)
        _state.value = _state.value.copy(
            ipAddress = ipAddress,
            since = _state.value.since ?: System.currentTimeMillis()
        )
    }

    fun connect(context: Context): ConnectResult {
        if (app == null) {
            _state.value = VpnState(status = VpnStatus.ERROR)
            return ConnectResult.Failed("libtailscale not initialized")
        }

        val vpnIntent = VpnService.prepare(context)
        if (vpnIntent != null) {
            return ConnectResult.NeedsPermission(vpnIntent)
        }

        startVpnService(context)
        return ConnectResult.Started
    }

    internal fun startVpnService(context: Context) {
        connectJob?.cancel()
        _state.value = _state.value.copy(status = VpnStatus.CONNECTING)

        connectJob = scope.launch {
            try {
                // Set WantRunning via prefs
                callLocalApi("PATCH", "/localapi/v0/prefs", """{"WantRunning":true}""")

                val intent = Intent(context, AliciaVpnService::class.java).apply {
                    action = AliciaVpnService.ACTION_START_VPN
                }
                context.startForegroundService(intent)

                // Apply saved exit node
                val settings = prefs.getVpnSettings()
                settings.selectedExitNodeId?.let { nodeId ->
                    callLocalApi("PATCH", "/localapi/v0/prefs", """{"ExitNodeID":"$nodeId"}""")
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to start VPN", e)
                _state.value = VpnState(status = VpnStatus.ERROR)
            }
        }
    }

    fun disconnect(context: Context) {
        connectJob?.cancel()
        connectJob = null
        scope.launch {
            try {
                callLocalApi("PATCH", "/localapi/v0/prefs", """{"WantRunning":false}""")
            } catch (e: Exception) {
                Log.w(TAG, "Failed to set WantRunning=false", e)
            }
            stopVpnInternal(context)
        }
    }

    private fun stopVpnInternal(context: Context) {
        try {
            context.startService(
                Intent(context, AliciaVpnService::class.java).apply {
                    action = AliciaVpnService.ACTION_STOP_VPN
                }
            )
        } catch (e: Exception) {
            Log.e(TAG, "Failed to send stop intent", e)
        }
        _state.value = VpnState(status = VpnStatus.DISCONNECTED)
    }

    fun setExitNode(nodeId: String, node: ExitNode? = null) {
        scope.launch {
            try {
                callLocalApi("PATCH", "/localapi/v0/prefs", """{"ExitNodeID":"$nodeId"}""")
                _state.value = _state.value.copy(exitNode = node)
                Log.i(TAG, "Exit node set to: ${node?.name ?: nodeId}")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to set exit node", e)
            }
        }
    }

    suspend fun getExitNodes(): List<ExitNode> = withContext(Dispatchers.IO) {
        try {
            val status = getBackendStatus() ?: return@withContext emptyList()
            val peers = status.optJSONObject("Peer") ?: return@withContext emptyList()
            val nodes = mutableListOf<ExitNode>()

            for (key in peers.keys()) {
                val peer = peers.getJSONObject(key)
                if (!peer.optBoolean("ExitNodeOption", false)) continue
                val hostName = peer.optString("HostName", "")
                val location = peer.optJSONObject("Location")
                nodes.add(
                    ExitNode(
                        id = peer.optString("ID", key),
                        name = hostName,
                        location = location?.let {
                            "${it.optString("CountryCode", "")} - ${it.optString("City", "")}"
                        } ?: hostName,
                        online = peer.optBoolean("Online", false),
                        countryCode = location?.optString("CountryCode", "") ?: ""
                    )
                )
            }
            nodes
        } catch (e: Exception) {
            Log.e(TAG, "Failed to get exit nodes", e)
            emptyList()
        }
    }

    suspend fun loginWithAuthKey(key: String): Boolean = withContext(Dispatchers.IO) {
        try {
            callLocalApi("PATCH", "/localapi/v0/prefs", """{"WantRunning":true}""")
            callLocalApi("POST", "/localapi/v0/start", """{"AuthKey":"$key"}""")
            Log.i(TAG, "Login with auth key initiated")
            // Verify backend reaches Running state
            repeat(15) {
                val status = getBackendStatus()
                val backendState = status?.optString("BackendState", "")
                if (backendState == "Running" || backendState == "NeedsMachineAuth") return@withContext true
                kotlinx.coroutines.delay(1000)
            }
            false
        } catch (e: Exception) {
            Log.e(TAG, "Failed to login with auth key", e)
            false
        }
    }

    suspend fun setControlUrl(url: String): Boolean = withContext(Dispatchers.IO) {
        try {
            callLocalApi("PATCH", "/localapi/v0/prefs", """{"ControlURL":"$url"}""")
            Log.i(TAG, "Control URL set to: $url")
            true
        } catch (e: Exception) {
            Log.e(TAG, "Failed to set control URL", e)
            false
        }
    }

    suspend fun forgetDevice(context: Context) = withContext(Dispatchers.IO) {
        try {
            connectJob?.cancel()
            connectJob = null
            callLocalApi("POST", "/localapi/v0/logout", null)
            stopVpnInternal(context)
            prefs.saveVpnSettings(VpnSettings())
            Log.i(TAG, "Device forgotten and logged out")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to forget device", e)
        }
    }

    fun shutdown() {
        notificationManager?.stop()
        notificationManager = null
        connectJob?.cancel()
        supervisorJob.cancel()
    }

    internal fun updateState(newState: VpnState) {
        _state.value = newState
    }

    private fun getBackendStatus(): JSONObject? {
        val body = callLocalApi("GET", "/localapi/v0/status", null) ?: return null
        return try { JSONObject(body) } catch (e: Exception) { null }
    }

    private fun callLocalApi(method: String, endpoint: String, body: String?): String? {
        val tsApp = app ?: return null
        val inputStream = body?.let {
            InputStreamAdapter(it.byteInputStream())
        }
        val response = tsApp.callLocalAPI(API_TIMEOUT_MS, method, endpoint, inputStream)
        val statusCode = response.statusCode()
        val responseBody = String(response.bodyBytes())
        if (statusCode !in 200..299) {
            Log.w(TAG, "LocalAPI $method $endpoint returned $statusCode: $responseBody")
        }
        return responseBody
    }
}
