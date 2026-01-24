package com.alicia.assistant.service

import android.content.Intent
import android.speech.RecognitionService
import android.speech.SpeechRecognizer
import android.util.Log

class AliciaRecognitionService : RecognitionService() {

    companion object {
        private const val TAG = "AliciaRecognition"
    }

    override fun onStartListening(intent: Intent?, callback: Callback?) {
        Log.d(TAG, "onStartListening")
        callback?.error(SpeechRecognizer.ERROR_CLIENT)
    }

    override fun onCancel(callback: Callback?) {
        Log.d(TAG, "onCancel")
    }

    override fun onStopListening(callback: Callback?) {
        Log.d(TAG, "onStopListening")
    }
}
