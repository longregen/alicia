package com.alicia.assistant.service

import android.media.AudioManager
import android.media.ToneGenerator
import android.util.Log

class AudioFeedbackManager {

    companion object {
        private const val TAG = "AudioFeedback"
        private const val TONE_VOLUME = 80
        private const val BEEP_DURATION_MS = 150
        private const val ERROR_DURATION_MS = 300
    }

    private var toneGenerator: ToneGenerator? = null

    init {
        try {
            toneGenerator = ToneGenerator(AudioManager.STREAM_VOICE_CALL, TONE_VOLUME)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to create ToneGenerator", e)
        }
    }

    fun playStartListening() {
        toneGenerator?.startTone(ToneGenerator.TONE_PROP_BEEP, BEEP_DURATION_MS)
    }

    fun playStopListening() {
        toneGenerator?.startTone(ToneGenerator.TONE_PROP_BEEP2, BEEP_DURATION_MS)
    }

    fun playError() {
        toneGenerator?.startTone(ToneGenerator.TONE_PROP_NACK, ERROR_DURATION_MS)
    }

    fun release() {
        toneGenerator?.release()
        toneGenerator = null
    }
}
