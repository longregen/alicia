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
 * Displays tool usage information with collapsible parameter and result details.
 * Matches web's ToolUseCard.tsx component.
 *
 * Features:
 * - Tool name with emoji icon
 * - Execution status badge
 * - Expandable parameter/result sections
 * - Voting controls
 */
@Composable
fun ToolUsageDisplay(
    toolUsages: List<ToolUsage>,
    modifier: Modifier = Modifier,
    isLatestMessage: Boolean = false,
    onVote: ((String, Boolean) -> Unit)? = null  // (toolUsageId, isUpvote) -> Unit
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
                isExpanded = isLatestMessage,
                onVote = onVote
            )
        }
    }
}

/**
 * Get emoji icon for tool based on name, matching web's toolIcons.
 */
private fun getToolIcon(toolName: String): String {
    val name = toolName.lowercase()
    return when {
        name == "memory_query" || name.contains("memory") -> "🧠"
        name.startsWith("web_") || name.contains("web") -> "🌐"
        name.contains("search") || name.contains("find") || name.contains("query") -> "🔍"
        name.contains("calculate") || name.contains("math") || name.contains("compute") -> "🔢"
        name.contains("file") || name.contains("read") || name.contains("write") -> "📄"
        name.contains("code") || name.contains("execute") || name.contains("run") -> "⚙️"
        name.contains("garden") || name.contains("sql") || name.contains("database") -> "🌱"
        else -> "🔧"
    }
}

/**
 * Get display-friendly name for tool, matching web's toolDisplayNames.
 */
private fun getToolDisplayName(toolName: String): String {
    return when (toolName) {
        "web_read" -> "Read Web Page"
        "web_fetch_raw" -> "Fetch Raw"
        "web_fetch_structured" -> "Extract Data"
        "web_search" -> "Web Search"
        "web_extract_links" -> "Extract Links"
        "web_extract_metadata" -> "Page Metadata"
        "web_screenshot" -> "Screenshot"
        "garden_describe_table" -> "Describe Table"
        "garden_execute_sql" -> "Execute SQL"
        "garden_schema_explore" -> "Explore Schema"
        "memory_query" -> "Memory Search"
        else -> toolName.replace("_", " ")
            .split(" ")
            .joinToString(" ") { it.replaceFirstChar { c -> c.uppercase() } }
    }
}

@Composable
private fun ToolUsageItem(
    usage: ToolUsage,
    modifier: Modifier = Modifier,
    isExpanded: Boolean = false,
    onVote: ((String, Boolean) -> Unit)? = null
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

    val toolIcon = getToolIcon(usage.request.toolName)
    val toolDisplayName = getToolDisplayName(usage.request.toolName)

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
                    // Tool emoji icon
                    Text(
                        text = toolIcon,
                        fontSize = 18.sp
                    )
                    Column {
                        Text(
                            text = toolDisplayName,
                            style = MaterialTheme.typography.labelLarge,
                            fontWeight = FontWeight.SemiBold,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }

                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Surface(
                        color = statusColor,
                        shape = RoundedCornerShape(12.dp)
                    ) {
                        Text(
                            text = statusText,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                            fontSize = 11.sp,
                            fontWeight = FontWeight.SemiBold,
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

                    // Voting controls
                    if (onVote != null) {
                        HorizontalDivider(
                            color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.2f),
                            modifier = Modifier.padding(vertical = 8.dp)
                        )
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = "Was this helpful?",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                            IconButton(
                                onClick = { onVote(usage.request.id, true) },
                                modifier = Modifier.size(28.dp)
                            ) {
                                Icon(
                                    imageVector = AppIcons.ThumbUp,
                                    contentDescription = "Helpful",
                                    modifier = Modifier.size(16.dp),
                                    tint = Color(0xFF10B981)
                                )
                            }
                            IconButton(
                                onClick = { onVote(usage.request.id, false) },
                                modifier = Modifier.size(28.dp)
                            ) {
                                Icon(
                                    imageVector = AppIcons.ThumbDown,
                                    contentDescription = "Not helpful",
                                    modifier = Modifier.size(16.dp),
                                    tint = Color(0xFFEF4444)
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
