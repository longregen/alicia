package org.localforge.alicia.feature.assistant.components.toolvisualizations

import androidx.compose.foundation.background
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

@Composable
fun GardenTableVisualization(
    result: Map<String, Any?>?,
    modifier: Modifier = Modifier
) {
    if (result == null) {
        ErrorVisualization(message = "No result data", modifier = modifier)
        return
    }

    val tableName = result["table_name"] as? String ?: "Unknown"
    val schema = result["schema"] as? String
    val columns = (result["columns"] as? List<*>)?.mapNotNull { it as? Map<*, *> } ?: emptyList()
    val primaryKey = result["primary_key"] as? List<*>
    val foreignKeys = (result["foreign_keys"] as? List<*>)?.mapNotNull { it as? Map<*, *> } ?: emptyList()
    val indexes = (result["indexes"] as? List<*>)?.mapNotNull { it as? Map<*, *> } ?: emptyList()
    val rowCount = (result["row_count"] as? Number)?.toLong()

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
                            Color(0xFF6366F1).copy(alpha = 0.3f),
                            Color(0xFF8B5CF6).copy(alpha = 0.3f)
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
                    Text(text = "📊", fontSize = 24.sp)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = tableName,
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            fontFamily = FontFamily.Monospace,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            schema?.let {
                                Text(
                                    text = "Schema: $it",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                            rowCount?.let {
                                Text(
                                    text = "• ${formatNumber(it)} rows",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                        }
                    }
                }
            }

            Column(
                modifier = Modifier
                    .padding(16.dp)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Columns
                if (columns.isNotEmpty()) {
                    TableSection(title = "Columns", icon = "📋", count = columns.size) {
                        columns.forEach { col ->
                            ColumnRow(col, primaryKey)
                        }
                    }
                }

                // Primary Key
                primaryKey?.let { pk ->
                    if (pk.isNotEmpty()) {
                        TableSection(title = "Primary Key", icon = "🔑", color = Color(0xFFFBBF24)) {
                            Text(
                                text = pk.joinToString(", "),
                                style = MaterialTheme.typography.bodySmall,
                                fontFamily = FontFamily.Monospace,
                                color = Color(0xFFFCD34D)
                            )
                        }
                    }
                }

                // Foreign Keys
                if (foreignKeys.isNotEmpty()) {
                    TableSection(title = "Foreign Keys", icon = "🔗", count = foreignKeys.size, color = Color(0xFF8B5CF6)) {
                        foreignKeys.forEach { fk ->
                            ForeignKeyRow(fk)
                        }
                    }
                }

                // Indexes
                if (indexes.isNotEmpty()) {
                    TableSection(title = "Indexes", icon = "📑", count = indexes.size, color = Color(0xFF10B981)) {
                        indexes.forEach { idx ->
                            IndexRow(idx)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun TableSection(
    title: String,
    icon: String,
    count: Int? = null,
    color: Color = MaterialTheme.colorScheme.primary,
    content: @Composable ColumnScope.() -> Unit
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        color = Color.Black.copy(alpha = 0.2f),
        shape = RoundedCornerShape(8.dp)
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(6.dp)
            ) {
                Text(text = icon, fontSize = 14.sp)
                Text(
                    text = title,
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold,
                    color = color
                )
                count?.let {
                    Text(
                        text = "($it)",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
            content()
        }
    }
}

@Composable
private fun ColumnRow(
    column: Map<*, *>,
    primaryKey: List<*>?
) {
    val name = column["name"] as? String ?: ""
    val type = column["type"] as? String ?: ""
    val nullable = column["nullable"] as? Boolean ?: true
    val defaultValue = column["default"] as? String
    val isPk = primaryKey?.contains(name) == true

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = name,
                style = MaterialTheme.typography.bodySmall,
                fontFamily = FontFamily.Monospace,
                fontWeight = if (isPk) FontWeight.Bold else FontWeight.Normal,
                color = if (isPk) Color(0xFFFCD34D) else MaterialTheme.colorScheme.onSurface
            )
            if (isPk) {
                Text(text = "🔑", fontSize = 10.sp)
            }
        }

        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            MetadataBadge(
                text = type,
                backgroundColor = Color(0xFF3B82F6).copy(alpha = 0.3f),
                textColor = Color(0xFF93C5FD)
            )
            if (!nullable) {
                MetadataBadge(
                    text = "NOT NULL",
                    backgroundColor = Color(0xFFEF4444).copy(alpha = 0.3f),
                    textColor = Color(0xFFFCA5A5)
                )
            }
        }
    }
}

@Composable
private fun ForeignKeyRow(fk: Map<*, *>) {
    val columns = (fk["columns"] as? List<*>)?.joinToString(", ") ?: ""
    val refTable = fk["referenced_table"] as? String ?: ""
    val refColumns = (fk["referenced_columns"] as? List<*>)?.joinToString(", ") ?: ""

    Text(
        text = "$columns → $refTable($refColumns)",
        style = MaterialTheme.typography.bodySmall,
        fontFamily = FontFamily.Monospace,
        color = Color(0xFFC4B5FD),
        modifier = Modifier.padding(vertical = 2.dp)
    )
}

@Composable
private fun IndexRow(index: Map<*, *>) {
    val name = index["name"] as? String ?: ""
    val columns = (index["columns"] as? List<*>)?.joinToString(", ") ?: ""
    val isUnique = index["unique"] as? Boolean ?: false

    Row(
        modifier = Modifier.padding(vertical = 2.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = name,
            style = MaterialTheme.typography.bodySmall,
            fontFamily = FontFamily.Monospace,
            color = Color(0xFF6EE7B7)
        )
        if (isUnique) {
            MetadataBadge(
                text = "UNIQUE",
                backgroundColor = Color(0xFFFBBF24).copy(alpha = 0.3f),
                textColor = Color(0xFFFCD34D)
            )
        }
        Text(
            text = "($columns)",
            style = MaterialTheme.typography.bodySmall,
            fontFamily = FontFamily.Monospace,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}

private fun formatNumber(num: Long): String {
    return when {
        num >= 1_000_000 -> "${num / 1_000_000}M"
        num >= 1_000 -> "${num / 1_000}K"
        else -> num.toString()
    }
}
