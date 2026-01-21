package org.localforge.alicia.core.data.sync

import org.localforge.alicia.core.domain.repository.ConversationRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.util.concurrent.atomic.AtomicLong
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.math.max
import kotlin.math.min
import kotlin.math.pow

@Singleton
class SyncManager @Inject constructor(
    private val conversationRepository: ConversationRepository
) {
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private var syncJob: Job? = null

    private val _isSyncing = MutableStateFlow(false)
    val isSyncing: StateFlow<Boolean> = _isSyncing.asStateFlow()

    private val _lastSyncTime = MutableStateFlow<Long?>(null)
    val lastSyncTime: StateFlow<Long?> = _lastSyncTime.asStateFlow()

    private val _syncError = MutableStateFlow<String?>(null)
    val syncError: StateFlow<String?> = _syncError.asStateFlow()

    private val retryDelayMs = AtomicLong(INITIAL_RETRY_DELAY_MS)
    private val lastActivityTimeMs = AtomicLong(System.currentTimeMillis())

    companion object {
        private const val BASE_SYNC_INTERVAL_MS = 5000L
        private const val MAX_SYNC_INTERVAL_MS = 60000L
        private const val INITIAL_RETRY_DELAY_MS = 1000L
        private const val MAX_RETRY_DELAY_MS = 30000L
        private const val IDLE_THRESHOLD_MS = 30000L
    }

    private fun getSyncInterval(): Long {
        val timeSinceActivity = System.currentTimeMillis() - lastActivityTimeMs.get()

        val baseInterval = if (timeSinceActivity < IDLE_THRESHOLD_MS) {
            // Active: use base interval
            BASE_SYNC_INTERVAL_MS
        } else {
            val idlePeriods = (timeSinceActivity / IDLE_THRESHOLD_MS).toInt()
            val backoffInterval = (BASE_SYNC_INTERVAL_MS * 2.0.pow(idlePeriods)).toLong()
            min(backoffInterval, MAX_SYNC_INTERVAL_MS)
        }

        return max(baseInterval, retryDelayMs.get())
    }

    fun markActivity() {
        lastActivityTimeMs.set(System.currentTimeMillis())
    }

    fun startPeriodicSync() {
        if (syncJob?.isActive == true) {
            return
        }

        syncJob = scope.launch {
            while (isActive) {
                performSync()
                val interval = getSyncInterval()
                delay(interval)
            }
        }
    }

    fun stopPeriodicSync() {
        syncJob?.cancel()
        syncJob = null
    }

    suspend fun syncNow() {
        performSync()
    }

    private suspend fun performSync() {
        if (_isSyncing.value) {
            return
        }

        try {
            _isSyncing.value = true
            _syncError.value = null

            val result = conversationRepository.syncWithServer()

            if (result.isSuccess) {
                _lastSyncTime.value = System.currentTimeMillis()
                retryDelayMs.set(INITIAL_RETRY_DELAY_MS)
            } else {
                val error = result.exceptionOrNull()
                _syncError.value = error?.message ?: "Sync failed"

                retryDelayMs.updateAndGet { current -> minOf(current * 2, MAX_RETRY_DELAY_MS) }
            }
        } catch (e: Exception) {
            _syncError.value = e.message ?: "Sync failed"
            retryDelayMs.updateAndGet { current -> minOf(current * 2, MAX_RETRY_DELAY_MS) }
        } finally {
            _isSyncing.value = false
        }
    }

    fun isSyncActive(): Boolean = syncJob?.isActive == true
}
