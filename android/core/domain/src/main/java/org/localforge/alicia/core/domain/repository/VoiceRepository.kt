package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.Voice
import kotlinx.coroutines.flow.Flow

interface VoiceRepository {
    suspend fun getAvailableVoices(): Result<List<Voice>>

    fun getCurrentVoice(): Flow<Voice?>

    suspend fun setVoice(voiceId: String): Result<Unit>
}
