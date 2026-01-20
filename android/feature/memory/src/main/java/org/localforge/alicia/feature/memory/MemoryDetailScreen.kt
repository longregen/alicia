package org.localforge.alicia.feature.memory

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import org.localforge.alicia.core.common.ui.AppIcons
import org.localforge.alicia.core.domain.model.Memory
import org.localforge.alicia.core.domain.model.MemoryCategory
import org.localforge.alicia.ui.theme.AliciaTheme
import java.text.SimpleDateFormat
import java.util.*

/**
 * MemoryDetailScreen - Displays detailed view of a single memory.
 * Matches the web frontend's MemoryDetail.tsx component.
 *
 * Features:
 * - Full content display
 * - Category badge with color
 * - Importance score
 * - Usage count
 * - Tags display
 * - Timestamps (created, updated)
 * - Actions: Pin, Edit, Archive, Delete
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MemoryDetailScreen(
    memoryId: String,
    viewModel: MemoryViewModel = hiltViewModel(),
    onNavigateBack: () -> Unit = {},
    onEditMemory: (Memory) -> Unit = {}
) {
    val memories by viewModel.memories.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()
    val errorMessage by viewModel.errorMessage.collectAsState()

    val memory = memories.find { it.id == memoryId }
    val extendedColors = AliciaTheme.extendedColors

    var showDeleteDialog by remember { mutableStateOf(false) }
    var showArchiveDialog by remember { mutableStateOf(false) }

    // Load memory if not in list
    LaunchedEffect(memoryId) {
        if (memory == null) {
            viewModel.loadMemory(memoryId)
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Memory Details") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = AppIcons.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    if (memory != null) {
                        // Pin button
                        IconButton(
                            onClick = { viewModel.togglePinMemory(memory.id) },
                            enabled = !isLoading
                        ) {
                            Icon(
                                imageVector = if (memory.pinned) AppIcons.Star else AppIcons.StarOutline,
                                contentDescription = if (memory.pinned) "Unpin" else "Pin",
                                tint = if (memory.pinned) extendedColors.accent else LocalContentColor.current
                            )
                        }

                        // Edit button
                        IconButton(
                            onClick = { onEditMemory(memory) },
                            enabled = !isLoading
                        ) {
                            Icon(
                                imageVector = AppIcons.Edit,
                                contentDescription = "Edit"
                            )
                        }

                        // Archive button
                        IconButton(
                            onClick = { showArchiveDialog = true },
                            enabled = !isLoading
                        ) {
                            Icon(
                                imageVector = AppIcons.Archive,
                                contentDescription = "Archive"
                            )
                        }

                        // Delete button
                        IconButton(
                            onClick = { showDeleteDialog = true },
                            enabled = !isLoading
                        ) {
                            Icon(
                                imageVector = AppIcons.Delete,
                                contentDescription = "Delete",
                                tint = extendedColors.destructive
                            )
                        }
                    }
                }
            )
        }
    ) { paddingValues ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
        ) {
            when {
                isLoading && memory == null -> {
                    // Loading state
                    Box(
                        modifier = Modifier.fillMaxSize(),
                        contentAlignment = Alignment.Center
                    ) {
                        Column(
                            horizontalAlignment = Alignment.CenterHorizontally,
                            verticalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            CircularProgressIndicator()
                            Text(
                                text = "Loading memory...",
                                style = MaterialTheme.typography.bodyMedium,
                                color = extendedColors.mutedForeground
                            )
                        }
                    }
                }

                errorMessage != null && memory == null -> {
                    // Error state
                    Box(
                        modifier = Modifier.fillMaxSize(),
                        contentAlignment = Alignment.Center
                    ) {
                        Column(
                            horizontalAlignment = Alignment.CenterHorizontally,
                            verticalArrangement = Arrangement.spacedBy(16.dp)
                        ) {
                            Icon(
                                imageVector = AppIcons.Error,
                                contentDescription = null,
                                tint = extendedColors.destructive,
                                modifier = Modifier.size(64.dp)
                            )
                            Text(
                                text = errorMessage ?: "Failed to load memory",
                                style = MaterialTheme.typography.bodyLarge,
                                color = extendedColors.destructive
                            )
                            TextButton(onClick = onNavigateBack) {
                                Text("Back to Memory List")
                            }
                        }
                    }
                }

                memory != null -> {
                    // Content
                    MemoryDetailContent(
                        memory = memory,
                        modifier = Modifier.fillMaxSize()
                    )
                }
            }
        }
    }

    // Delete confirmation dialog
    if (showDeleteDialog && memory != null) {
        AlertDialog(
            onDismissRequest = { showDeleteDialog = false },
            title = { Text("Delete Memory") },
            text = { Text("Are you sure you want to delete this memory? This action cannot be undone.") },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.deleteMemory(memory.id)
                        showDeleteDialog = false
                        onNavigateBack()
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

    // Archive confirmation dialog
    if (showArchiveDialog && memory != null) {
        AlertDialog(
            onDismissRequest = { showArchiveDialog = false },
            title = { Text("Archive Memory") },
            text = { Text("Archive this memory? It will be hidden from the main list.") },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.archiveMemory(memory.id)
                        showArchiveDialog = false
                        onNavigateBack()
                    }
                ) {
                    Text("Archive")
                }
            },
            dismissButton = {
                TextButton(onClick = { showArchiveDialog = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}

@Composable
private fun MemoryDetailContent(
    memory: Memory,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors
    val categoryColor = getCategoryColor(memory.category)

    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(24.dp)
    ) {
        // Category and metadata row
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Category badge
            Box(
                modifier = Modifier
                    .clip(RoundedCornerShape(4.dp))
                    .background(categoryColor.copy(alpha = 0.15f))
                    .border(1.dp, categoryColor, RoundedCornerShape(4.dp))
                    .padding(horizontal = 12.dp, vertical = 6.dp)
            ) {
                Text(
                    text = memory.category.name.lowercase()
                        .replaceFirstChar { it.uppercase() },
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Medium,
                    color = categoryColor
                )
            }

            // Importance
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = AppIcons.Star,
                    contentDescription = null,
                    tint = extendedColors.warning,
                    modifier = Modifier.size(16.dp)
                )
                Text(
                    text = "${(memory.importance * 100).toInt()}% importance",
                    style = MaterialTheme.typography.bodySmall,
                    color = extendedColors.mutedForeground
                )
            }

            // Usage count
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = AppIcons.History,
                    contentDescription = null,
                    tint = extendedColors.mutedForeground,
                    modifier = Modifier.size(16.dp)
                )
                Text(
                    text = "Used ${memory.usageCount} ${if (memory.usageCount == 1) "time" else "times"}",
                    style = MaterialTheme.typography.bodySmall,
                    color = extendedColors.mutedForeground
                )
            }
        }

        // Main content card
        Surface(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(8.dp),
            color = extendedColors.card,
            border = androidx.compose.foundation.BorderStroke(1.dp, extendedColors.border)
        ) {
            Text(
                text = memory.content,
                style = MaterialTheme.typography.bodyLarge,
                lineHeight = 24.sp,
                modifier = Modifier.padding(24.dp)
            )
        }

        // Tags
        if (memory.tags.isNotEmpty()) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    text = "Tags",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Medium,
                    color = extendedColors.mutedForeground
                )
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    memory.tags.forEach { tag ->
                        Box(
                            modifier = Modifier
                                .clip(RoundedCornerShape(4.dp))
                                .background(extendedColors.muted)
                                .border(1.dp, extendedColors.border, RoundedCornerShape(4.dp))
                                .padding(horizontal = 10.dp, vertical = 4.dp)
                        ) {
                            Text(
                                text = tag,
                                style = MaterialTheme.typography.bodySmall,
                                color = extendedColors.mutedForeground
                            )
                        }
                    }
                }
            }
        }

        // Timestamps
        HorizontalDivider(color = extendedColors.border)

        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            TimestampRow(
                label = "Created",
                timestamp = memory.createdAt
            )

            if (memory.updatedAt > memory.createdAt) {
                TimestampRow(
                    label = "Last updated",
                    timestamp = memory.updatedAt
                )
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "Memory ID",
                    style = MaterialTheme.typography.bodySmall,
                    color = extendedColors.mutedForeground
                )
                Box(
                    modifier = Modifier
                        .clip(RoundedCornerShape(4.dp))
                        .background(extendedColors.muted)
                        .padding(horizontal = 8.dp, vertical = 4.dp)
                ) {
                    Text(
                        text = memory.id,
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 11.sp,
                        color = extendedColors.mutedForeground
                    )
                }
            }
        }
    }
}

@Composable
private fun TimestampRow(
    label: String,
    timestamp: Long
) {
    val extendedColors = AliciaTheme.extendedColors
    val dateFormat = remember { SimpleDateFormat("MMM d, yyyy 'at' h:mm a", Locale.getDefault()) }
    val formattedDate = remember(timestamp) { dateFormat.format(Date(timestamp)) }
    val relativeTime = remember(timestamp) { formatRelativeTime(timestamp) }

    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodySmall,
            color = extendedColors.mutedForeground
        )
        Text(
            text = "$relativeTime ($formattedDate)",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onBackground
        )
    }
}

@Composable
private fun getCategoryColor(category: MemoryCategory): Color {
    val extendedColors = AliciaTheme.extendedColors
    return when (category) {
        MemoryCategory.PREFERENCE -> extendedColors.accent
        MemoryCategory.FACT -> extendedColors.success
        MemoryCategory.CONTEXT -> extendedColors.warning
        MemoryCategory.INSTRUCTION -> extendedColors.destructive
    }
}

private fun formatRelativeTime(timestamp: Long): String {
    val now = System.currentTimeMillis()
    val diff = now - timestamp

    return when {
        diff < 60000 -> "just now"
        diff < 3600000 -> "${diff / 60000}m ago"
        diff < 86400000 -> "${diff / 3600000}h ago"
        diff < 604800000 -> "${diff / 86400000}d ago"
        else -> {
            val date = Date(timestamp)
            SimpleDateFormat("MMM d", Locale.getDefault()).format(date)
        }
    }
}
