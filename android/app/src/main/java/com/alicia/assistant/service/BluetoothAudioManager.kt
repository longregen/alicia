package com.alicia.assistant.service

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.media.AudioDeviceCallback
import android.media.AudioDeviceInfo
import android.media.AudioManager
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.util.Log
import androidx.core.content.ContextCompat

/**
 * Manages Bluetooth audio device routing using setCommunicationDevice() API.
 *
 * This enables recording from Bluetooth headset microphones by routing audio through SCO/BLE.
 * SCO audio is limited to 8kHz mono, so callers should resample to 16kHz for Whisper/Vosk.
 */
class BluetoothAudioManager(private val context: Context) {

    companion object {
        private const val TAG = "BluetoothAudioManager"

        // SCO sample rate is 8kHz
        const val SCO_SAMPLE_RATE = 8000

        // Target sample rate for Whisper/Vosk
        const val TARGET_SAMPLE_RATE = 16000
    }

    private val audioManager = context.getSystemService(Context.AUDIO_SERVICE) as AudioManager
    private val mainHandler = Handler(Looper.getMainLooper())
    private val audioLock = Any()

    @Volatile
    private var currentBluetoothDevice: AudioDeviceInfo? = null

    @Volatile
    private var isBluetoothRouted = false

    private var deviceCallback: AudioDeviceCallback? = null
    @Volatile
    private var stateListener: BluetoothAudioStateListener? = null

    interface BluetoothAudioStateListener {
        fun onBluetoothDeviceConnected(device: AudioDeviceInfo)
        fun onBluetoothDeviceDisconnected()
    }

