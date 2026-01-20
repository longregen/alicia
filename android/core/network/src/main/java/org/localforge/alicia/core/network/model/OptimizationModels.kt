package org.localforge.alicia.core.network.model

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Request to create an optimization run
 */
@JsonClass(generateAdapter = true)
data class CreateOptimizationRequest(
    @Json(name = "name")
    val name: String,

    @Json(name = "prompt_type")
    val promptType: String,

    @Json(name = "baseline_prompt")
    val baselinePrompt: String? = null
)

/**
 * Optimization run response
 */
@JsonClass(generateAdapter = true)
data class OptimizationRunResponse(
    @Json(name = "id")
    val id: String,

    @Json(name = "name")
    val name: String,

    @Json(name = "prompt_type")
    val promptType: String,

    @Json(name = "status")
    val status: String,

    @Json(name = "best_score")
    val bestScore: Double,

    @Json(name = "iterations")
    val iterations: Int,

    @Json(name = "max_iterations")
    val maxIterations: Int,

    @Json(name = "config")
    val config: Map<String, Any>? = null,

    @Json(name = "created_at")
    val createdAt: String,

    @Json(name = "completed_at")
    val completedAt: String? = null,

    @Json(name = "dimension_weights")
    val dimensionWeights: Map<String, Double>? = null,

    @Json(name = "best_dim_scores")
    val bestDimScores: Map<String, Double>? = null
)

/**
 * Optimization candidate response
 */
@JsonClass(generateAdapter = true)
data class OptimizationCandidateResponse(
    @Json(name = "id")
    val id: String,

    @Json(name = "iteration")
    val iteration: Int,

    @Json(name = "prompt_text")
    val promptText: String,

    @Json(name = "score")
    val score: Double,

    @Json(name = "dimension_scores")
    val dimensionScores: Map<String, Double>? = null,

    @Json(name = "evaluation_count")
    val evaluationCount: Int,

    @Json(name = "success_count")
    val successCount: Int,

    @Json(name = "created_at")
    val createdAt: String
)

/**
 * Response for optimization candidates list
 */
@JsonClass(generateAdapter = true)
data class OptimizationCandidatesResponse(
    @Json(name = "candidates")
    val candidates: List<OptimizationCandidateResponse>
)

/**
 * Optimization run status values
 */
object OptimizationStatus {
    const val PENDING = "pending"
    const val RUNNING = "running"
    const val COMPLETED = "completed"
    const val FAILED = "failed"
    const val CANCELLED = "cancelled"
}
