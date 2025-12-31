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

/**
 * Implementation of VoiceRepository that fetches voices from API and manages selection via DataStore.
 */
@Singleton
class VoiceRepositoryImpl @Inject constructor(
    private val apiService: AliciaApiService,
    private val settingsDataStore: SettingsDataStore
) : VoiceRepository {

    // Use AtomicReference for thread-safe caching of voices
    private val cachedVoices = AtomicReference<List<Voice>?>(null)

    override suspend fun getAvailableVoices(): Result<List<Voice>> {
        return try {
            // Atomic cache check using get()
            val cached = cachedVoices.get()
            if (cached != null) {
                return Result.success(cached)
            }

            // Fetch from server
            val response = apiService.getAvailableVoices()

            if (response.isSuccessful && response.body() != null) {
                val voices = response.body()!!.toDomain()
                // Use compareAndSet to ensure thread-safe update.
                // If another thread already set the cache, get() returns their value.
                // The fallback `?: voices` handles the edge case where get() returns null
                // after a successful compareAndSet (shouldn't happen, but defensive).
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
            // May return null if getAvailableVoices() hasn't been called yet, or if no voice matches the selected ID
            // Thread-safe: uses AtomicReference to prevent race conditions during cache reads
            cachedVoices.get()?.find { it.id == voiceId }
        }
    }

    override suspend fun setVoice(voiceId: String): Result<Unit> {
        return try {
            // Ensure voices are cached: if cachedVoices is null, fetch from API first
            if (cachedVoices.get() == null) {
                getAvailableVoices()
            }

            // Verify the requested voice exists in the cached list
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
