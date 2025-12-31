package org.localforge.alicia.service.hotkey

import android.Manifest
import android.accessibilityservice.AccessibilityService
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.os.Handler
import android.os.Looper
import android.os.VibrationEffect
import android.os.Vibrator
import android.provider.Settings
import android.text.TextUtils
import android.view.KeyEvent
import timber.log.Timber
import android.view.accessibility.AccessibilityEvent
import android.view.accessibility.AccessibilityManager
import androidx.annotation.RequiresPermission
import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.core.intPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch

/**
 * DataStore extension for accessing settings in HotkeyAccessibilityService.
 */
private val Context.settingsDataStore by preferencesDataStore(name = "alicia_settings")

/**
 * Accessibility service that enables hardware button hotkey detection for activating
 * the Alicia voice assistant.
 *
 * Supported triggers:
 * - Volume up: triple-tap (default: 3, adjustable via VOLUME_TAP_THRESHOLD compile-time constant)
 * - Volume down: double-tap (triggers on 2nd press within timeout window)
 * - Power button: long-press (where supported on Android 12+)
 *
 * This service requires BIND_ACCESSIBILITY_SERVICE permission and must be
 * explicitly enabled by the user in system accessibility settings.
 */
@AndroidEntryPoint
class HotkeyAccessibilityService : AccessibilityService() {

    companion object {
        private const val TAG = "HotkeyAccessibilityService"

        // Volume button detection constants
        private const val VOLUME_TAP_INTERVAL_MS = 500L
        private const val VOLUME_TAP_THRESHOLD = 3

        // Power button detection constants
        private const val POWER_LONG_PRESS_MS = 1000L

        // Action intent constants
        const val ACTION_ACTIVATE = "org.localforge.alicia.ACTION_ACTIVATE_ASSISTANT"

        // Preference keys
        private val PREF_VOLUME_UP_ENABLED = booleanPreferencesKey("hotkey_volume_up_enabled")
        private val PREF_VOLUME_DOWN_ENABLED = booleanPreferencesKey("hotkey_volume_down_enabled")
        private val PREF_POWER_BUTTON_ENABLED = booleanPreferencesKey("hotkey_power_button_enabled")
        private val PREF_TAP_COUNT = intPreferencesKey("hotkey_tap_count")

        // Default values
        private const val DEFAULT_VOLUME_UP_ENABLED = true
        private const val DEFAULT_VOLUME_DOWN_ENABLED = false
        private const val DEFAULT_POWER_BUTTON_ENABLED = false
        private const val DEFAULT_TAP_COUNT = VOLUME_TAP_THRESHOLD
    }

    // Service lifecycle scope
    private val serviceScope = CoroutineScope(Dispatchers.Main + SupervisorJob())

    // Vibrator for haptic feedback
    private val vibrator: Vibrator by lazy {
        getSystemService(android.os.VibratorManager::class.java)?.defaultVibrator
            ?: throw IllegalStateException("Vibrator service not available")
    }

    // Handler for timing-related operations
    private val handler = Handler(Looper.getMainLooper())

    // Volume up tracking
    private var lastVolumeUpTime = 0L
    private var volumeUpCount = 0
    private var volumeUpResetRunnable: Runnable? = null

    // Volume down tracking
    private var lastVolumeDownTime = 0L
    private var volumeDownCount = 0
    private var volumeDownResetRunnable: Runnable? = null

    // Power button tracking
    private var powerButtonDownTime = 0L
    private var powerLongPressTriggered = false

    // Settings
    private var volumeUpEnabled = true
    private var volumeDownEnabled = false
    private var powerButtonEnabled = false
    private var requiredTapCount = VOLUME_TAP_THRESHOLD

    override fun onCreate() {
        super.onCreate()
        Timber.d("HotkeyAccessibilityService created")
        loadSettings()
    }

    override fun onServiceConnected() {
        super.onServiceConnected()
        Timber.i("HotkeyAccessibilityService connected and ready")
        // The service is now active and can intercept key events
    }

    override fun onAccessibilityEvent(event: AccessibilityEvent?) {
        // We're primarily interested in key events, not accessibility events
        // This is required to be implemented but we don't need to process these events
    }

    override fun onInterrupt() {
        Timber.w("HotkeyAccessibilityService interrupted")
    }

    override fun onKeyEvent(event: KeyEvent): Boolean {
        // Only process key events if the service is properly configured
        if (!isServiceEnabled()) {
            return super.onKeyEvent(event)
        }

        return when (event.keyCode) {
            KeyEvent.KEYCODE_VOLUME_UP -> {
                if (volumeUpEnabled) {
                    handleVolumeUp(event)
                }
                false // Don't consume the event - allow normal volume behavior
            }

            KeyEvent.KEYCODE_VOLUME_DOWN -> {
                if (volumeDownEnabled) {
                    handleVolumeDown(event)
                }
                false // Don't consume the event
            }

            KeyEvent.KEYCODE_POWER -> {
                if (powerButtonEnabled) {
                    handlePowerButton(event)
                } else {
                    false
                }
            }

            else -> super.onKeyEvent(event)
        }
    }

