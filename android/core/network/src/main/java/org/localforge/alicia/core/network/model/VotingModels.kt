package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class VoteRequest(
    @Json(name = "vote")
    val vote: String,

    @Json(name = "quick_feedback")
    val quickFeedback: String? = null
)

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

@JsonClass(generateAdapter = true)
data class QuickFeedbackRequest(
    @Json(name = "feedback")
    val feedback: String
)

@JsonClass(generateAdapter = true)
data class IrrelevanceReasonRequest(
    @Json(name = "reason")
    val reason: String
)

@JsonClass(generateAdapter = true)
data class QualityFeedbackRequest(
    @Json(name = "feedback")
    val feedback: String
)

object VoteType {
    const val UP = "up"
    const val DOWN = "down"
    const val CRITICAL = "critical"
}

object VoteTargetType {
    const val MESSAGE = "message"
    const val TOOL_USE = "tool_use"
    const val MEMORY = "memory"
    const val MEMORY_USAGE = "memory_usage"
    const val REASONING = "reasoning"
}
