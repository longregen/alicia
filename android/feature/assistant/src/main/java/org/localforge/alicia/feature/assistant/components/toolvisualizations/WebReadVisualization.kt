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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Beautiful visualization for web_read tool results.
 * Displays fetched web page content with metadata badges.
 */
@Composable
fun WebReadVisualization(
    result: Map<String, Any?>?,
    modifier: Modifier = Modifier
) {
    if (result == null) {
        ErrorVisualization(message = "No result data", modifier = modifier)
        return
    }

    val url = result["url"] as? String ?: ""
    val title = result["title"] as? String ?: "Web Page"
    val content = result["content"] as? String ?: ""
    val excerpt = result["excerpt"] as? String
    val author = result["author"] as? String
    val siteName = result["site_name"] as? String
    val wordCount = (result["word_count"] as? Number)?.toInt() ?: 0
    val estimatedTokens = (result["estimated_tokens"] as? Number)?.toInt() ?: 0

    var isExpanded by remember { mutableStateOf(false) }
    val contentPreview = content.take(500)
    val hasMore = content.length > 500

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = Color.Transparent
        )
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .background(
                    brush = Brush.linearGradient(
                        colors = listOf(
                            Color(0xFF1E3A5F).copy(alpha = 0.3f),
                            Color(0xFF2D1B69).copy(alpha = 0.3f)
                        )
                    )
                )
        ) {
            // Header
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Color.Black.copy(alpha = 0.2f))
                    .padding(16.dp)
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Text(
                        text = "📖",
                        fontSize = 24.sp
                    )
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = title,
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = MaterialTheme.colorScheme.onSurface,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis
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

                // Metadata badges
                FlowRow(
                    modifier = Modifier.padding(top = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    siteName?.let {
                        MetadataBadge(
                            text = it,
                            backgroundColor = Color(0xFF3B82F6).copy(alpha = 0.3f),
                            textColor = Color(0xFF93C5FD)
                        )
                    }
                    author?.let {
                        MetadataBadge(
                            text = "✍️ $it",
                            backgroundColor = Color(0xFF8B5CF6).copy(alpha = 0.3f),
                            textColor = Color(0xFFC4B5FD)
                        )
                    }
                    MetadataBadge(
                        text = "${formatNumber(wordCount)} words",
                        backgroundColor = Color(0xFF6B7280).copy(alpha = 0.3f),
                        textColor = Color(0xFFD1D5DB)
                    )
                    MetadataBadge(
                        text = "~${formatNumber(estimatedTokens)} tokens",
                        backgroundColor = Color(0xFF10B981).copy(alpha = 0.3f),
                        textColor = Color(0xFF6EE7B7)
                    )
                }
            }

            // Excerpt
            excerpt?.let {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    color = Color.Black.copy(alpha = 0.1f)
                ) {
                    Text(
                        text = "\"$it\"",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        fontStyle = androidx.compose.ui.text.font.FontStyle.Italic,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                    )
                }
            }

            // Content
            Column(modifier = Modifier.padding(16.dp)) {
                Surface(
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(max = if (isExpanded) 400.dp else 200.dp),
                    color = Color.Black.copy(alpha = 0.2f),
                    shape = RoundedCornerShape(8.dp)
                ) {
                    Text(
                        text = if (isExpanded) content else "$contentPreview${if (hasMore) "..." else ""}",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.9f),
                        modifier = Modifier
                            .padding(12.dp)
                            .verticalScroll(rememberScrollState())
                    )
                }

                if (hasMore) {
                    TextButton(
                        onClick = { isExpanded = !isExpanded },
                        modifier = Modifier.padding(top = 4.dp)
                    ) {
                        Text(
                            text = if (isExpanded) "← Show less" else "Show full content (${formatNumber(content.length)} chars)",
                            color = Color(0xFF60A5FA),
                            fontSize = 12.sp
                        )
                    }
                }
            }
        }
    }
}

@Composable
fun MetadataBadge(
    text: String,
    backgroundColor: Color,
    textColor: Color,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier,
        color = backgroundColor,
        shape = RoundedCornerShape(12.dp)
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            color = textColor,
            fontWeight = FontWeight.Medium,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
        )
    }
}

@Composable
fun ErrorVisualization(
    message: String,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = Color(0xFFDC2626).copy(alpha = 0.2f)
        ),
        shape = RoundedCornerShape(12.dp)
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(text = "❌", fontSize = 20.sp)
            Text(
                text = message,
                style = MaterialTheme.typography.bodyMedium,
                color = Color(0xFFFCA5A5)
            )
        }
    }
}

private fun formatNumber(num: Int): String {
    return when {
        num >= 1_000_000 -> "${num / 1_000_000}M"
        num >= 1_000 -> "${num / 1_000}K"
        else -> num.toString()
    }
}
