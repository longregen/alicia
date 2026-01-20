package org.localforge.alicia.feature.server

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import org.localforge.alicia.core.domain.model.*

/**
 * ServerScreen - Server information panel for Alicia
 *
 * Features matching the web frontend:
 * - Connection status with latency and quality indicator
 * - Model information (name and provider)
 * - MCP server statuses
 * - Session statistics
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ServerScreen(
    viewModel: ServerViewModel = hiltViewModel(),
    onNavigateBack: () -> Unit = {}
) {
    val serverInfo by viewModel.serverInfo.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()
    val error by viewModel.error.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Outlined.Dns,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.tertiary
                        )
                        Text("Server Info")
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Default.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.refresh() }) {
                        Icon(
                            imageVector = Icons.Outlined.Refresh,
                            contentDescription = "Refresh"
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
            when {
                isLoading -> {
                    LoadingState()
                }
                error != null -> {
                    ErrorState(
                        error = error!!,
                        onRetry = { viewModel.refresh() }
                    )
                }
                else -> {
                    ServerInfoContent(serverInfo = serverInfo)
                }
            }
        }
    }
}

@Composable
private fun LoadingState() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            CircularProgressIndicator()
            Text(
                text = "Loading...",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
private fun ErrorState(
    error: String,
    onRetry: () -> Unit
) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Card(
            modifier = Modifier.padding(16.dp),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.2f)
            ),
            border = androidx.compose.foundation.BorderStroke(
                1.dp,
                MaterialTheme.colorScheme.error.copy(alpha = 0.3f)
            )
        ) {
            Column(
                modifier = Modifier.padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                Icon(
                    imageVector = Icons.Outlined.ErrorOutline,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.error,
                    modifier = Modifier.size(48.dp)
                )
                Text(
                    text = error,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.error
                )
                TextButton(onClick = onRetry) {
                    Text("Retry")
                }
            }
        }
    }
}

@Composable
private fun ServerInfoContent(serverInfo: ServerInfo) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // Connection Panel
        ConnectionPanel(serverInfo = serverInfo)

        // Model Panel
        if (serverInfo.modelInfo != null) {
            ModelPanel(modelInfo = serverInfo.modelInfo!!)
        }

        // MCP Servers Panel
        if (serverInfo.mcpServers.isNotEmpty()) {
            MCPServersPanel(
                servers = serverInfo.mcpServers,
                summary = serverInfo.mcpServerSummary
            )
        }

        // Session Stats Panel
        SessionStatsPanel(stats = serverInfo.sessionStats)
    }
}

@Composable
private fun ConnectionPanel(serverInfo: ServerInfo) {
    InfoPanel(title = "Connection") {
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Status badge
            StatusBadge(
                text = serverInfo.connectionStatus.displayName,
                status = when (serverInfo.connectionStatus) {
                    ConnectionStatus.CONNECTED -> StatusType.SUCCESS
                    ConnectionStatus.CONNECTING, ConnectionStatus.RECONNECTING -> StatusType.WARNING
                    ConnectionStatus.DISCONNECTED -> StatusType.ERROR
                }
            )

            if (serverInfo.isConnected) {
                Text(
                    text = "·",
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Text(
                    text = "${serverInfo.latency}ms (${serverInfo.connectionQuality.displayName})",
                    style = MaterialTheme.typography.bodySmall,
                    color = getQualityColor(serverInfo.connectionQuality)
                )
            }

            if (serverInfo.isConnecting) {
                CircularProgressIndicator(
                    modifier = Modifier.size(16.dp),
                    strokeWidth = 2.dp,
                    color = MaterialTheme.colorScheme.tertiary
                )
            }
        }
    }
}

@Composable
private fun ModelPanel(modelInfo: ModelInfo) {
    InfoPanel(title = "Model") {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(
                text = modelInfo.name,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Medium
            )
            Text(
                text = "Provider: ${modelInfo.provider}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
private fun MCPServersPanel(
    servers: List<MCPServer>,
    summary: String
) {
    InfoPanel(
        title = "MCP Servers",
        trailing = {
            Text(
                text = summary,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            servers.forEach { server ->
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = server.name,
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.weight(1f)
                    )
                    StatusBadge(
                        text = server.status.name.lowercase(),
                        status = when (server.status) {
                            MCPServerStatus.CONNECTED -> StatusType.SUCCESS
                            MCPServerStatus.DISCONNECTED -> StatusType.ERROR
                            MCPServerStatus.ERROR -> StatusType.ERROR
                        }
                    )
                }
            }
        }
    }
}

@Composable
private fun SessionStatsPanel(stats: SessionStats) {
    InfoPanel(title = "Session") {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            StatItem(label = "Messages", value = stats.messageCount.toString())
            StatItem(label = "Tool Calls", value = stats.toolCallCount.toString())
            StatItem(label = "Memories", value = stats.memoriesUsed.toString())
            StatItem(label = "Duration", value = stats.formattedDuration)
        }
    }
}

@Composable
private fun InfoPanel(
    title: String,
    trailing: @Composable (() -> Unit)? = null,
    content: @Composable () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    fontWeight = FontWeight.SemiBold
                )
                trailing?.invoke()
            }
            content()
        }
    }
}

@Composable
private fun StatItem(
    label: String,
    value: String
) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium
        )
    }
}

private enum class StatusType {
    SUCCESS, WARNING, ERROR
}

@Composable
private fun StatusBadge(
    text: String,
    status: StatusType
) {
    val backgroundColor = when (status) {
        StatusType.SUCCESS -> Color(0xFF4DD488).copy(alpha = 0.15f) // Success green
        StatusType.WARNING -> Color(0xFFE5B94D).copy(alpha = 0.15f) // Warning yellow
        StatusType.ERROR -> MaterialTheme.colorScheme.error.copy(alpha = 0.15f)
    }

    val textColor = when (status) {
        StatusType.SUCCESS -> Color(0xFF4DD488)
        StatusType.WARNING -> Color(0xFFE5B94D)
        StatusType.ERROR -> MaterialTheme.colorScheme.error
    }

    Surface(
        shape = RoundedCornerShape(4.dp),
        color = backgroundColor
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            color = textColor,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
        )
    }
}

@Composable
private fun getQualityColor(quality: ConnectionQuality): Color {
    return when (quality) {
        ConnectionQuality.EXCELLENT -> Color(0xFF4DD488)
        ConnectionQuality.GOOD -> MaterialTheme.colorScheme.primary
        ConnectionQuality.FAIR -> Color(0xFFE5B94D)
        ConnectionQuality.POOR -> MaterialTheme.colorScheme.error
    }
}
