package org.localforge.alicia.core.data.preferences

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.emptyPreferences
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.catch
import kotlinx.coroutines.flow.map
import java.io.IOException

/**
 * Extension property to create a DataStore instance.
 */
val Context.settingsDataStore: DataStore<Preferences> by preferencesDataStore(
    name = "alicia_settings"
)

/**
 * DataStore wrapper for managing app settings.
 */
class SettingsDataStore(private val dataStore: DataStore<Preferences>) {

    /**
     * Wake word setting.
     */
    val wakeWord: Flow<String> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.WAKE_WORD] ?: PreferencesKeys.Defaults.WAKE_WORD
        }

    suspend fun setWakeWord(value: String) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.WAKE_WORD] = value
        }
    }

    /**
     * Wake word sensitivity (0.0 to 1.0).
     */
    val wakeWordSensitivity: Flow<Float> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.WAKE_WORD_SENSITIVITY]
                ?: PreferencesKeys.Defaults.WAKE_WORD_SENSITIVITY
        }

    suspend fun setWakeWordSensitivity(value: Float) {
        dataStore.edit { preferences ->
            // Clamp sensitivity to valid range [0.0, 1.0]
            preferences[PreferencesKeys.WAKE_WORD_SENSITIVITY] = value.coerceIn(0f, 1f)
        }
    }

    /**
     * Volume button activation enabled.
     */
    val volumeButtonEnabled: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.VOLUME_BUTTON_ENABLED]
                ?: PreferencesKeys.Defaults.VOLUME_BUTTON_ENABLED
        }

    suspend fun setVolumeButtonEnabled(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.VOLUME_BUTTON_ENABLED] = value
        }
    }

    /**
     * Shake to activate enabled.
     */
    val shakeEnabled: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.SHAKE_ENABLED] ?: PreferencesKeys.Defaults.SHAKE_ENABLED
        }

    suspend fun setShakeEnabled(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.SHAKE_ENABLED] = value
        }
    }

    /**
     * Floating button overlay enabled.
     */
    val floatingButtonEnabled: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.FLOATING_BUTTON_ENABLED]
                ?: PreferencesKeys.Defaults.FLOATING_BUTTON_ENABLED
        }

    suspend fun setFloatingButtonEnabled(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.FLOATING_BUTTON_ENABLED] = value
        }
    }

    /**
     * Selected voice ID for TTS.
     */
    val selectedVoice: Flow<String> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.SELECTED_VOICE] ?: PreferencesKeys.Defaults.SELECTED_VOICE
        }

    suspend fun setSelectedVoice(value: String) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.SELECTED_VOICE] = value
        }
    }

    /**
     * Speech rate for TTS (0.5 to 2.0).
     */
    val speechRate: Flow<Float> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.SPEECH_RATE] ?: PreferencesKeys.Defaults.SPEECH_RATE
        }

    suspend fun setSpeechRate(value: Float) {
        dataStore.edit { preferences ->
            // Clamp speech rate to valid range [0.5x, 2.0x]
            preferences[PreferencesKeys.SPEECH_RATE] = value.coerceIn(0.5f, 2.0f)
        }
    }

    /**
     * Server URL for Alicia backend.
     */
    val serverUrl: Flow<String> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.SERVER_URL] ?: PreferencesKeys.Defaults.SERVER_URL
        }

    suspend fun setServerUrl(value: String) {
        // Validate and normalize URL: trim whitespace, allow empty, require http/https prefix,
        // remove trailing slashes, and verify valid URL structure with java.net.URL
        val trimmedUrl = value.trim()

        // Allow empty string (user can clear the URL)
        if (trimmedUrl.isEmpty()) {
            dataStore.edit { preferences ->
                preferences[PreferencesKeys.SERVER_URL] = ""
            }
            return
        }

        // Basic URL validation
        require(trimmedUrl.startsWith("http://") || trimmedUrl.startsWith("https://")) {
            "Server URL must start with http:// or https://"
        }

        // Ensure no trailing slash for consistency
        val normalizedUrl = trimmedUrl.trimEnd('/')

        // Additional validation: check for valid URL structure
        try {
            java.net.URL(normalizedUrl)
        } catch (e: Exception) {
            throw IllegalArgumentException("Invalid URL format: ${e.message}")
        }

        dataStore.edit { preferences ->
            preferences[PreferencesKeys.SERVER_URL] = normalizedUrl
        }
    }

    /**
     * Save conversation history locally.
     */
    val saveHistory: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.SAVE_HISTORY] ?: PreferencesKeys.Defaults.SAVE_HISTORY
        }

    suspend fun setSaveHistory(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.SAVE_HISTORY] = value
        }
    }

    /**
     * Auto-start enabled on boot.
     */
    val autoStartEnabled: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.AUTO_START_ENABLED]
                ?: PreferencesKeys.Defaults.AUTO_START_ENABLED
        }

    suspend fun setAutoStartEnabled(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.AUTO_START_ENABLED] = value
        }
    }

    /**
     * Floating button auto-start on boot.
     */
    val floatingButtonAutoStart: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.FLOATING_BUTTON_AUTO_START]
                ?: PreferencesKeys.Defaults.FLOATING_BUTTON_AUTO_START
        }

    suspend fun setFloatingButtonAutoStart(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.FLOATING_BUTTON_AUTO_START] = value
        }
    }

    /**
     * Wake word detection auto-start on boot.
     */
    val wakeWordAutoStart: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.WAKE_WORD_AUTO_START]
                ?: PreferencesKeys.Defaults.WAKE_WORD_AUTO_START
        }

    suspend fun setWakeWordAutoStart(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.WAKE_WORD_AUTO_START] = value
        }
    }

    // ========== Appearance Settings ==========

    /**
     * Theme preference: "light", "dark", or "system".
     */
    val theme: Flow<String> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.THEME] ?: PreferencesKeys.Defaults.THEME
        }

    suspend fun setTheme(value: String) {
        require(value in listOf("light", "dark", "system")) {
            "Theme must be one of: light, dark, system"
        }
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.THEME] = value
        }
    }

    /**
     * Compact mode for smaller UI elements.
     */
    val compactMode: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.COMPACT_MODE] ?: PreferencesKeys.Defaults.COMPACT_MODE
        }

    suspend fun setCompactMode(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.COMPACT_MODE] = value
        }
    }

    /**
     * Reduce motion for accessibility.
     */
    val reduceMotion: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.REDUCE_MOTION] ?: PreferencesKeys.Defaults.REDUCE_MOTION
        }

    suspend fun setReduceMotion(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.REDUCE_MOTION] = value
        }
    }

    // ========== Audio Settings ==========

    /**
     * Audio output enabled.
     */
    val audioOutputEnabled: Flow<Boolean> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.AUDIO_OUTPUT_ENABLED] ?: PreferencesKeys.Defaults.AUDIO_OUTPUT_ENABLED
        }

    suspend fun setAudioOutputEnabled(value: Boolean) {
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.AUDIO_OUTPUT_ENABLED] = value
        }
    }

    // ========== Response Settings ==========

    /**
     * Response length preference: "concise", "balanced", or "detailed".
     */
    val responseLength: Flow<String> = dataStore.data
        .catch { exception ->
            if (exception is IOException) {
                emit(emptyPreferences())
            } else {
                throw exception
            }
        }
        .map { preferences ->
            preferences[PreferencesKeys.RESPONSE_LENGTH] ?: PreferencesKeys.Defaults.RESPONSE_LENGTH
        }

    suspend fun setResponseLength(value: String) {
        require(value in listOf("concise", "balanced", "detailed")) {
            "Response length must be one of: concise, balanced, detailed"
        }
        dataStore.edit { preferences ->
            preferences[PreferencesKeys.RESPONSE_LENGTH] = value
        }
    }

    /**
     * Clear all settings and restore defaults.
     */
    suspend fun clearAllSettings() {
        dataStore.edit { preferences ->
            preferences.clear()
        }
    }
}
