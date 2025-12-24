package org.localforge.alicia.core.network.protocol

/**
 * Protocol message types matching the server-side implementation
 */
enum class MessageType(val value: Int) {
    /** Error notification */
    ERROR_MESSAGE(1),

    /** User's text input */
    USER_MESSAGE(2),

    /** Complete assistant response (non-streaming) */
    ASSISTANT_MESSAGE(3),

    /** Raw audio data segment */
    AUDIO_CHUNK(4),

    /** Internal reasoning trace */
    REASONING_STEP(5),

    /** Request to execute a tool */
    TOOL_USE_REQUEST(6),

    /** Tool execution result */
    TOOL_USE_RESULT(7),

    /** Confirm receipt */
    ACKNOWLEDGEMENT(8),

    /** Speech-to-text output */
    TRANSCRIPTION(9),

    /** Stop current operation */
    CONTROL_STOP(10),

    /** Edit/vary previous message */
    CONTROL_VARIATION(11),

    /** Session configuration */
    CONFIGURATION(12),

    /** Begin streaming response */
    START_ANSWER(13),

    /** Memory retrieval log */
    MEMORY_TRACE(14),

    /** Assistant's internal commentary */
    COMMENTARY(15),

    /** Streaming response chunk */
    ASSISTANT_SENTENCE(16);

    companion object {
        private val map = values().associateBy(MessageType::value)
        fun fromInt(value: Int) = map[value] ?: ERROR_MESSAGE
    }
}
