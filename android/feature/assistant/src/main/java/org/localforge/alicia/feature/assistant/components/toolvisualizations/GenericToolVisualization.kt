package org.localforge.alicia.feature.assistant.components.toolvisualizations

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.google.gson.GsonBuilder

@Composable
fun GenericToolVisualization(
    toolName: String,
    result: Any?,
    modifier: Modifier = Modifier
) {
    val gson = remember { GsonBuilder().setPrettyPrinting().create() }
    var isExpanded by remember { mutableStateOf(false) }

    val icon = ToolIcons.getIcon(toolName)
    val displayName = ToolIcons.getDisplayName(toolName)

    val resultString = remember(result) {
        when (result) {
            is String -> result
            null -> "null"
            else -> try {
                gson.toJson(result)
            } catch (e: Exception) {
                result.toString()
            }
        }
    }

    val preview = resultString.take(300)
    val hasMore = resultString.length > 300

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = Color.Transparent)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .background(
                    brush = Brush.linearGradient(
                        colors = listOf(
                            Color(0xFF374151).copy(alpha = 0.3f),
                            Color(0xFF4B5563).copy(alpha = 0.3f)
                        )
                    )
                )
        ) {
            // Header
            Surface(
                modifier = Modifier.fillMaxWidth(),
                color = Color.Black.copy(alpha = 0.2f)
            ) {
                Row(
                    modifier = Modifier.padding(16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Text(text = icon, fontSize = 24.sp)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = displayName,
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Text(
                            text = toolName,
                            style = MaterialTheme.typography.bodySmall,
                            fontFamily = FontFamily.Monospace,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }

            // Result
            Column(modifier = Modifier.padding(16.dp)) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    color = Color.Black.copy(alpha = 0.2f),
                    shape = RoundedCornerShape(8.dp)
                ) {
                    Text(
                        text = if (isExpanded) resultString else "$preview${if (hasMore) "..." else ""}",
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.9f),
                        modifier = Modifier
                            .padding(12.dp)
                            .heightIn(max = if (isExpanded) 400.dp else 200.dp)
                            .verticalScroll(rememberScrollState())
                            .horizontalScroll(rememberScrollState())
                    )
                }

                if (hasMore) {
                    TextButton(
                        onClick = { isExpanded = !isExpanded },
                        contentPadding = PaddingValues(0.dp)
                    ) {
                        Text(
                            text = if (isExpanded) "← Show less" else "Show full result",
                            style = MaterialTheme.typography.labelSmall,
                            color = Color(0xFF60A5FA)
                        )
                    }
                }
            }
        }
    }
}
