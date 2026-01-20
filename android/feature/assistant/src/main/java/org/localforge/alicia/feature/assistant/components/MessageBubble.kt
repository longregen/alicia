package org.localforge.alicia.feature.assistant.components

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.*
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.KeyboardArrowUp
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.localforge.alicia.core.common.ui.AppIcons
import org.localforge.alicia.core.domain.model.Message
import org.localforge.alicia.core.domain.model.MessageRole
import org.localforge.alicia.core.domain.model.ToolUsage
import org.localforge.alicia.feature.assistant.BranchDirection
import org.localforge.alicia.feature.assistant.MessageBranchState
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/**
 * Message bubble component matching the web frontend's ChatBubble.
 *
 * Features:
 * - User messages aligned right with primary color
 * - Assistant messages aligned left with surface variant
 * - Reasoning blocks with collapsible content
 * - Streaming indicator for in-progress responses
 * - Tool usages inline display
 * - Timestamp display
 * - Edit button for user messages
 * - Voting buttons for assistant messages
 * - Copy functionality
 * - Branch navigation synced with backend (same as web frontend)
 */
@Composable
fun MessageBubble(
    message: Message,
    modifier: Modifier = Modifier,
    toolUsages: List<ToolUsage> = emptyList(),
    isLatestMessage: Boolean = false,
    isStreaming: Boolean = false,
    branchState: MessageBranchState? = null,
    onEdit: ((String, String) -> Unit)? = null,  // (messageId, newContent) -> Unit
    onVote: ((String, Boolean) -> Unit)? = null,  // (messageId, isUpvote) -> Unit
    onToolVote: ((String, Boolean) -> Unit)? = null,  // (toolUseId, isUpvote) -> Unit
    onCopy: ((String) -> Unit)? = null,  // (content) -> Unit
    onBranchNavigate: ((String, BranchDirection) -> Unit)? = null  // (messageId, direction) -> Unit
) {
    val isUser = message.role == MessageRole.USER
    var isEditing by remember { mutableStateOf(false) }
    var editedContent by remember { mutableStateOf(message.content) }
    var showActions by remember { mutableStateOf(false) }
    val focusRequester = remember { FocusRequester() }

    // Filter tool usages for this specific message
    val messageToolUsages = toolUsages.filter { it.request.messageId == message.id }

    // Get the effective content to display:
    // - If we have siblings and the current sibling's content differs, use the sibling's content
    // - Otherwise use the message's content
    val effectiveContent = branchState?.currentSibling?.content ?: message.content

    // Branch navigation data from server-synced state
    val hasBranches = (branchState?.count ?: 0) > 1
    val branchCount = branchState?.count ?: 0
    val currentBranchIndex = branchState?.currentIndex ?: 0
    val isLoadingBranches = branchState?.isLoading ?: false

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
            // Streaming indicator
            if (isStreaming && !isUser) {
                StreamingIndicator()
            }

            // Main message content with tap to show actions
            Box(
                modifier = Modifier.clickable { showActions = !showActions }
            ) {
                if (isEditing && isUser) {
                    // Edit mode for user messages
                    EditableMessageContent(
                        content = editedContent,
                        onContentChange = { editedContent = it },
                        onSave = {
                            onEdit?.invoke(message.id, editedContent)
                            isEditing = false
                        },
                        onCancel = {
                            editedContent = message.content
                            isEditing = false
                        },
                        focusRequester = focusRequester
                    )

                    // Request focus when entering edit mode
                    LaunchedEffect(isEditing) {
                        if (isEditing) {
                            focusRequester.requestFocus()
                        }
                    }
                } else {
                    MessageContent(
                        content = effectiveContent,
                        isUser = isUser,
                        isStreaming = isStreaming
                    )
                }
            }

            // Branch navigator (when multiple branches exist from server)
            // This matches the web frontend's BranchNavigator behavior
            if (hasBranches && onBranchNavigate != null) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    modifier = Modifier.padding(start = if (isUser) 0.dp else 4.dp)
                ) {
                    BranchNavigator(
                        currentIndex = currentBranchIndex,
                        totalBranches = branchCount,
                        onNavigate = { direction ->
                            // Trigger ViewModel to switch branch on server
                            // This will call PUT /conversations/{id}/switch-branch
                            onBranchNavigate(message.id, direction)
                        }
                    )

                    // Show loading indicator while switching branches
                    if (isLoadingBranches) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(12.dp),
                            strokeWidth = 1.5.dp,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }

            // Action buttons (shown on tap)
            AnimatedVisibility(
                visible = showActions && !isEditing,
                enter = expandVertically(),
                exit = shrinkVertically()
            ) {
                MessageActions(
                    isUser = isUser,
                    onEdit = if (isUser && onEdit != null) {
                        {
                            editedContent = effectiveContent
                            isEditing = true
                            showActions = false
                        }
                    } else null,
                    onCopy = { onCopy?.invoke(effectiveContent) },
                    onUpvote = if (!isUser && onVote != null) {
                        { onVote.invoke(message.id, true) }
                    } else null,
                    onDownvote = if (!isUser && onVote != null) {
                        { onVote.invoke(message.id, false) }
                    } else null
                )
            }

            // Tool usages for assistant messages
            if (!isUser && messageToolUsages.isNotEmpty()) {
                ToolUsageDisplay(
                    toolUsages = messageToolUsages,
                    isLatestMessage = isLatestMessage,
                    onVote = onToolVote
                )
            }

            // Timestamp
            MessageTimestamp(
                timestamp = message.createdAt,
                isUser = isUser
            )
        }
    }
}

