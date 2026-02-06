package com.alicia.assistant.service

import android.app.Activity
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
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject

object VpnManager {
    private const val TAG = "VpnManager"
    const val VPN_PERMISSION_REQUEST_CODE = 9001

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val _state = MutableStateFlow(VpnState())
    val state: StateFlow<VpnState> = _state.asStateFlow()

    private var backend: Any? = null // libtailscale backend handle
    private var isInitialized = false
    private var libtailscaleClass: Class<*>? = null
    private lateinit var appContext: Context
    private val prefs: PreferencesManager by lazy { PreferencesManager(appContext) }

    fun init(context: Context) {
        if (isInitialized) return
        isInitialized = true
        Log.i(TAG, "Initializing VPN manager")

        try {
            appContext = context.applicationContext
            val filesDir = appContext.filesDir.absolutePath
            val cacheDir = appContext.cacheDir.absolutePath

            // Try to load libtailscale classes via reflection to avoid hard compile dependency
            // when the AAR hasn't been built yet
            try {
                libtailscaleClass = Class.forName("com.tailscale.ipn.Libtailscale")
                val startMethod = libtailscaleClass?.getMethod("start", String::class.java, String::class.java)
                backend = startMethod?.invoke(null, filesDir, cacheDir)
                Log.i(TAG, "libtailscale backend started")
            } catch (e: ClassNotFoundException) {
                Log.w(TAG, "libtailscale not available - VPN features disabled")
            }

            scope.launch {
                val settings = prefs.getVpnSettings()
                if (settings.nodeRegistered) {
                    Log.i(TAG, "Node previously registered with Headscale")
                }
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to initialize VPN manager", e)
            _state.value = VpnState(status = VpnStatus.ERROR)
        }
    }

    fun connect(context: Context) {
        if (backend == null) {
            Log.w(TAG, "Cannot connect: libtailscale not initialized")
            _state.value = VpnState(status = VpnStatus.ERROR)
            return
        }

        val vpnIntent = VpnService.prepare(context)
        if (vpnIntent != null) {
            if (context is Activity) {
                context.startActivityForResult(vpnIntent, VPN_PERMISSION_REQUEST_CODE)
            } else {
                Log.w(TAG, "VPN permission required but no Activity context available")
                _state.value = VpnState(status = VpnStatus.ERROR)
            }
            return
        }

        startVpnService(context)
    }

    internal fun startVpnService(context: Context) {
        _state.value = _state.value.copy(status = VpnStatus.CONNECTING)

        scope.launch {
            try {
                val intent = Intent(context, AliciaVpnService::class.java).apply {
                    action = AliciaVpnService.ACTION_START_VPN
                }
                context.startForegroundService(intent)

                val settings = prefs.getVpnSettings()
                settings.selectedExitNodeId?.let { nodeId ->
                    setExitNode(nodeId)
                }

                var attempts = 0
                while (attempts < 30) {
                    val status = getBackendStatus()
                    if (status != null && status.optString("BackendState") == "Running") {
                        val selfNode = status.optJSONObject("Self")
                        val ipAddress = selfNode?.optJSONArray("TailscaleIPs")?.optString(0)
                        _state.value = VpnState(
                            status = VpnStatus.CONNECTED,
                            exitNode = _state.value.exitNode,
                            ipAddress = ipAddress,
                            since = System.currentTimeMillis()
                        )
                        Log.i(TAG, "VPN connected, IP: $ipAddress")
                        return@launch
                    }
                    attempts++
                    delay(1000)
                }

                Log.w(TAG, "VPN connection timed out")
                _state.value = VpnState(status = VpnStatus.ERROR)
            } catch (e: Exception) {
                Log.e(TAG, "Failed to connect VPN", e)
                _state.value = VpnState(status = VpnStatus.ERROR)
            }
        }
    }

    fun disconnect(context: Context) {
        scope.launch {
            try {
                val intent = Intent(context, AliciaVpnService::class.java).apply {
                    action = AliciaVpnService.ACTION_STOP_VPN
                }
                context.startService(intent)

                _state.value = VpnState(status = VpnStatus.DISCONNECTED)
                Log.i(TAG, "VPN disconnected")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to disconnect VPN", e)
            }
        }
    }

    fun setExitNode(nodeId: String) {
        scope.launch {
            try {
                val prefsJson = JSONObject().apply {
                    put("ExitNodeID", nodeId)
                }
                callLocalApi("POST", "/localapi/v0/prefs", prefsJson.toString())

                val nodes = getExitNodes()
                val selectedNode = nodes.find { it.id == nodeId }
                _state.value = _state.value.copy(exitNode = selectedNode)
                Log.i(TAG, "Exit node set to: ${selectedNode?.name ?: nodeId}")
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

            val keys = peers.keys()
            while (keys.hasNext()) {
                val key = keys.next()
                val peer = peers.getJSONObject(key)
                val exitNodeOption = peer.optBoolean("ExitNodeOption", false)
                if (exitNodeOption) {
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
            }
            nodes
        } catch (e: Exception) {
            Log.e(TAG, "Failed to get exit nodes", e)
            emptyList()
        }
    }

    fun loginWithAuthKey(key: String) {
        scope.launch {
            try {
                callLocalApi("POST", "/localapi/v0/login?key=$key", "")
                Log.i(TAG, "Login with auth key initiated")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to login with auth key", e)
            }
        }
    }

    fun loginWithUrl(url: String) {
        scope.launch {
            try {
                val body = JSONObject().apply {
                    put("ControlURL", url)
                }
                callLocalApi("POST", "/localapi/v0/prefs", body.toString())
                Log.i(TAG, "Control URL set to: $url")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to set login URL", e)
            }
        }
    }

    suspend fun isNodeRegistered(): Boolean = withContext(Dispatchers.IO) {
        try {
            val status = getBackendStatus()
            val state = status?.optString("BackendState", "")
            state == "Running"
        } catch (e: Exception) {
            false
        }
    }

    suspend fun forgetDevice(context: Context) = withContext(Dispatchers.IO) {
        try {
            callLocalApi("POST", "/localapi/v0/logout", "")
            disconnect(context)

            prefs.saveVpnSettings(VpnSettings())

            _state.value = VpnState(status = VpnStatus.DISCONNECTED)
            Log.i(TAG, "Device forgotten and logged out")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to forget device", e)
        }
    }

    internal fun updateState(newState: VpnState) {
        _state.value = newState
    }

    private fun getBackendStatus(): JSONObject? {
        return try {
            val response = callLocalApi("GET", "/localapi/v0/status", null)
            if (response != null) JSONObject(response) else null
        } catch (e: Exception) {
            Log.e(TAG, "Failed to get backend status", e)
            null
        }
    }

    private fun callLocalApi(method: String, endpoint: String, body: String?): String? {
        return try {
            if (backend == null) return null
            val callMethod = backend!!.javaClass.getMethod(
                "callLocalAPI", String::class.java, String::class.java, String::class.java
            )
            callMethod.invoke(backend, method, endpoint, body ?: "") as? String
        } catch (e: Exception) {
            Log.e(TAG, "Local API call failed: $method $endpoint", e)
            null
        }
    }
}
