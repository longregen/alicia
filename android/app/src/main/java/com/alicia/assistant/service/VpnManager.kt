package com.alicia.assistant.service

import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import android.net.LinkProperties
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.net.VpnService
import android.os.Handler
import android.os.Looper
import android.util.Log
import com.alicia.assistant.model.ExitNode
import com.alicia.assistant.model.VpnSettings
import com.alicia.assistant.model.VpnState
import com.alicia.assistant.model.VpnStatus
import com.alicia.assistant.storage.PreferencesManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import java.util.concurrent.atomic.AtomicBoolean
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import libtailscale.Libtailscale
import org.json.JSONObject

object VpnManager {
    private const val TAG = "VpnManager"
    private const val API_TIMEOUT_MS = 10_000L

    private const val NOTIFY_INITIAL_STATE = 2L
    private const val NOTIFY_PREFS = 4L
    private const val NOTIFY_NETMAP = 8L
    private const val NOTIFY_INITIAL_HEALTH = 128L

    sealed class ConnectResult {
        data object Started : ConnectResult()
        data class NeedsPermission(val intent: Intent) : ConnectResult()
        data class Failed(val reason: String) : ConnectResult()
    }

    private val supervisorJob = SupervisorJob()
    private val scope = CoroutineScope(supervisorJob + Dispatchers.IO)

    private val _state = MutableStateFlow(VpnState())
    val state: StateFlow<VpnState> = _state.asStateFlow()

    @Volatile
    private var app: libtailscale.Application? = null
    private var notificationManager: libtailscale.NotificationManager? = null
    @Volatile
    private var isInitialized = false
    private lateinit var appContext: Context
    private val prefs: PreferencesManager by lazy { PreferencesManager(appContext) }
    private var connectJob: Job? = null
    private val registrationInProgress = AtomicBoolean(false)
    private var networkCallback: ConnectivityManager.NetworkCallback? = null
    private val dnsDebounceHandler = Handler(Looper.getMainLooper())
    private var dnsDebounceRunnable: Runnable? = null

    @Synchronized
    fun init(context: Context) {
        if (isInitialized) return
        appContext = context.applicationContext
        Log.i(TAG, "Initializing VPN manager")

        try {
            val filesDir = appContext.filesDir.absolutePath
            val directFileRoot = appContext.getDir("tailscale", Context.MODE_PRIVATE).absolutePath
            val appCtx = AppContextImpl(appContext)
            app = Libtailscale.start(filesDir, directFileRoot, false, appCtx)
            isInitialized = true
            Log.i(TAG, "libtailscale backend started")
            notifyNetworkAvailable()
            registerNetworkCallback()
            startNotificationWatcher()
        } catch (e: UnsupportedOperationException) {
            Log.w(TAG, "libtailscale not available (stubs only) - VPN features disabled")
            isInitialized = true
        } catch (e: Exception) {
            Log.e(TAG, "Failed to initialize libtailscale", e)
            _state.value = VpnState(status = VpnStatus.ERROR)
        }
    }

    /**
     * Notify the Go network monitor about the current active interface.
     * Go's net.Interfaces() fails on Android 11+ (netlink permission denied),
     * so the monitor starts with SetNetworkUp(false). Calling onDNSConfigChanged
     * triggers InjectEvent which re-evaluates state using the Java bridge
     * (AppContext.getInterfacesAsJson) instead of the broken netlink path.
     */
    private fun notifyNetworkAvailable() {
        try {
            val cm = appContext.getSystemService(ConnectivityManager::class.java)
            val lp = cm.getLinkProperties(cm.activeNetwork)
            val ifname = lp?.interfaceName
            if (ifname != null) {
                Log.i(TAG, "Notifying Go network monitor: interface=$ifname")
                Libtailscale.onDNSConfigChanged(ifname)
            } else {
                Log.w(TAG, "No active network interface to notify Go monitor")
            }
        } catch (e: Exception) {
            Log.w(TAG, "Failed to notify Go network monitor", e)
        }
    }

