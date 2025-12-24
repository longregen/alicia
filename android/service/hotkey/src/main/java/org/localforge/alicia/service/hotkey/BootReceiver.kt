package org.localforge.alicia.service.hotkey

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import timber.log.Timber
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * DataStore extension for accessing settings in BootReceiver.
 */
private val Context.settingsDataStore by preferencesDataStore(name = "alicia_settings")

/**
 * Broadcast receiver that starts the Alicia voice service on device boot.
 *
 * This receiver listens for the BOOT_COMPLETED broadcast and automatically
 * starts the voice assistant service if auto-start is enabled in settings.
 *
 * Requirements:
 * - RECEIVE_BOOT_COMPLETED permission in AndroidManifest
 * - User must enable auto-start in app settings
 *
 * The receiver will:
 * 1. Check if auto-start is enabled
 * 2. Start the voice service as a foreground service
 * 3. Optionally start the floating button service
 */
@AndroidEntryPoint
class BootReceiver : BroadcastReceiver() {

    companion object {
        private const val TAG = "BootReceiver"

        // Intent actions
        private const val ACTION_BOOT_COMPLETED = Intent.ACTION_BOOT_COMPLETED
        private const val ACTION_LOCKED_BOOT_COMPLETED = "android.intent.action.LOCKED_BOOT_COMPLETED"
        private const val ACTION_MY_PACKAGE_REPLACED = Intent.ACTION_MY_PACKAGE_REPLACED

        // Preference keys
        private val PREF_AUTO_START = booleanPreferencesKey("auto_start_enabled")
        private val PREF_FLOATING_BUTTON_AUTO_START = booleanPreferencesKey("floating_button_auto_start")
        private val PREF_WAKE_WORD_AUTO_START = booleanPreferencesKey("wake_word_auto_start")

        // Default values
        private const val DEFAULT_AUTO_START = true
        private const val DEFAULT_FLOATING_BUTTON_AUTO_START = false
        private const val DEFAULT_WAKE_WORD_AUTO_START = true
    }

    override fun onReceive(context: Context, intent: Intent) {
        val action = intent.action
        Timber.i("Received broadcast: $action")

        when (action) {
            ACTION_BOOT_COMPLETED,
            ACTION_LOCKED_BOOT_COMPLETED -> {
                handleBootCompleted(context)
            }
            ACTION_MY_PACKAGE_REPLACED -> {
                handlePackageReplaced(context)
            }
            else -> {
                Timber.w("Received unexpected action: $action")
            }
        }
    }

    /**
     * Handle device boot completed
     */
    private fun handleBootCompleted(context: Context) {
        Timber.i("Device boot completed")

        // Use goAsync() for long-running operations
        val pendingResult = goAsync()

        // Create a scoped coroutine that will be cancelled after completion
        CoroutineScope(Dispatchers.IO + SupervisorJob()).launch {
            try {
                // Check if auto-start is enabled
                if (isAutoStartEnabled(context)) {
                    Timber.i("Auto-start is enabled, starting services")
                    startServices(context)
                } else {
                    Timber.d("Auto-start is disabled, not starting services")
                }
            } catch (e: Exception) {
                Timber.e(e, "Error handling boot completed")
            } finally {
                pendingResult.finish()
            }
        }
    }

    /**
     * Handle app package replaced (app updated)
     */
    private fun handlePackageReplaced(context: Context) {
        Timber.i("App package replaced (updated)")

        val pendingResult = goAsync()

        // Create a scoped coroutine that will be cancelled after completion
        CoroutineScope(Dispatchers.IO + SupervisorJob()).launch {
            try {
                // Restart services if they were running before update
                if (isAutoStartEnabled(context)) {
                    Timber.i("Restarting services after app update")
                    startServices(context)
                }
            } catch (e: Exception) {
                Timber.e(e, "Error handling package replaced")
            } finally {
                pendingResult.finish()
            }
        }
    }

    /**
     * Check if auto-start is enabled in settings.
     * Reads from DataStore preferences with fallback to default value.
     */
    private suspend fun isAutoStartEnabled(context: Context): Boolean {
        return context.settingsDataStore.data
            .map { preferences ->
                preferences[PREF_AUTO_START] ?: DEFAULT_AUTO_START
            }
            .first()
    }

    /**
     * Check if floating button auto-start is enabled.
     * Reads from DataStore preferences with fallback to default value.
     */
    private suspend fun isFloatingButtonAutoStartEnabled(context: Context): Boolean {
        return context.settingsDataStore.data
            .map { preferences ->
                preferences[PREF_FLOATING_BUTTON_AUTO_START] ?: DEFAULT_FLOATING_BUTTON_AUTO_START
            }
            .first()
    }

    /**
     * Check if wake word detection should auto-start.
     * Reads from DataStore preferences with fallback to default value.
     */
    private suspend fun isWakeWordAutoStartEnabled(context: Context): Boolean {
        return context.settingsDataStore.data
            .map { preferences ->
                preferences[PREF_WAKE_WORD_AUTO_START] ?: DEFAULT_WAKE_WORD_AUTO_START
            }
            .first()
    }

    /**
     * Start the necessary services
     */
    private suspend fun startServices(context: Context) {
        try {
            // Start voice service if wake word auto-start is enabled
            if (isWakeWordAutoStartEnabled(context)) {
                startVoiceService(context)
            }

            // Start floating button service if enabled
            if (isFloatingButtonAutoStartEnabled(context)) {
                startFloatingButtonService(context)
            }

            Timber.i("Services started successfully")
        } catch (e: Exception) {
            Timber.e(e, "Failed to start services")
        }
    }

    /**
     * Start the voice service
     */
    private fun startVoiceService(context: Context) {
        try {
            val intent = Intent().apply {
                // Points to the VoiceService class in the voice service module
                setClassName(context.packageName, "org.localforge.alicia.service.voice.VoiceService")
                action = "org.localforge.alicia.ACTION_START_WAKE_WORD"
            }

            context.startForegroundService(intent)

            Timber.d("Voice service started")
        } catch (e: Exception) {
            Timber.e(e, "Failed to start voice service")
        }
    }

    /**
     * Start the floating button service
     */
    private fun startFloatingButtonService(context: Context) {
        try {
            val intent = Intent(context, FloatingButtonService::class.java).apply {
                action = FloatingButtonService.ACTION_START
            }

            context.startForegroundService(intent)

            Timber.d("Floating button service started")
        } catch (e: Exception) {
            Timber.e(e, "Failed to start floating button service")
        }
    }
}
