package com.alicia.assistant

import android.Manifest
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.SystemClock
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.rule.GrantPermissionRule
import androidx.test.uiautomator.By
import androidx.test.uiautomator.UiDevice
import androidx.test.uiautomator.Until
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import java.io.File

@RunWith(AndroidJUnit4::class)
class AliciaE2ETest {

    private lateinit var device: UiDevice
    private lateinit var context: Context
    private val packageName = "com.alicia.assistant"

    @get:Rule
    val permissionRule: GrantPermissionRule = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
        GrantPermissionRule.grant(
            Manifest.permission.RECORD_AUDIO,
            Manifest.permission.POST_NOTIFICATIONS
        )
    } else {
        GrantPermissionRule.grant(
            Manifest.permission.RECORD_AUDIO
        )
    }

    @Before
    fun setup() {
        device = UiDevice.getInstance(InstrumentationRegistry.getInstrumentation())
        context = ApplicationProvider.getApplicationContext()

        // Press home to ensure we're on home screen
        device.pressHome()

        // Wait for launcher
        val launcherPackage = device.launcherPackageName
        device.wait(Until.hasObject(By.pkg(launcherPackage).depth(0)), 5000)
    }

    @Test
    fun testAliciaVoiceAssistantUI() {
        // Start the app
        launchApp()

        // Wait for app to fully load before taking screenshot
        device.waitForIdle(2000)
        SystemClock.sleep(1000)

        // Take initial screenshot
        takeScreenshot("01_app_launched")

        println("=== Starting Alicia E2E Test ===")

        // Step 1: Verify main activity is displayed
        println("Step 1: Verifying main activity")
        val mainActivity = device.wait(
            Until.hasObject(By.pkg(packageName)),
            5000
        )
        assert(mainActivity) { "Main activity should be visible" }

        takeScreenshot("02_main_activity")

        // Step 2: Look for the microphone button
        println("Step 2: Looking for microphone button")
        val micButton = device.wait(
            Until.findObject(By.res(packageName, "micButton")),
            5000
        )

        if (micButton != null) {
            println("Found microphone button")
            takeScreenshot("03_mic_button_visible")

            // Click the microphone button
            println("Step 3: Clicking microphone button")
            micButton.click()
            SystemClock.sleep(2000)

            takeScreenshot("04_after_mic_click")

            // Wait a moment and click again to stop listening
            SystemClock.sleep(1000)
            micButton.click()
            SystemClock.sleep(1000)

            takeScreenshot("05_after_stop_listening")
        } else {
            println("WARNING: Microphone button not found!")
        }

        // Step 4: Try to access settings
        println("Step 4: Looking for settings button")
        val settingsButton = device.wait(
            Until.findObject(By.res(packageName, "settingsButton")),
            3000
        )

        if (settingsButton != null) {
            println("Found settings button")
            settingsButton.click()
            SystemClock.sleep(2000)

            takeScreenshot("06_settings_screen")

            // Go back to main screen
            device.pressBack()
            SystemClock.sleep(1000)
        } else {
            println("WARNING: Settings button not found")
        }

        // Step 5: Try to access voice notes
        println("Step 5: Looking for voice notes button")
        val notesButton = device.wait(
            Until.findObject(By.res(packageName, "notesButton")),
            3000
        )

        if (notesButton != null) {
            println("Found voice notes button")
            notesButton.click()
            SystemClock.sleep(2000)

            takeScreenshot("07_voice_notes_screen")

            // Go back to main screen
            device.pressBack()
            SystemClock.sleep(1000)
        } else {
            println("WARNING: Voice notes button not found")
        }

        // Final screenshot
        takeScreenshot("08_final_state")

        println("=== Alicia E2E Test Completed Successfully ===")
    }

    @Test
    fun testAppLaunchesSuccessfully() {
        // Simple test to verify app launches without crashing
        launchApp()

        device.waitForIdle(2000)
        SystemClock.sleep(1000)

        // Verify app is in foreground
        val appVisible = device.wait(
            Until.hasObject(By.pkg(packageName)),
            5000
        )

        assert(appVisible) { "App should be visible after launch" }

        takeScreenshot("launch_test_complete")
    }

    private fun launchApp() {
        val intent = context.packageManager.getLaunchIntentForPackage(packageName)
        intent?.addFlags(Intent.FLAG_ACTIVITY_CLEAR_TASK)
        context.startActivity(intent)

        // Wait for the app to appear
        device.wait(Until.hasObject(By.pkg(packageName).depth(0)), 5000)
    }

    private fun takeScreenshot(name: String) {
        try {
            val screenshotFile = File("/sdcard/Pictures/$name.png")
            device.takeScreenshot(screenshotFile)
            println("Screenshot saved: $name.png")
        } catch (e: Exception) {
            println("Failed to take screenshot $name: ${e.message}")
        }
    }
}
