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

/**
 * Manages periodic synchronization with the server.
 * Uses adaptive polling with exponential backoff when idle to reduce battery and server load.
 */
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
        private const val BASE_SYNC_INTERVAL_MS = 5000L // Base interval: 5 seconds
        private const val MAX_SYNC_INTERVAL_MS = 60000L // Max interval when idle: 60 seconds
        private const val INITIAL_RETRY_DELAY_MS = 1000L // 1 second
        private const val MAX_RETRY_DELAY_MS = 30000L // 30 seconds
        private const val IDLE_THRESHOLD_MS = 30000L // Consider idle after 30 seconds of no new messages
    }

    /**
     * Calculate sync interval with exponential backoff when idle.
     * Matches web frontend behavior for consistency across platforms.
     * Also incorporates error-based retry delay when sync failures occur.
     */
    private fun getSyncInterval(): Long {
        val timeSinceActivity = System.currentTimeMillis() - lastActivityTimeMs.get()

        val baseInterval = if (timeSinceActivity < IDLE_THRESHOLD_MS) {
            // Active: use base interval
            BASE_SYNC_INTERVAL_MS
        } else {
            // Idle: use exponential backoff (2x for every 30s idle, up to max)
            val idlePeriods = (timeSinceActivity / IDLE_THRESHOLD_MS).toInt()
            val backoffInterval = (BASE_SYNC_INTERVAL_MS * 2.0.pow(idlePeriods)).toLong()
            min(backoffInterval, MAX_SYNC_INTERVAL_MS)
        }

        // Apply error-based retry delay (adds on top of idle backoff)
        return max(baseInterval, retryDelayMs.get())
    }

    /**
     * Mark activity to reset adaptive polling interval.
     * Call this when new messages arrive or user interacts.
     */
    fun markActivity() {
        lastActivityTimeMs.set(System.currentTimeMillis())
    }

    /**
     * Start periodic sync with adaptive polling.
     */
    fun startPeriodicSync() {
        if (syncJob?.isActive == true) {
            return // Already running
        }

        syncJob = scope.launch {
            while (isActive) {
                performSync()
                val interval = getSyncInterval()
                delay(interval)
            }
        }
    }

    /**
     * Stop periodic sync.
     */
    fun stopPeriodicSync() {
        syncJob?.cancel()
        syncJob = null
    }

    /**
     * Perform immediate sync.
     */
    suspend fun syncNow() {
        performSync()
    }

    /**
     * Perform the sync operation.
     */
    private suspend fun performSync() {
        if (_isSyncing.value) {
            return // Already syncing
        }

        try {
            _isSyncing.value = true
            _syncError.value = null

            val result = conversationRepository.syncWithServer()

            if (result.isSuccess) {
                _lastSyncTime.value = System.currentTimeMillis()
                retryDelayMs.set(INITIAL_RETRY_DELAY_MS) // Reset backoff on success

                // Note: We do NOT call markActivity() here to preserve adaptive polling.
                // Successful syncs should not reset the idle timer - only actual user
                // activity (new messages, interactions) should reset it via explicit
                // markActivity() calls from those events.
            } else {
                val error = result.exceptionOrNull()
                _syncError.value = error?.message ?: "Sync failed"

                // Exponential backoff
                retryDelayMs.updateAndGet { current -> minOf(current * 2, MAX_RETRY_DELAY_MS) }
            }
        } catch (e: Exception) {
            _syncError.value = e.message ?: "Sync failed"
            retryDelayMs.updateAndGet { current -> minOf(current * 2, MAX_RETRY_DELAY_MS) }
        } finally {
            _isSyncing.value = false
        }
    }

    /**
     * Check if sync is currently active.
     */
    fun isSyncActive(): Boolean = syncJob?.isActive == true
}
