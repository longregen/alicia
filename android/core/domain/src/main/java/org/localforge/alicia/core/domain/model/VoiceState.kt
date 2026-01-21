package org.localforge.alicia.core.domain.model

enum class VoiceState {
    IDLE,
    LISTENING_FOR_WAKE_WORD,
    ACTIVATED,
    LISTENING,
    PROCESSING,
    SPEAKING,
    ERROR,
    CONNECTING,
    DISCONNECTED;

    val isActive: Boolean
        get() = this in listOf(ACTIVATED, LISTENING, PROCESSING, SPEAKING)

    val canAcceptInput: Boolean
        get() = this in listOf(IDLE, LISTENING_FOR_WAKE_WORD, LISTENING)

    val isRecording: Boolean
        get() = this in listOf(LISTENING_FOR_WAKE_WORD, LISTENING)
}
