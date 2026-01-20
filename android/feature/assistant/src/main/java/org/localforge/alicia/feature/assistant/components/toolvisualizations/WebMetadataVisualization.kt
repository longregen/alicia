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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Visualization for web_extract_metadata tool results.
 * Displays Open Graph, Twitter Card, and other metadata.
 */
@Composable
fun WebMetadataVisualization(
    result: Map<String, Any?>?,
    modifier: Modifier = Modifier
) {
    if (result == null) {
        ErrorVisualization(message = "No result data", modifier = modifier)
        return
    }

    val url = result["url"] as? String ?: ""
    val title = result["title"] as? String
    val description = result["description"] as? String
    val author = result["author"] as? String
    val publishedDate = result["published_date"] as? String
    val ogData = result["open_graph"] as? Map<*, *>
    val twitterData = result["twitter_card"] as? Map<*, *>
    val jsonLd = result["json_ld"] as? List<*>

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
                            Color(0xFF0891B2).copy(alpha = 0.3f),
                            Color(0xFF0E7490).copy(alpha = 0.3f)
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
                    Text(text = "📋", fontSize = 24.sp)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = "Page Metadata",
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Text(
                            text = url,
                            style = MaterialTheme.typography.bodySmall,
                            color = Color(0xFF60A5FA),
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                }
            }

            Column(
                modifier = Modifier
                    .padding(16.dp)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Basic metadata
                if (title != null || description != null || author != null) {
                    MetadataSection(title = "Basic Info") {
                        title?.let { MetadataRow("Title", it) }
                        description?.let { MetadataRow("Description", it) }
                        author?.let { MetadataRow("Author", it) }
                        publishedDate?.let { MetadataRow("Published", it) }
                    }
                }

                // Open Graph
                ogData?.let { og ->
                    if (og.isNotEmpty()) {
                        MetadataSection(
                            title = "Open Graph",
                            icon = "📘",
                            color = Color(0xFF3B82F6)
                        ) {
                            og.forEach { (key, value) ->
                                if (value != null && value.toString().isNotBlank()) {
                                    MetadataRow(key.toString(), value.toString())
                                }
                            }
                        }
                    }
                }

                // Twitter Card
                twitterData?.let { twitter ->
                    if (twitter.isNotEmpty()) {
                        MetadataSection(
                            title = "Twitter Card",
                            icon = "🐦",
                            color = Color(0xFF1DA1F2)
                        ) {
                            twitter.forEach { (key, value) ->
                                if (value != null && value.toString().isNotBlank()) {
                                    MetadataRow(key.toString(), value.toString())
                                }
                            }
                        }
                    }
                }

                // JSON-LD
                jsonLd?.let { ld ->
                    if (ld.isNotEmpty()) {
                        MetadataSection(
                            title = "Structured Data (JSON-LD)",
                            icon = "📊",
                            color = Color(0xFF10B981)
                        ) {
                            Text(
                                text = "${ld.size} schema(s) found",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                            ld.forEachIndexed { index, schema ->
                                (schema as? Map<*, *>)?.let { s ->
                                    val type = s["@type"] as? String ?: "Unknown"
                                    MetadataBadge(
                                        text = type,
                                        backgroundColor = Color(0xFF10B981).copy(alpha = 0.3f),
                                        textColor = Color(0xFF6EE7B7),
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

@Composable
private fun MetadataSection(
    title: String,
    icon: String = "📄",
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
            }
            Spacer(modifier = Modifier.height(8.dp))
            content()
        }
    }
}

@Composable
private fun MetadataRow(
    label: String,
    value: String
) {
    Column(modifier = Modifier.padding(vertical = 4.dp)) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            fontWeight = FontWeight.Medium
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurface,
            maxLines = 3,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(top = 2.dp)
        )
    }
}
