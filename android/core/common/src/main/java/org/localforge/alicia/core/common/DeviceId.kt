package org.localforge.alicia.core.common

import android.content.Context
import androidx.core.content.edit
import java.util.UUID

object DeviceId {
    private const val PREFS_NAME = "alicia_device_prefs"
    private const val KEY_DEVICE_ID = "device_id"

    private var cachedDeviceId: String? = null

    private fun generateDeviceId(): String {
        val timestamp = System.currentTimeMillis()
        val random = UUID.randomUUID().toString().replace("-", "").substring(0, 12)
        return "android_${timestamp}_$random"
    }

    @Synchronized
    fun get(context: Context): String {
        cachedDeviceId?.let { return it }

        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        var deviceId = prefs.getString(KEY_DEVICE_ID, null)

        if (deviceId == null) {
            deviceId = generateDeviceId()
            prefs.edit {
                putString(KEY_DEVICE_ID, deviceId)
            }
        }

        cachedDeviceId = deviceId
        return deviceId
    }

    @Synchronized
    fun reset(context: Context) {
        cachedDeviceId = null
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE).edit {
            remove(KEY_DEVICE_ID)
        }
    }

    fun getParticipantId(context: Context): String {
        return "user_${get(context)}"
    }
}
