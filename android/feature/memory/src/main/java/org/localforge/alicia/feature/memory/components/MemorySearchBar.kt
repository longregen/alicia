package org.localforge.alicia.feature.memory.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import org.localforge.alicia.core.domain.model.MemoryCategory

/**
 * Search bar with category filter dropdown.
 * Matches the web frontend's MemorySearch component.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MemorySearchBar(
    searchQuery: String,
    selectedCategory: MemoryCategory?,
    onSearchQueryChange: (String) -> Unit,
    onCategoryChange: (MemoryCategory?) -> Unit,
    modifier: Modifier = Modifier
) {
    var categoryExpanded by remember { mutableStateOf(false) }

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Search field
        OutlinedTextField(
            value = searchQuery,
            onValueChange = onSearchQueryChange,
            modifier = Modifier.weight(1f),
            placeholder = { Text("Search memories...") },
            leadingIcon = {
                Icon(
                    imageVector = Icons.Outlined.Search,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
            },
            trailingIcon = {
                if (searchQuery.isNotEmpty()) {
                    IconButton(onClick = { onSearchQueryChange("") }) {
                        Icon(
                            imageVector = Icons.Outlined.Close,
                            contentDescription = "Clear search"
                        )
                    }
                }
            },
            singleLine = true,
            shape = RoundedCornerShape(8.dp)
        )

        // Category filter
        ExposedDropdownMenuBox(
            expanded = categoryExpanded,
            onExpandedChange = { categoryExpanded = it }
        ) {
            FilterChip(
                selected = selectedCategory != null,
                onClick = { categoryExpanded = true },
                label = {
                    Text(selectedCategory?.name?.lowercase()?.replaceFirstChar { it.uppercase() }
                        ?: "All")
                },
                trailingIcon = {
                    ExposedDropdownMenuDefaults.TrailingIcon(expanded = categoryExpanded)
                },
                modifier = Modifier.menuAnchor(MenuAnchorType.PrimaryEditable)
            )

            ExposedDropdownMenu(
                expanded = categoryExpanded,
                onDismissRequest = { categoryExpanded = false }
            ) {
                DropdownMenuItem(
                    text = { Text("All Categories") },
                    onClick = {
                        onCategoryChange(null)
                        categoryExpanded = false
                    },
                    leadingIcon = if (selectedCategory == null) {
                        { Icon(Icons.Outlined.Close, contentDescription = null) }
                    } else null
                )

                HorizontalDivider()

                MemoryCategory.entries.forEach { category ->
                    DropdownMenuItem(
                        text = { Text(category.name.lowercase().replaceFirstChar { it.uppercase() }) },
                        onClick = {
                            onCategoryChange(category)
                            categoryExpanded = false
                        },
                        leadingIcon = if (selectedCategory == category) {
                            { Icon(Icons.Outlined.Close, contentDescription = null) }
                        } else null
                    )
                }
            }
        }
    }
}
