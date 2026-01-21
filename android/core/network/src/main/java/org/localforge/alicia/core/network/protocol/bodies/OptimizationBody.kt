package org.localforge.alicia.core.network.protocol.bodies

data class DimensionWeights(
    val successRate: Float,
    val quality: Float,
    val efficiency: Float,
    val robustness: Float,
    val generalization: Float,
    val diversity: Float,
    val innovation: Float
)

data class DimensionScores(
    val successRate: Float,
    val quality: Float,
    val efficiency: Float,
    val robustness: Float,
    val generalization: Float,
    val diversity: Float,
    val innovation: Float
)

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

data class DimensionPreferenceBody(
    val conversationId: String,
    val weights: DimensionWeights,
    val preset: DimensionPreset? = null,
    val timestamp: Long
)

data class EliteSummary(
    val id: String,
    val label: String,
    val scores: DimensionScores,
    val description: String,
    val bestFor: String
)

data class EliteOptionsBody(
    val conversationId: String,
    val elites: List<EliteSummary>,
    val currentEliteId: String,
    val timestamp: Long
)

enum class OptimizationStatus(val value: String) {
    PENDING("pending"),
    RUNNING("running"),
    COMPLETED("completed"),
    FAILED("failed");

    companion object {
        fun fromString(value: String?) = entries.find { it.value == value }
    }
}

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

data class EliteSelectBody(
    val conversationId: String,
    val eliteId: String,
    val timestamp: Long
)
