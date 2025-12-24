package org.localforge.alicia.core.common

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.provider.Settings
import androidx.core.content.ContextCompat
import androidx.core.net.toUri
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Manages runtime permissions for the Alicia voice assistant app.
 * Handles checking and requesting various permissions needed for
 * voice recording, overlays, accessibility service, and more.
 */
@Singleton
class PermissionManager @Inject constructor(
    @ApplicationContext private val context: Context
) {
    companion object {
        const val PERMISSION_REQUEST_CODE = 1001
        const val OVERLAY_PERMISSION_REQUEST_CODE = 1002
    }

    /**
     * Core permissions required for basic app functionality
     */
    val requiredPermissions = listOf(
        Manifest.permission.RECORD_AUDIO
    )

    /**
     * Optional permissions for enhanced functionality
     */
    val optionalPermissions = mapOf(
        Manifest.permission.SYSTEM_ALERT_WINDOW to "Floating button overlay",
        Manifest.permission.VIBRATE to "Haptic feedback on activation",
        Manifest.permission.RECEIVE_BOOT_COMPLETED to "Auto-start on device boot"
    )

    /**
     * Check if microphone permission is granted
     */
    fun checkMicrophonePermission(): Boolean {
        return ContextCompat.checkSelfPermission(
            context,
            Manifest.permission.RECORD_AUDIO
        ) == PackageManager.PERMISSION_GRANTED
    }

    /**
     * Check if overlay permission is granted (for floating button)
     */
    fun checkOverlayPermission(): Boolean {
        return Settings.canDrawOverlays(context)
    }

    /**
     * Check if vibration permission is granted
     */
    fun checkVibratePermission(): Boolean {
        return ContextCompat.checkSelfPermission(
            context,
            Manifest.permission.VIBRATE
        ) == PackageManager.PERMISSION_GRANTED
    }

    /**
     * Check if the accessibility service is enabled for hardware button detection
     */
    fun isAccessibilityServiceEnabled(): Boolean {
        try {
            val accessibilityEnabled = Settings.Secure.getInt(
                context.contentResolver,
                Settings.Secure.ACCESSIBILITY_ENABLED,
                0
            )
            if (accessibilityEnabled != 1) return false

            val enabledServices = Settings.Secure.getString(
                context.contentResolver,
                Settings.Secure.ENABLED_ACCESSIBILITY_SERVICES
            ) ?: return false

            // Check if our service is in the list
            val serviceName = "${context.packageName}/org.localforge.alicia.service.hotkey.HotkeyAccessibilityService"
            return enabledServices.contains(serviceName)
        } catch (e: Settings.SettingNotFoundException) {
            return false
        }
    }

    /**
     * Check if boot receiver permission is granted
     */
    fun checkBootPermission(): Boolean {
        return ContextCompat.checkSelfPermission(
            context,
            Manifest.permission.RECEIVE_BOOT_COMPLETED
        ) == PackageManager.PERMISSION_GRANTED
    }

    /**
     * Check if all required permissions are granted
     */
    fun areAllRequiredPermissionsGranted(): Boolean {
        return requiredPermissions.all { permission ->
            ContextCompat.checkSelfPermission(
                context,
                permission
            ) == PackageManager.PERMISSION_GRANTED
        }
    }

    /**
     * Get a list of required permissions that are not yet granted
     */
    fun getMissingRequiredPermissions(): List<String> {
        return requiredPermissions.filter { permission ->
            ContextCompat.checkSelfPermission(
                context,
                permission
            ) != PackageManager.PERMISSION_GRANTED
        }
    }

    /**
     * Create an intent to open app settings
     */
    fun createAppSettingsIntent(): Intent {
        return Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS).apply {
            data = "package:${context.packageName}".toUri()
            flags = Intent.FLAG_ACTIVITY_NEW_TASK
        }
    }

    /**
     * Create an intent to open accessibility settings
     */
    fun createAccessibilitySettingsIntent(): Intent {
        return Intent(Settings.ACTION_ACCESSIBILITY_SETTINGS).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK
        }
    }

    /**
     * Create an intent to request overlay permission
     */
    fun createOverlayPermissionIntent(): Intent {
        return Intent(Settings.ACTION_MANAGE_OVERLAY_PERMISSION).apply {
            data = "package:${context.packageName}".toUri()
            flags = Intent.FLAG_ACTIVITY_NEW_TASK
        }
    }

    /**
     * Get a human-readable explanation for why a permission is needed
     */
    fun getPermissionRationale(permission: String): String {
        return when (permission) {
            Manifest.permission.RECORD_AUDIO ->
                "Microphone access is required to listen to your voice commands and communicate with the assistant."

            Manifest.permission.SYSTEM_ALERT_WINDOW ->
                "Overlay permission is needed to display a floating button that lets you quickly activate the assistant from any screen."

            Manifest.permission.VIBRATE ->
                "Vibration permission allows the app to provide haptic feedback when you activate the assistant."

            Manifest.permission.RECEIVE_BOOT_COMPLETED ->
                "This permission allows the voice assistant to start automatically when your device boots up."

            else -> "This permission is needed for proper app functionality."
        }
    }

    /**
     * Check all permissions and return a status report
     */
    fun getPermissionStatus(): PermissionStatus {
        return PermissionStatus(
            microphone = checkMicrophonePermission(),
            overlay = checkOverlayPermission(),
            vibrate = checkVibratePermission(),
            boot = checkBootPermission(),
            accessibilityService = isAccessibilityServiceEnabled()
        )
    }

    /**
     * Data class representing the current status of all permissions
     */
    data class PermissionStatus(
        val microphone: Boolean,
        val overlay: Boolean,
        val vibrate: Boolean,
        val boot: Boolean,
        val accessibilityService: Boolean
    ) {
        fun hasAllRequired(): Boolean = microphone

        fun hasAllOptional(): Boolean = overlay && vibrate && boot

        fun hasAccessibilityEnabled(): Boolean = accessibilityService

        fun getMissingPermissions(): List<String> {
            val missing = mutableListOf<String>()
            if (!microphone) missing.add("Microphone")
            if (!overlay) missing.add("Overlay")
            if (!vibrate) missing.add("Vibrate")
            if (!boot) missing.add("Boot")
            if (!accessibilityService) missing.add("Accessibility Service")
            return missing
        }
    }
}
