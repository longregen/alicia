package org.localforge.alicia.core.domain.model

/**
 * Domain model for a user note attached to messages, tool uses, or reasoning steps.
 * Matches the web frontend's note functionality.
 */
data class Note(
    val id: String,
    val targetId: String,
    val targetType: NoteTargetType,
    val content: String,
    val category: NoteCategory,
    val messageId: String? = null,
    val createdAt: Long,
    val updatedAt: Long
)

/**
 * Target types for notes
 */
enum class NoteTargetType(val value: String) {
    MESSAGE("message"),
    TOOL_USE("tool_use"),
    REASONING("reasoning");

    companion object {
        fun fromString(value: String?): NoteTargetType {
            return entries.find { it.value == value } ?: MESSAGE
        }
    }
}

/**
 * Categories for notes - matches web frontend
 */
enum class NoteCategory(val value: String) {
    IMPROVEMENT("improvement"),
    CORRECTION("correction"),
    CONTEXT("context"),
    GENERAL("general");

    companion object {
        fun fromString(value: String?): NoteCategory {
            return entries.find { it.value == value } ?: GENERAL
        }
    }
}
