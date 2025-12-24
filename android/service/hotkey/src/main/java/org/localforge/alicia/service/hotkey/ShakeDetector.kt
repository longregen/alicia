package org.localforge.alicia.service.hotkey

import android.content.Context
import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager
import timber.log.Timber
import kotlin.math.sqrt

/**
 * Detects device shake gestures using the accelerometer sensor.
 * Uses acceleration magnitude to detect shaking motion and triggers
 * a callback when a shake is detected.
 *
 * The detector requires a certain number of shake movements within a time
 * window to avoid false positives from normal device movement.
 *
 * @param context Android context for accessing the sensor service
 * @param onShakeDetected Callback invoked when a shake gesture is detected
 */
class ShakeDetector(
    private val context: Context,
    private val onShakeDetected: () -> Unit
) : SensorEventListener {

    companion object {
        private const val TAG = "ShakeDetector"

        // Shake detection thresholds
        private const val SHAKE_THRESHOLD = 12.0f // m/s² above gravity
        private const val SHAKE_INTERVAL_MS = 500L // Time window for counting shakes
        private const val SHAKE_COUNT_THRESHOLD = 2 // Minimum consecutive shakes needed
        private const val SHAKE_COOLDOWN_MS = 2000L // Cooldown period after detection

        // Sensor update rate
        private const val SENSOR_DELAY = SensorManager.SENSOR_DELAY_GAME
    }

    private val sensorManager: SensorManager =
        context.getSystemService(Context.SENSOR_SERVICE) as SensorManager

    private val accelerometer: Sensor? =
        sensorManager.getDefaultSensor(Sensor.TYPE_ACCELEROMETER)

    // Shake detection state
    private var lastShakeTime = 0L
    private var shakeCount = 0
    private var lastDetectionTime = 0L

    // Listener state
    private var isListening = false

    // Configuration
    private var threshold = SHAKE_THRESHOLD
    private var interval = SHAKE_INTERVAL_MS
    private var countThreshold = SHAKE_COUNT_THRESHOLD
    private var cooldown = SHAKE_COOLDOWN_MS

    init {
        if (accelerometer == null) {
            Timber.w("Accelerometer sensor not available on this device")
        } else {
            Timber.d("ShakeDetector initialized with accelerometer: ${accelerometer.name}")
        }
    }

    /**
     * Start listening for shake events
     * @return true if started successfully, false if accelerometer is not available
     */
    fun start(): Boolean {
        if (accelerometer == null) {
            Timber.e("Cannot start - accelerometer not available")
            return false
        }

        if (isListening) {
            Timber.w("ShakeDetector is already listening")
            return true
        }

        val registered = sensorManager.registerListener(
            this,
            accelerometer,
            SENSOR_DELAY
        )

        if (registered) {
            isListening = true
            resetState()
            Timber.i("ShakeDetector started")
        } else {
            Timber.e("Failed to register sensor listener")
        }

        return registered
    }

    /**
     * Stop listening for shake events
     */
    fun stop() {
        if (!isListening) {
            return
        }

        sensorManager.unregisterListener(this)
        isListening = false
        resetState()
        Timber.i("ShakeDetector stopped")
    }

    /**
     * Check if the detector is currently listening
     */
    fun isRunning(): Boolean = isListening

    /**
     * Configure shake detection sensitivity
     * @param threshold Acceleration threshold in m/s² (default: 12.0)
     * @param shakeCountThreshold Number of shakes required (default: 2)
     * @param intervalMs Time window for counting shakes in ms (default: 500)
     * @param cooldownMs Cooldown period after detection in ms (default: 2000)
     */
    fun configure(
        threshold: Float = SHAKE_THRESHOLD,
        shakeCountThreshold: Int = SHAKE_COUNT_THRESHOLD,
        intervalMs: Long = SHAKE_INTERVAL_MS,
        cooldownMs: Long = SHAKE_COOLDOWN_MS
    ) {
        this.threshold = threshold
        this.countThreshold = shakeCountThreshold
        this.interval = intervalMs
        this.cooldown = cooldownMs

        Timber.d("Configuration updated - threshold: $threshold, count: $shakeCountThreshold, interval: $intervalMs, cooldown: $cooldownMs")
    }

    override fun onSensorChanged(event: SensorEvent) {
        if (event.sensor.type != Sensor.TYPE_ACCELEROMETER) {
            return
        }

        // Get acceleration values
        val x = event.values[0]
        val y = event.values[1]
        val z = event.values[2]

        // Calculate magnitude of acceleration minus gravity
        val acceleration = sqrt(x * x + y * y + z * z) - SensorManager.GRAVITY_EARTH

        // Check if acceleration exceeds threshold
        if (acceleration > threshold) {
            val now = System.currentTimeMillis()

            // Check if we're in cooldown period
            if (now - lastDetectionTime < cooldown) {
                return
            }

            // Check if this shake is within the interval window
            if (now - lastShakeTime < interval) {
                shakeCount++
                Timber.d("Shake detected! Count: $shakeCount, Acceleration: ${"%.2f".format(acceleration)} m/s²")

                // Check if we've reached the threshold for triggering
                if (shakeCount >= countThreshold) {
                    handleShakeDetected(now)
                }
            } else {
                // Too much time passed, reset counter
                shakeCount = 1
                Timber.d("Shake detected (first), Acceleration: ${"%.2f".format(acceleration)} m/s²")
            }

            lastShakeTime = now
        }
    }

    override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {
        // Log accuracy changes for debugging
        val accuracyStr = when (accuracy) {
            SensorManager.SENSOR_STATUS_ACCURACY_HIGH -> "HIGH"
            SensorManager.SENSOR_STATUS_ACCURACY_MEDIUM -> "MEDIUM"
            SensorManager.SENSOR_STATUS_ACCURACY_LOW -> "LOW"
            SensorManager.SENSOR_STATUS_UNRELIABLE -> "UNRELIABLE"
            else -> "UNKNOWN"
        }
        Timber.d("Sensor accuracy changed: $accuracyStr")
    }

    /**
     * Handle shake detection event
     */
    private fun handleShakeDetected(now: Long) {
        Timber.i("Shake gesture detected! Triggering callback")

        // Update detection time for cooldown
        lastDetectionTime = now

        // Reset shake counter
        resetState()

        // Trigger callback
        try {
            onShakeDetected()
        } catch (e: Exception) {
            Timber.e(e, "Error in shake detection callback")
        }
    }

    /**
     * Reset shake detection state
     */
    private fun resetState() {
        shakeCount = 0
        lastShakeTime = 0L
    }

    /**
     * Check if accelerometer is available on this device
     */
    fun isAvailable(): Boolean = accelerometer != null

    /**
     * Get current shake detection statistics (for debugging/monitoring)
     */
    fun getStats(): ShakeStats {
        return ShakeStats(
            isListening = isListening,
            currentShakeCount = shakeCount,
            lastShakeTime = lastShakeTime,
            lastDetectionTime = lastDetectionTime,
            threshold = threshold,
            countThreshold = countThreshold
        )
    }

    /**
     * Data class for shake detection statistics
     */
    data class ShakeStats(
        val isListening: Boolean,
        val currentShakeCount: Int,
        val lastShakeTime: Long,
        val lastDetectionTime: Long,
        val threshold: Float,
        val countThreshold: Int
    )
}
