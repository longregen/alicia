package org.localforge.alicia.core.common

import android.content.Context
import androidx.core.content.edit
import java.util.UUID

/**
 * Device ID utility for persistent per-device identification.
 * Stores device ID in SharedPreferences to maintain consistency across app launches.
 */
object DeviceId {
    private const val PREFS_NAME = "alicia_device_prefs"
    private const val KEY_DEVICE_ID = "device_id"

    private var cachedDeviceId: String? = null

    /**
     * Generate a random device ID
     */
    private fun generateDeviceId(): String {
        val timestamp = System.currentTimeMillis()
        val random = UUID.randomUUID().toString().replace("-", "").substring(0, 12)
        return "android_${timestamp}_$random"
    }

    /**
     * Get or create persistent device ID from SharedPreferences
     */
    @Synchronized
    fun get(context: Context): String {
        // Return cached value if available
        cachedDeviceId?.let { return it }

        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

        // Try to get existing device ID from SharedPreferences
        var deviceId = prefs.getString(KEY_DEVICE_ID, null)

        if (deviceId == null) {
            // Generate new device ID if none exists
            deviceId = generateDeviceId()
            prefs.edit {
                putString(KEY_DEVICE_ID, deviceId)
            }
        }

        // Cache the device ID
        cachedDeviceId = deviceId

        return deviceId
    }

    /**
     * Reset device ID (useful for testing or logout)
     */
    @Synchronized
    fun reset(context: Context) {
        cachedDeviceId = null
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE).edit {
            remove(KEY_DEVICE_ID)
        }
    }

    /**
     * Get device ID formatted as LiveKit participant ID
     */
    fun getParticipantId(context: Context): String {
        return "user_${get(context)}"
    }
}
