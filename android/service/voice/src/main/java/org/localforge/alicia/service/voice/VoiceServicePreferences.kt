package org.localforge.alicia.service.voice

import android.content.Context
import android.content.SharedPreferences
import androidx.core.content.edit
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Manages preferences for the voice service.
 * Provides type-safe access to voice service settings.
 */
@Singleton
class VoiceServicePreferences @Inject constructor(
    @ApplicationContext context: Context
) {
    private val sharedPreferences: SharedPreferences = context.getSharedPreferences(
        PREFS_NAME,
        Context.MODE_PRIVATE
    )

    private val _settings = MutableStateFlow(loadSettings())
    val settings: StateFlow<VoiceSettings> = _settings.asStateFlow()

    init {
        // Listen for preference changes
        sharedPreferences.registerOnSharedPreferenceChangeListener { _, _ ->
            _settings.value = loadSettings()
        }
    }

    /**
     * Get the configured wake word.
     */
    fun getWakeWord(): WakeWordDetector.WakeWord {
        val wakeWordName = sharedPreferences.getString(KEY_WAKE_WORD, DEFAULT_WAKE_WORD)
        return try {
            WakeWordDetector.WakeWord.valueOf(wakeWordName ?: DEFAULT_WAKE_WORD)
        } catch (e: IllegalArgumentException) {
            WakeWordDetector.WakeWord.ALICIA
        }
    }

    /**
     * Set the wake word.
     */
    fun setWakeWord(wakeWord: WakeWordDetector.WakeWord) {
        sharedPreferences.edit {
            putString(KEY_WAKE_WORD, wakeWord.name)
        }
    }

    /**
     * Get wake word sensitivity (0.0 to 1.0).
     */
    fun getWakeWordSensitivity(): Float {
        return sharedPreferences.getFloat(KEY_WAKE_WORD_SENSITIVITY, DEFAULT_SENSITIVITY)
    }

    /**
     * Set wake word sensitivity.
     */
    fun setWakeWordSensitivity(sensitivity: Float) {
        sharedPreferences.edit {
            putFloat(KEY_WAKE_WORD_SENSITIVITY, sensitivity.coerceIn(0f, 1f))
        }
    }

    /**
     * Check if the service should auto-start on boot.
     */
    fun isAutoStartEnabled(): Boolean {
        return sharedPreferences.getBoolean(KEY_AUTO_START, DEFAULT_AUTO_START)
    }

    /**
     * Set auto-start on boot.
     */
    fun setAutoStartEnabled(enabled: Boolean) {
        sharedPreferences.edit {
            putBoolean(KEY_AUTO_START, enabled)
        }
    }

    /**
     * Check if battery optimization is enabled.
     */
    fun isBatteryOptimizationEnabled(): Boolean {
        return sharedPreferences.getBoolean(KEY_BATTERY_OPTIMIZATION, DEFAULT_BATTERY_OPTIMIZATION)
    }

    /**
     * Set battery optimization.
     */
    fun setBatteryOptimizationEnabled(enabled: Boolean) {
        sharedPreferences.edit {
            putBoolean(KEY_BATTERY_OPTIMIZATION, enabled)
        }
    }

    /**
     * Get silence detection threshold in milliseconds.
     */
    fun getSilenceThreshold(): Long {
        return sharedPreferences.getLong(KEY_SILENCE_THRESHOLD, DEFAULT_SILENCE_THRESHOLD)
    }

    /**
     * Set silence detection threshold.
     */
    fun setSilenceThreshold(thresholdMs: Long) {
        sharedPreferences.edit {
            putLong(KEY_SILENCE_THRESHOLD, thresholdMs.coerceAtLeast(500L))
        }
    }

    /**
     * Check if haptic feedback is enabled.
     */
    fun isHapticFeedbackEnabled(): Boolean {
        return sharedPreferences.getBoolean(KEY_HAPTIC_FEEDBACK, DEFAULT_HAPTIC_FEEDBACK)
    }

    /**
     * Set haptic feedback.
     */
    fun setHapticFeedbackEnabled(enabled: Boolean) {
        sharedPreferences.edit {
            putBoolean(KEY_HAPTIC_FEEDBACK, enabled)
        }
    }

    /**
     * Get server URL for LiveKit/API.
     */
    fun getServerUrl(): String {
        return sharedPreferences.getString(KEY_SERVER_URL, DEFAULT_SERVER_URL) ?: DEFAULT_SERVER_URL
    }

    /**
     * Set server URL.
     */
    fun setServerUrl(url: String) {
        sharedPreferences.edit {
            putString(KEY_SERVER_URL, url)
        }
    }

    /**
     * Check if the service is enabled.
     */
    fun isServiceEnabled(): Boolean {
        return sharedPreferences.getBoolean(KEY_SERVICE_ENABLED, DEFAULT_SERVICE_ENABLED)
    }

    /**
     * Set service enabled state.
     */
    fun setServiceEnabled(enabled: Boolean) {
        sharedPreferences.edit {
            putBoolean(KEY_SERVICE_ENABLED, enabled)
        }
    }

    /**
     * Reset all settings to defaults.
     */
    fun resetToDefaults() {
        sharedPreferences.edit {
            clear()
        }
    }

    private fun loadSettings(): VoiceSettings {
        return VoiceSettings(
            wakeWord = getWakeWord(),
            wakeWordSensitivity = getWakeWordSensitivity(),
            autoStartEnabled = isAutoStartEnabled(),
            batteryOptimizationEnabled = isBatteryOptimizationEnabled(),
            silenceThreshold = getSilenceThreshold(),
            hapticFeedbackEnabled = isHapticFeedbackEnabled(),
            serverUrl = getServerUrl(),
            serviceEnabled = isServiceEnabled()
        )
    }

    companion object {
        private const val PREFS_NAME = "voice_service_prefs"

        // Preference keys
        private const val KEY_WAKE_WORD = "wake_word"
        private const val KEY_WAKE_WORD_SENSITIVITY = "wake_word_sensitivity"
        private const val KEY_AUTO_START = "auto_start"
        private const val KEY_BATTERY_OPTIMIZATION = "battery_optimization"
        private const val KEY_SILENCE_THRESHOLD = "silence_threshold"
        private const val KEY_HAPTIC_FEEDBACK = "haptic_feedback"
        private const val KEY_SERVER_URL = "server_url"
        private const val KEY_SERVICE_ENABLED = "service_enabled"

        // Default values
        private const val DEFAULT_WAKE_WORD = "ALICIA"
        private const val DEFAULT_SENSITIVITY = 0.5f
        private const val DEFAULT_AUTO_START = false
        private const val DEFAULT_BATTERY_OPTIMIZATION = true
        private const val DEFAULT_SILENCE_THRESHOLD = 1500L
        private const val DEFAULT_HAPTIC_FEEDBACK = true
        private const val DEFAULT_SERVER_URL = "http://localhost:8080"
        private const val DEFAULT_SERVICE_ENABLED = true
    }
}

/**
 * Data class representing voice service settings.
 */
data class VoiceSettings(
    val wakeWord: WakeWordDetector.WakeWord,
    val wakeWordSensitivity: Float,
    val autoStartEnabled: Boolean,
    val batteryOptimizationEnabled: Boolean,
    val silenceThreshold: Long,
    val hapticFeedbackEnabled: Boolean,
    val serverUrl: String,
    val serviceEnabled: Boolean
)
