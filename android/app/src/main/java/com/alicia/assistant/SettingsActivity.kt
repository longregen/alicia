package com.alicia.assistant

import android.content.Intent
import android.os.Bundle
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.service.VoiceAssistantService
import com.alicia.assistant.storage.PreferencesManager
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.materialswitch.MaterialSwitch
import com.google.android.material.slider.Slider
import com.google.android.material.textfield.TextInputEditText
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

    private fun setupListeners() {
        wakeWordSwitch.setOnCheckedChangeListener { _, isChecked ->
            saveSettings()
            if (isChecked) {
                VoiceAssistantService.ensureRunning(this)
            } else {
                VoiceAssistantService.stop(this)
            }
        }

        voiceFeedbackSwitch.setOnCheckedChangeListener { _, _ -> saveSettings() }
        hapticFeedbackSwitch.setOnCheckedChangeListener { _, _ -> saveSettings() }
        ttsSpeedSlider.addOnChangeListener { _, value, _ ->
            ttsSpeedValue.text = String.format("%.1fx", value)
            saveSettings()
        }

        findViewById<android.view.View>(R.id.manageModelsButton).setOnClickListener {
            startActivity(Intent(this, ModelManagerActivity::class.java))
        }

    }

    private fun saveSettings() {
        val wakeWord = wakeWordInput.text?.toString()?.trim()?.ifEmpty { "alicia" } ?: "alicia"
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
