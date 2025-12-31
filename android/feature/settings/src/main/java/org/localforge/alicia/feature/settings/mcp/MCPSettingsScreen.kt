package org.localforge.alicia.feature.settings.mcp

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import org.localforge.alicia.core.common.ui.AppIcons
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.lifecycle.viewmodel.compose.hiltViewModel
import org.localforge.alicia.feature.settings.mcp.components.AddServerDialog
import org.localforge.alicia.feature.settings.mcp.components.EmptyServersState
import org.localforge.alicia.feature.settings.mcp.components.MCPServerCard

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MCPSettingsScreen(
    viewModel: MCPSettingsViewModel = hiltViewModel(),
    onNavigateBack: () -> Unit = {}
) {
    val uiState by viewModel.uiState.collectAsState()
    var showAddDialog by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("MCP Server Settings") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = AppIcons.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                }
            )
        },
        floatingActionButton = {
            FloatingActionButton(
                onClick = { showAddDialog = true }
            ) {
                Icon(
                    imageVector = AppIcons.Add,
                    contentDescription = "Add Server"
                )
            }
        }
    ) { paddingValues ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
        ) {
            when {
                uiState.isLoading -> {
                    Box(
                        modifier = Modifier.fillMaxSize(),
                        contentAlignment = androidx.compose.ui.Alignment.Center
                    ) {
                        CircularProgressIndicator()
                    }
                }

                uiState.error != null -> {
                    Box(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(16.dp),
                        contentAlignment = androidx.compose.ui.Alignment.Center
                    ) {
                        Column(
                            horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally,
                            verticalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            Text(
                                text = "Error: ${uiState.error}",
                                style = MaterialTheme.typography.bodyLarge,
                                color = MaterialTheme.colorScheme.error
                            )
                            Button(onClick = { viewModel.loadServers() }) {
                                Text("Retry")
                            }
                        }
                    }
                }

                uiState.servers.isEmpty() -> {
                    EmptyServersState(
                        onAddServerClick = { showAddDialog = true }
                    )
                }

                else -> {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        items(
                            items = uiState.servers,
                            // Note: Using server.name as key assumes server names are unique
                            key = { it.name }
                        ) { server ->
                            MCPServerCard(
                                server = server,
                                tools = uiState.tools.filter { tool ->
                                    server.tools.contains(tool.name)
                                },
                                onDelete = { viewModel.deleteServer(server.name) }
                            )
                        }
                    }
                }
            }

            // Clear success message after delay
            uiState.successMessage?.let { message ->
                LaunchedEffect(message) {
                    kotlinx.coroutines.delay(3000)
                    viewModel.clearSuccessMessage()
                }
            }
        }
    }

    if (showAddDialog) {
        AddServerDialog(
            existingServerNames = uiState.servers.map { it.name }.toSet(),
            onDismiss = { showAddDialog = false },
            onConfirm = { config ->
                viewModel.addServer(config)
                showAddDialog = false
            }
        )
    }
}
