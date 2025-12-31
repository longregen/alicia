package org.localforge.alicia.core.domain.model

/**
 * Represents the current state of the voice assistant.
 */
enum class VoiceState {
    /**
     * Voice assistant is idle, not listening or processing.
     */
    IDLE,

    /**
     * Background listening for wake word detection.
     */
    LISTENING_FOR_WAKE_WORD,

    /**
     * Wake word detected, assistant is activated.
     */
    ACTIVATED,

    /**
     * Actively listening to user input.
     */
    LISTENING,

    /**
     * Processing user input and generating response.
     */
    PROCESSING,

    /**
     * Assistant is speaking/playing audio response.
     */
    SPEAKING,

    /**
     * Error state - something went wrong.
     */
    ERROR,

    /**
     * Connecting to the server.
     */
    CONNECTING,

    /**
     * Disconnected from the server.
     */
    DISCONNECTED;

    /**
     * Check if the assistant is in an active state (activated, listening, processing, or speaking).
     */
    val isActive: Boolean
        get() = this in listOf(ACTIVATED, LISTENING, PROCESSING, SPEAKING)

    /**
     * Check if the assistant can accept user input or activation.
     * IDLE accepts manual activation, LISTENING_FOR_WAKE_WORD accepts wake word,
     * and LISTENING accepts voice input.
     */
    val canAcceptInput: Boolean
        get() = this in listOf(IDLE, LISTENING_FOR_WAKE_WORD, LISTENING)

    /**
     * Check if audio is being recorded.
     * Includes both wake word detection (buffered/low-power) and active listening (full recording).
     */
    val isRecording: Boolean
        get() = this in listOf(LISTENING_FOR_WAKE_WORD, LISTENING)
}
