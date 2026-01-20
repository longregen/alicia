package org.localforge.alicia.core.network.protocol

/**
 * Protocol message types matching the server-side implementation.
 * Keep in sync with frontend/src/types/protocol.ts and backend pkg/protocol/
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
    ASSISTANT_SENTENCE(16),

    // WebSocket sync message types
    /** Sync request */
    SYNC_REQUEST(17),

    /** Sync response */
    SYNC_RESPONSE(18),

    // Feedback protocol message types (20-27)
    /** User feedback on message/tool/memory */
    FEEDBACK(20),

    /** Confirmation of feedback received */
    FEEDBACK_CONFIRMATION(21),

    /** User note on message/tool */
    USER_NOTE(22),

    /** Confirmation of note saved */
    NOTE_CONFIRMATION(23),

    /** Memory action (create/update/delete) */
    MEMORY_ACTION(24),

    /** Confirmation of memory action */
    MEMORY_CONFIRMATION(25),

    /** Server info response */
    SERVER_INFO(26),

    /** Session statistics */
    SESSION_STATS(27),

    /** Conversation metadata update */
    CONVERSATION_UPDATE(28),

    // Dimension optimization message types (29-32) - match web protocol.ts
    /** Dimension preference selection */
    DIMENSION_PREFERENCE(29),

    /** Elite selection for optimization */
    ELITE_SELECT(30),

    /** Elite options for optimization */
    ELITE_OPTIONS(31),

    /** Optimization progress update */
    OPTIMIZATION_PROGRESS(32),

    // Subscription message types (40-43)
    /** Subscribe to conversation updates */
    SUBSCRIBE(40),

    /** Unsubscribe from conversation updates */
    UNSUBSCRIBE(41),

    /** Subscribe acknowledgement with missed messages */
    SUBSCRIBE_ACK(42),

    /** Unsubscribe acknowledgement */
    UNSUBSCRIBE_ACK(43);

    companion object {
        private val map = entries.associateBy(MessageType::value)
        fun fromInt(value: Int) = map[value] ?: ERROR_MESSAGE
    }
}
