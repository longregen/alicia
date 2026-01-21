package org.localforge.alicia.core.domain.model

data class DimensionWeights(
    val successRate: Float = 0.25f,
    val quality: Float = 0.2f,
    val efficiency: Float = 0.15f,
    val robustness: Float = 0.15f,
    val generalization: Float = 0.1f,
    val diversity: Float = 0.1f,
    val innovation: Float = 0.05f
) {
    fun normalize(): DimensionWeights {
        val sum = successRate + quality + efficiency + robustness +
                  generalization + diversity + innovation

        if (sum == 0f) return DEFAULT_WEIGHTS

        return DimensionWeights(
            successRate = successRate / sum,
            quality = quality / sum,
            efficiency = efficiency / sum,
            robustness = robustness / sum,
            generalization = generalization / sum,
            diversity = diversity / sum,
            innovation = innovation / sum
        )
    }

    companion object {
        val DEFAULT_WEIGHTS = DimensionWeights(
            successRate = 0.25f,
            quality = 0.2f,
            efficiency = 0.15f,
            robustness = 0.15f,
            generalization = 0.1f,
            diversity = 0.1f,
            innovation = 0.05f
        )
    }
}

enum class PresetId {
    ACCURACY,
    SPEED,
    RELIABLE,
    CREATIVE,
    BALANCED
}

data class PivotPreset(
    val id: PresetId,
    val label: String,
    val icon: String,
    val weights: DimensionWeights,
    val description: String
)

data class DimensionConfig(
    val key: DimensionKey,
    val label: String,
    val icon: String
)

enum class DimensionKey {
    SUCCESS_RATE,
    QUALITY,
    EFFICIENCY,
    ROBUSTNESS,
    GENERALIZATION,
    DIVERSITY,
    INNOVATION
}

object PivotPresets {
    val ACCURACY = PivotPreset(
        id = PresetId.ACCURACY,
        label = "Accurate",
        icon = "✓",
        weights = DimensionWeights(
            successRate = 0.4f,
            quality = 0.25f,
            efficiency = 0.1f,
            robustness = 0.1f,
            generalization = 0.1f,
            diversity = 0.03f,
            innovation = 0.02f
        ),
        description = "Prioritize correct answers over speed"
    )

    val SPEED = PivotPreset(
        id = PresetId.SPEED,
        label = "Fast",
        icon = "⚡",
        weights = DimensionWeights(
            successRate = 0.2f,
            quality = 0.15f,
            efficiency = 0.35f,
            robustness = 0.15f,
            generalization = 0.1f,
            diversity = 0.03f,
            innovation = 0.02f
        ),
        description = "Quick responses with reasonable accuracy"
    )

    val RELIABLE = PivotPreset(
        id = PresetId.RELIABLE,
        label = "Reliable",
        icon = "🛡️",
        weights = DimensionWeights(
            successRate = 0.25f,
            quality = 0.2f,
            efficiency = 0.1f,
            robustness = 0.3f,
            generalization = 0.1f,
            diversity = 0.03f,
            innovation = 0.02f
        ),
        description = "Consistent results across different inputs"
    )

    val CREATIVE = PivotPreset(
        id = PresetId.CREATIVE,
        label = "Creative",
        icon = "🎨",
        weights = DimensionWeights(
            successRate = 0.15f,
            quality = 0.2f,
            efficiency = 0.1f,
            robustness = 0.1f,
            generalization = 0.1f,
            diversity = 0.2f,
            innovation = 0.15f
        ),
        description = "Novel approaches and varied solutions"
    )

    val BALANCED = PivotPreset(
        id = PresetId.BALANCED,
        label = "Balanced",
        icon = "⚖️",
        weights = DimensionWeights(
            successRate = 0.25f,
            quality = 0.2f,
            efficiency = 0.15f,
            robustness = 0.15f,
            generalization = 0.1f,
            diversity = 0.1f,
            innovation = 0.05f
        ),
        description = "Moderate emphasis favoring success and quality"
    )

    val ALL = listOf(ACCURACY, SPEED, RELIABLE, CREATIVE, BALANCED)

    fun getById(id: PresetId): PivotPreset = when (id) {
        PresetId.ACCURACY -> ACCURACY
        PresetId.SPEED -> SPEED
        PresetId.RELIABLE -> RELIABLE
        PresetId.CREATIVE -> CREATIVE
        PresetId.BALANCED -> BALANCED
    }
}

object DimensionConfigs {
    val ALL = listOf(
        DimensionConfig(DimensionKey.SUCCESS_RATE, "Accuracy", "✓"),
        DimensionConfig(DimensionKey.QUALITY, "Quality", "★"),
        DimensionConfig(DimensionKey.EFFICIENCY, "Speed", "⚡"),
        DimensionConfig(DimensionKey.ROBUSTNESS, "Reliability", "🛡️"),
        DimensionConfig(DimensionKey.GENERALIZATION, "Adaptability", "🔄"),
        DimensionConfig(DimensionKey.DIVERSITY, "Creativity", "🎨"),
        DimensionConfig(DimensionKey.INNOVATION, "Novelty", "💡")
    )
}

fun DimensionWeights.getByKey(key: DimensionKey): Float = when (key) {
    DimensionKey.SUCCESS_RATE -> successRate
    DimensionKey.QUALITY -> quality
    DimensionKey.EFFICIENCY -> efficiency
    DimensionKey.ROBUSTNESS -> robustness
    DimensionKey.GENERALIZATION -> generalization
    DimensionKey.DIVERSITY -> diversity
    DimensionKey.INNOVATION -> innovation
}

fun DimensionWeights.withKey(key: DimensionKey, value: Float): DimensionWeights = when (key) {
    DimensionKey.SUCCESS_RATE -> copy(successRate = value)
    DimensionKey.QUALITY -> copy(quality = value)
    DimensionKey.EFFICIENCY -> copy(efficiency = value)
    DimensionKey.ROBUSTNESS -> copy(robustness = value)
    DimensionKey.GENERALIZATION -> copy(generalization = value)
    DimensionKey.DIVERSITY -> copy(diversity = value)
    DimensionKey.INNOVATION -> copy(innovation = value)
}