/**
 * Action buttons for messages (edit, copy, vote).
 */
@Composable
private fun MessageActions(
    isUser: Boolean,
    onEdit: (() -> Unit)?,
    onCopy: () -> Unit,
    onUpvote: (() -> Unit)?,
    onDownvote: (() -> Unit)?
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.padding(horizontal = 4.dp)
    ) {
        // Edit button (user messages only)
        onEdit?.let {
            IconButton(
                onClick = it,
                modifier = Modifier.size(32.dp)
            ) {
                Icon(
                    imageVector = AppIcons.Edit,
                    contentDescription = "Edit",
                    modifier = Modifier.size(16.dp),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }

        // Copy button
        IconButton(
            onClick = onCopy,
            modifier = Modifier.size(32.dp)
        ) {
            Icon(
                imageVector = AppIcons.ContentCopy,
                contentDescription = "Copy",
                modifier = Modifier.size(16.dp),
                tint = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }

        // Vote buttons (assistant messages only)
        if (!isUser) {
            Spacer(modifier = Modifier.width(8.dp))

            onUpvote?.let {
                IconButton(
                    onClick = it,
                    modifier = Modifier.size(32.dp)
                ) {
                    Icon(
                        imageVector = AppIcons.ThumbUp,
                        contentDescription = "Upvote",
                        modifier = Modifier.size(16.dp),
                        tint = Color(0xFF10B981) // Success green
                    )
                }
            }

            onDownvote?.let {
                IconButton(
                    onClick = it,
                    modifier = Modifier.size(32.dp)
                ) {
                    Icon(
                        imageVector = AppIcons.ThumbDown,
                        contentDescription = "Downvote",
                        modifier = Modifier.size(16.dp),
                        tint = Color(0xFFEF4444) // Destructive red
                    )
                }
            }
        }
    }
}

/**
 * Editable message content for user message editing.
 */
@Composable
private fun EditableMessageContent(
    content: String,
    onContentChange: (String) -> Unit,
    onSave: () -> Unit,
    onCancel: () -> Unit,
    focusRequester: FocusRequester
) {
    Surface(
        shape = RoundedCornerShape(16.dp),
        color = MaterialTheme.colorScheme.primary.copy(alpha = 0.9f)
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            BasicTextField(
                value = content,
                onValueChange = onContentChange,
                modifier = Modifier
                    .fillMaxWidth()
                    .focusRequester(focusRequester),
                textStyle = TextStyle(
                    color = MaterialTheme.colorScheme.onPrimary,
                    fontSize = 16.sp
                ),
                cursorBrush = SolidColor(MaterialTheme.colorScheme.onPrimary)
            )

            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.align(Alignment.End)
            ) {
                TextButton(
                    onClick = onCancel,
                    colors = ButtonDefaults.textButtonColors(
                        contentColor = MaterialTheme.colorScheme.onPrimary.copy(alpha = 0.7f)
                    )
                ) {
                    Text("Cancel", fontSize = 12.sp)
                }
                Button(
                    onClick = onSave,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = MaterialTheme.colorScheme.onPrimary,
                        contentColor = MaterialTheme.colorScheme.primary
                    ),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 4.dp)
                ) {
                    Text("Save", fontSize = 12.sp)
                }
            }
        }
    }
}

