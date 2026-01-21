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

val Context.settingsDataStore: DataStore<Preferences> by preferencesDataStore(
    name = "alicia_settings"
)

class SettingsDataStore(private val dataStore: DataStore<Preferences>) {

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
            preferences[PreferencesKeys.WAKE_WORD_SENSITIVITY] = value.coerceIn(0f, 1f)
        }
    }

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
            preferences[PreferencesKeys.SPEECH_RATE] = value.coerceIn(0.5f, 2.0f)
        }
    }

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
        val trimmedUrl = value.trim()

        if (trimmedUrl.isEmpty()) {
            dataStore.edit { preferences ->
                preferences[PreferencesKeys.SERVER_URL] = ""
            }
            return
        }

        require(trimmedUrl.startsWith("http://") || trimmedUrl.startsWith("https://")) {
            "Server URL must start with http:// or https://"
        }

        val normalizedUrl = trimmedUrl.trimEnd('/')

        try {
            java.net.URL(normalizedUrl)
        } catch (e: Exception) {
            throw IllegalArgumentException("Invalid URL format: ${e.message}")
        }

        dataStore.edit { preferences ->
            preferences[PreferencesKeys.SERVER_URL] = normalizedUrl
        }
    }

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

    suspend fun clearAllSettings() {
        dataStore.edit { preferences ->
            preferences.clear()
        }
    }
}
