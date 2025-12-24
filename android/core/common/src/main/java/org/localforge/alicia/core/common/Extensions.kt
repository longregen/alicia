package org.localforge.alicia.core.common

import android.Manifest
import android.content.Context
import android.content.Intent
import android.os.VibrationEffect
import android.widget.Toast
import androidx.annotation.RequiresPermission
import androidx.annotation.StringRes

/**
 * Extension functions for common Android operations
 */

/**
 * Show a short toast message
 */
fun Context.showToast(message: String, duration: Int = Toast.LENGTH_SHORT) {
    Toast.makeText(this, message, duration).show()
}

/**
 * Show a short toast message from string resource
 */
fun Context.showToast(@StringRes messageRes: Int, duration: Int = Toast.LENGTH_SHORT) {
    Toast.makeText(this, messageRes, duration).show()
}

/**
 * Vibrate the device for a short duration
 * @param durationMs Duration in milliseconds (default: 50ms)
 * @param amplitude Vibration amplitude (default: default amplitude)
 */
@RequiresPermission(Manifest.permission.VIBRATE)
fun Context.vibrate(durationMs: Long = 50, amplitude: Int = VibrationEffect.DEFAULT_AMPLITUDE) {
    val vibrator = getSystemService(android.os.VibratorManager::class.java)?.defaultVibrator ?: return

    if (!vibrator.hasVibrator()) return

    try {
        val effect = VibrationEffect.createOneShot(durationMs, amplitude)
        vibrator.vibrate(effect)
    } catch (e: Exception) {
        // Silently fail - vibration is not critical
    }
}

/**
 * Vibrate with a pattern
 * @param pattern Array of alternating durations of off/on (e.g., [0, 50, 100, 50])
 */
@RequiresPermission(Manifest.permission.VIBRATE)
fun Context.vibratePattern(pattern: LongArray) {
    val vibrator = getSystemService(android.os.VibratorManager::class.java)?.defaultVibrator ?: return

    if (!vibrator.hasVibrator()) return

    try {
        val effect = VibrationEffect.createWaveform(pattern, -1)
        vibrator.vibrate(effect)
    } catch (e: Exception) {
        // Silently fail
    }
}

/**
 * Check if a service is running
 *
 * Note: ActivityManager.getRunningServices() is deprecated as of API 26 with no replacement.
 * This method only works reliably for services within the same application.
 * For cross-app service detection, there is no alternative due to privacy restrictions.
 *
 * Since we only need to check our own services, this implementation is acceptable.
 */
fun Context.isServiceRunning(serviceClass: Class<*>): Boolean {
    return try {
        val manager = getSystemService(Context.ACTIVITY_SERVICE) as android.app.ActivityManager
        // This deprecated API is the only way to check if a service is running.
        // It still works for the app's own services as of Android 14.
        @Suppress("DEPRECATION")
        manager.getRunningServices(Integer.MAX_VALUE)
            .any { it.service.className == serviceClass.name }
    } catch (e: Exception) {
        false
    }
}

/**
 * Get screen width in pixels
 */
fun Context.getScreenWidth(): Int {
    val windowManager = getSystemService(Context.WINDOW_SERVICE) as android.view.WindowManager
    val metrics = windowManager.currentWindowMetrics
    return metrics.bounds.width()
}

/**
 * Get screen height in pixels
 */
fun Context.getScreenHeight(): Int {
    val windowManager = getSystemService(Context.WINDOW_SERVICE) as android.view.WindowManager
    val metrics = windowManager.currentWindowMetrics
    return metrics.bounds.height()
}

/**
 * Convert dp to pixels
 */
fun Context.dpToPx(dp: Float): Int {
    return (dp * resources.displayMetrics.density).toInt()
}

/**
 * Convert pixels to dp
 */
fun Context.pxToDp(px: Int): Float {
    return px / resources.displayMetrics.density
}

/**
 * Execute block and return result, catching exceptions
 */
inline fun <T> tryOrNull(block: () -> T): T? {
    return try {
        block()
    } catch (e: Exception) {
        null
    }
}

/**
 * Execute block and return result or default value on exception
 */
inline fun <T> tryOrDefault(default: T, block: () -> T): T {
    return try {
        block()
    } catch (e: Exception) {
        default
    }
}

/**
 * Extension to check if a string is a valid URL
 */
fun String.isValidUrl(): Boolean {
    return try {
        val url = java.net.URL(this)
        url.toURI()
        true
    } catch (e: Exception) {
        false
    }
}

/**
 * Format milliseconds to human-readable duration
 */
fun Long.formatDuration(): String {
    val seconds = this / 1000
    val minutes = seconds / 60
    val hours = minutes / 60

    return when {
        hours > 0 -> "${hours}h ${minutes % 60}m"
        minutes > 0 -> "${minutes}m ${seconds % 60}s"
        else -> "${seconds}s"
    }
}

/**
 * Check if the current time is between two timestamps
 */
fun Long.isBetween(start: Long, end: Long): Boolean {
    return this in start..end
}

/**
 * Safe intent extra retrieval
 */
inline fun <reified T> Intent.getSafeExtra(key: String, default: T): T {
    return try {
        when (T::class) {
            String::class -> getStringExtra(key) as? T ?: default
            Int::class -> getIntExtra(key, default as Int) as T
            Long::class -> getLongExtra(key, default as Long) as T
            Boolean::class -> getBooleanExtra(key, default as Boolean) as T
            Float::class -> getFloatExtra(key, default as Float) as T
            Double::class -> getDoubleExtra(key, default as Double) as T
            else -> default
        }
    } catch (e: Exception) {
        default
    }
}
