package org.localforge.alicia.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import org.localforge.alicia.core.domain.repository.ConversationRepository
import org.localforge.alicia.core.domain.repository.SettingsRepository
import org.localforge.alicia.core.domain.repository.VoiceRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * Represents a voice option for text-to-speech.
 *
 * @property id Unique identifier for the voice
 * @property name Display name of the voice
 * @property description Human-readable description of the voice characteristics
 */
data class VoiceOption(
    val id: String,
    val name: String,
    val description: String
)

/**
 * Application settings data class.
 *
 * @property wakeWord Selected wake word identifier
 * @property wakeWordSensitivity Sensitivity threshold for wake word detection (0.0 to 1.0)
 * @property volumeButtonEnabled Whether volume button activation is enabled
 * @property shakeEnabled Whether shake-to-activate is enabled
 * @property floatingButtonEnabled Whether floating button overlay is enabled
 * @property availableVoices List of available voice options
 * @property selectedVoice Currently selected voice identifier
 * @property speechRate Speech rate multiplier (1.0 is normal speed)
 * @property serverUrl Backend server URL
 * @property isConnected Current connection status to the server
 * @property lastConnectionCheck Timestamp of last connection check
 * @property saveHistory Whether conversation history should be saved
 */
data class AppSettings(
    // Activation settings
    val wakeWord: String = "alicia",
    val wakeWordSensitivity: Float = 0.7f,
    val volumeButtonEnabled: Boolean = false,
    val shakeEnabled: Boolean = false,
    val floatingButtonEnabled: Boolean = false,

    // Voice settings
    val availableVoices: List<VoiceOption> = emptyList(),
    val selectedVoice: String = "default",
    val speechRate: Float = 1.0f,

    // Server settings
    val serverUrl: String = "",
    val isConnected: Boolean = false,
    val lastConnectionCheck: Long = 0,

    // Privacy settings
    val saveHistory: Boolean = true
)

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val settingsRepository: SettingsRepository,
    private val voiceRepository: VoiceRepository,
    private val conversationRepository: ConversationRepository
    // LiveKitManager is not injected here as voice features are handled by VoiceController
) : ViewModel() {

    private val _settings = MutableStateFlow(AppSettings())
    val settings: StateFlow<AppSettings> = _settings.asStateFlow()

    init {
        loadSettings()
    }

    /**
     * Loads settings from repository and combines them into the UI state.
     * Connection status is intentionally ephemeral - always starts disconnected.
     * Use testConnection() to verify server connectivity.
     */
    private fun loadSettings() {
        viewModelScope.launch {
            // Fetch available voices from repository
            val voicesResult = voiceRepository.getAvailableVoices()
            val availableVoices = voicesResult.getOrNull()?.map { voice ->
                VoiceOption(
                    id = voice.id,
                    name = voice.name,
                    description = voice.description ?: "${voice.gender} voice"
                )
            } ?: emptyList()

            // Combine all settings flows
            combine(
                settingsRepository.wakeWord,
                settingsRepository.wakeWordSensitivity,
                settingsRepository.volumeButtonEnabled,
                settingsRepository.shakeEnabled,
                settingsRepository.floatingButtonEnabled,
                settingsRepository.selectedVoice,
                settingsRepository.speechRate,
                settingsRepository.serverUrl,
                settingsRepository.saveHistory
            ) { values: Array<Any?> ->
                val wakeWord = values[0] as String
                val sensitivity = values[1] as Float
                val volumeBtn = values[2] as Boolean
                val shake = values[3] as Boolean
                val floatingBtn = values[4] as Boolean
                val voice = values[5] as String
                val rate = values[6] as Float
                val url = values[7] as String
                val history = values[8] as Boolean
                AppSettings(
                    wakeWord = wakeWord,
                    wakeWordSensitivity = sensitivity,
                    volumeButtonEnabled = volumeBtn,
                    shakeEnabled = shake,
                    floatingButtonEnabled = floatingBtn,
                    availableVoices = availableVoices,
                    selectedVoice = voice,
                    speechRate = rate,
                    serverUrl = url,
                    saveHistory = history
                )
            }.collect { newSettings ->
                _settings.value = newSettings
            }
        }
    }

    /**
     * Sets the wake word for voice activation.
     *
     * @param wakeWord The wake word identifier to set
     */
    fun setWakeWord(wakeWord: String) {
        viewModelScope.launch {
            settingsRepository.setWakeWord(wakeWord)
        }
    }

    /**
     * Sets the wake word detection sensitivity.
     *
     * @param sensitivity Sensitivity value between 0.0 (least sensitive) and 1.0 (most sensitive)
     */
    fun setWakeWordSensitivity(sensitivity: Float) {
        viewModelScope.launch {
            settingsRepository.setWakeWordSensitivity(sensitivity)
        }
    }

    /**
     * Enables or disables volume button activation.
     *
     * Note: Accessibility service management should be done at the Activity level
     * by prompting the user to enable/disable it in Android Settings.
     *
     * @param enabled Whether volume button activation should be enabled
     */
    fun setVolumeButtonEnabled(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setVolumeButtonEnabled(enabled)
            // Note: Accessibility service management should be done at the Activity level
            // by prompting the user to enable/disable it in Android Settings.
            // The ViewModel only updates the preference here.
        }
    }

    /**
     * Enables or disables shake-to-activate.
     *
     * Note: Shake detector service lifecycle should be managed by a foreground service
     * that observes this setting.
     *
     * @param enabled Whether shake activation should be enabled
     */
    fun setShakeEnabled(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setShakeEnabled(enabled)
            // Note: Shake detector service lifecycle should be managed by a foreground service
            // that observes this setting. The service will start/stop based on this preference.
            // The ViewModel only updates the preference here.
        }
    }

    /**
     * Enables or disables the floating button overlay.
     *
     * Note: Floating button overlay service should be managed by a foreground service
     * that observes this setting and requires SYSTEM_ALERT_WINDOW permission.
     *
     * @param enabled Whether floating button should be enabled
     */
    fun setFloatingButtonEnabled(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setFloatingButtonEnabled(enabled)
            // Note: Floating button overlay service should be managed by a foreground service
            // that observes this setting and requires SYSTEM_ALERT_WINDOW permission.
            // The service will start/stop based on this preference.
            // The ViewModel only updates the preference here.
        }
    }

    /**
     * Sets the voice for text-to-speech in the voice repository.
     *
     * @param voiceId The voice identifier to set
     */
    fun setVoice(voiceId: String) {
        viewModelScope.launch {
            voiceRepository.setVoice(voiceId)
        }
    }

    /**
     * Sets the speech rate for text-to-speech.
     *
     * @param rate Speech rate multiplier (1.0 is normal speed)
     */
    fun setSpeechRate(rate: Float) {
        viewModelScope.launch {
            settingsRepository.setSpeechRate(rate)
        }
    }

    /**
     * Sets the backend server URL.
     *
     * @param url The server URL to set
     */
    fun setServerUrl(url: String) {
        viewModelScope.launch {
            settingsRepository.setServerUrl(url)
        }
    }

    /**
     * Tests the connection to the backend server.
     * Updates the connection status and timestamp in the UI state.
     */
    fun testConnection() {
        viewModelScope.launch {
            try {
                // Test connection by fetching conversations list
                val result = conversationRepository.syncWithServer()

                val isConnected = result.isSuccess

                _settings.update {
                    it.copy(
                        isConnected = isConnected,
                        lastConnectionCheck = System.currentTimeMillis()
                    )
                }
            } catch (e: Exception) {
                _settings.update {
                    it.copy(
                        isConnected = false,
                        lastConnectionCheck = System.currentTimeMillis()
                    )
                }
            }
        }
    }

    /**
     * Enables or disables conversation history saving.
     *
     * @param enabled Whether conversation history should be saved
     */
    fun setSaveHistory(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setSaveHistory(enabled)
        }
    }

    /**
     * Deletes all conversation history from the database.
     */
    fun clearHistory() {
        viewModelScope.launch {
            conversationRepository.deleteAllConversations()
        }
    }
}
