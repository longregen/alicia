package org.localforge.alicia.feature.assistant.components.toolvisualizations

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
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

@Composable
fun WebLinksVisualization(
    result: Map<String, Any?>?,
    modifier: Modifier = Modifier
) {
    if (result == null) {
        ErrorVisualization(message = "No result data", modifier = modifier)
        return
    }

    val url = result["url"] as? String ?: ""
    val links = (result["links"] as? List<*>)?.mapNotNull { it as? Map<*, *> } ?: emptyList()
    val totalCount = (result["total_count"] as? Number)?.toInt() ?: links.size
    val truncated = result["truncated"] as? Boolean ?: false

    var selectedFilter by remember { mutableStateOf("all") }

    val filteredLinks = remember(selectedFilter, links) {
        when (selectedFilter) {
            "internal" -> links.filter { (it["is_internal"] as? Boolean) == true }
            "external" -> links.filter { (it["is_internal"] as? Boolean) != true }
            else -> links
        }
    }

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
                            Color(0xFF7C3AED).copy(alpha = 0.3f),
                            Color(0xFF6366F1).copy(alpha = 0.3f)
                        )
                    )
                )
        ) {
            // Header
            Surface(
                modifier = Modifier.fillMaxWidth(),
                color = Color.Black.copy(alpha = 0.2f)
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        Text(text = "🔗", fontSize = 24.sp)
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = "Extract Links",
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onSurface
                            )
                            Text(
                                text = "$totalCount links found${if (truncated) " (truncated)" else ""}",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    }

                    // Filter tabs
                    Row(
                        modifier = Modifier.padding(top = 12.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        FilterChip(
                            selected = selectedFilter == "all",
                            onClick = { selectedFilter = "all" },
                            label = { Text("All") },
                            colors = FilterChipDefaults.filterChipColors(
                                selectedContainerColor = Color(0xFF8B5CF6).copy(alpha = 0.3f)
                            )
                        )
                        FilterChip(
                            selected = selectedFilter == "internal",
                            onClick = { selectedFilter = "internal" },
                            label = { Text("Internal") },
                            colors = FilterChipDefaults.filterChipColors(
                                selectedContainerColor = Color(0xFF10B981).copy(alpha = 0.3f)
                            )
                        )
                        FilterChip(
                            selected = selectedFilter == "external",
                            onClick = { selectedFilter = "external" },
                            label = { Text("External") },
                            colors = FilterChipDefaults.filterChipColors(
                                selectedContainerColor = Color(0xFFF59E0B).copy(alpha = 0.3f)
                            )
                        )
                    }
                }
            }

            // Links list
            if (filteredLinks.isEmpty()) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(32.dp),
                    contentAlignment = Alignment.Center
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(text = "🔗", fontSize = 32.sp)
                        Spacer(modifier = Modifier.height(8.dp))
                        Text(
                            text = "No links found",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            } else {
                Column(
                    modifier = Modifier
                        .heightIn(max = 300.dp)
                        .padding(8.dp)
                ) {
                    filteredLinks.forEachIndexed { index, link ->
                        LinkItem(
                            link = link,
                            index = index + 1
                        )
                        if (index < filteredLinks.lastIndex) {
                            HorizontalDivider(
                                color = Color.White.copy(alpha = 0.1f),
                                thickness = 1.dp,
                                modifier = Modifier.padding(vertical = 4.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun LinkItem(
    link: Map<*, *>,
    index: Int
) {
    val url = link["url"] as? String ?: ""
    val text = link["text"] as? String
    val isInternal = link["is_internal"] as? Boolean ?: false

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 8.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.Top
    ) {
        // Index number
        Text(
            text = "$index.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.width(24.dp)
        )

        Column(modifier = Modifier.weight(1f)) {
            // Link text
            text?.let {
                if (it.isNotBlank()) {
                    Text(
                        text = it,
                        style = MaterialTheme.typography.bodySmall,
                        fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onSurface,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            }

            // URL
            Text(
                text = url,
                style = MaterialTheme.typography.bodySmall,
                color = Color(0xFF60A5FA),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
        }

        // Internal/External badge
        Surface(
            color = if (isInternal) Color(0xFF10B981).copy(alpha = 0.3f) else Color(0xFFF59E0B).copy(alpha = 0.3f),
            shape = RoundedCornerShape(4.dp)
        ) {
            Text(
                text = if (isInternal) "int" else "ext",
                style = MaterialTheme.typography.labelSmall,
                color = if (isInternal) Color(0xFF6EE7B7) else Color(0xFFFCD34D),
                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
            )
        }
    }
}
