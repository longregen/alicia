package org.localforge.alicia.core.data.repository

import org.localforge.alicia.core.data.preferences.SettingsDataStore
import org.localforge.alicia.core.domain.repository.SettingsRepository
import kotlinx.coroutines.flow.Flow
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Implementation of SettingsRepository using DataStore for preferences.
 */
@Singleton
class SettingsRepositoryImpl @Inject constructor(
    private val settingsDataStore: SettingsDataStore
) : SettingsRepository {

    override val wakeWord: Flow<String>
        get() = settingsDataStore.wakeWord

    override suspend fun setWakeWord(value: String) {
        settingsDataStore.setWakeWord(value)
    }

    override val wakeWordSensitivity: Flow<Float>
        get() = settingsDataStore.wakeWordSensitivity

    override suspend fun setWakeWordSensitivity(value: Float) {
        settingsDataStore.setWakeWordSensitivity(value)
    }

    override val volumeButtonEnabled: Flow<Boolean>
        get() = settingsDataStore.volumeButtonEnabled

    override suspend fun setVolumeButtonEnabled(value: Boolean) {
        settingsDataStore.setVolumeButtonEnabled(value)
    }

    override val shakeEnabled: Flow<Boolean>
        get() = settingsDataStore.shakeEnabled

    override suspend fun setShakeEnabled(value: Boolean) {
        settingsDataStore.setShakeEnabled(value)
    }

    override val floatingButtonEnabled: Flow<Boolean>
        get() = settingsDataStore.floatingButtonEnabled

    override suspend fun setFloatingButtonEnabled(value: Boolean) {
        settingsDataStore.setFloatingButtonEnabled(value)
    }

    override val selectedVoice: Flow<String>
        get() = settingsDataStore.selectedVoice

    override suspend fun setSelectedVoice(value: String) {
        settingsDataStore.setSelectedVoice(value)
    }

    override val speechRate: Flow<Float>
        get() = settingsDataStore.speechRate

    override suspend fun setSpeechRate(value: Float) {
        settingsDataStore.setSpeechRate(value)
    }

    override val serverUrl: Flow<String>
        get() = settingsDataStore.serverUrl

    override suspend fun setServerUrl(value: String) {
        settingsDataStore.setServerUrl(value)
    }

    override val saveHistory: Flow<Boolean>
        get() = settingsDataStore.saveHistory

    override suspend fun setSaveHistory(value: Boolean) {
        settingsDataStore.setSaveHistory(value)
    }

    override val autoStartEnabled: Flow<Boolean>
        get() = settingsDataStore.autoStartEnabled

    override suspend fun setAutoStartEnabled(value: Boolean) {
        settingsDataStore.setAutoStartEnabled(value)
    }

    override val floatingButtonAutoStart: Flow<Boolean>
        get() = settingsDataStore.floatingButtonAutoStart

    override suspend fun setFloatingButtonAutoStart(value: Boolean) {
        settingsDataStore.setFloatingButtonAutoStart(value)
    }

    override val wakeWordAutoStart: Flow<Boolean>
        get() = settingsDataStore.wakeWordAutoStart

    override suspend fun setWakeWordAutoStart(value: Boolean) {
        settingsDataStore.setWakeWordAutoStart(value)
    }

    override suspend fun clearAllSettings() {
        settingsDataStore.clearAllSettings()
    }
}
