package org.localforge.alicia.core.network.protocol.bodies

/**
 * Feedback target types
 */
enum class FeedbackTargetType(val value: String) {
    MESSAGE("message"),
    TOOL_USE("tool_use"),
    MEMORY("memory"),
    REASONING("reasoning");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * Vote type for feedback
 */
enum class VoteType(val value: String) {
    UP("up"),
    DOWN("down"),
    CRITICAL("critical"),
    REMOVE("remove");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * Feedback body (Type 20)
 * User feedback on message/tool/memory
 */
data class FeedbackBody(
    val id: String,
    val conversationId: String,
    val messageId: String,
    val targetType: FeedbackTargetType,
    val targetId: String,
    val vote: VoteType,
    val quickFeedback: String? = null,
    val note: String? = null,
    val timestamp: Long
)

/**
 * Feedback aggregates
 */
data class FeedbackAggregates(
    val upvotes: Int,
    val downvotes: Int,
    val specialVotes: Map<String, Int>? = null
)

/**
 * Feedback confirmation body (Type 21)
 */
data class FeedbackConfirmationBody(
    val feedbackId: String,
    val targetType: FeedbackTargetType,
    val targetId: String,
    val aggregates: FeedbackAggregates,
    val userVote: VoteType?
)

/**
 * Note category
 */
enum class NoteCategory(val value: String) {
    IMPROVEMENT("improvement"),
    CORRECTION("correction"),
    CONTEXT("context"),
    GENERAL("general");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * Note action
 */
enum class NoteAction(val value: String) {
    CREATE("create"),
    UPDATE("update"),
    DELETE("delete");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * User note body (Type 22)
 */
data class UserNoteBody(
    val id: String,
    val messageId: String,
    val content: String,
    val category: NoteCategory,
    val action: NoteAction,
    val timestamp: Long
)

/**
 * Note confirmation body (Type 23)
 */
data class NoteConfirmationBody(
    val noteId: String,
    val messageId: String,
    val success: Boolean
)

/**
 * Memory category for protocol
 */
enum class ProtocolMemoryCategory(val value: String) {
    PREFERENCE("preference"),
    FACT("fact"),
    CONTEXT("context"),
    INSTRUCTION("instruction");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * Memory action type
 */
enum class MemoryActionType(val value: String) {
    CREATE("create"),
    UPDATE("update"),
    DELETE("delete"),
    PIN("pin"),
    ARCHIVE("archive");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * Memory data for actions
 */
data class MemoryData(
    val content: String,
    val category: ProtocolMemoryCategory,
    val pinned: Boolean? = null
)

/**
 * Memory action body (Type 24)
 */
data class MemoryActionBody(
    val id: String,
    val action: MemoryActionType,
    val memory: MemoryData? = null,
    val timestamp: Long
)

/**
 * Memory confirmation body (Type 25)
 */
data class MemoryConfirmationBody(
    val memoryId: String,
    val action: MemoryActionType,
    val success: Boolean
)