    /**
     * Register a persistent network callback so that any network change
     * (including after /start recreates the backend) triggers onDNSConfigChanged.
     * This complements the one-shot notifyNetworkAvailable() call.
     *
     * Uses a NetworkRequest with INTERNET + NOT_VPN capabilities for per-network
     * tracking (instead of registerDefaultNetworkCallback) so we receive callbacks
     * for all non-VPN internet-capable networks. Includes onLost to notify the Go
     * backend when a network disappears. DNS change notifications are debounced
     * by 300ms to prevent rapid WiFi/cellular toggling from overwhelming the
     * Go-side buffer=1 channel.
     */
    private fun registerNetworkCallback() {
        try {
            val cm = appContext.getSystemService(ConnectivityManager::class.java)
            val request = NetworkRequest.Builder()
                .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                .addCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN)
                .build()
            val callback = object : ConnectivityManager.NetworkCallback() {
                override fun onAvailable(network: Network) {
                    val lp = cm.getLinkProperties(network)
                    val ifname = lp?.interfaceName ?: return
                    Log.d(TAG, "NetworkCallback onAvailable: $ifname")
                    notifyDnsChanged(ifname)
                }

                override fun onLinkPropertiesChanged(network: Network, lp: LinkProperties) {
                    val ifname = lp.interfaceName ?: return
                    Log.d(TAG, "NetworkCallback onLinkPropertiesChanged: $ifname")
                    notifyDnsChanged(ifname)
                }

                override fun onLost(network: Network) {
                    Log.d(TAG, "NetworkCallback onLost: $network")
                    notifyDnsChanged("")
                }
            }
            cm.registerNetworkCallback(request, callback)
            networkCallback = callback
            Log.i(TAG, "Network callback registered")
        } catch (e: Exception) {
            Log.w(TAG, "Failed to register network callback", e)
        }
    }

    /**
     * Debounced DNS change notification. Cancels any pending notification and
     * schedules a new one after 300ms. This prevents rapid WiFi/cellular toggling
     * from overwhelming the Go-side buffer=1 channel.
     */
    private fun notifyDnsChanged(ifname: String) {
        dnsDebounceRunnable?.let { dnsDebounceHandler.removeCallbacks(it) }
        val runnable = Runnable {
            try {
                Libtailscale.onDNSConfigChanged(ifname)
            } catch (e: Exception) {
                Log.w(TAG, "Failed to notify DNS config changed", e)
            }
        }
        dnsDebounceRunnable = runnable
        dnsDebounceHandler.postDelayed(runnable, 300)
    }

    private fun startNotificationWatcher() {
        val tsApp = app ?: return
        val mask = NOTIFY_INITIAL_STATE or NOTIFY_PREFS or NOTIFY_NETMAP or NOTIFY_INITIAL_HEALTH
        try {
            notificationManager = tsApp.watchNotifications(mask) { notification ->
                try {
                    val json = JSONObject(String(notification, Charsets.UTF_8))
                    json.optInt("State", -1).takeIf { it >= 0 }?.let { stateInt ->
                        val newStatus = when (stateInt) {
                            1 -> VpnStatus.IN_USE_OTHER_USER
                            3 -> VpnStatus.PENDING_APPROVAL
                            in 0..4 -> VpnStatus.DISCONNECTED
                            5 -> VpnStatus.CONNECTING
                            6 -> VpnStatus.CONNECTED
                            else -> null
                        }
                        if (newStatus != null && newStatus != _state.value.status) {
                            _state.value = _state.value.copy(status = newStatus)
                            if (newStatus == VpnStatus.CONNECTED) {
                                scope.launch { refreshConnectionInfo() }
                            }
                        }
                    }
                    // Parse health warnings
                    json.optJSONObject("Health")?.let { health ->
                        val warnings = health.optJSONArray("Warnings")
                        val warningText = if (warnings != null && warnings.length() > 0) {
                            val first = warnings.getJSONObject(0)
                            first.optString("Text", first.optString("Title", null))
                        } else null
                        if (_state.value.healthWarning != warningText) {
                            _state.value = _state.value.copy(healthWarning = warningText)
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

        // Parse current exit node from backend status
        val exitNodeId = status.optJSONObject("Prefs")?.optString("ExitNodeID", "")
        val exitNode = if (!exitNodeId.isNullOrEmpty()) {
            val peers = status.optJSONObject("Peer")
            var found: ExitNode? = null
            if (peers != null) {
                for (key in peers.keys()) {
                    val peer = peers.getJSONObject(key)
                    if (peer.optString("ID", "") == exitNodeId) {
                        val hostName = peer.optString("HostName", "")
                        val location = peer.optJSONObject("Location")
                        found = ExitNode(
                            id = exitNodeId,
                            name = hostName,
                            location = location?.let {
                                "${it.optString("CountryCode", "")} - ${it.optString("City", "")}"
                            } ?: hostName,
                            online = peer.optBoolean("Online", false),
                            countryCode = location?.optString("CountryCode", "") ?: ""
                        )
                        break
                    }
                }
            }
            found
        } else null

        _state.value = _state.value.copy(
            ipAddress = ipAddress,
            exitNode = exitNode ?: _state.value.exitNode,
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
                patchPrefs(JSONObject().put("WantRunning", true))
                // Check if cancelled between patchPrefs and service start
                ensureActive()

                val intent = Intent(context, AliciaVpnService::class.java).apply {
                    action = AliciaVpnService.ACTION_START_VPN
                }
                try {
                    context.startForegroundService(intent)
                } catch (e: android.app.ForegroundServiceStartNotAllowedException) {
                    Log.w(TAG, "Cannot start foreground service from background", e)
                    _state.value = VpnState(status = VpnStatus.ERROR)
                    return@launch
                }

                val settings = prefs.getVpnSettings()
                settings.selectedExitNodeId?.let { nodeId ->
                    patchPrefs(JSONObject().put("ExitNodeID", nodeId))
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to start VPN", e)
                _state.value = VpnState(status = VpnStatus.ERROR)
            }
        }
    }

    fun disconnect(context: Context) {
        val job = connectJob
        connectJob = null
        job?.cancel()
        scope.launch {
            // Wait for any in-flight connect coroutine to finish cancellation
            // before stopping, to prevent the service restart race
            job?.join()
            try {
                patchPrefs(JSONObject().put("WantRunning", false))
            } catch (e: Exception) {
                Log.w(TAG, "Failed to set WantRunning=false", e)
            }
            stopVpnInternal(context)
        }
    }

    private fun stopVpnInternal(context: Context) {
        _state.value = VpnState(status = VpnStatus.DISCONNECTED)
        try {
            context.startService(
                Intent(context, AliciaVpnService::class.java).apply {
                    action = AliciaVpnService.ACTION_STOP_VPN
                }
            )
        } catch (e: Exception) {
            Log.e(TAG, "Failed to send stop intent", e)
        }
    }

    fun setExitNode(nodeId: String, node: ExitNode? = null) {
        scope.launch {
            try {
                patchPrefs(JSONObject().put("ExitNodeID", nodeId))
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

    /**
     * Register with a Headscale server using a pre-auth key.
     *
     * Follows the official Tailscale Android app pattern:
     * 1. Ensure Go network monitor has current interface state (for unpause)
     * 2. POST /start with AuthKey + UpdatePrefs (creates new control client)
     * 3. POST /login-interactive (sets loginGoal and wakes auth routine)
     *
     * Key details:
     * - /start creates a new control client with the AuthKey stored in
     *   Direct.authKey, but does NOT call cc.Login() on Android because
     *   hasNodeKeyLocked()=false and confWantRunning=false (no daemon config).
     * - /login-interactive calls cc.Login(LoginInteractive) which sets
     *   loginGoal and cancels authCtx, waking the auth routine.
     * - The auth routine then calls TryLogin which includes the AuthKey
     *   in the RegisterRequest sent to the control server.
     * - The control client must NOT be paused for auth to proceed.
     *   Pause depends on AnyInterfaceUp() which checks HaveV4/HaveV6
     *   (set from IP addresses reported by getInterfacesAsJson).
     */
    suspend fun loginWithAuthKey(context: Context, controlUrl: String, key: String): Boolean = withContext(Dispatchers.IO) {
        if (!registrationInProgress.compareAndSet(false, true)) {
            Log.w(TAG, "Registration already in progress, ignoring duplicate call")
            return@withContext false
        }
        try {
            kotlinx.coroutines.withTimeoutOrNull(60_000L) {
                // Ensure Go network monitor has up-to-date interface state so the
                // control client won't be paused. AnyInterfaceUp() requires HaveV4
                // or HaveV6, which depend on IP addresses from getInterfacesAsJson.
                notifyNetworkAvailable()
                kotlinx.coroutines.delay(500)

                val body = JSONObject().apply {
                    put("AuthKey", key)
                    put("UpdatePrefs", JSONObject().apply {
                        put("ControlURL", controlUrl)
                        put("WantRunning", true)
                    })
                }

                Log.i(TAG, "Calling /start with ControlURL=$controlUrl")
                if (callLocalApi("POST", "/localapi/v0/start", body.toString()) == null) {
                    Log.e(TAG, "/start failed")
                    return@withTimeoutOrNull false
                }

                // Brief wait for control client to initialize.
                kotlinx.coroutines.delay(500)

                // Kick the auth routine. /start alone does NOT call cc.Login()
                // on Android. /login-interactive sets loginGoal and cancels
                // authCtx, waking the auth routine to use our AuthKey.
                Log.i(TAG, "Calling /login-interactive to kick auth routine")
                if (callLocalApi("POST", "/localapi/v0/login-interactive", null) == null) {
                    Log.e(TAG, "/login-interactive failed")
                    return@withTimeoutOrNull false
                }

                // Poll for backend to progress past NeedsLogin.
                repeat(45) { pollIdx ->
                    kotlinx.coroutines.delay(1000)
                    val state = getBackendStatus()?.optString("BackendState", "")
                    Log.d(TAG, "Polling backend state: $state (poll $pollIdx)")
                    if (state == "Running" || state == "NeedsMachineAuth") return@withTimeoutOrNull true
                }
                Log.w(TAG, "Login failed: backend did not reach Running state")
                false
            } ?: run {
                Log.e(TAG, "Login timed out after 60 seconds")
                false
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to login with auth key", e)
            false
        } finally {
            registrationInProgress.set(false)
        }
    }

    suspend fun forgetDevice(context: Context) = withContext(Dispatchers.IO) {
        try {
            connectJob?.cancel()
            connectJob = null
            // Stop notification watcher - backend session is being destroyed
            notificationManager?.stop()
            notificationManager = null
            callLocalApi("POST", "/localapi/v0/logout", null)
            stopVpnInternal(context)
            prefs.saveVpnSettings(VpnSettings())
            // Clear Go backend's encrypted state to prevent stale ControlURL/node keys
            try {
                val masterKey = MasterKey.Builder(appContext)
                    .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
                    .build()
                EncryptedSharedPreferences.create(
                    appContext,
                    "tailscale_encrypted_prefs",
                    masterKey,
                    EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                    EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
                ).edit().clear().apply()
            } catch (e: Exception) {
                Log.w(TAG, "Failed to clear Go backend prefs", e)
            }
            Log.i(TAG, "Device forgotten and logged out")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to forget device", e)
        }
    }

    fun shutdown() {
        try {
            networkCallback?.let {
                val cm = appContext.getSystemService(ConnectivityManager::class.java)
                cm.unregisterNetworkCallback(it)
            }
        } catch (e: Exception) {
            Log.w(TAG, "Failed to unregister network callback", e)
        }
        networkCallback = null
        dnsDebounceRunnable?.let { dnsDebounceHandler.removeCallbacks(it) }
        dnsDebounceRunnable = null
        notificationManager?.stop()
        notificationManager = null
        connectJob?.cancel()
        supervisorJob.cancel()
    }

    internal fun updateState(newState: VpnState) {
        _state.value = newState
    }

    private fun patchPrefs(prefs: JSONObject) {
        callLocalApi("PATCH", "/localapi/v0/prefs", prefs.toString())
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
        val bytes = response.bodyBytes()
        val responseBody = if (bytes != null) String(bytes, Charsets.UTF_8) else ""
        if (statusCode !in 200..299) {
            Log.w(TAG, "LocalAPI $method $endpoint returned $statusCode: $responseBody")
            return null
        }
        return responseBody
    }
}
