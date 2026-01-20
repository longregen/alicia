package org.localforge.alicia.core.network.protocol.bodies

/**
 * Dimension weights for optimization
 */
data class DimensionWeights(
    val successRate: Float,
    val quality: Float,
    val efficiency: Float,
    val robustness: Float,
    val generalization: Float,
    val diversity: Float,
    val innovation: Float
)

/**
 * Dimension scores
 */
data class DimensionScores(
    val successRate: Float,
    val quality: Float,
    val efficiency: Float,
    val robustness: Float,
    val generalization: Float,
    val diversity: Float,
    val innovation: Float
)

/**
 * Dimension preset
 */
enum class DimensionPreset(val value: String) {
    ACCURACY("accuracy"),
    SPEED("speed"),
    RELIABLE("reliable"),
    CREATIVE("creative"),
    BALANCED("balanced");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * Dimension preference body (Type 30)
 */
data class DimensionPreferenceBody(
    val conversationId: String,
    val weights: DimensionWeights,
    val preset: DimensionPreset? = null,
    val timestamp: Long
)

/**
 * Elite summary
 */
data class EliteSummary(
    val id: String,
    val label: String,
    val scores: DimensionScores,
    val description: String,
    val bestFor: String
)

/**
 * Elite options body (Type 31)
 */
data class EliteOptionsBody(
    val conversationId: String,
    val elites: List<EliteSummary>,
    val currentEliteId: String,
    val timestamp: Long
)

/**
 * Optimization status
 */
enum class OptimizationStatus(val value: String) {
    PENDING("pending"),
    RUNNING("running"),
    COMPLETED("completed"),
    FAILED("failed");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

/**
 * Optimization progress body (Type 32)
 */
data class OptimizationProgressBody(
    val runId: String,
    val status: OptimizationStatus,
    val iteration: Int,
    val maxIterations: Int,
    val currentScore: Float,
    val bestScore: Float,
    val dimensionScores: Map<String, Float>? = null,
    val message: String? = null,
    val timestamp: Long
)

/**
 * Elite select body (Type 33)
 */
data class EliteSelectBody(
    val conversationId: String,
    val eliteId: String,
    val timestamp: Long
)
