package org.localforge.alicia.feature.assistant

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.lifecycle.viewmodel.compose.hiltViewModel
import org.localforge.alicia.feature.assistant.components.*
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Keyboard
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.History
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.automirrored.filled.Send

/**
 * Main assistant screen that provides both voice and text interaction with the AI assistant.
 *
 * This composable displays:
 * - Conversation history with message bubbles
 * - Protocol messages (errors, reasoning steps, tool usages, memory traces, commentaries)
 * - Voice state indicator and manual activation button (in voice mode)
 * - Text input field with send button (in text mode)
 * - Response controls for stopping/regenerating responses
 *
 * @param conversationId Optional ID of a specific conversation to load. If null, loads the most recent conversation.
 * @param viewModel The ViewModel managing assistant state and interactions.
 * @param onNavigateToConversations Callback invoked when navigating to conversation history.
 * @param onNavigateToSettings Callback invoked when navigating to settings.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AssistantScreen(
    conversationId: String? = null,
    viewModel: AssistantViewModel = hiltViewModel(),
    onNavigateToConversations: () -> Unit = {},
    onNavigateToSettings: () -> Unit = {}
) {
    // Load specific conversation if provided
    LaunchedEffect(conversationId) {
        conversationId?.let {
            viewModel.loadSpecificConversation(it)
        }
    }
    val messages by viewModel.messages.collectAsState()
    val voiceState by viewModel.voiceState.collectAsState()
    val currentTranscription by viewModel.currentTranscription.collectAsState()
    val inputMode by viewModel.inputMode.collectAsState()
    val textInput by viewModel.textInput.collectAsState()
    val isSendingMessage by viewModel.isSendingMessage.collectAsState()
    val isGenerating by viewModel.isGenerating.collectAsState()

    // Protocol messages
    val errors by viewModel.errors.collectAsState()
    val reasoningSteps by viewModel.reasoningSteps.collectAsState()
    val toolUsages by viewModel.toolUsages.collectAsState()
    val memoryTraces by viewModel.memoryTraces.collectAsState()
    val commentaries by viewModel.commentaries.collectAsState()

    val listState = rememberLazyListState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Alicia") },
                actions = {
                    IconButton(onClick = { viewModel.toggleInputMode() }) {
                        Icon(
                            imageVector = if (inputMode == InputMode.Voice) {
                                Icons.Default.Keyboard
                            } else {
                                Icons.Default.Mic
                            },
                            contentDescription = if (inputMode == InputMode.Voice) "Switch to Text" else "Switch to Voice"
                        )
                    }
                    IconButton(onClick = onNavigateToConversations) {
                        Icon(
                            imageVector = Icons.Default.History,
                            contentDescription = "Conversations"
                        )
                    }
                    IconButton(onClick = onNavigateToSettings) {
                        Icon(
                            imageVector = Icons.Default.Settings,
                            contentDescription = "Settings"
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
                .background(MaterialTheme.colorScheme.background)
        ) {
            Column(
                modifier = Modifier.fillMaxSize(),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                // Conversation history
                LazyColumn(
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp),
                    state = listState,
                    reverseLayout = true,
                    contentPadding = PaddingValues(vertical = 8.dp)
                ) {
                    // Protocol messages at top (when reversed, they appear at bottom)
                    item {
                        ProtocolDisplay(
                            errors = errors,
                            reasoningSteps = reasoningSteps,
                            toolUsages = toolUsages,
                            memoryTraces = memoryTraces,
                            commentaries = commentaries,
                            modifier = Modifier.padding(vertical = 8.dp)
                        )
                    }

                    items(messages.reversed()) { message ->
                        val isLatest = message == messages.lastOrNull()
                        MessageBubble(
                            message = message,
                            toolUsages = toolUsages,
                            isLatestMessage = isLatest,
                            modifier = Modifier.padding(vertical = 4.dp)
                        )
                    }
                }

                // Response controls (stop/regenerate) - shown in voice mode only
                if (inputMode == InputMode.Voice) {
                    ResponseControls(
                        isGenerating = isGenerating,
                        hasMessages = messages.isNotEmpty(),
                        onStop = { viewModel.stopGeneration() },
                        onRegenerate = { viewModel.regenerateResponse() }
                    )
                }

                // Input area - conditional based on mode
                when (inputMode) {
                    InputMode.Voice -> {
                        // Current transcription overlay
                        if (currentTranscription.isNotEmpty()) {
                            TranscriptionOverlay(
                                text = currentTranscription,
                                isFinal = false,
                                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                            )
                        }

                        // Voice state indicator with animation
                        VoiceStateIndicator(
                            state = voiceState,
                            modifier = Modifier.padding(24.dp)
                        )

                        // Manual activation button
                        AssistantButton(
                            state = voiceState,
                            onClick = { viewModel.toggleListening() },
                            onLongClick = { viewModel.startNewConversation() },
                            modifier = Modifier.padding(bottom = 32.dp)
                        )
                    }
                    InputMode.Text -> {
                        // Text input field
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp, vertical = 16.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            OutlinedTextField(
                                value = textInput,
                                onValueChange = { viewModel.updateTextInput(it) },
                                modifier = Modifier.weight(1f),
                                placeholder = { Text("Type a message...") },
                                enabled = !isSendingMessage,
                                maxLines = 4,
                                keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
                                    imeAction = androidx.compose.ui.text.input.ImeAction.Send
                                ),
                                keyboardActions = androidx.compose.foundation.text.KeyboardActions(
                                    onSend = {
                                        if (textInput.isNotBlank()) {
                                            viewModel.sendTextMessage()
                                        }
                                    }
                                )
                            )

                            Spacer(modifier = Modifier.width(8.dp))

                            IconButton(
                                onClick = { viewModel.sendTextMessage() },
                                enabled = !isSendingMessage && textInput.isNotBlank()
                            ) {
                                if (isSendingMessage) {
                                    CircularProgressIndicator(
                                        modifier = Modifier.size(24.dp)
                                    )
                                } else {
                                    Icon(
                                        imageVector = Icons.AutoMirrored.Filled.Send,
                                        contentDescription = "Send"
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    // Auto-scroll to bottom when new messages arrive
    LaunchedEffect(messages.size) {
        if (messages.isNotEmpty()) {
            listState.animateScrollToItem(0)
        }
    }
}
