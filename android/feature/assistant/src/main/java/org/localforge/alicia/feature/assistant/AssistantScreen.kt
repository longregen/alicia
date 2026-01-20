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
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.unit.dp
import androidx.hilt.lifecycle.viewmodel.compose.hiltViewModel
import org.localforge.alicia.feature.assistant.components.*
import org.localforge.alicia.core.common.ui.AppIcons

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
    onNavigateToSettings: () -> Unit = {},
    onOpenDrawer: () -> Unit = {}
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

    // Branch states for message versioning
    val branchStates by viewModel.branchStates.collectAsState()

    val listState = rememberLazyListState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Alicia") },
                navigationIcon = {
                    IconButton(onClick = onOpenDrawer) {
                        Icon(
                            imageVector = AppIcons.Menu,
                            contentDescription = "Open menu"
                        )
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.toggleInputMode() }) {
                        Icon(
                            imageVector = if (inputMode == InputMode.Voice) {
                                AppIcons.Keyboard
                            } else {
                                AppIcons.Mic
                            },
                            contentDescription = if (inputMode == InputMode.Voice) "Switch to Text" else "Switch to Voice"
                        )
                    }
                    IconButton(onClick = onNavigateToConversations) {
                        Icon(
                            imageVector = AppIcons.History,
                            contentDescription = "Conversations"
                        )
                    }
                    IconButton(onClick = onNavigateToSettings) {
                        Icon(
                            imageVector = AppIcons.Settings,
                            contentDescription = "Settings"
                        )
                    }
                }
            )
        }
    ) { paddingValues ->
        val clipboardManager = LocalClipboardManager.current

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

                        // Note: Siblings are now fetched from server automatically in ViewModel
                        // when messages are loaded, so no initialization is needed here

                        MessageBubble(
                            message = message,
                            toolUsages = toolUsages,
                            isLatestMessage = isLatest,
                            isStreaming = isLatest && isGenerating,
                            branchState = branchStates[message.id],
                            modifier = Modifier.padding(vertical = 4.dp),
                            onEdit = { messageId, newContent ->
                                viewModel.editMessage(messageId, newContent)
                            },
                            onVote = { messageId, isUpvote ->
                                viewModel.voteOnMessage(messageId, isUpvote)
                            },
                            onToolVote = { toolUseId, isUpvote ->
                                viewModel.voteOnToolUse(toolUseId, isUpvote)
                            },
                            onCopy = { content ->
                                clipboardManager.setText(AnnotatedString(content))
                            },
                            onBranchNavigate = { messageId, direction ->
                                viewModel.navigateBranch(messageId, direction)
                            }
                        )
                    }
                }

                // Response controls (stop/regenerate) - shown in both modes
                ResponseControls(
                    isGenerating = isGenerating,
                    hasMessages = messages.isNotEmpty(),
                    onStop = { viewModel.stopGeneration() },
                    onRegenerate = { viewModel.regenerateResponse() }
                )

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
                        // Text input area matching web's InputArea.tsx layout
                        InputArea(
                            textInput = textInput,
                            onTextInputChange = { viewModel.updateTextInput(it) },
                            onSend = { viewModel.sendTextMessage() },
                            onVoiceClick = { viewModel.toggleInputMode() },
                            disabled = isSendingMessage,
                            placeholder = "Type a message...",
                            conversationId = conversationId
                        )
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
