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
import org.localforge.alicia.feature.settings.components.ResponseLength
import org.localforge.alicia.feature.settings.components.ThemeOption
import javax.inject.Inject

data class VoiceOption(
    val id: String,
    val name: String,
    val description: String
)

data class AppSettings(
    val wakeWord: String = "alicia",
    val wakeWordSensitivity: Float = 0.7f,
    val volumeButtonEnabled: Boolean = false,
    val shakeEnabled: Boolean = false,
    val floatingButtonEnabled: Boolean = false,
    val availableVoices: List<VoiceOption> = emptyList(),
    val selectedVoice: String = "default",
    val speechRate: Float = 1.0f,
    val audioOutputEnabled: Boolean = true,
    val responseLength: ResponseLength = ResponseLength.BALANCED,
    val autoPinMemories: Boolean = true,
    val confirmDeleteMemories: Boolean = true,
    val showRelevanceScores: Boolean = true,
    val theme: ThemeOption = ThemeOption.SYSTEM,
    val compactMode: Boolean = false,
    val reduceMotion: Boolean = false,
    val soundNotifications: Boolean = true,
    val messagePreviews: Boolean = true,
    val serverUrl: String = "",
    val isConnected: Boolean = false,
    val lastConnectionCheck: Long = 0,
    val saveHistory: Boolean = true
)

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val settingsRepository: SettingsRepository,
    private val voiceRepository: VoiceRepository,
    private val conversationRepository: ConversationRepository
) : ViewModel() {

    private val _settings = MutableStateFlow(AppSettings())
    val settings: StateFlow<AppSettings> = _settings.asStateFlow()

    init {
        loadSettings()
    }

    private fun loadSettings() {
        viewModelScope.launch {
            val voicesResult = voiceRepository.getAvailableVoices()
            val availableVoices = voicesResult.getOrNull()?.map { voice ->
                VoiceOption(
                    id = voice.id,
                    name = voice.name,
                    description = voice.description ?: "${voice.gender} voice"
                )
            } ?: emptyList()

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

    fun setWakeWord(wakeWord: String) {
        viewModelScope.launch {
            settingsRepository.setWakeWord(wakeWord)
        }
    }

    fun setWakeWordSensitivity(sensitivity: Float) {
        viewModelScope.launch {
            settingsRepository.setWakeWordSensitivity(sensitivity)
        }
    }

    fun setVolumeButtonEnabled(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setVolumeButtonEnabled(enabled)
        }
    }

    fun setShakeEnabled(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setShakeEnabled(enabled)
        }
    }

    fun setFloatingButtonEnabled(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setFloatingButtonEnabled(enabled)
        }
    }

    fun setVoice(voiceId: String) {
        viewModelScope.launch {
            voiceRepository.setVoice(voiceId)
        }
    }

    fun setSpeechRate(rate: Float) {
        viewModelScope.launch {
            settingsRepository.setSpeechRate(rate)
        }
    }

    fun setServerUrl(url: String) {
        viewModelScope.launch {
            settingsRepository.setServerUrl(url)
        }
    }

    fun testConnection() {
        viewModelScope.launch {
            try {
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

    fun setSaveHistory(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setSaveHistory(enabled)
        }
    }

    fun clearHistory() {
        viewModelScope.launch {
            conversationRepository.deleteAllConversations()
        }
    }

    fun setAudioOutputEnabled(enabled: Boolean) {
        _settings.update { it.copy(audioOutputEnabled = enabled) }
        // TODO: Persist to settings repository when added
    }

    fun setResponseLength(length: ResponseLength) {
        _settings.update { it.copy(responseLength = length) }
        // TODO: Persist to settings repository when added
    }

    fun setAutoPinMemories(enabled: Boolean) {
        _settings.update { it.copy(autoPinMemories = enabled) }
        // TODO: Persist to settings repository when added
    }

    fun setConfirmDeleteMemories(enabled: Boolean) {
        _settings.update { it.copy(confirmDeleteMemories = enabled) }
        // TODO: Persist to settings repository when added
    }

    fun setShowRelevanceScores(enabled: Boolean) {
        _settings.update { it.copy(showRelevanceScores = enabled) }
        // TODO: Persist to settings repository when added
    }

    fun setTheme(theme: ThemeOption) {
        _settings.update { it.copy(theme = theme) }
        // TODO: Persist to settings repository and apply theme change
    }

    fun setCompactMode(enabled: Boolean) {
        _settings.update { it.copy(compactMode = enabled) }
        // TODO: Persist to settings repository when added
    }

    fun setReduceMotion(enabled: Boolean) {
        _settings.update { it.copy(reduceMotion = enabled) }
        // TODO: Persist to settings repository when added
    }

    fun setSoundNotifications(enabled: Boolean) {
        _settings.update { it.copy(soundNotifications = enabled) }
        // TODO: Persist to settings repository when added
    }

    fun setMessagePreviews(enabled: Boolean) {
        _settings.update { it.copy(messagePreviews = enabled) }
        // TODO: Persist to settings repository when added
    }
}
