package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Request to create a note
 */
@JsonClass(generateAdapter = true)
data class CreateNoteRequest(
    @Json(name = "content")
    val content: String,

    @Json(name = "category")
    val category: String = "general"
)

/**
 * Request to update a note
 */
@JsonClass(generateAdapter = true)
data class UpdateNoteRequest(
    @Json(name = "content")
    val content: String
)

/**
 * Response model for a single note
 */
@JsonClass(generateAdapter = true)
data class NoteResponse(
    @Json(name = "id")
    val id: String,

    @Json(name = "message_id")
    val messageId: String? = null,

    @Json(name = "target_id")
    val targetId: String,

    @Json(name = "target_type")
    val targetType: String,

    @Json(name = "content")
    val content: String,

    @Json(name = "category")
    val category: String,

    @Json(name = "created_at")
    val createdAt: Long,

    @Json(name = "updated_at")
    val updatedAt: Long
)

/**
 * Response model for note list
 */
@JsonClass(generateAdapter = true)
data class NoteListResponse(
    @Json(name = "notes")
    val notes: List<NoteResponse>,

    @Json(name = "total")
    val total: Int
)

/**
 * Note target types
 */
object NoteTargetType {
    const val MESSAGE = "message"
    const val TOOL_USE = "tool_use"
    const val REASONING = "reasoning"
}

/**
 * Note categories - matches web frontend
 */
object NoteCategory {
    const val IMPROVEMENT = "improvement"
    const val CORRECTION = "correction"
    const val CONTEXT = "context"
    const val GENERAL = "general"
}