    /**
     * Handle volume up key events for triple-tap detection
     */
    @RequiresPermission(Manifest.permission.VIBRATE)
    private fun handleVolumeUp(event: KeyEvent): Boolean {
        if (event.action != KeyEvent.ACTION_DOWN) {
            return false
        }

        val now = System.currentTimeMillis()

        // Check if this press is within the tap interval
        if (now - lastVolumeUpTime < VOLUME_TAP_INTERVAL_MS) {
            volumeUpCount++
            Timber.d("Volume up tap count: $volumeUpCount")

            if (volumeUpCount >= requiredTapCount) {
                // Triple-tap detected!
                Timber.i("Volume up triple-tap detected - activating assistant")
                activateAssistant()
                resetVolumeUpCounter()
                return false
            }
        } else {
            // Too much time passed, reset counter
            volumeUpCount = 1
        }

        lastVolumeUpTime = now

        // Schedule a reset of the counter if no more taps come
        volumeUpResetRunnable?.let { handler.removeCallbacks(it) }
        volumeUpResetRunnable = Runnable { resetVolumeUpCounter() }
        handler.postDelayed(volumeUpResetRunnable!!, VOLUME_TAP_INTERVAL_MS + 100)

        return false
    }

    /**
     * Handle volume down key events for double-tap detection
     */
    @RequiresPermission(Manifest.permission.VIBRATE)
    private fun handleVolumeDown(event: KeyEvent): Boolean {
        if (event.action != KeyEvent.ACTION_DOWN) {
            return false
        }

        val now = System.currentTimeMillis()

        if (now - lastVolumeDownTime < VOLUME_TAP_INTERVAL_MS) {
            volumeDownCount++
            Timber.d("Volume down tap count: $volumeDownCount")

            if (volumeDownCount >= 2) { // Double-tap for volume down
                Timber.i("Volume down double-tap detected - activating assistant")
                activateAssistant()
                resetVolumeDownCounter()
                return false
            }
        } else {
            volumeDownCount = 1
        }

        lastVolumeDownTime = now

        volumeDownResetRunnable?.let { handler.removeCallbacks(it) }
        volumeDownResetRunnable = Runnable { resetVolumeDownCounter() }
        handler.postDelayed(volumeDownResetRunnable!!, VOLUME_TAP_INTERVAL_MS + 100)

        return false
    }

    /**
     * Handle power button key events for long-press detection
     * Note: This may not work on all devices due to system-level power button handling
     */
    @RequiresPermission(Manifest.permission.VIBRATE)
    private fun handlePowerButton(event: KeyEvent): Boolean {
        when (event.action) {
            KeyEvent.ACTION_DOWN -> {
                if (powerButtonDownTime == 0L) {
                    powerButtonDownTime = System.currentTimeMillis()
                    powerLongPressTriggered = false

                    // Schedule long-press detection
                    handler.postDelayed({
                        if (powerButtonDownTime > 0 && !powerLongPressTriggered) {
                            val pressDuration = System.currentTimeMillis() - powerButtonDownTime
                            if (pressDuration >= POWER_LONG_PRESS_MS) {
                                Timber.i("Power button long-press detected - activating assistant")
                                powerLongPressTriggered = true
                                activateAssistant()
                            }
                        }
                    }, POWER_LONG_PRESS_MS)
                }
            }

            KeyEvent.ACTION_UP -> {
                powerButtonDownTime = 0L
            }
        }

        // Return false to allow normal power button behavior
        return false
    }

    /**
     * Reset the volume up counter
     */
    private fun resetVolumeUpCounter() {
        volumeUpCount = 0
        volumeUpResetRunnable?.let { handler.removeCallbacks(it) }
    }

    /**
     * Reset the volume down counter
     */
    private fun resetVolumeDownCounter() {
        volumeDownCount = 0
        volumeDownResetRunnable?.let { handler.removeCallbacks(it) }
    }

    /**
     * Activate the Alicia voice assistant
     */
    @RequiresPermission(Manifest.permission.VIBRATE)
    private fun activateAssistant() {
        // Provide haptic feedback
        provideHapticFeedback()

        // Send broadcast or start service to activate the assistant
        val intent = Intent(ACTION_ACTIVATE).apply {
            setPackage(packageName)
        }

        // Send as broadcast so any component can receive it
        sendBroadcast(intent)

        Timber.i("Assistant activation intent sent")
    }

