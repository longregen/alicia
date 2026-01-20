package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Request to vote on an entity
 */
@JsonClass(generateAdapter = true)
data class VoteRequest(
    @Json(name = "vote")
    val vote: String,

    @Json(name = "quick_feedback")
    val quickFeedback: String? = null
)

/**
 * Response model for voting operations
 */
@JsonClass(generateAdapter = true)
data class VoteResponse(
    @Json(name = "target_id")
    val targetId: String,

    @Json(name = "target_type")
    val targetType: String,

    @Json(name = "upvotes")
    val upvotes: Int,

    @Json(name = "downvotes")
    val downvotes: Int,

    @Json(name = "user_vote")
    val userVote: String?,

    @Json(name = "special")
    val special: Map<String, Int>? = null
)

/**
 * Request to submit quick feedback for tool use
 */
@JsonClass(generateAdapter = true)
data class QuickFeedbackRequest(
    @Json(name = "feedback")
    val feedback: String
)

/**
 * Request to submit irrelevance reason for memory usage
 */
@JsonClass(generateAdapter = true)
data class IrrelevanceReasonRequest(
    @Json(name = "reason")
    val reason: String
)

/**
 * Request to submit quality feedback for memory extraction
 */
@JsonClass(generateAdapter = true)
data class QualityFeedbackRequest(
    @Json(name = "feedback")
    val feedback: String
)

/**
 * Vote types
 */
object VoteType {
    const val UP = "up"
    const val DOWN = "down"
    const val CRITICAL = "critical"
}

/**
 * Target types for voting
 */
object VoteTargetType {
    const val MESSAGE = "message"
    const val TOOL_USE = "tool_use"
    const val MEMORY = "memory"
    const val MEMORY_USAGE = "memory_usage"
    const val REASONING = "reasoning"
}
