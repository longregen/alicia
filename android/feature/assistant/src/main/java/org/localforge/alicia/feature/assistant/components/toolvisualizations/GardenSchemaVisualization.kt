package org.localforge.alicia.feature.assistant.components.toolvisualizations

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
fun GardenSchemaVisualization(
    result: Map<String, Any?>?,
    modifier: Modifier = Modifier
) {
    if (result == null) {
        ErrorVisualization(message = "No result data", modifier = modifier)
        return
    }

    val schemas = (result["schemas"] as? List<*>)?.mapNotNull { it as? Map<*, *> } ?: emptyList()
    val tables = (result["tables"] as? List<*>)?.mapNotNull { it as? Map<*, *> } ?: emptyList()
    val relationships = (result["relationships"] as? List<*>)?.mapNotNull { it as? Map<*, *> } ?: emptyList()
    val totalTables = (result["total_tables"] as? Number)?.toInt() ?: tables.size
    val totalSchemas = (result["total_schemas"] as? Number)?.toInt() ?: schemas.size

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
                            Color(0xFF0284C7).copy(alpha = 0.3f),
                            Color(0xFF0891B2).copy(alpha = 0.3f)
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
                    Text(text = "🗺️", fontSize = 24.sp)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = "Schema Explorer",
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Text(
                            text = "$totalSchemas schema(s), $totalTables table(s)",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }

            Column(
                modifier = Modifier
                    .padding(16.dp)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                // Schemas
                if (schemas.isNotEmpty()) {
                    SchemaSection(title = "Schemas", icon = "📁", count = schemas.size) {
                        schemas.forEach { schema ->
                            SchemaItem(schema)
                        }
                    }
                }

                // Tables
                if (tables.isNotEmpty()) {
                    SchemaSection(title = "Tables", icon = "📊", count = tables.size) {
                        tables.forEach { table ->
                            TableItem(table)
                        }
                    }
                }

                // Relationships
                if (relationships.isNotEmpty()) {
                    SchemaSection(
                        title = "Relationships",
                        icon = "🔗",
                        count = relationships.size,
                        color = Color(0xFF8B5CF6)
                    ) {
                        relationships.forEach { rel ->
                            RelationshipItem(rel)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SchemaSection(
    title: String,
    icon: String,
    count: Int? = null,
    color: Color = Color(0xFF0EA5E9),
    content: @Composable ColumnScope.() -> Unit
) {
    var expanded by remember { mutableStateOf(true) }

    Surface(
        modifier = Modifier.fillMaxWidth(),
        color = Color.Black.copy(alpha = 0.2f),
        shape = RoundedCornerShape(8.dp)
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { expanded = !expanded },
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
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
                Text(
                    text = if (expanded) "▼" else "▶",
                    style = MaterialTheme.typography.labelSmall,
                    color = color
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
private fun SchemaItem(schema: Map<*, *>) {
    val name = schema["name"] as? String ?: ""
    val tableCount = (schema["table_count"] as? Number)?.toInt()

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = name,
            style = MaterialTheme.typography.bodySmall,
            fontFamily = FontFamily.Monospace,
            color = MaterialTheme.colorScheme.onSurface
        )
        tableCount?.let {
            MetadataBadge(
                text = "$it tables",
                backgroundColor = Color(0xFF6B7280).copy(alpha = 0.3f),
                textColor = Color(0xFFD1D5DB)
            )
        }
    }
}

@Composable
private fun TableItem(table: Map<*, *>) {
    val name = table["name"] as? String ?: ""
    val schema = table["schema"] as? String
    val columnCount = (table["column_count"] as? Number)?.toInt()
    val rowCount = (table["row_count"] as? Number)?.toLong()

    var expanded by remember { mutableStateOf(false) }
    val columns = (table["columns"] as? List<*>)?.mapNotNull { it as? Map<*, *> }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable(enabled = columns != null) { expanded = !expanded },
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = if (schema != null) "$schema.$name" else name,
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Medium,
                    color = Color(0xFF38BDF8)
                )
                if (columns != null) {
                    Text(
                        text = if (expanded) "▼" else "▶",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color(0xFF38BDF8)
                    )
                }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                columnCount?.let {
                    MetadataBadge(
                        text = "$it cols",
                        backgroundColor = Color(0xFF3B82F6).copy(alpha = 0.3f),
                        textColor = Color(0xFF93C5FD)
                    )
                }
                rowCount?.let {
                    MetadataBadge(
                        text = "${formatNumber(it)} rows",
                        backgroundColor = Color(0xFF10B981).copy(alpha = 0.3f),
                        textColor = Color(0xFF6EE7B7)
                    )
                }
            }
        }

        // Expanded columns
        columns?.let { cols ->
            AnimatedVisibility(
                visible = expanded,
                enter = expandVertically(),
                exit = shrinkVertically()
            ) {
                Surface(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 8.dp),
                    color = Color.Black.copy(alpha = 0.2f),
                    shape = RoundedCornerShape(4.dp)
                ) {
                    Column(modifier = Modifier.padding(8.dp)) {
                        cols.forEach { col ->
                            val colName = col["name"] as? String ?: ""
                            val colType = col["type"] as? String ?: ""
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(vertical = 2.dp),
                                horizontalArrangement = Arrangement.SpaceBetween
                            ) {
                                Text(
                                    text = colName,
                                    style = MaterialTheme.typography.bodySmall,
                                    fontFamily = FontFamily.Monospace,
                                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
                                )
                                Text(
                                    text = colType,
                                    style = MaterialTheme.typography.bodySmall,
                                    fontFamily = FontFamily.Monospace,
                                    color = Color(0xFF93C5FD)
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
private fun RelationshipItem(rel: Map<*, *>) {
    val fromTable = rel["from_table"] as? String ?: ""
    val fromColumn = rel["from_column"] as? String ?: ""
    val toTable = rel["to_table"] as? String ?: ""
    val toColumn = rel["to_column"] as? String ?: ""
    val type = rel["type"] as? String

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = "$fromTable.$fromColumn",
            style = MaterialTheme.typography.bodySmall,
            fontFamily = FontFamily.Monospace,
            color = Color(0xFFC4B5FD)
        )
        Text(
            text = "→",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Text(
            text = "$toTable.$toColumn",
            style = MaterialTheme.typography.bodySmall,
            fontFamily = FontFamily.Monospace,
            color = Color(0xFFC4B5FD)
        )
        type?.let {
            MetadataBadge(
                text = it,
                backgroundColor = Color(0xFF8B5CF6).copy(alpha = 0.3f),
                textColor = Color(0xFFC4B5FD)
            )
        }
    }
}

private fun formatNumber(num: Long): String {
    return when {
        num >= 1_000_000 -> "${num / 1_000_000}M"
        num >= 1_000 -> "${num / 1_000}K"
        else -> num.toString()
    }
}
