package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.Voice
import kotlinx.coroutines.flow.Flow

/**
 * Repository interface for managing TTS voices.
 */
interface VoiceRepository {
    /**
     * Get all available voices from the server.
     * @return List of available voices.
     */
    suspend fun getAvailableVoices(): Result<List<Voice>>

    /**
     * Get the currently selected voice.
     * Returns null when no voice has been explicitly selected by the user,
     * which allows the system to use the first available voice as default.
     * @return Flow of the current voice, or null if not set.
     */
    fun getCurrentVoice(): Flow<Voice?>

    /**
     * Set the active voice.
     * @param voiceId The voice ID to set as active.
     */
    suspend fun setVoice(voiceId: String): Result<Unit>
}