@Composable
private fun StreamingIndicator() {
    Surface(
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.7f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp)
        ) {
            StreamingDot()
            Text(
                text = "Streaming",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
private fun StreamingDot() {
    val infiniteTransition = rememberInfiniteTransition(label = "streaming")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.3f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(600, easing = EaseInOut),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Box(
        modifier = Modifier
            .size(6.dp)
            .clip(RoundedCornerShape(3.dp))
            .background(Color(0xFF4DD4C5).copy(alpha = alpha)) // Accent color
    )
}

@Composable
private fun MessageContent(
    content: String,
    isUser: Boolean,
    isStreaming: Boolean
) {
    // Parse reasoning blocks from content
    val parts = parseMessageContent(content)

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
            MaterialTheme.colorScheme.surfaceVariant,
        modifier = Modifier.animateContentSize()
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            parts.forEach { part ->
                when (part) {
                    is MessagePart.Text -> {
                        Text(
                            text = part.content,
                            style = MaterialTheme.typography.bodyLarge,
                            color = if (isUser)
                                MaterialTheme.colorScheme.onPrimary
                            else
                                MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    is MessagePart.Reasoning -> {
                        ReasoningBlock(
                            content = part.content,
                            sequence = part.sequence
                        )
                    }
                }
            }

            // Cursor for streaming
            if (isStreaming && !isUser) {
                Box(
                    modifier = Modifier
                        .width(2.dp)
                        .height(16.dp)
                        .background(MaterialTheme.colorScheme.primary)
                )
            }
        }
    }
}

@Composable
private fun ReasoningBlock(
    content: String,
    sequence: Int
) {
    var isExpanded by remember { mutableStateOf(false) }
    val accentColor = Color(0xFF4DD4C5) // Accent color from theme

    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(topEnd = 8.dp, bottomEnd = 8.dp),
        color = accentColor.copy(alpha = 0.1f)
    ) {
        Column(
            modifier = Modifier
                .background(
                    color = Color.Transparent
                )
                .drawLeftBorder(accentColor, 4.dp)
                .padding(12.dp)
        ) {
            // Header
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { isExpanded = !isExpanded },
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "Reasoning",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = accentColor
                )
                Icon(
                    imageVector = if (isExpanded) Icons.Default.KeyboardArrowUp else Icons.Default.KeyboardArrowDown,
                    contentDescription = if (isExpanded) "Collapse" else "Expand",
                    tint = accentColor,
                    modifier = Modifier.size(20.dp)
                )
            }

            // Content
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = if (isExpanded || content.length <= 100) content else content.take(100) + "...",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurface
            )

            // Show more button
            if (!isExpanded && content.length > 100) {
                Text(
                    text = "Show more",
                    style = MaterialTheme.typography.labelSmall,
                    color = accentColor,
                    modifier = Modifier
                        .padding(top = 4.dp)
                        .clickable { isExpanded = true }
                )
            }
        }
    }
}

@Composable
private fun MessageTimestamp(
    timestamp: Long,
    isUser: Boolean
) {
    val formattedTime = remember(timestamp) {
        val instant = Instant.ofEpochMilli(timestamp)
        val localTime = instant.atZone(ZoneId.systemDefault()).toLocalTime()
        localTime.format(DateTimeFormatter.ofPattern("h:mm a"))
    }

    Text(
        text = formattedTime,
        style = MaterialTheme.typography.labelSmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f),
        modifier = Modifier.padding(horizontal = 4.dp)
    )
}

// Helper to draw left border
private fun Modifier.drawLeftBorder(color: Color, width: androidx.compose.ui.unit.Dp) = this.then(
    Modifier.drawWithContent {
        drawContent()
        drawRect(
            color = color,
            topLeft = Offset.Zero,
            size = Size(width.toPx(), size.height)
        )
    }
)

// Message content parsing
private sealed class MessagePart {
    data class Text(val content: String) : MessagePart()
    data class Reasoning(val content: String, val sequence: Int) : MessagePart()
}

private fun parseMessageContent(content: String): List<MessagePart> {
    val parts = mutableListOf<MessagePart>()
    val reasoningPattern = "<reasoning(?:\\s+data-sequence=\"(\\d+)\")?>(.*?)</reasoning>".toRegex(RegexOption.DOT_MATCHES_ALL)

    var lastIndex = 0
    var sequenceCounter = 0

    reasoningPattern.findAll(content).forEach { match ->
        // Add text before this reasoning block
        if (match.range.first > lastIndex) {
            val textBefore = content.substring(lastIndex, match.range.first).trim()
            if (textBefore.isNotEmpty()) {
                parts.add(MessagePart.Text(textBefore))
            }
        }

        // Add reasoning block
        val sequence = match.groupValues[1].toIntOrNull() ?: sequenceCounter++
        val reasoningContent = match.groupValues[2].trim()
        parts.add(MessagePart.Reasoning(reasoningContent, sequence))

        lastIndex = match.range.last + 1
    }

    // Add remaining text
    if (lastIndex < content.length) {
        val remainingText = content.substring(lastIndex).trim()
        if (remainingText.isNotEmpty()) {
            parts.add(MessagePart.Text(remainingText))
        }
    }

    // If no parts were added, the entire content is text
    if (parts.isEmpty() && content.isNotEmpty()) {
        parts.add(MessagePart.Text(content))
    }

    return parts
}
