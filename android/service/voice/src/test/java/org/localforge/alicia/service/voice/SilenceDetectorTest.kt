package org.localforge.alicia.service.voice

import kotlinx.coroutines.test.runTest
import org.junit.After
import org.junit.Before
import org.junit.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for SilenceDetector.
 */
class SilenceDetectorTest {

    private lateinit var silenceDetector: SilenceDetector
    private var silenceDetectedCount = 0

    @Before
    fun setup() {
        silenceDetectedCount = 0
        silenceDetector = SilenceDetector(
            silenceThresholdMs = 1000L,
            energyThreshold = 500f,
            onSilenceDetected = {
                silenceDetectedCount++
            }
        )
    }

    @After
    fun tearDown() {
        silenceDetector.stop()
    }

    @Test
    fun `processAudio detects silence when energy is below threshold`() = runTest {
        // Create silent audio (all zeros)
        val silentAudio = ByteArray(3200) { 0 }

        silenceDetector.start()

        // Process silent audio
        Thread.sleep(500)
        silenceDetector.processAudio(silentAudio)

        // Wait for silence threshold to pass
        Thread.sleep(1100)

        // Silence should be detected
        assertTrue(silenceDetectedCount > 0, "Silence should have been detected")
    }

    @Test
    fun `processAudio does not detect silence when energy is above threshold`() = runTest {
        // Create audio with some energy
        val audioWithEnergy = ByteArray(3200) { i ->
            ((i % 256) - 128).toByte() // Simple sine-like wave
        }

        silenceDetector.start()

        // Process audio with energy
        repeat(10) {
            silenceDetector.processAudio(audioWithEnergy)
            Thread.sleep(100)
        }

        // Silence should not be detected
        assertEquals(0, silenceDetectedCount, "Silence should not have been detected")
    }

    @Test
    fun `reset resets the silence timer`() = runTest {
        val silentAudio = ByteArray(3200) { 0 }

        silenceDetector.start()

        // Process silent audio
        silenceDetector.processAudio(silentAudio)
        Thread.sleep(500)

        // Reset the timer
        silenceDetector.reset()

        // Wait less than the threshold
        Thread.sleep(500)

        // Silence should not be detected yet
        assertEquals(0, silenceDetectedCount, "Silence should not be detected after reset")
    }

    @Test
    fun `stop stops silence detection`() = runTest {
        silenceDetector.start()
        silenceDetector.stop()

        val silentAudio = ByteArray(3200) { 0 }

        // Process silent audio after stopping
        silenceDetector.processAudio(silentAudio)
        Thread.sleep(1200)

        // Silence should not be detected (detector is stopped)
        assertEquals(0, silenceDetectedCount, "Silence should not be detected when stopped")
    }
}
