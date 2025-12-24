package org.localforge.alicia.feature.settings.mcp.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import org.localforge.alicia.core.common.Logger
import org.localforge.alicia.core.domain.model.MCPServerConfig
import org.localforge.alicia.core.domain.model.MCPTransport

/**
 * Character used to separate multiple command arguments in the input field.
 */
private const val ARG_SEPARATOR = ","

/**
 * Character used to separate environment variable key-value pairs.
 */
private const val ENV_SEPARATOR = "="

/**
 * Maximum number of parts when splitting an environment variable (key=value).
 */
private const val ENV_PARTS_LIMIT = 2

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddServerDialog(
    onDismiss: () -> Unit,
    onConfirm: (MCPServerConfig) -> Unit
) {
    val logger = remember { Logger.forTag("AddServerDialog") }
    var name by remember { mutableStateOf("") }
    var transport by remember { mutableStateOf(MCPTransport.STDIO) }
    var command by remember { mutableStateOf("") }
    var argsText by remember { mutableStateOf("") }
    var envText by remember { mutableStateOf("") }
    var nameError by remember { mutableStateOf<String?>(null) }
    var commandError by remember { mutableStateOf<String?>(null) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add MCP Server") },
        text = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Server Name
                OutlinedTextField(
                    value = name,
                    onValueChange = {
                        name = it
                        nameError = null
                    },
                    label = { Text("Server Name *") },
                    placeholder = { Text("my-mcp-server") },
                    isError = nameError != null,
                    supportingText = nameError?.let { { Text(it) } },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth()
                )

                // Transport Type
                Column(modifier = Modifier.fillMaxWidth()) {
                    Text(
                        text = "Transport Type *",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        FilterChip(
                            selected = transport == MCPTransport.STDIO,
                            onClick = { transport = MCPTransport.STDIO },
                            label = { Text("stdio") },
                            modifier = Modifier.weight(1f)
                        )
                        FilterChip(
                            selected = transport == MCPTransport.SSE,
                            onClick = { transport = MCPTransport.SSE },
                            label = { Text("SSE") },
                            modifier = Modifier.weight(1f)
                        )
                    }
                }

                // Command
                OutlinedTextField(
                    value = command,
                    onValueChange = {
                        command = it
                        commandError = null
                    },
                    label = { Text("Command *") },
                    placeholder = { Text("/path/to/executable or npx package-name") },
                    isError = commandError != null,
                    supportingText = commandError?.let { { Text(it) } },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth()
                )

                // Arguments
                OutlinedTextField(
                    value = argsText,
                    onValueChange = { argsText = it },
                    label = { Text("Arguments (comma-separated)") },
                    placeholder = { Text("arg1, arg2, arg3") },
                    modifier = Modifier.fillMaxWidth()
                )

                // Environment Variables
                OutlinedTextField(
                    value = envText,
                    onValueChange = { envText = it },
                    label = { Text("Environment Variables") },
                    placeholder = { Text("KEY1=value1\nKEY2=value2") },
                    supportingText = { Text("One per line: KEY=value") },
                    minLines = 3,
                    maxLines = 5,
                    modifier = Modifier.fillMaxWidth()
                )
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    // Validate inputs
                    var hasError = false

                    if (name.isBlank()) {
                        nameError = "Server name is required"
                        hasError = true
                    }

                    if (command.isBlank()) {
                        commandError = "Command is required"
                        hasError = true
                    }

                    if (hasError) return@TextButton

                    // Parse args
                    val args = if (argsText.isNotBlank()) {
                        argsText.split(ARG_SEPARATOR)
                            .map { it.trim() }
                            .filter { it.isNotEmpty() }
                    } else {
                        emptyList()
                    }

                    // Parse env
                    val env = if (envText.isNotBlank()) {
                        envText.lines()
                            .mapNotNull { line ->
                                val trimmedLine = line.trim()
                                if (trimmedLine.isEmpty()) {
                                    return@mapNotNull null
                                }
                                val parts = trimmedLine.split(ENV_SEPARATOR, limit = ENV_PARTS_LIMIT)
                                if (parts.size == ENV_PARTS_LIMIT && parts[0].isNotBlank()) {
                                    parts[0].trim() to parts[1].trim()
                                } else {
                                    // Log invalid environment variable lines
                                    logger.w("Invalid environment variable format: '$trimmedLine'. Expected format: KEY=value")
                                    null
                                }
                            }
                            .toMap()
                            .takeIf { it.isNotEmpty() }
                    } else {
                        null
                    }

                    val config = MCPServerConfig(
                        name = name.trim(),
                        transport = transport,
                        command = command.trim(),
                        args = args,
                        env = env
                    )

                    onConfirm(config)
                }
            ) {
                Text("Add Server")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}
