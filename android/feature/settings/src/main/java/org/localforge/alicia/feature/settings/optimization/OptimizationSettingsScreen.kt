package org.localforge.alicia.feature.settings.optimization

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.localforge.alicia.core.common.ui.AppIcons
import org.localforge.alicia.core.domain.model.*
import org.localforge.alicia.ui.theme.AliciaTheme

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OptimizationSettingsScreen(
    onNavigateBack: () -> Unit = {},
    conversationId: String? = null,
    onPresetChanged: ((PresetId) -> Unit)? = null,
    onWeightsChanged: ((DimensionWeights) -> Unit)? = null
) {
    val extendedColors = AliciaTheme.extendedColors

    var selectedPreset by remember { mutableStateOf<PresetId?>(PresetId.BALANCED) }
    var weights by remember { mutableStateOf(DimensionWeights.DEFAULT_WEIGHTS) }
    var showAdvanced by remember { mutableStateOf(false) }

    // Update weights when preset changes
    LaunchedEffect(selectedPreset) {
        selectedPreset?.let { presetId ->
            val preset = PivotPresets.getById(presetId)
            weights = preset.weights
            onPresetChanged?.invoke(presetId)
            onWeightsChanged?.invoke(preset.weights)
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Optimization") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = AppIcons.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                }
            )
        }
    ) { paddingValues ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(24.dp)
        ) {
            item {
                ResponseStyleSection(
                    selectedPreset = selectedPreset,
                    onPresetSelected = { presetId ->
                        selectedPreset = presetId
                    },
                    showAdvanced = showAdvanced,
                    onToggleAdvanced = { showAdvanced = !showAdvanced },
                    weights = weights,
                    onWeightChanged = { key, value ->
                        weights = weights.withKey(key, value).normalize()
                        selectedPreset = null // Custom weights
                        onWeightsChanged?.invoke(weights)
                    },
                    onResetToBalanced = {
                        selectedPreset = PresetId.BALANCED
                    },
                    conversationId = conversationId
                )
            }

            if (conversationId == null) {
                item {
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(8.dp),
                        color = extendedColors.muted
                    ) {
                        Text(
                            text = "Select a conversation to sync optimization preferences with the server",
                            style = MaterialTheme.typography.bodyMedium,
                            color = extendedColors.mutedForeground,
                            modifier = Modifier.padding(16.dp)
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ResponseStyleSection(
    selectedPreset: PresetId?,
    onPresetSelected: (PresetId) -> Unit,
    showAdvanced: Boolean,
    onToggleAdvanced: () -> Unit,
    weights: DimensionWeights,
    onWeightChanged: (DimensionKey, Float) -> Unit,
    onResetToBalanced: () -> Unit,
    conversationId: String?
) {
    val extendedColors = AliciaTheme.extendedColors
    val isDisabled = conversationId == null

    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(text = "⚙️", fontSize = 20.sp)
            Text(
                text = "Response Style",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
        }

        // Preset buttons
        Surface(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(12.dp),
            color = extendedColors.card,
            border = androidx.compose.foundation.BorderStroke(1.dp, extendedColors.border)
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    PivotPresets.ALL.forEach { preset ->
                        PresetButton(
                            preset = preset,
                            isSelected = selectedPreset == preset.id,
                            onClick = { if (!isDisabled) onPresetSelected(preset.id) },
                            modifier = Modifier.weight(1f),
                            enabled = !isDisabled
                        )
                    }
                }

                TextButton(
                    onClick = onToggleAdvanced,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text(
                        text = "Custom weights ${if (showAdvanced) "▲" else "▼"}",
                        style = MaterialTheme.typography.bodySmall,
                        color = extendedColors.mutedForeground
                    )
                }

                AnimatedVisibility(
                    visible = showAdvanced,
                    enter = expandVertically(),
                    exit = shrinkVertically()
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 8.dp),
                        verticalArrangement = Arrangement.spacedBy(16.dp)
                    ) {
                        HorizontalDivider(color = extendedColors.border.copy(alpha = 0.5f))

                        DimensionConfigs.ALL.forEach { config ->
                            DimensionSlider(
                                config = config,
                                value = weights.getByKey(config.key),
                                onValueChange = { onWeightChanged(config.key, it) },
                                enabled = !isDisabled
                            )
                        }

                        // Reset button
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.End
                        ) {
                            OutlinedButton(
                                onClick = onResetToBalanced,
                                enabled = !isDisabled && selectedPreset != PresetId.BALANCED
                            ) {
                                Text(
                                    text = "Reset to Balanced",
                                    style = MaterialTheme.typography.bodySmall
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun PresetButton(
    preset: PivotPreset,
    isSelected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true
) {
    val extendedColors = AliciaTheme.extendedColors

    val backgroundColor = if (isSelected) {
        extendedColors.accent.copy(alpha = 0.2f)
    } else {
        extendedColors.muted
    }

    val borderColor = if (isSelected) {
        extendedColors.accent
    } else {
        extendedColors.border
    }

    val textColor = if (isSelected) {
        extendedColors.accent
    } else {
        extendedColors.mutedForeground
    }

    Box(
        modifier = modifier
            .clip(RoundedCornerShape(8.dp))
            .background(backgroundColor)
            .border(1.dp, borderColor, RoundedCornerShape(8.dp))
            .clickable(enabled = enabled, onClick = onClick)
            .padding(vertical = 10.dp, horizontal = 4.dp),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(2.dp)
        ) {
            Text(
                text = preset.icon,
                fontSize = 14.sp
            )
            Text(
                text = preset.label,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = if (isSelected) FontWeight.Medium else FontWeight.Normal,
                color = textColor
            )
        }
    }
}

@Composable
private fun DimensionSlider(
    config: DimensionConfig,
    value: Float,
    onValueChange: (Float) -> Unit,
    enabled: Boolean = true
) {
    val extendedColors = AliciaTheme.extendedColors

    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        Row(
            modifier = Modifier.width(100.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            Text(
                text = config.icon,
                fontSize = 14.sp
            )
            Text(
                text = config.label,
                style = MaterialTheme.typography.bodySmall,
                color = extendedColors.mutedForeground
            )
        }

        Slider(
            value = value,
            onValueChange = onValueChange,
            valueRange = 0f..1f,
            enabled = enabled,
            modifier = Modifier.weight(1f),
            colors = SliderDefaults.colors(
                thumbColor = extendedColors.accent,
                activeTrackColor = extendedColors.accent,
                inactiveTrackColor = extendedColors.muted
            )
        )

        Text(
            text = "${(value * 100).toInt()}%",
            style = MaterialTheme.typography.bodySmall,
            fontWeight = FontWeight.Medium,
            modifier = Modifier.width(40.dp),
            color = MaterialTheme.colorScheme.onBackground
        )
    }
}
