package org.localforge.alicia.core.data.preferences

import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.core.floatPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey

/**
 * DataStore preference keys for Alicia app settings.
 */
object PreferencesKeys {
    // Wake word settings
    val WAKE_WORD = stringPreferencesKey("wake_word")
    val WAKE_WORD_SENSITIVITY = floatPreferencesKey("wake_word_sensitivity")

    // Activation methods
    val VOLUME_BUTTON_ENABLED = booleanPreferencesKey("volume_button_enabled")
    val SHAKE_ENABLED = booleanPreferencesKey("shake_enabled")
    val FLOATING_BUTTON_ENABLED = booleanPreferencesKey("floating_button_enabled")

    // Voice settings
    val SELECTED_VOICE = stringPreferencesKey("selected_voice")
    val SPEECH_RATE = floatPreferencesKey("speech_rate")

    // Server settings
    val SERVER_URL = stringPreferencesKey("server_url")

    // Privacy settings
    val SAVE_HISTORY = booleanPreferencesKey("save_history")

    // Auto-start settings
    val AUTO_START_ENABLED = booleanPreferencesKey("auto_start_enabled")
    val FLOATING_BUTTON_AUTO_START = booleanPreferencesKey("floating_button_auto_start")
    val WAKE_WORD_AUTO_START = booleanPreferencesKey("wake_word_auto_start")

    // Default values
    object Defaults {
        const val WAKE_WORD = "alicia"
        const val WAKE_WORD_SENSITIVITY = 0.7f
        const val VOLUME_BUTTON_ENABLED = true
        const val SHAKE_ENABLED = false
        const val FLOATING_BUTTON_ENABLED = false
        const val SELECTED_VOICE = "af_sarah"
        const val SPEECH_RATE = 1.0f

        /**
         * Default server URL.
         *
         * IMPORTANT: This is an empty string by default, requiring users to configure
         * their server URL before the app can connect. The localhost default was removed
         * as it only works in development and should not ship in production builds.
         *
         * For development: Set to "http://localhost:8080" or "http://10.0.2.2:8080" (Android emulator)
         * For production: Users must configure their actual server URL in settings
         */
        const val SERVER_URL = ""
        const val SAVE_HISTORY = true
        const val AUTO_START_ENABLED = true
        const val FLOATING_BUTTON_AUTO_START = false
        const val WAKE_WORD_AUTO_START = true
    }
}
