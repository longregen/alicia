package com.alicia.assistant

import android.content.Intent
import android.content.res.ColorStateList
import android.os.Bundle
import android.provider.Settings
import android.util.Log
import android.util.TypedValue
import android.view.View
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.result.contract.ActivityResultContracts
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.model.ExitNode
import com.alicia.assistant.model.VpnStatus
import com.alicia.assistant.service.AliciaApiClient
import com.alicia.assistant.service.ApiClient
import com.alicia.assistant.service.VpnManager
import com.alicia.assistant.storage.PreferencesManager
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.button.MaterialButton
import com.google.android.material.card.MaterialCardView
import com.google.android.material.color.MaterialColors
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.materialswitch.MaterialSwitch
import com.google.android.material.progressindicator.LinearProgressIndicator
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject

class VpnSettingsActivity : ComponentActivity() {

    private lateinit var preferencesManager: PreferencesManager
    private lateinit var heroShieldIcon: ImageView
    private lateinit var heroStatusText: TextView
    private lateinit var heroDetailText: TextView
    private lateinit var connectButton: MaterialButton
    private lateinit var autoConnectSwitch: MaterialSwitch
    private lateinit var exitNodeListContainer: LinearLayout
    private lateinit var exitNodeNoneCheck: ImageView

    private var currentExitNodeId: String? = null

    /** Pending action after VPN permission is granted */
    private var pendingAction: (() -> Unit)? = null

    private val vpnPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { result ->
        if (result.resultCode == RESULT_OK) {
            VpnManager.startVpnService(this)
            pendingAction?.invoke()
        } else {
            findViewById<MaterialButton>(R.id.provisionButton).isEnabled = true
            findViewById<LinearProgressIndicator>(R.id.provisionProgress).visibility = View.GONE
        }
        pendingAction = null
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_vpn_settings)

        preferencesManager = PreferencesManager(this)
        heroShieldIcon = findViewById(R.id.heroShieldIcon)
        heroStatusText = findViewById(R.id.heroStatusText)
        heroDetailText = findViewById(R.id.heroDetailText)
        connectButton = findViewById(R.id.connectButton)
        autoConnectSwitch = findViewById(R.id.autoConnectSwitch)
        exitNodeListContainer = findViewById(R.id.exitNodeListContainer)
        exitNodeNoneCheck = findViewById(R.id.exitNodeNoneCheck)

        val toolbar = findViewById<MaterialToolbar>(R.id.toolbar)
        toolbar.setNavigationOnClickListener { finish() }

