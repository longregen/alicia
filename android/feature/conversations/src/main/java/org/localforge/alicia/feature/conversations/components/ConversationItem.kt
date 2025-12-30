package org.localforge.alicia.feature.conversations.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import org.localforge.alicia.core.common.ui.AppIcons
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import org.localforge.alicia.feature.conversations.Conversation
import java.text.SimpleDateFormat
import java.util.*

// Time interval constants for timestamp formatting
private const val MILLIS_PER_SECOND = 1000L
private const val MILLIS_PER_MINUTE = 60 * MILLIS_PER_SECOND
private const val MILLIS_PER_HOUR = 60 * MILLIS_PER_MINUTE
private const val MILLIS_PER_DAY = 24 * MILLIS_PER_HOUR
private const val MILLIS_PER_WEEK = 7 * MILLIS_PER_DAY

@Composable
fun ConversationItem(
    conversation: Conversation,
    onClick: () -> Unit,
    onDeleteClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(
                modifier = Modifier.weight(1f)
            ) {
                // Title, or default 'Conversation' if not set
                Text(
                    text = conversation.title ?: "Conversation",
                    style = MaterialTheme.typography.titleMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )

                Spacer(modifier = Modifier.height(4.dp))

                // Last message preview
                if (conversation.lastMessage != null) {
                    Text(
                        text = conversation.lastMessage,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis
                    )
                }

                Spacer(modifier = Modifier.height(8.dp))

                // Metadata row
                Row(
                    horizontalArrangement = Arrangement.spacedBy(16.dp)
                ) {
                    Text(
                        text = formatTimestamp(conversation.timestamp),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.7f)
                    )
                    Text(
                        text = "${conversation.messageCount} messages",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.7f)
                    )
                }
            }

            // Delete button
            IconButton(onClick = onDeleteClick) {
                Icon(
                    imageVector = AppIcons.Delete,
                    contentDescription = "Delete",
                    tint = MaterialTheme.colorScheme.error
                )
            }
        }
    }
}

/**
 * Thread-safe date formatter for timestamps older than a week.
 * SimpleDateFormat is not thread-safe, so we use ThreadLocal to ensure each thread
 * has its own instance.
 */
private val dateFormatter = ThreadLocal.withInitial {
    SimpleDateFormat("MMM d", Locale.getDefault())
}

/**
 * Formats a timestamp into a human-readable relative time string.
 *
 * The formatting follows these rules based on the time difference:
 * - Future timestamps: "Just now"
 * - Less than 1 minute: "Just now"
 * - Less than 1 hour: "Xm ago" (e.g., "5m ago")
 * - Less than 1 day: "Xh ago" (e.g., "3h ago")
 * - Less than 1 week: "Xd ago" (e.g., "2d ago")
 * - 1 week or more: Formatted as "MMM d" (e.g., "Jan 15")
 *
 * @param timestamp Unix timestamp in milliseconds to format
 * @return Human-readable relative time string
 */
private fun formatTimestamp(timestamp: Long): String {
    val now = System.currentTimeMillis()
    val diff = now - timestamp

    // Handle future timestamps (clock skew or invalid data)
    if (diff < 0) {
        return "Just now"
    }

    return when {
        diff < MILLIS_PER_MINUTE -> "Just now"
        diff < MILLIS_PER_HOUR -> {
            val minutes = (diff / MILLIS_PER_MINUTE).coerceAtLeast(1)
            "${minutes}m ago"
        }
        diff < MILLIS_PER_DAY -> "${diff / MILLIS_PER_HOUR}h ago"
        diff < MILLIS_PER_WEEK -> "${diff / MILLIS_PER_DAY}d ago"
        else -> dateFormatter.get()!!.format(Date(timestamp))
    }
}
