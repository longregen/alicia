package com.alicia.assistant

import android.content.Intent
import android.content.res.ColorStateList
import android.os.Bundle
import android.provider.Settings
import android.view.View
import android.widget.ImageView
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.result.contract.ActivityResultContracts
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.model.ExitNode
import com.alicia.assistant.model.VpnStatus
import com.alicia.assistant.service.VpnManager
import com.alicia.assistant.storage.PreferencesManager
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.button.MaterialButton
import com.google.android.material.color.MaterialColors
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.materialswitch.MaterialSwitch
import com.google.android.material.textfield.TextInputEditText
import kotlinx.coroutines.launch

class VpnSettingsActivity : ComponentActivity() {

    private lateinit var preferencesManager: PreferencesManager
    private lateinit var heroShieldIcon: ImageView
    private lateinit var heroStatusText: TextView
    private lateinit var heroDetailText: TextView
    private lateinit var connectButton: MaterialButton
    private lateinit var serverUrlInput: TextInputEditText
    private lateinit var authKeyInput: TextInputEditText
    private lateinit var autoConnectSwitch: MaterialSwitch

    /** Pending registration action after VPN permission is granted */
    private var pendingRegistration: (() -> Unit)? = null

    private val vpnPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { result ->
        if (result.resultCode == RESULT_OK) {
            VpnManager.startVpnService(this)
            // Resume pending registration if VPN permission was requested for it
            pendingRegistration?.invoke()
            pendingRegistration = null
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_vpn_settings)

        preferencesManager = PreferencesManager(this)
        heroShieldIcon = findViewById(R.id.heroShieldIcon)
        heroStatusText = findViewById(R.id.heroStatusText)
        heroDetailText = findViewById(R.id.heroDetailText)
        connectButton = findViewById(R.id.connectButton)
        serverUrlInput = findViewById(R.id.serverUrlInput)
        authKeyInput = findViewById(R.id.authKeyInput)
        autoConnectSwitch = findViewById(R.id.autoConnectSwitch)

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

        findViewById<View>(R.id.scanQrCard).setOnClickListener {
            startActivity(Intent(this, VpnQrScanActivity::class.java))
        }

        findViewById<MaterialButton>(R.id.registerButton).setOnClickListener {
            val serverUrl = serverUrlInput.text.toString().trim()
            val authKey = authKeyInput.text.toString().trim()

            if (serverUrl.isEmpty()) {
                Toast.makeText(this, R.string.vpn_server_url_required, Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            // The libtailscale backend pauses its control client until the VPN
            // service is running (requestVPN). Start the service before registering.
            val doRegister: () -> Unit = {
                val registerButton = findViewById<MaterialButton>(R.id.registerButton)
                registerButton.isEnabled = false
                registerButton.text = getString(R.string.vpn_registering)

                lifecycleScope.launch {
                    val registered = if (authKey.isNotEmpty()) {
                        VpnManager.loginWithAuthKey(this@VpnSettingsActivity, serverUrl, authKey)
                    } else {
                        true
                    }

                    preferencesManager.saveVpnSettings(
                        preferencesManager.getVpnSettings().copy(
                            headscaleUrl = serverUrl,
                            authKey = authKey,
                            nodeRegistered = registered
                        )
                    )
                    registerButton.isEnabled = true
                    registerButton.text = getString(R.string.vpn_register)

                    if (registered) {
                        updateSections(true)
                        Toast.makeText(this@VpnSettingsActivity, R.string.vpn_credentials_saved, Toast.LENGTH_SHORT).show()
                    } else {
                        Toast.makeText(this@VpnSettingsActivity, R.string.vpn_registration_failed, Toast.LENGTH_SHORT).show()
                    }
                }
            }

            // Ensure VPN service is running so the backend's control client is active
            when (val result = VpnManager.connect(this)) {
                is VpnManager.ConnectResult.NeedsPermission -> {
                    pendingRegistration = doRegister
                    vpnPermissionLauncher.launch(result.intent)
                }
                else -> doRegister()
            }
        }

        findViewById<View>(R.id.currentExitNodeCard).setOnClickListener {
            showExitNodePicker()
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

    private fun loadSettings() {
        lifecycleScope.launch {
            val settings = preferencesManager.getVpnSettings()
            serverUrlInput.setText(settings.headscaleUrl)
            if (settings.authKey.isNotEmpty()) {
                authKeyInput.setText(R.string.vpn_auth_key_placeholder)
                authKeyInput.isEnabled = false
            }
            autoConnectSwitch.isChecked = settings.autoConnect
            updateSections(settings.nodeRegistered)
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
                            MaterialColors.getColor(heroShieldIcon, com.google.android.material.R.attr.colorPrimary)
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
                    VpnStatus.ERROR -> {
                        heroShieldIcon.imageTintList = ColorStateList.valueOf(
                            MaterialColors.getColor(heroShieldIcon, com.google.android.material.R.attr.colorError)
                        )
                        heroStatusText.text = getString(R.string.vpn_error)
                        heroDetailText.text = getString(R.string.vpn_connection_failed)
                        connectButton.text = getString(R.string.vpn_retry)
                    }
                }
            }
        }
    }

    private fun updateSections(registered: Boolean) {
        findViewById<View>(R.id.setupSection).visibility = if (registered) View.GONE else View.VISIBLE
        findViewById<View>(R.id.exitNodeSection).visibility = if (registered) View.VISIBLE else View.GONE
    }

    private fun showExitNodePicker() {
        lifecycleScope.launch {
            val nodes = VpnManager.getExitNodes()
            if (nodes.isEmpty()) {
                Toast.makeText(this@VpnSettingsActivity, R.string.vpn_no_exit_nodes, Toast.LENGTH_SHORT).show()
                return@launch
            }

            val nodeNames = nodes.map { node ->
                val flag = countryCodeToFlag(node.countryCode)
                val status = if (node.online) "" else " (offline)"
                "$flag ${node.name} - ${node.location}$status"
            }.toTypedArray()

            MaterialAlertDialogBuilder(this@VpnSettingsActivity)
                .setTitle(R.string.vpn_select_exit_node)
                .setItems(nodeNames) { _, which ->
                    val selected = nodes[which]
                    VpnManager.setExitNode(selected.id, selected)
                    lifecycleScope.launch {
                        val settings = preferencesManager.getVpnSettings()
                        preferencesManager.saveVpnSettings(settings.copy(selectedExitNodeId = selected.id))
                    }
                    updateExitNodeDisplay(selected)
                }
                .show()
        }
    }

    private fun updateExitNodeDisplay(node: ExitNode) {
        findViewById<TextView>(R.id.exitNodeFlag).text = countryCodeToFlag(node.countryCode)
        findViewById<TextView>(R.id.exitNodeName).text = node.name
        findViewById<TextView>(R.id.exitNodeLocation).text = node.location
    }

    private fun countryCodeToFlag(countryCode: String): String {
        if (countryCode.length != 2) return ""
        val first = Character.codePointAt(countryCode.uppercase(), 0) - 0x41 + 0x1F1E6
        val second = Character.codePointAt(countryCode.uppercase(), 1) - 0x41 + 0x1F1E6
        return String(Character.toChars(first)) + String(Character.toChars(second))
    }

}
