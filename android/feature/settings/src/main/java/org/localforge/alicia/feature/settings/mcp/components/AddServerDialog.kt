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

private const val ARG_SEPARATOR = ","
private const val ENV_SEPARATOR = "="
private const val ENV_PARTS_LIMIT = 2

private fun validateEnvLines(envText: String): Pair<Map<String, String>, List<String>> {
    if (envText.isBlank()) return emptyMap<String, String>() to emptyList()

    val validEntries = mutableMapOf<String, String>()
    val invalidLines = mutableListOf<String>()

    envText.lines().forEachIndexed { index, line ->
        val trimmedLine = line.trim()
        if (trimmedLine.isEmpty()) return@forEachIndexed

        val parts = trimmedLine.split("=", limit = 2)
        if (parts.size == 2 && parts[0].isNotBlank()) {
            validEntries[parts[0].trim()] = parts[1].trim()
        } else {
            invalidLines.add("Line ${index + 1}: \"$trimmedLine\"")
        }
    }

    return validEntries to invalidLines
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddServerDialog(
    existingServerNames: Set<String> = emptySet(),
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
    var envWarnings by remember { mutableStateOf<List<String>>(emptyList()) }

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

                OutlinedTextField(
                    value = argsText,
                    onValueChange = { argsText = it },
                    label = { Text("Arguments (comma-separated)") },
                    placeholder = { Text("arg1, arg2, arg3") },
                    modifier = Modifier.fillMaxWidth()
                )

                OutlinedTextField(
                    value = envText,
                    onValueChange = { newValue ->
                        envText = newValue
                        val (_, warnings) = validateEnvLines(newValue)
                        envWarnings = warnings
                    },
                    label = { Text("Environment Variables") },
                    placeholder = { Text("KEY1=value1\nKEY2=value2") },
                    isError = envWarnings.isNotEmpty(),
                    supportingText = {
                        if (envWarnings.isNotEmpty()) {
                            Text(
                                text = "Invalid format (will be skipped): ${envWarnings.take(2).joinToString(", ")}",
                                color = MaterialTheme.colorScheme.error
                            )
                        } else {
                            Text("One per line: KEY=value")
                        }
                    },
                    minLines = 3,
                    maxLines = 5,
                    modifier = Modifier.fillMaxWidth()
                )
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    var hasError = false

                    if (name.isBlank()) {
                        nameError = "Server name is required"
                        hasError = true
                    } else if (existingServerNames.contains(name.trim())) {
                        nameError = "A server with this name already exists"
                        hasError = true
                    }

                    if (command.isBlank()) {
                        commandError = "Command is required"
                        hasError = true
                    }

                    if (hasError) return@TextButton

                    val args = if (argsText.isNotBlank()) {
                        argsText.split(ARG_SEPARATOR)
                            .map { it.trim() }
                            .filter { it.isNotEmpty() }
                    } else {
                        emptyList()
                    }

                    val (env, _) = validateEnvLines(envText)
                    val envMap = env.takeIf { it.isNotEmpty() }

                    val config = MCPServerConfig(
                        name = name.trim(),
                        transport = transport,
                        command = command.trim(),
                        args = args,
                        env = envMap
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
