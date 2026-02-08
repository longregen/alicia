package com.alicia.assistant

import android.content.Intent
import android.os.Bundle
import android.view.View
import android.widget.TextView
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.service.VoiceAssistantService
import com.alicia.assistant.storage.PreferencesManager
import com.alicia.assistant.telemetry.AliciaTelemetry
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.materialswitch.MaterialSwitch
import com.google.android.material.slider.Slider
import com.google.android.material.textfield.TextInputEditText
import com.alicia.assistant.model.VpnStatus
import com.alicia.assistant.service.VpnManager
import kotlinx.coroutines.launch

class SettingsActivity : ComponentActivity() {

    private lateinit var preferencesManager: PreferencesManager
    private lateinit var wakeWordSwitch: MaterialSwitch
    private lateinit var voiceFeedbackSwitch: MaterialSwitch
    private lateinit var hapticFeedbackSwitch: MaterialSwitch
    private lateinit var ttsSpeedSlider: Slider
    private lateinit var ttsSpeedValue: TextView
    private lateinit var wakeWordInput: TextInputEditText

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_settings)

        preferencesManager = PreferencesManager(this)

        val toolbar = findViewById<MaterialToolbar>(R.id.toolbar)
        toolbar.setNavigationOnClickListener { finish() }

        wakeWordSwitch = findViewById(R.id.wakeWordSwitch)
        voiceFeedbackSwitch = findViewById(R.id.voiceFeedbackSwitch)
        hapticFeedbackSwitch = findViewById(R.id.hapticFeedbackSwitch)
        ttsSpeedSlider = findViewById(R.id.ttsSpeedSlider)
        ttsSpeedValue = findViewById(R.id.ttsSpeedValue)
        wakeWordInput = findViewById(R.id.wakeWordInput)

        loadSettings()
        setupListeners()
    }

    override fun onResume() {
        super.onResume()
        VoiceAssistantService.ensureRunning(this)

        // Update VPN status text
        val vpnStatusText = findViewById<TextView>(R.id.vpnSettingsStatus)
        when (VpnManager.state.value.status) {
            VpnStatus.CONNECTED -> vpnStatusText.text = getString(R.string.vpn_connected)
            VpnStatus.CONNECTING -> vpnStatusText.text = getString(R.string.vpn_connecting_status)
            VpnStatus.DISCONNECTED -> vpnStatusText.text = getString(R.string.vpn_status_off)
            VpnStatus.PENDING_APPROVAL -> vpnStatusText.text = getString(R.string.vpn_pending_approval)
            VpnStatus.ERROR -> vpnStatusText.text = getString(R.string.vpn_error)
        }
    }

    override fun onPause() {
        super.onPause()
        saveSettings()
    }

    private fun loadSettings() {
        lifecycleScope.launch {
            val settings = preferencesManager.getSettings()
            wakeWordSwitch.isChecked = settings.wakeWordEnabled
            voiceFeedbackSwitch.isChecked = settings.voiceFeedbackEnabled
            hapticFeedbackSwitch.isChecked = settings.hapticFeedbackEnabled
            ttsSpeedSlider.value = settings.ttsSpeed
            ttsSpeedValue.text = String.format("%.1fx", settings.ttsSpeed)
            wakeWordInput.setText(settings.wakeWord)
        }
    }

    private fun onSettingToggled(name: String, enabled: Boolean, action: (Boolean) -> Unit) {
        AliciaTelemetry.withSpan("settings.toggle.$name") { span ->
            span.setAttribute("setting", name)
            span.setAttribute("enabled", enabled)
            action(enabled)
        }
    }

    private fun setupListeners() {
        wakeWordSwitch.setOnCheckedChangeListener { _, isChecked ->
            onSettingToggled("wake_word_enabled", isChecked) {
                saveSettings()
                if (isChecked) {
                    VoiceAssistantService.ensureRunning(this)
                } else {
                    VoiceAssistantService.stop(this)
                }
            }
        }

        voiceFeedbackSwitch.setOnCheckedChangeListener { _, isChecked ->
            onSettingToggled("voice_feedback_enabled", isChecked) {
                saveSettings()
            }
        }
        hapticFeedbackSwitch.setOnCheckedChangeListener { _, isChecked ->
            onSettingToggled("haptic_feedback_enabled", isChecked) {
                saveSettings()
            }
        }
        ttsSpeedSlider.addOnChangeListener { _, value, _ ->
            ttsSpeedValue.text = String.format("%.1fx", value)
        }
        ttsSpeedSlider.addOnSliderTouchListener(object : Slider.OnSliderTouchListener {
            override fun onStartTrackingTouch(slider: Slider) {}
            override fun onStopTrackingTouch(slider: Slider) {
                onSettingToggled("tts_speed", true) {
                    saveSettings()
                }
            }
        })

        findViewById<View>(R.id.manageModelsButton).setOnClickListener {
            startActivity(Intent(this, ModelManagerActivity::class.java))
        }

        findViewById<View>(R.id.vpnSettingsCard).setOnClickListener {
            startActivity(Intent(this, VpnSettingsActivity::class.java))
        }

    }

    private fun saveSettings() {
        val wakeWord = wakeWordInput.text.toString().trim().ifEmpty { "alicia" }
        lifecycleScope.launch {
            val current = preferencesManager.getSettings()
            preferencesManager.saveSettings(current.copy(
                wakeWordEnabled = wakeWordSwitch.isChecked,
                wakeWord = wakeWord,
                voiceFeedbackEnabled = voiceFeedbackSwitch.isChecked,
                hapticFeedbackEnabled = hapticFeedbackSwitch.isChecked,
                ttsSpeed = ttsSpeedSlider.value
            ))
        }
    }
}
