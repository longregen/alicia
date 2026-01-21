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
    onArchiveClick: () -> Unit,
    onUnarchiveClick: () -> Unit,
    onRenameClick: () -> Unit,
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
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Text(
                        text = conversation.title ?: "Conversation",
                        style = MaterialTheme.typography.titleMedium,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f, fill = false)
                    )
                    if (conversation.isArchived) {
                        Surface(
                            shape = MaterialTheme.shapes.small,
                            color = MaterialTheme.colorScheme.secondaryContainer
                        ) {
                            Text(
                                text = "Archived",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSecondaryContainer,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.height(4.dp))

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

            IconButton(onClick = onRenameClick) {
                Icon(
                    imageVector = AppIcons.Edit,
                    contentDescription = "Rename",
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            IconButton(
                onClick = if (conversation.isArchived) onUnarchiveClick else onArchiveClick
            ) {
                Icon(
                    imageVector = if (conversation.isArchived) AppIcons.Unarchive else AppIcons.Archive,
                    contentDescription = if (conversation.isArchived) "Unarchive" else "Archive",
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

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

// SimpleDateFormat is not thread-safe, so we use ThreadLocal as a defensive measure
private val dateFormatter = ThreadLocal.withInitial {
    SimpleDateFormat("MMM d", Locale.getDefault())
}

private fun formatTimestamp(timestamp: Long): String {
    val now = System.currentTimeMillis()
    val diff = now - timestamp

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
