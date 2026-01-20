package org.localforge.alicia.feature.memory.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import org.localforge.alicia.core.domain.model.Memory
import org.localforge.alicia.core.domain.model.MemoryCategory

/**
 * Dialog for creating or editing a memory.
 * Matches the web frontend's MemoryEditor component.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MemoryEditorDialog(
    memory: Memory?,
    onSave: (String, MemoryCategory) -> Unit,
    onDismiss: () -> Unit
) {
    var content by remember(memory) { mutableStateOf(memory?.content ?: "") }
    var category by remember(memory) { mutableStateOf(memory?.category ?: MemoryCategory.PREFERENCE) }
    var categoryExpanded by remember { mutableStateOf(false) }

    val isEditing = memory != null

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Card(
            modifier = Modifier
                .fillMaxWidth(0.95f)
                .padding(16.dp),
            shape = RoundedCornerShape(16.dp)
        ) {
            Column(
                modifier = Modifier.padding(24.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Title
                Text(
                    text = if (isEditing) "Edit Memory" else "Create Memory",
                    style = MaterialTheme.typography.headlineSmall
                )

                // Content field
                OutlinedTextField(
                    value = content,
                    onValueChange = { content = it },
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 120.dp),
                    label = { Text("Memory Content") },
                    placeholder = { Text("Enter the memory content...") },
                    minLines = 4,
                    maxLines = 8,
                    shape = RoundedCornerShape(8.dp)
                )

                // Category selector
                ExposedDropdownMenuBox(
                    expanded = categoryExpanded,
                    onExpandedChange = { categoryExpanded = it }
                ) {
                    OutlinedTextField(
                        value = category.name.lowercase().replaceFirstChar { it.uppercase() },
                        onValueChange = {},
                        modifier = Modifier
                            .fillMaxWidth()
                            .menuAnchor(MenuAnchorType.PrimaryEditable),
                        readOnly = true,
                        label = { Text("Category") },
                        trailingIcon = {
                            ExposedDropdownMenuDefaults.TrailingIcon(expanded = categoryExpanded)
                        },
                        shape = RoundedCornerShape(8.dp)
                    )

                    ExposedDropdownMenu(
                        expanded = categoryExpanded,
                        onDismissRequest = { categoryExpanded = false }
                    ) {
                        MemoryCategory.entries.forEach { cat ->
                            DropdownMenuItem(
                                text = {
                                    Column {
                                        Text(cat.name.lowercase().replaceFirstChar { it.uppercase() })
                                        Text(
                                            text = getCategoryDescription(cat),
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant
                                        )
                                    }
                                },
                                onClick = {
                                    category = cat
                                    categoryExpanded = false
                                }
                            )
                        }
                    }
                }

                // Actions
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.End
                ) {
                    TextButton(onClick = onDismiss) {
                        Text("Cancel")
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(
                        onClick = { onSave(content, category) },
                        enabled = content.isNotBlank()
                    ) {
                        Text(if (isEditing) "Save" else "Create")
                    }
                }
            }
        }
    }
}

private fun getCategoryDescription(category: MemoryCategory): String {
    return when (category) {
        MemoryCategory.PREFERENCE -> "User preferences and settings"
        MemoryCategory.FACT -> "Facts about the user or their environment"
        MemoryCategory.CONTEXT -> "Contextual information for conversations"
        MemoryCategory.INSTRUCTION -> "Instructions for how to behave"
    }
}
