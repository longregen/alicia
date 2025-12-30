package org.localforge.alicia.feature.assistant.components

import androidx.compose.foundation.layout.*
import org.localforge.alicia.core.common.ui.AppIcons
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay

@Composable
fun ResponseControls(
    isGenerating: Boolean,
    hasMessages: Boolean,
    onStop: () -> Unit,
    onRegenerate: () -> Unit,
    modifier: Modifier = Modifier
) {
    var isStopping by remember { mutableStateOf(false) }

    // Reset stopping state after a delay
    // The 1 second timeout allows the UI to show "Stopping..." feedback before returning to normal state
    // This provides user feedback even if the stop operation doesn't have a completion callback
    LaunchedEffect(isStopping) {
        if (isStopping) {
            delay(1000)
            isStopping = false
        }
    }

    if (!hasMessages) {
        return
    }

    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically
    ) {
        if (isGenerating) {
            // Stop button
            Button(
                onClick = {
                    isStopping = true
                    onStop()
                },
                enabled = !isStopping,
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.error,
                    contentColor = MaterialTheme.colorScheme.onError
                ),
                modifier = Modifier.padding(4.dp)
            ) {
                Icon(
                    imageVector = AppIcons.Stop,
                    contentDescription = "Stop",
                    modifier = Modifier.size(18.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(if (isStopping) "Stopping..." else "Stop")
            }
        } else {
            // Regenerate button
            OutlinedButton(
                onClick = onRegenerate,
                modifier = Modifier.padding(4.dp)
            ) {
                Icon(
                    imageVector = AppIcons.Refresh,
                    contentDescription = "Regenerate",
                    modifier = Modifier.size(18.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text("Regenerate")
            }
        }
    }
}
