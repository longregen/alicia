package org.localforge.alicia.core.network.protocol

enum class MessageType(val value: Int) {
    ERROR_MESSAGE(1),
    USER_MESSAGE(2),
    ASSISTANT_MESSAGE(3),
    AUDIO_CHUNK(4),
    REASONING_STEP(5),
    TOOL_USE_REQUEST(6),
    TOOL_USE_RESULT(7),
    ACKNOWLEDGEMENT(8),
    TRANSCRIPTION(9),
    CONTROL_STOP(10),
    CONTROL_VARIATION(11),
    CONFIGURATION(12),
    START_ANSWER(13),
    MEMORY_TRACE(14),
    COMMENTARY(15),
    ASSISTANT_SENTENCE(16),
    SYNC_REQUEST(17),
    SYNC_RESPONSE(18),
    FEEDBACK(20),
    FEEDBACK_CONFIRMATION(21),
    USER_NOTE(22),
    NOTE_CONFIRMATION(23),
    MEMORY_ACTION(24),
    MEMORY_CONFIRMATION(25),
    SERVER_INFO(26),
    SESSION_STATS(27),
    CONVERSATION_UPDATE(28),
    DIMENSION_PREFERENCE(29),
    ELITE_SELECT(30),
    ELITE_OPTIONS(31),
    OPTIMIZATION_PROGRESS(32),
    SUBSCRIBE(40),
    UNSUBSCRIBE(41),
    SUBSCRIBE_ACK(42),
    UNSUBSCRIBE_ACK(43);

    companion object {
        private val map = entries.associateBy(MessageType::value)
        fun fromInt(value: Int) = map[value] ?: ERROR_MESSAGE
    }
}
