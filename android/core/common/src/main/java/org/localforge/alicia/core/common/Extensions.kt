package org.localforge.alicia.core.common

import android.Manifest
import android.content.Context
import android.content.Intent
import android.os.VibrationEffect
import android.widget.Toast
import androidx.annotation.RequiresPermission
import androidx.annotation.StringRes

fun Context.showToast(message: String, duration: Int = Toast.LENGTH_SHORT) {
    Toast.makeText(this, message, duration).show()
}

fun Context.showToast(@StringRes messageRes: Int, duration: Int = Toast.LENGTH_SHORT) {
    Toast.makeText(this, messageRes, duration).show()
}

@RequiresPermission(Manifest.permission.VIBRATE)
fun Context.vibrate(durationMs: Long = 50, amplitude: Int = VibrationEffect.DEFAULT_AMPLITUDE) {
    val vibrator = getSystemService(android.os.VibratorManager::class.java)?.defaultVibrator ?: return

    if (!vibrator.hasVibrator()) return

    try {
        val effect = VibrationEffect.createOneShot(durationMs, amplitude)
        vibrator.vibrate(effect)
    } catch (e: Exception) {
        // Vibration is not critical - silent failure is acceptable
    }
}

@RequiresPermission(Manifest.permission.VIBRATE)
fun Context.vibratePattern(pattern: LongArray) {
    val vibrator = getSystemService(android.os.VibratorManager::class.java)?.defaultVibrator ?: return

    if (!vibrator.hasVibrator()) return

    try {
        val effect = VibrationEffect.createWaveform(pattern, -1)
        vibrator.vibrate(effect)
    } catch (e: Exception) {
        // Vibration is not critical - silent failure is acceptable
    }
}

// ActivityManager.getRunningServices() is deprecated but still works for same-app services
fun Context.isServiceRunning(serviceClass: Class<*>): Boolean {
    return try {
        val manager = getSystemService(Context.ACTIVITY_SERVICE) as android.app.ActivityManager
        @Suppress("DEPRECATION")
        manager.getRunningServices(Integer.MAX_VALUE)
            .any { it.service.className == serviceClass.name }
    } catch (e: Exception) {
        false
    }
}

fun Context.getScreenWidth(): Int {
    val windowManager = getSystemService(Context.WINDOW_SERVICE) as android.view.WindowManager
    val metrics = windowManager.currentWindowMetrics
    return metrics.bounds.width()
}

fun Context.getScreenHeight(): Int {
    val windowManager = getSystemService(Context.WINDOW_SERVICE) as android.view.WindowManager
    val metrics = windowManager.currentWindowMetrics
    return metrics.bounds.height()
}

fun Context.dpToPx(dp: Float): Int {
    return (dp * resources.displayMetrics.density).toInt()
}

fun Context.pxToDp(px: Int): Float {
    return px / resources.displayMetrics.density
}

inline fun <T> tryOrNull(block: () -> T): T? {
    return try {
        block()
    } catch (e: Exception) {
        null
    }
}

inline fun <T> tryOrDefault(default: T, block: () -> T): T {
    return try {
        block()
    } catch (e: Exception) {
        default
    }
}

fun String.isValidUrl(): Boolean {
    return try {
        val url = java.net.URL(this)
        url.toURI()
        true
    } catch (e: Exception) {
        false
    }
}

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

fun Long.isBetween(start: Long, end: Long): Boolean {
    return this in start..end
}

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
