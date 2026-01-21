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

enum class FetchType {
    RAW, STRUCTURED
}

@Composable
fun WebFetchVisualization(
    result: Map<String, Any?>?,
    type: FetchType,
    modifier: Modifier = Modifier
) {
    if (result == null) {
        ErrorVisualization(message = "No result data", modifier = modifier)
        return
    }

    val gson = remember { GsonBuilder().setPrettyPrinting().create() }

    val gradientColors = when (type) {
        FetchType.RAW -> listOf(
            Color(0xFF1E40AF).copy(alpha = 0.3f),
            Color(0xFF3730A3).copy(alpha = 0.3f)
        )
        FetchType.STRUCTURED -> listOf(
            Color(0xFF065F46).copy(alpha = 0.3f),
            Color(0xFF047857).copy(alpha = 0.3f)
        )
    }

    val icon = when (type) {
        FetchType.RAW -> "🌐"
        FetchType.STRUCTURED -> "🏗️"
    }

    val title = when (type) {
        FetchType.RAW -> "Fetch Raw"
        FetchType.STRUCTURED -> "Fetch Structured"
    }

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = Color.Transparent)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .background(brush = Brush.linearGradient(gradientColors))
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
                            text = title,
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        val url = result["url"] as? String
                        url?.let {
                            Text(
                                text = it,
                                style = MaterialTheme.typography.bodySmall,
                                color = Color(0xFF60A5FA),
                                maxLines = 1
                            )
                        }
                    }
                }
            }

            when (type) {
                FetchType.RAW -> RawFetchContent(result = result, gson = gson)
                FetchType.STRUCTURED -> StructuredFetchContent(result = result, gson = gson)
            }
        }
    }
}

@Composable
private fun RawFetchContent(
    result: Map<String, Any?>,
    gson: com.google.gson.Gson
) {
    var showHeaders by remember { mutableStateOf(false) }
    var isExpanded by remember { mutableStateOf(false) }

    val statusCode = (result["status_code"] as? Number)?.toInt()
    val headers = result["headers"] as? Map<*, *>
    val body = result["body"] as? String ?: ""
    val contentType = result["content_type"] as? String

    Column(modifier = Modifier.padding(16.dp)) {
        // Status and content type
        FlowRow(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            statusCode?.let {
                val statusColor = when {
                    it in 200..299 -> Color(0xFF10B981)
                    it in 300..399 -> Color(0xFFFBBF24)
                    it in 400..499 -> Color(0xFFF59E0B)
                    else -> Color(0xFFEF4444)
                }
                MetadataBadge(
                    text = "Status: $it",
                    backgroundColor = statusColor.copy(alpha = 0.3f),
                    textColor = statusColor
                )
            }
            contentType?.let {
                MetadataBadge(
                    text = it,
                    backgroundColor = Color(0xFF6B7280).copy(alpha = 0.3f),
                    textColor = Color(0xFFD1D5DB)
                )
            }
        }

        // Headers toggle
        headers?.let {
            TextButton(
                onClick = { showHeaders = !showHeaders },
                modifier = Modifier.padding(top = 8.dp),
                contentPadding = PaddingValues(0.dp)
            ) {
                Text(
                    text = if (showHeaders) "▼ Hide headers" else "▶ Show headers (${it.size})",
                    style = MaterialTheme.typography.labelSmall,
                    color = Color(0xFF60A5FA)
                )
            }

            AnimatedVisibility(
                visible = showHeaders,
                enter = expandVertically(),
                exit = shrinkVertically()
            ) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    color = Color.Black.copy(alpha = 0.2f),
                    shape = RoundedCornerShape(8.dp)
                ) {
                    Column(
                        modifier = Modifier
                            .padding(12.dp)
                            .heightIn(max = 150.dp)
                            .verticalScroll(rememberScrollState())
                    ) {
                        it.forEach { (key, value) ->
                            Row(modifier = Modifier.padding(vertical = 2.dp)) {
                                Text(
                                    text = "$key: ",
                                    style = MaterialTheme.typography.bodySmall,
                                    fontFamily = FontFamily.Monospace,
                                    fontWeight = FontWeight.Bold,
                                    color = Color(0xFF93C5FD)
                                )
                                Text(
                                    text = value.toString(),
                                    style = MaterialTheme.typography.bodySmall,
                                    fontFamily = FontFamily.Monospace,
                                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
                                )
                            }
                        }
                    }
                }
            }
        }

        // Body
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = "Response Body",
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface
        )

        Surface(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 4.dp),
            color = Color.Black.copy(alpha = 0.2f),
            shape = RoundedCornerShape(8.dp)
        ) {
            val displayBody = if (isExpanded) body else body.take(500)
            Text(
                text = if (!isExpanded && body.length > 500) "$displayBody..." else displayBody,
                style = MaterialTheme.typography.bodySmall,
                fontFamily = FontFamily.Monospace,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.9f),
                modifier = Modifier
                    .padding(12.dp)
                    .heightIn(max = if (isExpanded) 300.dp else 150.dp)
                    .verticalScroll(rememberScrollState())
                    .horizontalScroll(rememberScrollState())
            )
        }

        if (body.length > 500) {
            TextButton(
                onClick = { isExpanded = !isExpanded },
                contentPadding = PaddingValues(0.dp)
            ) {
                Text(
                    text = if (isExpanded) "← Show less" else "Show full response",
                    style = MaterialTheme.typography.labelSmall,
                    color = Color(0xFF60A5FA)
                )
            }
        }
    }
}

@Composable
private fun StructuredFetchContent(
    result: Map<String, Any?>,
    gson: com.google.gson.Gson
) {
    val data = result["data"] as? Map<*, *>
    val error = result["error"] as? String

    Column(modifier = Modifier.padding(16.dp)) {
        if (error != null) {
            Surface(
                modifier = Modifier.fillMaxWidth(),
                color = Color(0xFFDC2626).copy(alpha = 0.2f),
                shape = RoundedCornerShape(8.dp)
            ) {
                Text(
                    text = error,
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFFFCA5A5),
                    modifier = Modifier.padding(12.dp)
                )
            }
        } else if (data != null) {
            Text(
                text = "Extracted Data",
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface
            )

            Surface(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 8.dp),
                color = Color.Black.copy(alpha = 0.2f),
                shape = RoundedCornerShape(8.dp)
            ) {
                Column(
                    modifier = Modifier
                        .padding(12.dp)
                        .heightIn(max = 300.dp)
                        .verticalScroll(rememberScrollState())
                ) {
                    data.forEach { (key, value) ->
                        Column(modifier = Modifier.padding(vertical = 4.dp)) {
                            Text(
                                text = key.toString(),
                                style = MaterialTheme.typography.labelSmall,
                                fontWeight = FontWeight.Bold,
                                color = Color(0xFF10B981)
                            )
                            Text(
                                text = when (value) {
                                    is List<*> -> value.joinToString("\n")
                                    is Map<*, *> -> gson.toJson(value)
                                    else -> value.toString()
                                },
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.9f),
                                modifier = Modifier.padding(top = 2.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}
