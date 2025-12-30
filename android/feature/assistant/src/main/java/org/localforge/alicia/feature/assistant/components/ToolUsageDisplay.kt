package org.localforge.alicia.feature.assistant.components

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
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
import org.localforge.alicia.core.domain.model.ToolUsage
import com.google.gson.GsonBuilder

/**
 * Display tool usages inline with messages.
 * Similar to web's ToolUsageDisplay.tsx
 */
@Composable
fun ToolUsageDisplay(
    toolUsages: List<ToolUsage>,
    modifier: Modifier = Modifier,
    isLatestMessage: Boolean = false
) {
    if (toolUsages.isEmpty()) {
        return
    }

    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(top = 8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        toolUsages.forEach { usage ->
            ToolUsageItem(
                usage = usage,
                isExpanded = isLatestMessage
            )
        }
    }
}

@Composable
private fun ToolUsageItem(
    usage: ToolUsage,
    modifier: Modifier = Modifier,
    isExpanded: Boolean = false
) {
    var expanded by remember(isExpanded) { mutableStateOf(isExpanded) }
    val gson = remember { GsonBuilder().setPrettyPrinting().create() }

    val hasResult = usage.result != null
    val isSuccess = usage.result?.success == true
    val isPending = !hasResult

    val statusColor = when {
        isPending -> Color(0xFFFF9800) // Orange for running
        isSuccess -> Color(0xFF4CAF50) // Green for success
        else -> Color(0xFFF44336) // Red for failed
    }

    val statusText = when {
        isPending -> "Running..."
        isSuccess -> "Success"
        else -> "Failed"
    }

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.6f)
        )
    ) {
        Column(modifier = Modifier.fillMaxWidth()) {
            // Header
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { expanded = !expanded }
                    .padding(12.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(
                        imageVector = AppIcons.Build,
                        contentDescription = "Tool",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.size(20.dp)
                    )
                    Text(
                        text = usage.request.toolName,
                        style = MaterialTheme.typography.labelLarge,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
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
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                            fontSize = 11.sp,
                            fontWeight = FontWeight.Bold,
                            color = Color.White
                        )
                    }
                    Icon(
                        imageVector = if (expanded) AppIcons.ExpandLess else AppIcons.ExpandMore,
                        contentDescription = if (expanded) "Collapse" else "Expand",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.size(20.dp)
                    )
                }
            }

            // Body
            AnimatedVisibility(
                visible = expanded,
                enter = expandVertically(),
                exit = shrinkVertically()
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 12.dp)
                        .padding(bottom = 12.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    HorizontalDivider(
                        color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.2f),
                        modifier = Modifier.padding(bottom = 4.dp)
                    )

                    // Parameters section
                    Text(
                        text = "Parameters",
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        color = MaterialTheme.colorScheme.surface,
                        shape = RoundedCornerShape(4.dp)
                    ) {
                        Text(
                            text = gson.toJson(usage.request.parameters),
                            modifier = Modifier.padding(8.dp),
                            style = MaterialTheme.typography.bodySmall,
                            fontFamily = FontFamily.Monospace,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                    }

                    // Result section
                    if (hasResult) {
                        Text(
                            text = if (isSuccess) "Result" else "Error",
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Surface(
                            modifier = Modifier.fillMaxWidth(),
                            color = MaterialTheme.colorScheme.surface,
                            shape = RoundedCornerShape(4.dp)
                        ) {
                            usage.result?.let { result ->
                                if (isSuccess) {
                                    Text(
                                        text = gson.toJson(result.result),
                                        modifier = Modifier.padding(8.dp),
                                        style = MaterialTheme.typography.bodySmall,
                                        fontFamily = FontFamily.Monospace,
                                        color = MaterialTheme.colorScheme.onSurface
                                    )
                                } else {
                                    Column(modifier = Modifier.padding(8.dp)) {
                                        Text(
                                            text = result.errorMessage ?: "Unknown error",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.error
                                        )
                                        result.errorCode?.let { code ->
                                            Text(
                                                text = "Code: $code",
                                                style = MaterialTheme.typography.bodySmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                modifier = Modifier.padding(top = 4.dp)
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
