package org.localforge.alicia.core.domain.repository

import kotlinx.coroutines.flow.Flow

/**
 * Repository interface for managing app settings using DataStore.
 */
interface SettingsRepository {
    /**
     * Wake word setting.
     */
    val wakeWord: Flow<String>
    /**
     * Set the wake word.
     * @param value Wake word string
     */
    suspend fun setWakeWord(value: String)

    /**
     * Wake word sensitivity (0.0 to 1.0).
     */
    val wakeWordSensitivity: Flow<Float>
    /**
     * Set the wake word sensitivity.
     * @param value Sensitivity value (0.0 to 1.0)
     */
    suspend fun setWakeWordSensitivity(value: Float)

    /**
     * Volume button activation enabled.
     */
    val volumeButtonEnabled: Flow<Boolean>
    /**
     * Set whether volume button activation is enabled.
     * @param value True to enable, false to disable
     */
    suspend fun setVolumeButtonEnabled(value: Boolean)

    /**
     * Shake to activate enabled.
     */
    val shakeEnabled: Flow<Boolean>
    /**
     * Set whether shake to activate is enabled.
     * @param value True to enable, false to disable
     */
    suspend fun setShakeEnabled(value: Boolean)

    /**
     * Floating button overlay enabled.
     */
    val floatingButtonEnabled: Flow<Boolean>
    /**
     * Set whether floating button overlay is enabled.
     * @param value True to enable, false to disable
     */
    suspend fun setFloatingButtonEnabled(value: Boolean)

    /**
     * Selected voice ID for TTS.
     */
    val selectedVoice: Flow<String>
    /**
     * Set the selected voice ID.
     * @param value Voice ID string
     */
    suspend fun setSelectedVoice(value: String)

    /**
     * Speech rate for TTS (0.5 to 2.0).
     */
    val speechRate: Flow<Float>
    /**
     * Set the speech rate.
     * @param value Speech rate (0.5 to 2.0)
     */
    suspend fun setSpeechRate(value: Float)

    /**
     * Server URL for Alicia backend.
     */
    val serverUrl: Flow<String>
    /**
     * Set the server URL.
     * @param value Server URL string
     */
    suspend fun setServerUrl(value: String)

    /**
     * Save conversation history locally.
     */
    val saveHistory: Flow<Boolean>
    /**
     * Set whether to save conversation history locally.
     * @param value True to save history, false otherwise
     */
    suspend fun setSaveHistory(value: Boolean)

    /**
     * Auto-start enabled on boot.
     */
    val autoStartEnabled: Flow<Boolean>
    /**
     * Set whether auto-start is enabled on boot.
     * @param value True to enable, false to disable
     */
    suspend fun setAutoStartEnabled(value: Boolean)

    /**
     * Floating button auto-start on boot.
     */
    val floatingButtonAutoStart: Flow<Boolean>
    /**
     * Set whether floating button auto-starts on boot.
     * @param value True to enable, false to disable
     */
    suspend fun setFloatingButtonAutoStart(value: Boolean)

    /**
     * Wake word detection auto-start on boot.
     */
    val wakeWordAutoStart: Flow<Boolean>
    /**
     * Set whether wake word detection auto-starts on boot.
     * @param value True to enable, false to disable
     */
    suspend fun setWakeWordAutoStart(value: Boolean)

    /**
     * Clear all settings and restore defaults.
     */
    suspend fun clearAllSettings()
}
