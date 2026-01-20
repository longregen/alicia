package org.localforge.alicia.feature.assistant.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import org.localforge.alicia.ui.theme.AliciaTheme

/**
 * InputArea - Unified input area matching the web's InputArea.tsx.
 *
 * Layout: [MicrophoneButton] [TextField] [SendButton]
 *
 * Features:
 * - Voice input button with VAD visualization
 * - Text input with rounded corners
 * - Send button with paper airplane icon
 * - Autofocus when conversation changes
 * - Enter key to send
 */
@Composable
fun InputArea(
    textInput: String,
    onTextInputChange: (String) -> Unit,
    onSend: () -> Unit,
    onVoiceClick: () -> Unit = {},
    microphoneStatus: MicrophoneStatus = MicrophoneStatus.Inactive,
    isSpeaking: Boolean = false,
    speechProbability: Float = 0f,
    disabled: Boolean = false,
    isVoiceActive: Boolean = false,
    placeholder: String = "Type a message...",
    conversationId: String? = null,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors
    val focusRequester = remember { FocusRequester() }

    // Autofocus when conversation changes
    LaunchedEffect(conversationId) {
        if (conversationId != null) {
            try {
                focusRequester.requestFocus()
            } catch (_: Exception) {
                // Focus request might fail if component not yet laid out
            }
        }
    }

    val canSend = textInput.isNotBlank()
    val isRecording = isVoiceActive && microphoneStatus == MicrophoneStatus.Recording

    Row(
        modifier = modifier
            .fillMaxWidth()
            .background(extendedColors.elevated)
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.Bottom,
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        // Voice input button
        MicrophoneButton(
            microphoneStatus = microphoneStatus,
            isSpeaking = isSpeaking,
            speechProbability = speechProbability,
            onClick = onVoiceClick,
            disabled = disabled
        )

        // Text input field
        OutlinedTextField(
            value = textInput,
            onValueChange = onTextInputChange,
            modifier = Modifier
                .weight(1f)
                .focusRequester(focusRequester),
            placeholder = {
                Text(
                    text = placeholder,
                    color = extendedColors.mutedForeground
                )
            },
            enabled = !disabled && !isRecording,
            maxLines = 4,
            shape = RoundedCornerShape(24.dp),
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = extendedColors.accent,
                unfocusedBorderColor = extendedColors.border,
                disabledBorderColor = extendedColors.border.copy(alpha = 0.5f),
                focusedContainerColor = extendedColors.card,
                unfocusedContainerColor = extendedColors.card,
                disabledContainerColor = extendedColors.muted
            ),
            keyboardOptions = KeyboardOptions(
                imeAction = ImeAction.Send
            ),
            keyboardActions = KeyboardActions(
                onSend = {
                    if (canSend) {
                        onSend()
                    }
                }
            )
        )

        // Send button
        InputSendButton(
            onSend = onSend,
            canSend = canSend,
            disabled = disabled || isRecording
        )
    }
}
