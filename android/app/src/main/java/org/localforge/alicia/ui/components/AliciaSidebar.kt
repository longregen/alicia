package org.localforge.alicia.ui.components

import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.localforge.alicia.core.domain.model.Conversation
import org.localforge.alicia.ui.theme.AliciaTheme
import java.time.Instant
import java.time.ZoneId

/**
 * Alicia Sidebar Component
 *
 * A drawer-style sidebar that matches the web frontend's Sidebar component.
 *
 * Features:
 * - "New Chat" button
 * - Active conversations list with context menu (rename, archive, delete)
 * - Archived conversations (collapsible)
 * - Navigation to Memory, Server, and Settings
 * - Connection status indicator
 */
@Composable
fun AliciaSidebar(
    conversations: List<Conversation>,
    selectedConversationId: String?,
    isConnected: Boolean,
    isLoading: Boolean,
    onNewConversation: () -> Unit,
    onSelectConversation: (String) -> Unit,
    onRenameConversation: (String, String) -> Unit,
    onArchiveConversation: (String) -> Unit,
    onUnarchiveConversation: (String) -> Unit,
    onDeleteConversation: (String) -> Unit,
    onNavigateToMemory: () -> Unit,
    onNavigateToServer: () -> Unit,
    onNavigateToSettings: () -> Unit,
    onClose: () -> Unit,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors
    var archivedExpanded by remember { mutableStateOf(false) }

    // Separate active and archived conversations
    val activeConversations = remember(conversations) {
        conversations.filter { !it.isDeleted }
            .sortedByDescending { it.updatedAt }
    }
    val archivedConversations = remember(conversations) {
        conversations.filter { it.isDeleted }
            .sortedByDescending { it.updatedAt }
    }

    Column(
        modifier = modifier
            .fillMaxHeight()
            .width(280.dp)
            .background(extendedColors.sidebar)
    ) {
        // Header
        SidebarHeader(
            onNewConversation = onNewConversation,
            onClose = onClose
        )

        // Conversation list
        LazyColumn(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth()
                .padding(horizontal = 10.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            // Loading state
            if (isLoading) {
                item {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(32.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(24.dp),
                            strokeWidth = 2.dp
                        )
                    }
                }
            }

            // Empty state
            if (!isLoading && conversations.isEmpty()) {
                item {
                    Text(
                        text = "No conversations yet",
                        style = MaterialTheme.typography.bodyMedium,
                        color = extendedColors.mutedForeground,
                        modifier = Modifier.padding(32.dp)
                    )
                }
            }

            // Active conversations section
            if (activeConversations.isNotEmpty()) {
                item {
                    Text(
                        text = "ACTIVE (${activeConversations.size})",
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.SemiBold,
                        color = extendedColors.mutedForeground,
                        letterSpacing = 0.5.sp,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 8.dp)
                    )
                }

                items(activeConversations, key = { it.id }) { conversation ->
                    ConversationItem(
                        conversation = conversation,
                        isSelected = conversation.id == selectedConversationId,
                        isArchived = false,
                        onClick = { onSelectConversation(conversation.id) },
                        onRename = { newTitle -> onRenameConversation(conversation.id, newTitle) },
                        onArchive = { onArchiveConversation(conversation.id) },
                        onDelete = { onDeleteConversation(conversation.id) }
                    )
                }
            }

            // Archived conversations section (collapsible)
            if (archivedConversations.isNotEmpty()) {
                item {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(4.dp))
                            .clickable { archivedExpanded = !archivedExpanded }
                            .padding(horizontal = 8.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = "ARCHIVED (${archivedConversations.size})",
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = extendedColors.mutedForeground,
                            letterSpacing = 0.5.sp
                        )
                        Icon(
                            imageVector = if (archivedExpanded) Icons.Default.KeyboardArrowUp else Icons.Default.KeyboardArrowDown,
                            contentDescription = if (archivedExpanded) "Collapse" else "Expand",
                            modifier = Modifier.size(16.dp),
                            tint = extendedColors.mutedForeground
                        )
                    }
                }

                if (archivedExpanded) {
                    items(archivedConversations, key = { it.id }) { conversation ->
                        ConversationItem(
                            conversation = conversation,
                            isSelected = conversation.id == selectedConversationId,
                            isArchived = true,
                            onClick = { onSelectConversation(conversation.id) },
                            onRename = { newTitle -> onRenameConversation(conversation.id, newTitle) },
                            onUnarchive = { onUnarchiveConversation(conversation.id) },
                            onDelete = { onDeleteConversation(conversation.id) }
                        )
                    }
                }
            }
        }

        // Bottom navigation
        SidebarBottomNav(
            isConnected = isConnected,
            onNavigateToMemory = onNavigateToMemory,
            onNavigateToServer = onNavigateToServer,
            onNavigateToSettings = onNavigateToSettings
        )
    }
}

