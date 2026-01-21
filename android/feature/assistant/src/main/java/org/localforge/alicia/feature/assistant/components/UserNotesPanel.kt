package org.localforge.alicia.feature.assistant.components

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import org.localforge.alicia.core.common.ui.AppIcons
import org.localforge.alicia.core.domain.model.Note
import org.localforge.alicia.core.domain.model.NoteCategory
import org.localforge.alicia.core.domain.model.NoteTargetType
import java.time.Instant
import java.time.ZoneId
import java.time.temporal.ChronoUnit

@Composable
fun UserNotesPanel(
    targetType: NoteTargetType,
    targetId: String,
    notes: List<Note>,
    isLoading: Boolean,
    error: String?,
    modifier: Modifier = Modifier,
    compact: Boolean = false,
    onAddNote: (content: String, category: NoteCategory) -> Unit,
    onUpdateNote: (noteId: String, content: String) -> Unit,
    onDeleteNote: (noteId: String) -> Unit
) {
    var isAddingNote by remember { mutableStateOf(false) }
    var newNoteContent by remember { mutableStateOf("") }
    var newNoteCategory by remember { mutableStateOf(NoteCategory.GENERAL) }
    var editingNoteId by remember { mutableStateOf<String?>(null) }
    var editContent by remember { mutableStateOf("") }

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surface,
        tonalElevation = 1.dp
    ) {
        Column(
            modifier = Modifier.padding(if (compact) 12.dp else 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "Notes",
                        style = if (compact) MaterialTheme.typography.titleSmall else MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold
                    )
                    if (notes.isNotEmpty()) {
                        Text(
                            text = "(${notes.size})",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }

                if (!isAddingNote) {
                    Button(
                        onClick = { isAddingNote = true },
                        enabled = !isLoading,
                        contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp)
                    ) {
                        Icon(
                            imageVector = AppIcons.Add,
                            contentDescription = null,
                            modifier = Modifier.size(16.dp)
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Add Note", style = MaterialTheme.typography.labelMedium)
                    }
                }
            }

            error?.let {
                Surface(
                    color = MaterialTheme.colorScheme.errorContainer,
                    shape = RoundedCornerShape(8.dp)
                ) {
                    Text(
                        text = it,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onErrorContainer,
                        modifier = Modifier.padding(8.dp)
                    )
                }
            }

            AnimatedVisibility(
                visible = isAddingNote,
                enter = expandVertically(),
                exit = shrinkVertically()
            ) {
                AddNoteForm(
                    content = newNoteContent,
                    category = newNoteCategory,
                    isLoading = isLoading,
                    onContentChange = { newNoteContent = it },
                    onCategoryChange = { newNoteCategory = it },
                    onSave = {
                        if (newNoteContent.isNotBlank()) {
                            onAddNote(newNoteContent, newNoteCategory)
                            newNoteContent = ""
                            newNoteCategory = NoteCategory.GENERAL
                            isAddingNote = false
                        }
                    },
                    onCancel = {
                        isAddingNote = false
                        newNoteContent = ""
                        newNoteCategory = NoteCategory.GENERAL
                    }
                )
            }

            if (notes.isEmpty() && !isAddingNote) {
                Text(
                    text = "No notes yet. Add one to get started.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 16.dp),
                    textAlign = androidx.compose.ui.text.style.TextAlign.Center
                )
            } else {
                LazyColumn(
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.heightIn(max = 300.dp)
                ) {
                    items(notes, key = { it.id }) { note ->
                        NoteItem(
                            note = note,
                            isEditing = editingNoteId == note.id,
                            editContent = if (editingNoteId == note.id) editContent else note.content,
                            isLoading = isLoading,
                            onEditStart = {
                                editingNoteId = note.id
                                editContent = note.content
                            },
                            onEditCancel = {
                                editingNoteId = null
                                editContent = ""
                            },
                            onEditSave = {
                                if (editContent.isNotBlank()) {
                                    onUpdateNote(note.id, editContent)
                                    editingNoteId = null
                                    editContent = ""
                                }
                            },
                            onEditContentChange = { editContent = it },
                            onDelete = { onDeleteNote(note.id) }
                        )
                    }
                }
            }

            if (isLoading) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 8.dp),
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(16.dp),
                        strokeWidth = 2.dp
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        text = "Processing...",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        }
    }
}

@Composable
private fun AddNoteForm(
    content: String,
    category: NoteCategory,
    isLoading: Boolean,
    onContentChange: (String) -> Unit,
    onCategoryChange: (NoteCategory) -> Unit,
    onSave: () -> Unit,
    onCancel: () -> Unit
) {
    Surface(
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                NoteCategory.entries.forEach { cat ->
                    CategoryChip(
                        category = cat,
                        isSelected = category == cat,
                        onClick = { onCategoryChange(cat) }
                    )
                }
            }

            OutlinedTextField(
                value = content,
                onValueChange = onContentChange,
                placeholder = { Text("Write your note here...") },
                modifier = Modifier.fillMaxWidth(),
                minLines = 3,
                maxLines = 5,
                textStyle = MaterialTheme.typography.bodyMedium
            )

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End,
                verticalAlignment = Alignment.CenterVertically
            ) {
                TextButton(
                    onClick = onCancel,
                    enabled = !isLoading
                ) {
                    Text("Cancel")
                }
                Spacer(modifier = Modifier.width(8.dp))
                Button(
                    onClick = onSave,
                    enabled = content.isNotBlank() && !isLoading
                ) {
                    Text("Save")
                }
            }
        }
    }
}