        setupUI()
        loadSettings()
        observeVpnState()
    }

    private fun setupUI() {
        connectButton.setOnClickListener {
            val state = VpnManager.state.value
            if (state.status == VpnStatus.CONNECTED || state.status == VpnStatus.CONNECTING) {
                VpnManager.disconnect(this)
            } else {
                when (val result = VpnManager.connect(this)) {
                    is VpnManager.ConnectResult.NeedsPermission ->
                        vpnPermissionLauncher.launch(result.intent)
                    else -> {}
                }
            }
        }

        findViewById<MaterialButton>(R.id.provisionButton).setOnClickListener {
            provisionAndConnect()
        }

        // "None" exit node card
        findViewById<View>(R.id.exitNodeNoneCard).setOnClickListener {
            VpnManager.setExitNode("", null)
            currentExitNodeId = null
            lifecycleScope.launch {
                val settings = preferencesManager.getVpnSettings()
                preferencesManager.saveVpnSettings(settings.copy(selectedExitNodeId = null))
            }
            updateExitNodeSelection()
        }

        autoConnectSwitch.setOnCheckedChangeListener { _, isChecked ->
            lifecycleScope.launch {
                val settings = preferencesManager.getVpnSettings()
                preferencesManager.saveVpnSettings(settings.copy(autoConnect = isChecked))
            }
        }

        findViewById<View>(R.id.alwaysOnCard).setOnClickListener {
            startActivity(Intent(Settings.ACTION_VPN_SETTINGS))
        }

        findViewById<View>(R.id.forgetDeviceCard).setOnClickListener {
            MaterialAlertDialogBuilder(this)
                .setTitle(R.string.vpn_forget_device)
                .setMessage(R.string.vpn_forget_confirm_message)
                .setPositiveButton(R.string.vpn_forget) { _, _ ->
                    lifecycleScope.launch {
                        VpnManager.forgetDevice(this@VpnSettingsActivity)
                        updateSections(false)
                        Toast.makeText(this@VpnSettingsActivity, R.string.vpn_device_forgotten, Toast.LENGTH_SHORT).show()
                    }
                }
                .setNegativeButton(R.string.vpn_cancel, null)
                .show()
        }
    }

    private fun provisionAndConnect() {
        val provisionButton = findViewById<MaterialButton>(R.id.provisionButton)
        val provisionProgress = findViewById<LinearProgressIndicator>(R.id.provisionProgress)

        provisionButton.isEnabled = false
        provisionProgress.visibility = View.VISIBLE

        lifecycleScope.launch {
            try {
                val (serverUrl, authKey) = fetchAuthKey()

                // Ensure VPN service is running before registration
                val doRegister: () -> Unit = {
                    lifecycleScope.launch {
                        val registered = VpnManager.loginWithAuthKey(
                            this@VpnSettingsActivity, serverUrl, authKey
                        )

                        provisionButton.isEnabled = true
                        provisionProgress.visibility = View.GONE

                        if (registered) {
                            preferencesManager.saveVpnSettings(
                                preferencesManager.getVpnSettings().copy(
                                    headscaleUrl = serverUrl,
                                    authKey = "",
                                    nodeRegistered = true
                                )
                            )
                            updateSections(true)
                            loadExitNodes()
                            Toast.makeText(
                                this@VpnSettingsActivity,
                                R.string.vpn_credentials_saved,
                                Toast.LENGTH_SHORT
                            ).show()
                        } else {
                            Toast.makeText(
                                this@VpnSettingsActivity,
                                R.string.vpn_provision_failed,
                                Toast.LENGTH_SHORT
                            ).show()
                        }
                    }
                }

                when (val result = VpnManager.connect(this@VpnSettingsActivity)) {
                    is VpnManager.ConnectResult.NeedsPermission -> {
                        pendingAction = doRegister
                        vpnPermissionLauncher.launch(result.intent)
                    }
                    else -> doRegister()
                }
            } catch (e: Exception) {
                provisionButton.isEnabled = true
                provisionProgress.visibility = View.GONE
                Toast.makeText(
                    this@VpnSettingsActivity,
                    R.string.vpn_provision_failed,
                    Toast.LENGTH_SHORT
                ).show()
            }
        }
    }

    private suspend fun fetchAuthKey(): Pair<String, String> = withContext(Dispatchers.IO) {
        val url = "${ApiClient.BASE_URL}/api/v1/vpn/auth-key"
        val body = "{}".toRequestBody("application/json".toMediaType())
        val request = Request.Builder()
            .url(url)
            .header("X-User-ID", AliciaApiClient.USER_ID)
            .header("Accept", "application/json")
            .post(body)
            .build()

        ApiClient.httpClient.newCall(request).execute().use { response ->
            val responseBody = response.body?.string() ?: ""
            if (!response.isSuccessful) {
                Log.e("VpnSettings", "Auth key request failed: ${response.code}: $responseBody")
                throw Exception("Auth key request failed: ${response.code}")
            }
            val json = JSONObject(responseBody)
            Pair(json.getString("server_url"), json.getString("auth_key"))
        }
    }

    private fun loadSettings() {
        lifecycleScope.launch {
            val settings = preferencesManager.getVpnSettings()
            autoConnectSwitch.isChecked = settings.autoConnect
            currentExitNodeId = settings.selectedExitNodeId
            updateSections(settings.nodeRegistered)
            if (settings.nodeRegistered) {
                loadExitNodes()
            }
        }
    }

    private fun loadExitNodes() {
        val exitNodeProgress = findViewById<LinearProgressIndicator>(R.id.exitNodeProgress)
        exitNodeProgress.visibility = View.VISIBLE
        exitNodeListContainer.removeAllViews()

        lifecycleScope.launch {
            val nodes = VpnManager.getExitNodes()
            val exitNodeId = VpnManager.getCurrentExitNodeId()
            if (exitNodeId != null) {
                currentExitNodeId = exitNodeId
            }

            exitNodeProgress.visibility = View.GONE

            for (node in nodes) {
                val card = createExitNodeCard(node)
                exitNodeListContainer.addView(card)
            }

            updateExitNodeSelection()
        }
    }

    private fun createExitNodeCard(node: ExitNode): MaterialCardView {
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

        val row = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = android.view.Gravity.CENTER_VERTICAL
            setPadding(dpToPx(16), dpToPx(16), dpToPx(16), dpToPx(16))
            val typedValue = TypedValue()
            context.theme.resolveAttribute(android.R.attr.selectableItemBackground, typedValue, true)
            setBackgroundResource(typedValue.resourceId)
        }

        val checkIcon = ImageView(this).apply {
            layoutParams = LinearLayout.LayoutParams(dpToPx(20), dpToPx(20)).apply {
                marginEnd = dpToPx(12)
            }
            setImageResource(R.drawable.ic_check)
            imageTintList = ColorStateList.valueOf(
                MaterialColors.getColor(this, android.R.attr.colorPrimary)
            )
            visibility = if (node.id == currentExitNodeId) View.VISIBLE else View.INVISIBLE
            tag = "check_${node.id}"
        }
        row.addView(checkIcon)

        val flag = countryCodeToFlag(node.countryCode)
        if (flag.isNotEmpty()) {
            val flagView = TextView(this).apply {
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply {
                    marginEnd = dpToPx(12)
                }
                textSize = 24f
                text = flag
            }
            row.addView(flagView)
        }

        val textCol = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
        }

        val nameView = TextView(this).apply {
            text = node.name
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
            setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurface))
        }
        textCol.addView(nameView)

        val locationView = TextView(this).apply {
            text = node.location
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTextColor(MaterialColors.getColor(this, com.google.android.material.R.attr.colorOnSurfaceVariant))
        }
        textCol.addView(locationView)

        row.addView(textCol)

        val statusDot = ImageView(this).apply {
            layoutParams = LinearLayout.LayoutParams(dpToPx(8), dpToPx(8))
            setImageResource(
                if (node.online) R.drawable.status_dot_online
                else R.drawable.status_dot_offline
            )
        }
        row.addView(statusDot)

        card.addView(row)

        card.setOnClickListener {
            VpnManager.setExitNode(node.id, node)
            currentExitNodeId = node.id
            lifecycleScope.launch {
                val settings = preferencesManager.getVpnSettings()
                preferencesManager.saveVpnSettings(settings.copy(selectedExitNodeId = node.id))
            }
            updateExitNodeSelection()
        }

        return card
    }

    private fun updateExitNodeSelection() {
        exitNodeNoneCheck.visibility = if (currentExitNodeId.isNullOrEmpty()) View.VISIBLE else View.INVISIBLE

        for (i in 0 until exitNodeListContainer.childCount) {
            val card = exitNodeListContainer.getChildAt(i) as? MaterialCardView ?: continue
            val row = card.getChildAt(0) as? LinearLayout ?: continue
            for (j in 0 until row.childCount) {
                val child = row.getChildAt(j)
                if (child is ImageView && child.tag is String) {
                    val tag = child.tag as String
                    if (tag.startsWith("check_")) {
                        val nodeId = tag.removePrefix("check_")
                        child.visibility = if (nodeId == currentExitNodeId) View.VISIBLE else View.INVISIBLE
                    }
                }
            }
        }
    }

    private fun observeVpnState() {
        lifecycleScope.launch {
            VpnManager.state.collect { vpnState ->
                when (vpnState.status) {
                    VpnStatus.DISCONNECTED -> {
                        heroShieldIcon.imageTintList = ColorStateList.valueOf(
                            MaterialColors.getColor(heroShieldIcon, com.google.android.material.R.attr.colorOnSurfaceVariant)
                        )
                        heroStatusText.text = getString(R.string.vpn_disconnected)
                        heroDetailText.text = getString(R.string.vpn_not_connected)
                        connectButton.text = getString(R.string.vpn_connect)
                    }
                    VpnStatus.CONNECTING -> {
                        heroShieldIcon.imageTintList = ColorStateList.valueOf(
                            MaterialColors.getColor(heroShieldIcon, android.R.attr.colorPrimary)
                        )
                        heroStatusText.text = getString(R.string.vpn_connecting_status)
                        heroDetailText.text = getString(R.string.vpn_setting_up_tunnel)
                        connectButton.text = getString(R.string.vpn_cancel)
                    }
                    VpnStatus.CONNECTED -> {
                        heroShieldIcon.imageTintList = ColorStateList.valueOf(
                            getColor(R.color.vpn_connected)
                        )
                        heroStatusText.text = getString(R.string.vpn_connected)
                        heroDetailText.text = vpnState.exitNode?.let { getString(R.string.vpn_connected_via, it.location) }
                            ?: vpnState.ipAddress?.let { "IP: $it" }
                            ?: getString(R.string.vpn_connected_to_tailnet)
                        connectButton.text = getString(R.string.vpn_disconnect)
                    }
                    VpnStatus.PENDING_APPROVAL -> {
                        heroShieldIcon.imageTintList = ColorStateList.valueOf(
                            MaterialColors.getColor(heroShieldIcon, com.google.android.material.R.attr.colorTertiary)
                        )
                        heroStatusText.text = getString(R.string.vpn_pending_approval)
                        heroDetailText.text = getString(R.string.vpn_pending_approval_detail)
                        connectButton.text = getString(R.string.vpn_disconnect)
                    }
                    VpnStatus.IN_USE_OTHER_USER -> {
                        heroShieldIcon.imageTintList = ColorStateList.valueOf(
                            MaterialColors.getColor(heroShieldIcon, android.R.attr.colorError)
                        )
                        heroStatusText.text = getString(R.string.vpn_in_use)
                        heroDetailText.text = getString(R.string.vpn_in_use_other_user)
                        connectButton.text = getString(R.string.vpn_connect)
                    }
                    VpnStatus.ERROR -> {
                        heroShieldIcon.imageTintList = ColorStateList.valueOf(
                            MaterialColors.getColor(heroShieldIcon, android.R.attr.colorError)
                        )
                        heroStatusText.text = getString(R.string.vpn_error)
                        heroDetailText.text = getString(R.string.vpn_connection_failed)
                        connectButton.text = getString(R.string.vpn_retry)
                    }
                }
                // Show health warning if present
                if (vpnState.healthWarning != null && vpnState.status == VpnStatus.CONNECTED) {
                    heroDetailText.text = vpnState.healthWarning
                }
            }
        }
    }

    private fun updateSections(registered: Boolean) {
        findViewById<View>(R.id.setupSection).visibility = if (registered) View.GONE else View.VISIBLE
        findViewById<View>(R.id.exitNodeSection).visibility = if (registered) View.VISIBLE else View.GONE
    }

    private fun countryCodeToFlag(countryCode: String): String {
        if (countryCode.length != 2) return ""
        val first = Character.codePointAt(countryCode.uppercase(), 0) - 0x41 + 0x1F1E6
        val second = Character.codePointAt(countryCode.uppercase(), 1) - 0x41 + 0x1F1E6
        return String(Character.toChars(first)) + String(Character.toChars(second))
    }

    private fun dpToPx(dp: Int): Int {
        return TypedValue.applyDimension(
            TypedValue.COMPLEX_UNIT_DIP, dp.toFloat(), resources.displayMetrics
        ).toInt()
    }
}
