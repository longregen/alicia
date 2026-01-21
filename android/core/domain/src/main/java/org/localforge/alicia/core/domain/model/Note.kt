package org.localforge.alicia.core.domain.model

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