    /**
     * Check if BLUETOOTH_CONNECT permission is granted.
     * On Android 11 and below, returns true (permission not required at runtime).
     * On Android 12+, checks the runtime permission.
     */
    private fun hasBluetoothPermission(): Boolean {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            ContextCompat.checkSelfPermission(context, Manifest.permission.BLUETOOTH_CONNECT) ==
                PackageManager.PERMISSION_GRANTED
        } else {
            true
        }
    }

    /**
     * Find available Bluetooth audio input devices (SCO or BLE headset).
     * Returns null if BLUETOOTH_CONNECT permission is not granted (Android 12+).
     */
    fun findBluetoothInputDevice(): AudioDeviceInfo? {
        if (!hasBluetoothPermission()) {
            Log.d(TAG, "BLUETOOTH_CONNECT permission not granted, skipping Bluetooth device search")
            return null
        }

        val devices = audioManager.availableCommunicationDevices

        // Prefer BLE headset (better quality) over classic SCO
        val bleDevice = devices.find { it.type == AudioDeviceInfo.TYPE_BLE_HEADSET }
        if (bleDevice != null) {
            Log.d(TAG, "Found BLE headset: ${bleDevice.productName}")
            return bleDevice
        }

        val scoDevice = devices.find { it.type == AudioDeviceInfo.TYPE_BLUETOOTH_SCO }
        if (scoDevice != null) {
            Log.d(TAG, "Found Bluetooth SCO device: ${scoDevice.productName}")
            return scoDevice
        }

        Log.d(TAG, "No Bluetooth audio device available")
        return null
    }

    /**
     * Check if a Bluetooth audio device is available for input.
     */
    fun isBluetoothAvailable(): Boolean {
        return findBluetoothInputDevice() != null
    }

    /**
     * Check if audio is currently routed through Bluetooth.
     */
    fun isBluetoothActive(): Boolean = isBluetoothRouted

    /**
     * Get the effective sample rate based on current routing.
     * Returns 8000 if Bluetooth SCO is active, otherwise the default rate.
     */
    fun getEffectiveSampleRate(defaultRate: Int): Int {
        if (!isBluetoothRouted) return defaultRate

        val device = currentBluetoothDevice ?: return defaultRate

        // BLE Audio can support higher sample rates
        if (device.type == AudioDeviceInfo.TYPE_BLE_HEADSET) {
            val sampleRates = device.sampleRates
            if (sampleRates.isNotEmpty()) {
                // Use highest available, up to target
                val best = sampleRates.filter { it <= TARGET_SAMPLE_RATE }.maxOrNull()
                if (best != null) {
                    Log.d(TAG, "BLE headset sample rate: $best")
                    return best
                }
            }
            // BLE Audio default is typically 16kHz or higher
            return TARGET_SAMPLE_RATE
        }

        // Classic SCO is always 8kHz
        return SCO_SAMPLE_RATE
    }

    /**
     * Route audio through Bluetooth device if available.
     * Returns true if Bluetooth routing was enabled.
     * Returns false if BLUETOOTH_CONNECT permission is not granted (Android 12+).
     */
    fun enableBluetoothAudio(): Boolean {
        if (!hasBluetoothPermission()) {
            Log.d(TAG, "BLUETOOTH_CONNECT permission not granted, falling back to built-in mic")
            return false
        }

        synchronized(audioLock) {
            val device = findBluetoothInputDevice()
            if (device == null) {
                Log.d(TAG, "No Bluetooth device to enable")
                return false
            }

            return try {
                val success = audioManager.setCommunicationDevice(device)
                if (success) {
                    currentBluetoothDevice = device
                    isBluetoothRouted = true
                    Log.i(TAG, "Bluetooth audio enabled: ${device.productName} (type=${device.type})")
                } else {
                    Log.w(TAG, "setCommunicationDevice returned false")
                }
                success
            } catch (e: Exception) {
                Log.e(TAG, "Failed to set communication device", e)
                false
            }
        }
    }

    /**
     * Stop routing audio through Bluetooth, revert to default device.
     */
    fun disableBluetoothAudio() {
        synchronized(audioLock) {
            if (!isBluetoothRouted) return

            try {
                audioManager.clearCommunicationDevice()
                Log.i(TAG, "Bluetooth audio disabled")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to clear communication device", e)
            } finally {
                currentBluetoothDevice = null
                isBluetoothRouted = false
            }
        }
    }

    /**
     * Register for Bluetooth device connection/disconnection events.
     * Does nothing if BLUETOOTH_CONNECT permission is not granted (Android 12+).
     */
    fun setStateListener(listener: BluetoothAudioStateListener?) {
        synchronized(audioLock) {
            stateListener = listener
        }

        if (listener != null && deviceCallback == null && hasBluetoothPermission()) {
            deviceCallback = object : AudioDeviceCallback() {
                override fun onAudioDevicesAdded(addedDevices: Array<out AudioDeviceInfo>) {
                    for (device in addedDevices) {
                        if (device.type == AudioDeviceInfo.TYPE_BLUETOOTH_SCO ||
                            device.type == AudioDeviceInfo.TYPE_BLE_HEADSET) {
                            Log.d(TAG, "Bluetooth device added: ${device.productName}")
                            // Capture listener inside synchronized block to avoid race with setStateListener(null)
                            val capturedListener = synchronized(audioLock) { stateListener } ?: return
                            capturedListener.onBluetoothDeviceConnected(device)
                        }
                    }
                }

                override fun onAudioDevicesRemoved(removedDevices: Array<out AudioDeviceInfo>) {
                    for (device in removedDevices) {
                        // Capture listener and check device state atomically inside synchronized block
                        val (capturedListener, shouldNotify) = synchronized(audioLock) {
                            val l = stateListener
                            val notify = if (device.id == currentBluetoothDevice?.id) {
                                Log.d(TAG, "Current Bluetooth device removed: ${device.productName}")
                                isBluetoothRouted = false
                                currentBluetoothDevice = null
                                true
                            } else {
                                false
                            }
                            Pair(l, notify)
                        }
                        if (shouldNotify && capturedListener != null) {
                            capturedListener.onBluetoothDeviceDisconnected()
                        }
                    }
                }
            }
            try {
                audioManager.registerAudioDeviceCallback(deviceCallback, mainHandler)
            } catch (e: Exception) {
                Log.e(TAG, "Failed to register audio device callback", e)
                deviceCallback = null
            }
        } else if (listener == null && deviceCallback != null) {
            audioManager.unregisterAudioDeviceCallback(deviceCallback)
            deviceCallback = null
        }
    }

    /**
     * Release resources. Call when done with the manager.
     */
    fun release() {
        disableBluetoothAudio()
        setStateListener(null)
    }
}