    /**
     * Provide haptic feedback to the user
     */
    @RequiresPermission(Manifest.permission.VIBRATE)
    private fun provideHapticFeedback() {
        if (!vibrator.hasVibrator()) return

        try {
            val effect = VibrationEffect.createOneShot(
                50, // Duration in milliseconds
                VibrationEffect.DEFAULT_AMPLITUDE
            )
            vibrator.vibrate(effect)
        } catch (e: Exception) {
            Timber.e(e, "Failed to provide haptic feedback")
        }
    }

    /**
     * Check if the accessibility service is properly enabled in system settings.
     *
     * This method verifies that this accessibility service is currently enabled
     * in Android's accessibility settings by checking both:
     * 1. The list of enabled accessibility services from AccessibilityManager
     * 2. The Settings.Secure provider as a fallback
     *
     * @return true if the service is enabled and running, false otherwise
     */
    private fun isServiceEnabled(): Boolean {
        try {
            val accessibilityManager = getSystemService(Context.ACCESSIBILITY_SERVICE) as? AccessibilityManager
                ?: return false

            // Get the component name for this service
            val expectedComponentName = ComponentName(this, HotkeyAccessibilityService::class.java)

            // Method 1: Check using AccessibilityManager's enabled service list
            val enabledServices = accessibilityManager.getEnabledAccessibilityServiceList(
                android.accessibilityservice.AccessibilityServiceInfo.FEEDBACK_ALL_MASK
            )

            val isEnabledViaManager = enabledServices.any { serviceInfo ->
                serviceInfo.resolveInfo?.serviceInfo?.let { info ->
                    val componentName = ComponentName(info.packageName, info.name)
                    componentName == expectedComponentName
                } ?: false
            }

            if (isEnabledViaManager) {
                return true
            }

            // Method 2: Fallback to Settings.Secure (more reliable on some devices)
            val enabledServicesSetting = Settings.Secure.getString(
                contentResolver,
                Settings.Secure.ENABLED_ACCESSIBILITY_SERVICES
            ) ?: return false

            val colonSplitter = TextUtils.SimpleStringSplitter(':')
            colonSplitter.setString(enabledServicesSetting)

            while (colonSplitter.hasNext()) {
                val componentNameString = colonSplitter.next()
                val enabledComponent = ComponentName.unflattenFromString(componentNameString)
                if (enabledComponent != null && enabledComponent == expectedComponentName) {
                    return true
                }
            }

            return false
        } catch (e: Exception) {
            Timber.e(e, "Error checking if accessibility service is enabled")
            // In case of error, assume service might not be enabled to be safe
            return false
        }
    }

    /**
     * Load settings from DataStore.
     *
     * Reads hotkey preferences from DataStore including:
     * - Volume up enabled state
     * - Volume down enabled state
     * - Power button enabled state
     * - Required tap count threshold
     *
     * Falls back to default values if preferences are not set or if an error occurs.
     */
    private fun loadSettings() {
        serviceScope.launch {
            try {
                // Read volume up enabled setting
                volumeUpEnabled = settingsDataStore.data
                    .map { preferences ->
                        preferences[PREF_VOLUME_UP_ENABLED] ?: DEFAULT_VOLUME_UP_ENABLED
                    }
                    .first()

                // Read volume down enabled setting
                volumeDownEnabled = settingsDataStore.data
                    .map { preferences ->
                        preferences[PREF_VOLUME_DOWN_ENABLED] ?: DEFAULT_VOLUME_DOWN_ENABLED
                    }
                    .first()

                // Read power button enabled setting
                powerButtonEnabled = settingsDataStore.data
                    .map { preferences ->
                        preferences[PREF_POWER_BUTTON_ENABLED] ?: DEFAULT_POWER_BUTTON_ENABLED
                    }
                    .first()

                // Read tap count threshold setting
                requiredTapCount = settingsDataStore.data
                    .map { preferences ->
                        preferences[PREF_TAP_COUNT] ?: DEFAULT_TAP_COUNT
                    }
                    .first()

                Timber.d("Settings loaded from DataStore - Volume Up: $volumeUpEnabled, Volume Down: $volumeDownEnabled, Power: $powerButtonEnabled, Tap Count: $requiredTapCount")
            } catch (e: Exception) {
                Timber.e(e, "Failed to load settings from DataStore, using defaults")
                // Fall back to default values on error
                volumeUpEnabled = DEFAULT_VOLUME_UP_ENABLED
                volumeDownEnabled = DEFAULT_VOLUME_DOWN_ENABLED
                powerButtonEnabled = DEFAULT_POWER_BUTTON_ENABLED
                requiredTapCount = DEFAULT_TAP_COUNT
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()

        // Clean up handlers
        handler.removeCallbacksAndMessages(null)

        // Cancel coroutine scope
        serviceScope.cancel()

        Timber.d("HotkeyAccessibilityService destroyed")
    }
}
