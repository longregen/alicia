package com.alicia.assistant.service

/**
 * Audio resampling utilities for converting between sample rates.
 *
 * Used to convert 8kHz Bluetooth SCO audio to 16kHz for Whisper/Vosk processing.
 */
object AudioResampler {

    /**
     * Resample audio from source rate to target rate using linear interpolation.
     *
     * @param input PCM samples as ShortArray
     * @param srcRate Source sample rate (e.g., 8000)
     * @param dstRate Target sample rate (e.g., 16000)
     * @return Resampled PCM samples
     */
    fun resample(input: ShortArray, srcRate: Int, dstRate: Int): ShortArray {
        if (srcRate == dstRate) return input
        if (input.isEmpty()) return input

        val ratio = dstRate.toDouble() / srcRate
        val outputLength = (input.size * ratio).toInt()
        val output = ShortArray(outputLength)

        for (i in 0 until outputLength) {
            val srcPos = i / ratio
            val srcIndex = srcPos.toInt()
            val frac = srcPos - srcIndex

            val sample = if (srcIndex + 1 < input.size) {
                // Linear interpolation between two samples
                val s0 = input[srcIndex].toDouble()
                val s1 = input[srcIndex + 1].toDouble()
                (s0 + frac * (s1 - s0)).toInt().coerceIn(Short.MIN_VALUE.toInt(), Short.MAX_VALUE.toInt())
            } else {
                input[srcIndex].toInt()
            }

            output[i] = sample.toShort()
        }

        return output
    }

    /**
     * Resample audio from source rate to target rate (ByteArray version).
     * Assumes signed 16-bit little-endian PCM audio.
     *
     * @param input PCM bytes (signed 16-bit LE, must have even length)
     * @param srcRate Source sample rate (e.g., 8000)
     * @param dstRate Target sample rate (e.g., 16000)
     * @return Resampled PCM bytes
     * @throws IllegalArgumentException if input has odd length
     */
    fun resampleBytes(input: ByteArray, srcRate: Int, dstRate: Int): ByteArray {
        if (srcRate == dstRate) return input
        if (input.isEmpty()) return input
        require(input.size % 2 == 0) { "Input byte array must have even length for 16-bit PCM samples" }

        // Convert bytes to shorts (signed 16-bit little-endian PCM).
        // Low byte: must mask with 0xFF to prevent sign extension from corrupting the value
        // (e.g., byte 0xFF should become int 255, not -1).
        // High byte: no mask needed - any sign extension is discarded when we shift left
        // by 8 bits and convert the combined value to Short.
        val shortInput = ShortArray(input.size / 2)
        for (i in shortInput.indices) {
            val lo = input[i * 2].toInt() and 0xFF
            val hi = input[i * 2 + 1].toInt()
            shortInput[i] = ((hi shl 8) or lo).toShort()
        }

        // Resample
        val shortOutput = resample(shortInput, srcRate, dstRate)

        // Convert back to bytes (16-bit LE)
        val output = ByteArray(shortOutput.size * 2)
        for (i in shortOutput.indices) {
            val sample = shortOutput[i].toInt()
            output[i * 2] = (sample and 0xFF).toByte()
            output[i * 2 + 1] = ((sample shr 8) and 0xFF).toByte()
        }

        return output
    }

    /**
     * Resample a single frame of audio (used for real-time VAD processing).
     * Maintains state for continuous resampling across frame boundaries.
     *
     * Thread-safe: all access to mutable state is synchronized.
     */
    class StreamResampler(private val srcRate: Int, private val dstRate: Int) {
        private val lock = Any()
        @Volatile private var lastSample: Short = 0

        fun resampleFrame(input: ShortArray): ShortArray {
            if (srcRate == dstRate) return input
            if (input.isEmpty()) return input

            val ratio = dstRate.toDouble() / srcRate
            val outputLength = (input.size * ratio).toInt()
            val output = ShortArray(outputLength)

            synchronized(lock) {
                // Capture lastSample at start for consistent interpolation
                val previousFrameLastSample = lastSample

                for (i in 0 until outputLength) {
                    val srcPos = i / ratio
                    val srcIndex = srcPos.toInt()
                    val frac = srcPos - srcIndex

                    // Determine s0 (the sample before the interpolation point)
                    // When srcIndex == 0, we need the sample "before" index 0:
                    // - If frac == 0, srcPos is exactly at input[0], so s0 = input[0]
                    // - If frac > 0, srcPos is between input[-1] and input[0], so s0 = previousFrameLastSample
                    val s0: Double
                    val s1: Double

                    if (srcIndex == 0 && frac > 0) {
                        // Interpolating between previous frame's last sample and input[0]
                        s0 = previousFrameLastSample.toDouble()
                        s1 = input[0].toDouble()
                    } else if (srcIndex < input.size) {
                        s0 = input[srcIndex].toDouble()
                        s1 = if (srcIndex + 1 < input.size) {
                            input[srcIndex + 1].toDouble()
                        } else {
                            // At the end of input, no next sample to interpolate to
                            input[srcIndex].toDouble()
                        }
                    } else {
                        // srcIndex beyond input bounds (shouldn't happen with correct outputLength)
                        s0 = input.last().toDouble()
                        s1 = s0
                    }

                    val sample = (s0 + frac * (s1 - s0)).toInt()
                        .coerceIn(Short.MIN_VALUE.toInt(), Short.MAX_VALUE.toInt())
                    output[i] = sample.toShort()
                }

                // Save last sample for next frame's boundary interpolation
                lastSample = input.last()
            }

            return output
        }

        fun reset() {
            synchronized(lock) {
                lastSample = 0
            }
        }
    }
}
