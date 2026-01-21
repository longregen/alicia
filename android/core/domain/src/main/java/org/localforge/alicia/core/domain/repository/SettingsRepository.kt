package org.localforge.alicia.core.domain.repository

import kotlinx.coroutines.flow.Flow

interface SettingsRepository {
    val wakeWord: Flow<String>
    suspend fun setWakeWord(value: String)

    val wakeWordSensitivity: Flow<Float>
    suspend fun setWakeWordSensitivity(value: Float)

    val volumeButtonEnabled: Flow<Boolean>
    suspend fun setVolumeButtonEnabled(value: Boolean)

    val shakeEnabled: Flow<Boolean>
    suspend fun setShakeEnabled(value: Boolean)

    val floatingButtonEnabled: Flow<Boolean>
    suspend fun setFloatingButtonEnabled(value: Boolean)

    val selectedVoice: Flow<String>
    suspend fun setSelectedVoice(value: String)

    val speechRate: Flow<Float>
    suspend fun setSpeechRate(value: Float)

    val serverUrl: Flow<String>
    suspend fun setServerUrl(value: String)

    val saveHistory: Flow<Boolean>
    suspend fun setSaveHistory(value: Boolean)

    val autoStartEnabled: Flow<Boolean>
    suspend fun setAutoStartEnabled(value: Boolean)

    val floatingButtonAutoStart: Flow<Boolean>
    suspend fun setFloatingButtonAutoStart(value: Boolean)

    val wakeWordAutoStart: Flow<Boolean>
    suspend fun setWakeWordAutoStart(value: Boolean)

    val theme: Flow<String>
    suspend fun setTheme(value: String)

    val compactMode: Flow<Boolean>
    suspend fun setCompactMode(value: Boolean)

    val reduceMotion: Flow<Boolean>
    suspend fun setReduceMotion(value: Boolean)

    val audioOutputEnabled: Flow<Boolean>
    suspend fun setAudioOutputEnabled(value: Boolean)

    val responseLength: Flow<String>
    suspend fun setResponseLength(value: String)

    suspend fun clearAllSettings()
}
