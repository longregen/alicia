package org.localforge.alicia.feature.assistant.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import org.localforge.alicia.core.domain.model.Message
import org.localforge.alicia.core.domain.model.MessageRole
import org.localforge.alicia.core.domain.model.ToolUsage

@Composable
fun MessageBubble(
    message: Message,
    modifier: Modifier = Modifier,
    toolUsages: List<ToolUsage> = emptyList(),
    isLatestMessage: Boolean = false
) {
    val isUser = message.role == MessageRole.USER

    // Filter tool usages for this specific message
    val messageToolUsages = toolUsages.filter { it.request.messageId == message.id }

    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = if (isUser) Arrangement.End else Arrangement.Start
    ) {
        Column(
            modifier = Modifier.widthIn(max = 320.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Surface(
                shape = RoundedCornerShape(
                    topStart = 16.dp,
                    topEnd = 16.dp,
                    bottomStart = if (isUser) 16.dp else 4.dp,
                    bottomEnd = if (isUser) 4.dp else 16.dp
                ),
                color = if (isUser)
                    MaterialTheme.colorScheme.primary
                else
                    MaterialTheme.colorScheme.surfaceVariant
            ) {
                Text(
                    text = message.content,
                    modifier = Modifier.padding(12.dp),
                    style = MaterialTheme.typography.bodyLarge,
                    color = if (isUser)
                        MaterialTheme.colorScheme.onPrimary
                    else
                        MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            // Show tool usages inline for assistant messages
            if (!isUser && messageToolUsages.isNotEmpty()) {
                ToolUsageDisplay(
                    toolUsages = messageToolUsages,
                    isLatestMessage = isLatestMessage
                )
            }
        }
    }
}
