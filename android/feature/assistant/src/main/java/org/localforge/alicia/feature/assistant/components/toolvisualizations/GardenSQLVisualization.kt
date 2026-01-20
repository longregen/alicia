package org.localforge.alicia.feature.assistant.components.toolvisualizations

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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Visualization for garden_execute_sql tool results.
 * Displays SQL query results with pagination.
 */
@Composable
fun GardenSQLVisualization(
    result: Map<String, Any?>?,
    modifier: Modifier = Modifier
) {
    if (result == null) {
        ErrorVisualization(message = "No result data", modifier = modifier)
        return
    }

    val success = result["success"] as? Boolean ?: false
    val error = result["error"] as? String
    val columns = (result["columns"] as? List<*>)?.mapNotNull { it as? String } ?: emptyList()
    val rows = (result["rows"] as? List<*>)?.mapNotNull { it as? Map<*, *> } ?: emptyList()
    val rowCount = (result["row_count"] as? Number)?.toInt() ?: rows.size
    val truncated = result["truncated"] as? Boolean ?: false

    var currentPage by remember { mutableStateOf(0) }
    val rowsPerPage = 10
    val totalPages = (rows.size + rowsPerPage - 1) / rowsPerPage
    val displayedRows = rows.drop(currentPage * rowsPerPage).take(rowsPerPage)

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
                        colors = if (success) {
                            listOf(
                                Color(0xFF059669).copy(alpha = 0.3f),
                                Color(0xFF10B981).copy(alpha = 0.3f)
                            )
                        } else {
                            listOf(
                                Color(0xFFDC2626).copy(alpha = 0.3f),
                                Color(0xFFEF4444).copy(alpha = 0.3f)
                            )
                        }
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
                    Text(
                        text = if (success) "⚡" else "❌",
                        fontSize = 24.sp
                    )
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = "Query Results",
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Text(
                            text = "$rowCount row${if (rowCount != 1) "s" else ""} returned${if (truncated) " (truncated)" else ""}",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    Surface(
                        color = if (success) Color(0xFF10B981).copy(alpha = 0.3f) else Color(0xFFEF4444).copy(alpha = 0.3f),
                        shape = RoundedCornerShape(12.dp)
                    ) {
                        Text(
                            text = if (success) "✓ Success" else "✗ Failed",
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.Bold,
                            color = if (success) Color(0xFF6EE7B7) else Color(0xFFFCA5A5),
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                        )
                    }
                }
            }

            if (!success && error != null) {
                // Error display
                Surface(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    color = Color(0xFFDC2626).copy(alpha = 0.2f),
                    shape = RoundedCornerShape(8.dp)
                ) {
                    Text(
                        text = error,
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                        color = Color(0xFFFCA5A5),
                        modifier = Modifier.padding(12.dp)
                    )
                }
            } else if (rows.isEmpty()) {
                // Empty state
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(32.dp),
                    contentAlignment = Alignment.Center
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(text = "📭", fontSize = 32.sp)
                        Spacer(modifier = Modifier.height(8.dp))
                        Text(
                            text = "Query returned no rows",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            } else {
                // Results table
                Column(modifier = Modifier.padding(16.dp)) {
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        color = Color.Black.copy(alpha = 0.2f),
                        shape = RoundedCornerShape(8.dp)
                    ) {
                        Column(
                            modifier = Modifier
                                .horizontalScroll(rememberScrollState())
                                .verticalScroll(rememberScrollState())
                                .heightIn(max = 300.dp)
                        ) {
                            // Header row
                            Row(
                                modifier = Modifier
                                    .background(Color.Black.copy(alpha = 0.2f))
                                    .padding(horizontal = 8.dp, vertical = 8.dp)
                            ) {
                                // Row number column
                                Text(
                                    text = "#",
                                    style = MaterialTheme.typography.labelSmall,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    modifier = Modifier.width(40.dp),
                                    textAlign = TextAlign.Center
                                )
                                columns.forEach { col ->
                                    Text(
                                        text = col,
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = FontWeight.Bold,
                                        fontFamily = FontFamily.Monospace,
                                        color = MaterialTheme.colorScheme.onSurface,
                                        modifier = Modifier
                                            .width(120.dp)
                                            .padding(horizontal = 8.dp)
                                    )
                                }
                            }

                            HorizontalDivider(color = Color.White.copy(alpha = 0.1f))

                            // Data rows
                            displayedRows.forEachIndexed { index, row ->
                                Row(
                                    modifier = Modifier
                                        .padding(horizontal = 8.dp, vertical = 6.dp)
                                ) {
                                    // Row number
                                    Text(
                                        text = "${currentPage * rowsPerPage + index + 1}",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        modifier = Modifier.width(40.dp),
                                        textAlign = TextAlign.Center
                                    )
                                    columns.forEach { col ->
                                        val value = row[col]
                                        Text(
                                            text = formatValue(value),
                                            style = MaterialTheme.typography.bodySmall,
                                            fontFamily = FontFamily.Monospace,
                                            color = getValueColor(value),
                                            maxLines = 1,
                                            overflow = TextOverflow.Ellipsis,
                                            modifier = Modifier
                                                .width(120.dp)
                                                .padding(horizontal = 8.dp)
                                        )
                                    }
                                }
                                if (index < displayedRows.lastIndex) {
                                    HorizontalDivider(
                                        color = Color.White.copy(alpha = 0.05f),
                                        modifier = Modifier.padding(horizontal = 8.dp)
                                    )
                                }
                            }
                        }
                    }

                    // Pagination
                    if (totalPages > 1) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 12.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = "Page ${currentPage + 1} of $totalPages",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                TextButton(
                                    onClick = { currentPage = maxOf(0, currentPage - 1) },
                                    enabled = currentPage > 0
                                ) {
                                    Text("← Prev")
                                }
                                TextButton(
                                    onClick = { currentPage = minOf(totalPages - 1, currentPage + 1) },
                                    enabled = currentPage < totalPages - 1
                                ) {
                                    Text("Next →")
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

private fun formatValue(value: Any?): String {
    return when (value) {
        null -> "NULL"
        is Map<*, *>, is List<*> -> value.toString()
        else -> value.toString()
    }
}

@Composable
private fun getValueColor(value: Any?): Color {
    return when (value) {
        null -> Color(0xFF9CA3AF)
        is Number -> Color(0xFF60A5FA)
        is Boolean -> if (value) Color(0xFF10B981) else Color(0xFFEF4444)
        else -> MaterialTheme.colorScheme.onSurface.copy(alpha = 0.9f)
    }
}
