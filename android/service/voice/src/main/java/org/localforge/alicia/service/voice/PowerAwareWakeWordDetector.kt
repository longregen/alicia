package org.localforge.alicia.service.voice

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager
import android.os.BatteryManager
import android.os.PowerManager
import androidx.annotation.RequiresPermission
import timber.log.Timber
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.util.concurrent.atomic.AtomicBoolean
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.math.sqrt

/**
 * Power-aware wrapper for wake word detection.
 * Optimizes battery usage by:
 * - Adjusting sample rate based on device state
 * - Managing wake locks efficiently
 * - Reducing processing when phone is face down or in pocket
 * - Adapting to battery saver mode
 */
@Singleton
class PowerAwareWakeWordDetector @Inject constructor(
    @ApplicationContext private val context: Context,
    private val wakeWordDetector: WakeWordDetector
) : SensorEventListener {

    private val powerManager = context.getSystemService(Context.POWER_SERVICE) as PowerManager
    private val batteryManager = context.getSystemService(Context.BATTERY_SERVICE) as BatteryManager
    private val sensorManager = context.getSystemService(Context.SENSOR_SERVICE) as SensorManager

    private var wakeLock: PowerManager.WakeLock? = null
    private var deviceStateMonitorJob: Job? = null
    private val monitorScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private val isPhoneFaceDown = AtomicBoolean(false)
    private val isInPocket = AtomicBoolean(false)
    private var currentDeviceState = DeviceState.SCREEN_ON_ACTIVE

    private val accelerometer = sensorManager.getDefaultSensor(Sensor.TYPE_ACCELEROMETER)
    private val proximity = sensorManager.getDefaultSensor(Sensor.TYPE_PROXIMITY)

    /**
     * Device states for power optimization.
     */
    enum class DeviceState {
        SCREEN_ON_ACTIVE,      // Screen on, user actively using device
        SCREEN_OFF_STATIONARY, // Screen off, device not moving
        SCREEN_OFF_MOVING,     // Screen off, device moving (in pocket/bag)
        BATTERY_SAVER,         // Battery saver mode enabled
        FACE_DOWN,             // Phone placed face down on surface
        IN_POCKET              // Phone in pocket/bag
    }

    /**
     * Start power-aware wake word detection.
     */
    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    suspend fun start(
        wakeWord: WakeWordDetector.WakeWord,
        onDetected: () -> Unit
    ) {
        acquireWakeLock()
        startDeviceStateMonitoring()

        // Get optimal configuration based on current device state
        val sensitivity = getOptimalSensitivity()

        wakeWordDetector.start(
            wakeWord = wakeWord,
            sensitivity = sensitivity,
            onDetected = {
                onWakeWordDetected(onDetected)
            }
        )

        Timber.i("Power-aware wake word detection started")
    }

    /**
     * Stop wake word detection and release resources.
     */
    fun stop() {
        stopDeviceStateMonitoring()
        releaseWakeLock()
        wakeWordDetector.stop()

        Timber.i("Power-aware wake word detection stopped")
    }

    /**
     * Pause wake word detection.
     */
    fun pause() {
        wakeWordDetector.pause()
        releaseWakeLock()
    }

    /**
     * Resume wake word detection.
     */
    fun resume() {
        acquireWakeLock()
        wakeWordDetector.resume()
    }

    private fun onWakeWordDetected(callback: () -> Unit) {
        // Temporarily release wake lock when processing user input
        releaseWakeLock()
        callback()
    }

    /**
     * Acquire partial wake lock to keep CPU running for wake word detection.
     */
    private fun acquireWakeLock() {
        if (wakeLock?.isHeld == true) {
            return
        }

        wakeLock = powerManager.newWakeLock(
            PowerManager.PARTIAL_WAKE_LOCK,
            "Alicia::PowerAwareWakeWordDetector"
        ).apply {
            // Use timeout to prevent battery drain if something goes wrong
            acquire(WAKE_LOCK_TIMEOUT)
        }

        Timber.d("Wake lock acquired")
    }

    /**
     * Release wake lock.
     */
    private fun releaseWakeLock() {
        wakeLock?.let {
            if (it.isHeld) {
                it.release()
                Timber.d("Wake lock released")
            }
        }
        wakeLock = null
    }

    /**
     * Start monitoring device state for power optimization.
     */
    private fun startDeviceStateMonitoring() {
        // Register sensors
        accelerometer?.let {
            sensorManager.registerListener(
                this,
                it,
                SensorManager.SENSOR_DELAY_NORMAL
            )
        }

        proximity?.let {
            sensorManager.registerListener(
                this,
                it,
                SensorManager.SENSOR_DELAY_NORMAL
            )
        }

        // Monitor device state changes
        deviceStateMonitorJob = monitorScope.launch {
            while (isActive) {
                updateDeviceState()
                adaptToDeviceState()
                delay(STATE_CHECK_INTERVAL)
            }
        }
    }

    /**
     * Stop monitoring device state.
     */
    private fun stopDeviceStateMonitoring() {
        sensorManager.unregisterListener(this)
        deviceStateMonitorJob?.cancel()
        deviceStateMonitorJob = null
    }

    /**
     * Update current device state based on sensors and system state.
     */
    private fun updateDeviceState() {
        currentDeviceState = when {
            isPowerSaveMode() -> DeviceState.BATTERY_SAVER
            isPhoneFaceDown.get() -> DeviceState.FACE_DOWN
            isInPocket.get() -> DeviceState.IN_POCKET
            powerManager.isInteractive -> DeviceState.SCREEN_ON_ACTIVE
            else -> DeviceState.SCREEN_OFF_STATIONARY
        }
    }

    /**
     * Adapt wake word detection parameters based on device state.
     */
    @SuppressLint("MissingPermission")
    private suspend fun adaptToDeviceState() {
        val newSensitivity = getOptimalSensitivity()

        // Update wake word detector sensitivity based on device state
        wakeWordDetector.setSensitivity(newSensitivity)

        // In very low power states, pause detection temporarily
        when (currentDeviceState) {
            DeviceState.FACE_DOWN -> {
                // Reduce processing when face down for extended period
                if (!powerManager.isInteractive) {
                    Timber.d("Device face down with screen off - reducing wake word sensitivity")
                }
            }
            DeviceState.IN_POCKET -> {
                // Phone is in pocket - can still detect wake word but lower sensitivity
                Timber.d("Device in pocket - adjusting wake word detection")
            }
            else -> {
                // Normal operation
            }
        }
    }

    /**
     * Get optimal wake word sensitivity based on device state.
     */
    private fun getOptimalSensitivity(): Float {
        return when (currentDeviceState) {
            DeviceState.SCREEN_ON_ACTIVE -> 0.5f      // Normal sensitivity
            DeviceState.SCREEN_OFF_STATIONARY -> 0.6f // Slightly higher threshold
            DeviceState.SCREEN_OFF_MOVING -> 0.7f     // Higher threshold in motion
            DeviceState.BATTERY_SAVER -> 0.7f         // Higher threshold to save battery
            DeviceState.FACE_DOWN -> 0.75f            // Much higher threshold
            DeviceState.IN_POCKET -> 0.7f             // Higher threshold
        }
    }

    /**
     * Check if device is in power save mode.
     */
    private fun isPowerSaveMode(): Boolean {
        return powerManager.isPowerSaveMode
    }

    /**
     * Get current battery level (0-100).
     */
    private fun getBatteryLevel(): Int {
        return batteryManager.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY)
    }

    // SensorEventListener implementation

    override fun onSensorChanged(event: SensorEvent) {
        when (event.sensor.type) {
            Sensor.TYPE_ACCELEROMETER -> {
                handleAccelerometerData(event.values)
            }
            Sensor.TYPE_PROXIMITY -> {
                handleProximityData(event.values[0])
            }
        }
    }

    /**
     * Required override from SensorEventListener. Accuracy changes don't affect wake word detection
     * as we only need relative motion/orientation data, not precise measurements.
     */
    override fun onAccuracyChanged(sensor: Sensor, accuracy: Int) {
        // No action needed - sensor accuracy changes don't affect wake word detection
    }

    /**
     * Handle accelerometer data to detect phone orientation and movement.
     */
    private fun handleAccelerometerData(values: FloatArray) {
        val x = values[0]
        val y = values[1]
        val z = values[2]

        // Check if phone is face down (z-axis pointing down)
        isPhoneFaceDown.set(z < -8.0f)

        // Calculate movement magnitude
        val movement = sqrt(x * x + y * y + z * z)

        // Detect if in pocket/bag (significant movement with screen off)
        if (!powerManager.isInteractive && movement > IN_POCKET_MOVEMENT_THRESHOLD) {
            isInPocket.set(true)
        } else if (powerManager.isInteractive) {
            // Clear in-pocket state when screen is on (user is likely interacting with device)
            isInPocket.set(false)
        }
    }

    /**
     * Handle proximity sensor data.
     */
    private fun handleProximityData(distance: Float) {
        // Proximity sensor detects if something is near the phone
        // Can be used in combination with other sensors to detect pocket
        val isNear = distance < 5.0f // Within 5cm

        if (isNear && !powerManager.isInteractive) {
            isInPocket.set(true)
            Timber.d("Proximity sensor: near object detected")
        } else if (powerManager.isInteractive) {
            isInPocket.set(false)
        }
    }

    companion object {
        // Wake lock configuration
        private const val WAKE_LOCK_TIMEOUT = 30 * 60 * 1000L // 30 minutes

        // Monitoring intervals
        private const val STATE_CHECK_INTERVAL = 5000L // Check device state every 5 seconds

        // Movement thresholds
        private const val IN_POCKET_MOVEMENT_THRESHOLD = 12.0f
    }
}
