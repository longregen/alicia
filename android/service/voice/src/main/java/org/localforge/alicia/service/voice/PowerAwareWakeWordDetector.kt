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

    enum class DeviceState {
        SCREEN_ON_ACTIVE,
        SCREEN_OFF_STATIONARY,
        SCREEN_OFF_MOVING,
        BATTERY_SAVER,
        FACE_DOWN,
        IN_POCKET
    }

    @RequiresPermission(Manifest.permission.RECORD_AUDIO)
    suspend fun start(
        wakeWord: WakeWordDetector.WakeWord,
        onDetected: () -> Unit
    ) {
        acquireWakeLock()
        startDeviceStateMonitoring()

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

    fun stop() {
        stopDeviceStateMonitoring()
        releaseWakeLock()
        wakeWordDetector.stop()

        Timber.i("Power-aware wake word detection stopped")
    }

    fun pause() {
        wakeWordDetector.pause()
        releaseWakeLock()
    }

    fun resume() {
        acquireWakeLock()
        wakeWordDetector.resume()
    }

    private fun onWakeWordDetected(callback: () -> Unit) {
        releaseWakeLock()
        callback()
    }

    private fun acquireWakeLock() {
        if (wakeLock?.isHeld == true) {
            return
        }

        wakeLock = powerManager.newWakeLock(
            PowerManager.PARTIAL_WAKE_LOCK,
            "Alicia::PowerAwareWakeWordDetector"
        ).apply {
            // Timeout prevents battery drain if wake lock is never released due to a bug
            acquire(WAKE_LOCK_TIMEOUT)
        }

        Timber.d("Wake lock acquired")
    }

    private fun releaseWakeLock() {
        wakeLock?.let {
            if (it.isHeld) {
                it.release()
                Timber.d("Wake lock released")
            }
        }
        wakeLock = null
    }

    private fun startDeviceStateMonitoring() {
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

        deviceStateMonitorJob = monitorScope.launch {
            while (isActive) {
                updateDeviceState()
                adaptToDeviceState()
                delay(STATE_CHECK_INTERVAL)
            }
        }
    }

    private fun stopDeviceStateMonitoring() {
        sensorManager.unregisterListener(this)
        deviceStateMonitorJob?.cancel()
        deviceStateMonitorJob = null
    }

    private fun updateDeviceState() {
        currentDeviceState = when {
            isPowerSaveMode() -> DeviceState.BATTERY_SAVER
            isPhoneFaceDown.get() -> DeviceState.FACE_DOWN
            isInPocket.get() -> DeviceState.IN_POCKET
            powerManager.isInteractive -> DeviceState.SCREEN_ON_ACTIVE
            else -> DeviceState.SCREEN_OFF_STATIONARY
        }
    }

    @SuppressLint("MissingPermission")
    private suspend fun adaptToDeviceState() {
        val newSensitivity = getOptimalSensitivity()
        wakeWordDetector.setSensitivity(newSensitivity)

        when (currentDeviceState) {
            DeviceState.FACE_DOWN -> {
                // Reduce processing when face down for extended period
                if (!powerManager.isInteractive) {
                    Timber.d("Device face down with screen off - reducing wake word sensitivity")
                }
            }
            DeviceState.IN_POCKET -> {
                Timber.d("Device in pocket - adjusting wake word detection")
            }
            else -> { }
        }
    }

    private fun getOptimalSensitivity(): Float {
        return when (currentDeviceState) {
            DeviceState.SCREEN_ON_ACTIVE -> 0.5f
            DeviceState.SCREEN_OFF_STATIONARY -> 0.6f
            DeviceState.SCREEN_OFF_MOVING -> 0.7f
            DeviceState.BATTERY_SAVER -> 0.7f
            DeviceState.FACE_DOWN -> 0.75f
            DeviceState.IN_POCKET -> 0.7f
        }
    }

    private fun isPowerSaveMode(): Boolean {
        return powerManager.isPowerSaveMode
    }

    private fun getBatteryLevel(): Int {
        return batteryManager.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY)
    }

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

    override fun onAccuracyChanged(sensor: Sensor, accuracy: Int) { }

    private fun handleAccelerometerData(values: FloatArray) {
        val x = values[0]
        val y = values[1]
        val z = values[2]

        isPhoneFaceDown.set(z < -8.0f)

        val movement = sqrt(x * x + y * y + z * z)

        if (!powerManager.isInteractive && movement > IN_POCKET_MOVEMENT_THRESHOLD) {
            isInPocket.set(true)
        } else if (powerManager.isInteractive) {
            isInPocket.set(false)
        }
    }

    private fun handleProximityData(distance: Float) {
        val isNear = distance < 5.0f

        if (isNear && !powerManager.isInteractive) {
            isInPocket.set(true)
            Timber.d("Proximity sensor: near object detected")
        } else if (powerManager.isInteractive) {
            isInPocket.set(false)
        }
    }

    companion object {
        private const val WAKE_LOCK_TIMEOUT = 30 * 60 * 1000L
        private const val STATE_CHECK_INTERVAL = 5000L
        private const val IN_POCKET_MOVEMENT_THRESHOLD = 12.0f
    }
}
