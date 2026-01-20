package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Request to create a new memory
 */
@JsonClass(generateAdapter = true)
data class CreateMemoryRequest(
    @Json(name = "content")
    val content: String,

    @Json(name = "category")
    val category: String
)

/**
 * Request to update a memory
 */
@JsonClass(generateAdapter = true)
data class UpdateMemoryRequest(
    @Json(name = "content")
    val content: String
)

/**
 * Request to add tags to a memory
 */
@JsonClass(generateAdapter = true)
data class AddMemoryTagsRequest(
    @Json(name = "tags")
    val tags: List<String>
)

/**
 * Request to pin/unpin a memory
 */
@JsonClass(generateAdapter = true)
data class PinMemoryRequest(
    @Json(name = "pinned")
    val pinned: Boolean
)

/**
 * Request to set memory importance
 */
@JsonClass(generateAdapter = true)
data class SetMemoryImportanceRequest(
    @Json(name = "importance")
    val importance: Int
)

/**
 * Request to search memories
 */
@JsonClass(generateAdapter = true)
data class SearchMemoriesRequest(
    @Json(name = "query")
    val query: String,

    @Json(name = "limit")
    val limit: Int = 10
)

/**
 * Response model for a single memory
 */
@JsonClass(generateAdapter = true)
data class MemoryResponse(
    @Json(name = "id")
    val id: String,

    @Json(name = "content")
    val content: String,

    @Json(name = "category")
    val category: String,

    @Json(name = "importance")
    val importance: Int,

    @Json(name = "tags")
    val tags: List<String> = emptyList(),

    @Json(name = "pinned")
    val pinned: Boolean = false,

    @Json(name = "archived")
    val archived: Boolean = false,

    @Json(name = "createdAt")
    val createdAt: Long,

    @Json(name = "updatedAt")
    val updatedAt: Long
)

/**
 * Response model for memory list
 */
@JsonClass(generateAdapter = true)
data class MemoryListResponse(
    @Json(name = "memories")
    val memories: List<MemoryResponse>,

    @Json(name = "total")
    val total: Int
)

/**
 * Memory categories
 */
object MemoryCategory {
    const val PREFERENCE = "preference"
    const val FACT = "fact"
    const val CONTEXT = "context"
    const val INSTRUCTION = "instruction"
}
