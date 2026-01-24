package com.alicia.assistant.model

sealed class RecognitionResult {
    data class Success(val text: String) : RecognitionResult()
    data class Error(val reason: ErrorReason, val cause: Throwable? = null) : RecognitionResult()
}

enum class ErrorReason {
    RECORDING_FAILED,
    RECORDING_STOPPED_EARLY,
    NO_SPEECH_DETECTED,
    SERVER_ERROR,
    NETWORK_ERROR
}