@Composable
private fun NoteItem(
    note: Note,
    isEditing: Boolean,
    editContent: String,
    isLoading: Boolean,
    onEditStart: () -> Unit,
    onEditCancel: () -> Unit,
    onEditSave: () -> Unit,
    onEditContentChange: (String) -> Unit,
    onDelete: () -> Unit
) {
    Surface(
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.surface,
        border = androidx.compose.foundation.BorderStroke(
            1.dp,
            MaterialTheme.colorScheme.outlineVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                CategoryBadge(category = note.category)
                Text(
                    text = formatTimestamp(note.createdAt),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            if (isEditing) {
                OutlinedTextField(
                    value = editContent,
                    onValueChange = onEditContentChange,
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 2,
                    maxLines = 5,
                    textStyle = MaterialTheme.typography.bodyMedium
                )

                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Button(
                        onClick = onEditSave,
                        enabled = editContent.isNotBlank() && !isLoading,
                        contentPadding = PaddingValues(horizontal = 12.dp, vertical = 4.dp)
                    ) {
                        Text("Save", style = MaterialTheme.typography.labelSmall)
                    }
                    TextButton(
                        onClick = onEditCancel,
                        enabled = !isLoading
                    ) {
                        Text("Cancel", style = MaterialTheme.typography.labelSmall)
                    }
                }
            } else {
                Text(
                    text = note.content,
                    style = MaterialTheme.typography.bodyMedium
                )

                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "Edit",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.clickable(enabled = !isLoading) { onEditStart() }
                    )
                    Text(
                        text = "•",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Text(
                        text = "Delete",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.error,
                        modifier = Modifier.clickable(enabled = !isLoading) { onDelete() }
                    )

                    if (note.updatedAt != note.createdAt) {
                        Text(
                            text = "•",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Text(
                            text = "edited ${formatTimestamp(note.updatedAt)}",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun CategoryChip(
    category: NoteCategory,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    val (backgroundColor, contentColor, borderColor) = getCategoryColors(category)

    Surface(
        shape = RoundedCornerShape(4.dp),
        color = if (isSelected) backgroundColor else Color.Transparent,
        border = androidx.compose.foundation.BorderStroke(
            1.dp,
            if (isSelected) borderColor else MaterialTheme.colorScheme.outlineVariant
        ),
        modifier = Modifier.clickable { onClick() }
    ) {
        Text(
            text = getCategoryLabel(category),
            style = MaterialTheme.typography.labelSmall,
            color = if (isSelected) contentColor else MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
        )
    }
}

@Composable
private fun CategoryBadge(category: NoteCategory) {
    val (backgroundColor, contentColor, borderColor) = getCategoryColors(category)

    Surface(
        shape = RoundedCornerShape(4.dp),
        color = backgroundColor,
        border = androidx.compose.foundation.BorderStroke(1.dp, borderColor)
    ) {
        Text(
            text = getCategoryLabel(category),
            style = MaterialTheme.typography.labelSmall,
            color = contentColor,
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
        )
    }
}

private fun getCategoryLabel(category: NoteCategory): String {
    return when (category) {
        NoteCategory.IMPROVEMENT -> "Improvement"
        NoteCategory.CORRECTION -> "Correction"
        NoteCategory.CONTEXT -> "Context"
        NoteCategory.GENERAL -> "General"
    }
}

@Composable
private fun getCategoryColors(category: NoteCategory): Triple<Color, Color, Color> {
    return when (category) {
        NoteCategory.IMPROVEMENT -> Triple(
            Color(0xFFD1FAE5), // Light green (accent subtle)
            Color(0xFF059669), // Green (accent)
            Color(0xFF059669)
        )
        NoteCategory.CORRECTION -> Triple(
            Color(0xFFFEE2E2), // Light red (error subtle)
            Color(0xFFDC2626), // Red (error)
            Color(0xFFDC2626)
        )
        NoteCategory.CONTEXT -> Triple(
            Color(0xFFFEF3C7), // Light yellow (warning subtle)
            Color(0xFFD97706), // Amber (warning)
            Color(0xFFD97706)
        )
        NoteCategory.GENERAL -> Triple(
            MaterialTheme.colorScheme.surface,
            MaterialTheme.colorScheme.onSurface,
            MaterialTheme.colorScheme.outline
        )
    }
}

private fun formatTimestamp(timestamp: Long): String {
    val instant = Instant.ofEpochMilli(timestamp)
    val now = Instant.now()
    val diffMins = ChronoUnit.MINUTES.between(instant, now)
    val diffHours = ChronoUnit.HOURS.between(instant, now)
    val diffDays = ChronoUnit.DAYS.between(instant, now)

    return when {
        diffMins < 1 -> "just now"
        diffMins < 60 -> "${diffMins}m ago"
        diffHours < 24 -> "${diffHours}h ago"
        diffDays < 7 -> "${diffDays}d ago"
        else -> {
            val localDate = instant.atZone(ZoneId.systemDefault()).toLocalDate()
            localDate.toString()
        }
    }
}
