package org.localforge.alicia.feature.conversations

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import org.localforge.alicia.core.common.ui.AppIcons
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.lifecycle.viewmodel.compose.hiltViewModel
import org.localforge.alicia.feature.conversations.components.ConversationItem

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ConversationsScreen(
    viewModel: ConversationsViewModel = hiltViewModel(),
    onNavigateBack: () -> Unit = {},
    onConversationClick: (String) -> Unit = {}
) {
    val conversations by viewModel.conversations.collectAsState()
    var showDeleteDialog by remember { mutableStateOf(false) }
    var conversationToDelete by remember { mutableStateOf<Conversation?>(null) }
    var showRenameDialog by remember { mutableStateOf(false) }
    var conversationToRename by remember { mutableStateOf<Conversation?>(null) }
    var renameText by remember { mutableStateOf("") }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Conversation History") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = AppIcons.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.clearAllConversations() }) {
                        Icon(
                            imageVector = AppIcons.Delete,
                            contentDescription = "Clear all"
                        )
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
            if (conversations.isEmpty()) {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(32.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center
                ) {
                    Text(
                        text = "No conversations yet",
                        style = MaterialTheme.typography.titleLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = "Start talking to Alicia to see your conversation history here",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.7f)
                    )
                }
            } else {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(16.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    items(conversations, key = { it.id }) { conversation ->
                        ConversationItem(
                            conversation = conversation,
                            onClick = { onConversationClick(conversation.id) },
                            onDeleteClick = {
                                conversationToDelete = conversation
                                showDeleteDialog = true
                            },
                            onArchiveClick = {
                                viewModel.archiveConversation(conversation.id)
                            },
                            onUnarchiveClick = {
                                viewModel.unarchiveConversation(conversation.id)
                            },
                            onRenameClick = {
                                conversationToRename = conversation
                                renameText = conversation.title ?: ""
                                showRenameDialog = true
                            }
                        )
                    }
                }
            }
        }
    }

    if (showDeleteDialog && conversationToDelete != null) {
        AlertDialog(
            onDismissRequest = { showDeleteDialog = false },
            title = { Text("Delete Conversation") },
            text = { Text("Are you sure you want to delete this conversation?") },
            confirmButton = {
                TextButton(
                    onClick = {
                        conversationToDelete?.let { viewModel.deleteConversation(it.id) }
                        showDeleteDialog = false
                        conversationToDelete = null
                    }
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

    if (showRenameDialog && conversationToRename != null) {
        AlertDialog(
            onDismissRequest = {
                showRenameDialog = false
                conversationToRename = null
                renameText = ""
            },
            title = { Text("Rename Conversation") },
            text = {
                OutlinedTextField(
                    value = renameText,
                    onValueChange = { renameText = it },
                    label = { Text("Title") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth()
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        conversationToRename?.let {
                            if (renameText.isNotBlank()) {
                                viewModel.renameConversation(it.id, renameText.trim())
                            }
                        }
                        showRenameDialog = false
                        conversationToRename = null
                        renameText = ""
                    },
                    enabled = renameText.isNotBlank()
                ) {
                    Text("Save")
                }
            },
            dismissButton = {
                TextButton(
                    onClick = {
                        showRenameDialog = false
                        conversationToRename = null
                        renameText = ""
                    }
                ) {
                    Text("Cancel")
                }
            }
        )
    }
}
