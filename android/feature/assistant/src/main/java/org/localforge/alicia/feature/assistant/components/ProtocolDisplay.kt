package org.localforge.alicia.feature.assistant.components

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import org.localforge.alicia.core.common.ui.AppIcons
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.localforge.alicia.core.domain.model.*
import org.localforge.alicia.feature.assistant.components.toolvisualizations.ToolVisualizationRouter
import org.localforge.alicia.feature.assistant.components.toolvisualizations.ToolIcons
import com.google.gson.GsonBuilder

@Composable
fun ProtocolDisplay(
    errors: List<ErrorMessage>,
    reasoningSteps: List<ReasoningStep>,
    toolUsages: List<ToolUsage>,
    memoryTraces: List<MemoryTrace>,
    commentaries: List<Commentary>,
    modifier: Modifier = Modifier
) {
    val hasMessages = errors.isNotEmpty() || reasoningSteps.isNotEmpty() ||
            toolUsages.isNotEmpty() || memoryTraces.isNotEmpty() || commentaries.isNotEmpty()

    if (!hasMessages) {
        return
    }

    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp)
    ) {
        // Errors
        errors.forEach { error ->
            ErrorMessageItem(error)
            Spacer(modifier = Modifier.height(8.dp))
        }

        // Reasoning Steps
        if (reasoningSteps.isNotEmpty()) {
            ProtocolSection(
                title = "Reasoning",
                color = MaterialTheme.colorScheme.primary
            ) {
                reasoningSteps.forEach { step ->
                    ReasoningStepItem(step)
                    Spacer(modifier = Modifier.height(4.dp))
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
        }

        // Tool Usage
        if (toolUsages.isNotEmpty()) {
            ProtocolSection(
                title = "Tool Usage",
                color = MaterialTheme.colorScheme.tertiary
            ) {
                toolUsages.forEach { usage ->
                    ToolUsageItem(usage)
                    Spacer(modifier = Modifier.height(4.dp))
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
        }

        // Memory Traces
        if (memoryTraces.isNotEmpty()) {
            ProtocolSection(
                title = "Retrieved Memories",
                color = MaterialTheme.colorScheme.secondary
            ) {
                memoryTraces.forEach { trace ->
                    MemoryTraceItem(trace)
                    Spacer(modifier = Modifier.height(4.dp))
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
        }

        // Commentaries
        if (commentaries.isNotEmpty()) {
            ProtocolSection(
                title = "System Commentary",
                color = Color(0xFF9E9E9E)
            ) {
                commentaries.forEach { commentary ->
                    CommentaryItem(commentary)
                    Spacer(modifier = Modifier.height(4.dp))
                }
            }
        }
    }
}

@Composable
fun ProtocolSection(
    title: String,
    color: Color,
    modifier: Modifier = Modifier,
    content: @Composable ColumnScope.() -> Unit
) {
    var expanded by remember { mutableStateOf(true) }

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
        )
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { expanded = !expanded },
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.Bold,
                    color = color
                )
                Icon(
                    imageVector = if (expanded) AppIcons.ExpandLess else AppIcons.ExpandMore,
                    contentDescription = if (expanded) "Collapse" else "Expand",
                    tint = color
                )
            }

            AnimatedVisibility(
                visible = expanded,
                enter = expandVertically(),
                exit = shrinkVertically()
            ) {
                Column(modifier = Modifier.padding(top = 8.dp)) {
                    content()
                }
            }
        }
    }
}

@Composable
fun ErrorMessageItem(error: ErrorMessage) {
    val severityColor = when (error.severity) {
        Severity.INFO -> Color(0xFF2196F3)
        Severity.WARNING -> Color(0xFFFF9800)
        Severity.ERROR -> Color(0xFFF44336)
        Severity.CRITICAL -> Color(0xFF9C27B0)
    }

    val severityLabel = when (error.severity) {
        Severity.INFO -> "INFO"
        Severity.WARNING -> "WARNING"
        Severity.ERROR -> "ERROR"
        Severity.CRITICAL -> "CRITICAL"
    }

    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.3f)
        )
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .border(2.dp, severityColor, RoundedCornerShape(8.dp))
                .padding(12.dp)
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Surface(
                    color = severityColor,
                    shape = RoundedCornerShape(4.dp)
                ) {
                    Text(
                        text = severityLabel,
                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                        fontSize = 10.sp,
                        fontWeight = FontWeight.Bold,
                        color = Color.White
                    )
                }
                Text(
                    text = "Error",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold
                )
            }
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = error.message,
                style = MaterialTheme.typography.bodyMedium
            )
            if (error.code != 0) {
                Text(
                    text = "Code: ${error.code}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 4.dp)
                )
            }
        }
    }
}

@Composable
fun ReasoningStepItem(step: ReasoningStep) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(6.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
        )
    ) {
        Column(modifier = Modifier.padding(10.dp)) {
            Text(
                text = "Step ${step.sequence}",
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.primary
            )
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = step.content,
                style = MaterialTheme.typography.bodyMedium
            )
        }
    }
}