@Composable
private fun SidebarHeader(
    onNewConversation: () -> Unit,
    onClose: () -> Unit
) {
    val extendedColors = AliciaTheme.extendedColors

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "Alicia",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.SemiBold,
                color = extendedColors.sidebarForeground
            )
            IconButton(
                onClick = onClose,
                modifier = Modifier.size(32.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = "Close sidebar",
                    tint = extendedColors.sidebarForeground,
                    modifier = Modifier.size(20.dp)
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        Button(
            onClick = onNewConversation,
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(8.dp),
            colors = ButtonDefaults.buttonColors(
                containerColor = MaterialTheme.colorScheme.secondary,
                contentColor = MaterialTheme.colorScheme.onSecondary
            )
        ) {
            Text("New Chat")
        }
    }

    HorizontalDivider(color = extendedColors.border)
}

@Composable
private fun ConversationItem(
    conversation: Conversation,
    isSelected: Boolean,
    isArchived: Boolean,
    onClick: () -> Unit,
    onRename: (String) -> Unit = {},
    onArchive: () -> Unit = {},
    onUnarchive: () -> Unit = {},
    onDelete: () -> Unit
) {
    val extendedColors = AliciaTheme.extendedColors
    var showMenu by remember { mutableStateOf(false) }
    var showRenameDialog by remember { mutableStateOf(false) }
    var showDeleteDialog by remember { mutableStateOf(false) }

    Box {
        Surface(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(6.dp))
                .clickable(onClick = onClick),
            color = if (isSelected) extendedColors.sidebarAccent else extendedColors.sidebar,
            shape = RoundedCornerShape(6.dp)
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(12.dp),
                verticalAlignment = Alignment.Top
            ) {
                // Selection indicator
                if (isSelected) {
                    Box(
                        modifier = Modifier
                            .width(2.dp)
                            .height(40.dp)
                            .background(extendedColors.accent, RoundedCornerShape(1.dp))
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                }

                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Text(
                        text = conversation.title ?: "New Conversation",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = extendedColors.sidebarForeground,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                    Text(
                        text = formatRelativeTime(conversation.updatedAt),
                        style = MaterialTheme.typography.bodySmall,
                        fontSize = 12.sp,
                        color = extendedColors.mutedForeground
                    )
                }

                // Context menu trigger
                IconButton(
                    onClick = { showMenu = true },
                    modifier = Modifier.size(24.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.MoreVert,
                        contentDescription = "More options",
                        tint = extendedColors.mutedForeground,
                        modifier = Modifier.size(16.dp)
                    )
                }
            }
        }

        // Context menu
        DropdownMenu(
            expanded = showMenu,
            onDismissRequest = { showMenu = false }
        ) {
            DropdownMenuItem(
                text = { Text("Rename") },
                onClick = {
                    showMenu = false
                    showRenameDialog = true
                },
                leadingIcon = {
                    Icon(Icons.Outlined.Edit, contentDescription = null)
                }
            )

            if (isArchived) {
                DropdownMenuItem(
                    text = { Text("Unarchive") },
                    onClick = {
                        showMenu = false
                        onUnarchive()
                    },
                    leadingIcon = {
                        Icon(Icons.Outlined.Unarchive, contentDescription = null)
                    }
                )
            } else {
                DropdownMenuItem(
                    text = { Text("Archive") },
                    onClick = {
                        showMenu = false
                        onArchive()
                    },
                    leadingIcon = {
                        Icon(Icons.Outlined.Archive, contentDescription = null)
                    }
                )
            }

            HorizontalDivider()

            DropdownMenuItem(
                text = {
                    Text(
                        "Delete",
                        color = extendedColors.destructive
                    )
                },
                onClick = {
                    showMenu = false
                    showDeleteDialog = true
                },
                leadingIcon = {
                    Icon(
                        Icons.Outlined.Delete,
                        contentDescription = null,
                        tint = extendedColors.destructive
                    )
                }
            )
        }
    }

    // Rename dialog
    if (showRenameDialog) {
        RenameDialog(
            currentTitle = conversation.title ?: "New Conversation",
            onConfirm = { newTitle ->
                onRename(newTitle)
                showRenameDialog = false
            },
            onDismiss = { showRenameDialog = false }
        )
    }

    // Delete confirmation dialog
    if (showDeleteDialog) {
        AlertDialog(
            onDismissRequest = { showDeleteDialog = false },
            title = { Text("Delete Conversation") },
            text = { Text("Are you sure you want to delete this conversation? This action cannot be undone.") },
            confirmButton = {
                TextButton(
                    onClick = {
                        onDelete()
                        showDeleteDialog = false
                    },
                    colors = ButtonDefaults.textButtonColors(
                        contentColor = extendedColors.destructive
                    )
                ) {
                    Text("Delete")
                }
            },
            dismissButton = {
                TextButton(onClick = { showDeleteDialog = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}

@Composable
private fun RenameDialog(
    currentTitle: String,
    onConfirm: (String) -> Unit,
    onDismiss: () -> Unit
) {
    var title by remember { mutableStateOf(currentTitle) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Rename Conversation") },
        text = {
            OutlinedTextField(
                value = title,
                onValueChange = { title = it },
                label = { Text("Title") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth()
            )
        },
        confirmButton = {
            TextButton(
                onClick = { onConfirm(title) },
                enabled = title.isNotBlank()
            ) {
                Text("Save")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}

@Composable
private fun SidebarBottomNav(
    isConnected: Boolean,
    onNavigateToMemory: () -> Unit,
    onNavigateToServer: () -> Unit,
    onNavigateToSettings: () -> Unit
) {
    val extendedColors = AliciaTheme.extendedColors

    Column(
        modifier = Modifier
            .fillMaxWidth()
    ) {
        HorizontalDivider(color = extendedColors.border)

        // Connection status
        ConnectionStatusIndicator(
            isConnected = isConnected,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp)
        )

        // Navigation buttons
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(10.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            SidebarNavButton(
                icon = Icons.Outlined.Memory,
                label = "Memory",
                onClick = onNavigateToMemory
            )
            SidebarNavButton(
                icon = Icons.Outlined.Dns,
                label = "Server",
                onClick = onNavigateToServer
            )
            SidebarNavButton(
                icon = Icons.Outlined.Settings,
                label = "Settings",
                onClick = onNavigateToSettings
            )
        }
    }
}

@Composable
private fun SidebarNavButton(
    icon: ImageVector,
    label: String,
    onClick: () -> Unit,
    isActive: Boolean = false
) {
    val extendedColors = AliciaTheme.extendedColors

    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(6.dp))
            .clickable(onClick = onClick),
        color = if (isActive) extendedColors.sidebarAccent else extendedColors.sidebar,
        shape = RoundedCornerShape(6.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                modifier = Modifier.size(18.dp),
                tint = if (isActive) extendedColors.accent else extendedColors.sidebarForeground
            )
            Text(
                text = label,
                style = MaterialTheme.typography.bodyMedium,
                color = if (isActive) extendedColors.accent else extendedColors.sidebarForeground
            )
        }
    }
}

/**
 * Connection status enum matching web's ConnectionStatus
 */
enum class ConnectionState {
    Connected,
    Connecting,
    Reconnecting,
    Disconnected,
    Error
}

@Composable
private fun ConnectionStatusIndicator(
    isConnected: Boolean,
    connectionState: ConnectionState = if (isConnected) ConnectionState.Connected else ConnectionState.Disconnected,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors

    val statusColor = when (connectionState) {
        ConnectionState.Connected -> extendedColors.success
        ConnectionState.Connecting, ConnectionState.Reconnecting -> extendedColors.warning
        ConnectionState.Disconnected, ConnectionState.Error -> extendedColors.destructive
    }

    val statusText = when (connectionState) {
        ConnectionState.Connected -> "Connected"
        ConnectionState.Connecting -> "Connecting..."
        ConnectionState.Reconnecting -> "Reconnecting..."
        ConnectionState.Disconnected -> "Disconnected"
        ConnectionState.Error -> "Error"
    }

    val isAnimated = connectionState == ConnectionState.Connecting ||
            connectionState == ConnectionState.Reconnecting

    // Pulsing animation for connecting states
    val pulseAlpha by if (isAnimated) {
        rememberInfiniteTransition(label = "pulse").animateFloat(
            initialValue = 0.5f,
            targetValue = 1f,
            animationSpec = infiniteRepeatable(
                animation = tween(600),
                repeatMode = RepeatMode.Reverse
            ),
            label = "pulse_alpha"
        )
    } else {
        remember { mutableFloatStateOf(1f) }
    }

    Row(
        modifier = modifier,
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Box(
            modifier = Modifier
                .size(8.dp)
                .clip(CircleShape)
                .background(statusColor.copy(alpha = pulseAlpha))
        )
        Text(
            text = statusText,
            style = MaterialTheme.typography.bodySmall,
            color = extendedColors.mutedForeground
        )
    }
}

private fun formatRelativeTime(timestamp: Long): String {
    val now = System.currentTimeMillis()
    val diffMs = now - timestamp
    val diffMins = diffMs / 60000
    val diffHours = diffMs / 3600000
    val diffDays = diffMs / 86400000

    return when {
        diffMins < 1 -> "Just now"
        diffMins < 60 -> "${diffMins}m ago"
        diffHours < 24 -> "${diffHours}h ago"
        diffDays == 1L -> "Yesterday"
        diffDays < 7 -> "${diffDays}d ago"
        else -> {
            val instant = Instant.ofEpochMilli(timestamp)
            val date = java.time.LocalDate.ofInstant(instant, ZoneId.systemDefault())
            "${date.monthValue}/${date.dayOfMonth}/${date.year}"
        }
    }
}
