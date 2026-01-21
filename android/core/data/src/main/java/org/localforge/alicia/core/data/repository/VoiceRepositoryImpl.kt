package org.localforge.alicia.core.data.repository

import org.localforge.alicia.core.data.mapper.toDomain
import org.localforge.alicia.core.data.preferences.SettingsDataStore
import org.localforge.alicia.core.domain.model.Voice
import org.localforge.alicia.core.domain.repository.VoiceRepository
import org.localforge.alicia.core.network.api.AliciaApiService
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import java.util.concurrent.atomic.AtomicReference
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class VoiceRepositoryImpl @Inject constructor(
    private val apiService: AliciaApiService,
    private val settingsDataStore: SettingsDataStore
) : VoiceRepository {

    private val cachedVoices = AtomicReference<List<Voice>?>(null)

    override suspend fun getAvailableVoices(): Result<List<Voice>> {
        return try {
            val cached = cachedVoices.get()
            if (cached != null) {
                return Result.success(cached)
            }

            val response = apiService.getAvailableVoices()

            if (response.isSuccessful && response.body() != null) {
                val voices = response.body()!!.toDomain()
                cachedVoices.compareAndSet(null, voices)
                Result.success(cachedVoices.get() ?: voices)
            } else {
                Result.failure(
                    Exception("Failed to fetch voices: ${response.code()} ${response.message()}")
                )
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override fun getCurrentVoice(): Flow<Voice?> {
        return settingsDataStore.selectedVoice.map { voiceId ->
            cachedVoices.get()?.find { it.id == voiceId }
        }
    }

    override suspend fun setVoice(voiceId: String): Result<Unit> {
        return try {
            if (cachedVoices.get() == null) {
                getAvailableVoices()
            }

            val voiceExists = cachedVoices.get()?.any { it.id == voiceId } == true

            if (voiceExists) {
                settingsDataStore.setSelectedVoice(voiceId)
                Result.success(Unit)
            } else {
                Result.failure(IllegalArgumentException("Voice with ID $voiceId not found"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