@Composable
fun ToolUsageItem(usage: ToolUsage) {
    var expanded by remember { mutableStateOf(false) }
    val gson = remember { GsonBuilder().setPrettyPrinting().create() }

    val result = usage.result
    val toolName = usage.request.toolName
    val isNativeTool = toolName.startsWith("web_") || toolName.startsWith("garden_")

    val statusColor = when {
        result == null -> Color(0xFFFF9800) // Orange for pending
        result.success -> Color(0xFF4CAF50) // Green for success
        else -> Color(0xFFF44336) // Red for failed
    }

    val statusText = when {
        result == null -> "Running..."
        result.success -> "Success"
        else -> "Failed"
    }

    // Use beautiful visualization for native tools with results
    if (isNativeTool && result?.success == true && result.result != null) {
        Column(modifier = Modifier.fillMaxWidth()) {
            // Compact header with expand toggle
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { expanded = !expanded },
                shape = RoundedCornerShape(8.dp),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)
                )
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(12.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = ToolIcons.getIcon(toolName),
                            fontSize = 18.sp
                        )
                        Text(
                            text = ToolIcons.getDisplayName(toolName),
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                    }
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Surface(
                            color = statusColor,
                            shape = RoundedCornerShape(4.dp)
                        ) {
                            Text(
                                text = statusText,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                                fontSize = 10.sp,
                                fontWeight = FontWeight.Bold,
                                color = Color.White
                            )
                        }
                        Icon(
                            imageVector = if (expanded) AppIcons.ExpandLess else AppIcons.ExpandMore,
                            contentDescription = if (expanded) "Collapse" else "Expand",
                            tint = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }

            // Expanded visualization
            AnimatedVisibility(
                visible = expanded,
                enter = expandVertically(),
                exit = shrinkVertically()
            ) {
                Column(modifier = Modifier.padding(top = 8.dp)) {
                    ToolVisualizationRouter(
                        toolName = toolName,
                        result = result.result
                    )
                }
            }
        }
    } else {
        // Fallback to original display for non-native tools or pending/failed results
        Card(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(6.dp),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.3f)
            )
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { expanded = !expanded }
                    .padding(10.dp)
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = ToolIcons.getIcon(toolName),
                            fontSize = 16.sp
                        )
                        Text(
                            text = toolName,
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.tertiary
                        )
                    }
                    Surface(
                        color = statusColor,
                        shape = RoundedCornerShape(4.dp)
                    ) {
                        Text(
                            text = statusText,
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                            fontSize = 10.sp,
                            fontWeight = FontWeight.Bold,
                            color = Color.White
                        )
                    }
                }

                AnimatedVisibility(visible = expanded) {
                    Column(modifier = Modifier.padding(top = 8.dp)) {
                        Text(
                            text = "Parameters",
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.Bold
                        )
                        Surface(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 4.dp),
                            color = MaterialTheme.colorScheme.surface,
                            shape = RoundedCornerShape(4.dp)
                        ) {
                            Text(
                                text = gson.toJson(usage.request.parameters),
                                modifier = Modifier.padding(8.dp),
                                style = MaterialTheme.typography.bodySmall,
                                fontFamily = FontFamily.Monospace
                            )
                        }

                        usage.result?.let { result ->
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                text = "Result",
                                style = MaterialTheme.typography.labelSmall,
                                fontWeight = FontWeight.Bold
                            )
                            Surface(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(top = 4.dp),
                                color = MaterialTheme.colorScheme.surface,
                                shape = RoundedCornerShape(4.dp)
                            ) {
                                if (result.success) {
                                    Text(
                                        text = gson.toJson(result.result),
                                        modifier = Modifier.padding(8.dp),
                                        style = MaterialTheme.typography.bodySmall,
                                        fontFamily = FontFamily.Monospace
                                    )
                                } else {
                                    Column(modifier = Modifier.padding(8.dp)) {
                                        Text(
                                            text = "Error: ${result.errorMessage}",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.error
                                        )
                                        result.errorCode?.let { code ->
                                            Text(
                                                text = "Code: $code",
                                                style = MaterialTheme.typography.bodySmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant
                                            )
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun MemoryTraceItem(trace: MemoryTrace) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(6.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.secondaryContainer.copy(alpha = 0.3f)
        )
    ) {
        Column(modifier = Modifier.padding(10.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "Memory ${trace.memoryId.take(8)}",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.secondary
                )
                Text(
                    text = "Relevance: ${(trace.relevance * 100).toInt()}%",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = trace.content,
                style = MaterialTheme.typography.bodyMedium
            )
        }
    }
}

@Composable
fun CommentaryItem(commentary: Commentary) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(6.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
        )
    ) {
        Column(modifier = Modifier.padding(10.dp)) {
            Text(
                text = commentary.content,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            commentary.commentaryType?.let { type ->
                Text(
                    text = "Type: $type",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 4.dp)
                )
            }
        }
    }
}
