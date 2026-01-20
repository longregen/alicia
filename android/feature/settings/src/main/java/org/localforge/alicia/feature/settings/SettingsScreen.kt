package org.localforge.alicia.feature.settings

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import org.localforge.alicia.core.common.ui.AppIcons
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.lifecycle.viewmodel.compose.hiltViewModel
import org.localforge.alicia.feature.settings.components.*

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    viewModel: SettingsViewModel = hiltViewModel(),
    onNavigateBack: () -> Unit = {},
    onNavigateToMCPSettings: () -> Unit = {},
    onNavigateToOptimizationSettings: () -> Unit = {}
) {
    val settings by viewModel.settings.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Settings") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = AppIcons.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                }
            )
        }
    ) { paddingValues ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues),
            contentPadding = PaddingValues(vertical = 8.dp)
        ) {
            // Activation Settings
            item {
                SettingsSection(title = "Activation") {
                    WakeWordSelector(
                        selectedWakeWord = settings.wakeWord,
                        onWakeWordSelected = { viewModel.setWakeWord(it) }
                    )

                    SliderSetting(
                        title = "Wake Word Sensitivity",
                        subtitle = "Adjust how easily the wake word is detected",
                        value = settings.wakeWordSensitivity,
                        valueRange = 0f..1f,
                        onValueChange = { viewModel.setWakeWordSensitivity(it) }
                    )

                    SwitchSetting(
                        title = "Volume Button Activation",
                        subtitle = "Triple-tap volume up to activate",
                        checked = settings.volumeButtonEnabled,
                        onCheckedChange = { viewModel.setVolumeButtonEnabled(it) }
                    )

                    SwitchSetting(
                        title = "Shake to Activate",
                        subtitle = "Shake your phone to start listening",
                        checked = settings.shakeEnabled,
                        onCheckedChange = { viewModel.setShakeEnabled(it) }
                    )

                    SwitchSetting(
                        title = "Floating Button",
                        subtitle = "Show always-visible activation button",
                        checked = settings.floatingButtonEnabled,
                        onCheckedChange = { viewModel.setFloatingButtonEnabled(it) }
                    )
                }
            }

            // Voice Settings
            item {
                SettingsSection(title = "Voice") {
                    VoiceSelector(
                        voices = settings.availableVoices,
                        selectedVoice = settings.selectedVoice,
                        onVoiceSelected = { viewModel.setVoice(it) }
                    )

                    SliderSetting(
                        title = "Speech Rate",
                        subtitle = "Adjust how fast Alicia speaks",
                        value = settings.speechRate,
                        valueRange = 0.5f..2.0f,
                        onValueChange = { viewModel.setSpeechRate(it) },
                        valueLabel = { String.format(java.util.Locale.US, "%.1fx", it) }
                    )

                    SwitchSetting(
                        title = "Audio Output",
                        subtitle = "Enable voice responses from Alicia",
                        checked = settings.audioOutputEnabled,
                        onCheckedChange = { viewModel.setAudioOutputEnabled(it) }
                    )
                }
            }

            // Response Settings
            item {
                SettingsSection(title = "Response") {
                    ResponseLengthSelector(
                        selectedLength = settings.responseLength,
                        onLengthSelected = { viewModel.setResponseLength(it) }
                    )

                    ButtonSetting(
                        title = "Response Optimization",
                        subtitle = "Configure dimension weights and response style",
                        buttonText = "Configure",
                        onClick = onNavigateToOptimizationSettings
                    )
                }
            }

            // Memory Settings
            item {
                SettingsSection(title = "Memory") {
                    SwitchSetting(
                        title = "Auto-pin important memories",
                        subtitle = "Automatically pin memories marked as important",
                        checked = settings.autoPinMemories,
                        onCheckedChange = { viewModel.setAutoPinMemories(it) }
                    )

                    SwitchSetting(
                        title = "Confirm before deleting",
                        subtitle = "Show confirmation dialog when deleting memories",
                        checked = settings.confirmDeleteMemories,
                        onCheckedChange = { viewModel.setConfirmDeleteMemories(it) }
                    )

                    SwitchSetting(
                        title = "Show relevance scores",
                        subtitle = "Display relevance scores on memory cards",
                        checked = settings.showRelevanceScores,
                        onCheckedChange = { viewModel.setShowRelevanceScores(it) }
                    )
                }
            }

            // Appearance Settings
            item {
                SettingsSection(title = "Appearance") {
                    ThemeSelector(
                        selectedTheme = settings.theme,
                        onThemeSelected = { viewModel.setTheme(it) }
                    )

                    SwitchSetting(
                        title = "Compact mode",
                        subtitle = "Use smaller spacing and fonts",
                        checked = settings.compactMode,
                        onCheckedChange = { viewModel.setCompactMode(it) }
                    )

                    SwitchSetting(
                        title = "Reduce motion",
                        subtitle = "Minimize animations for accessibility",
                        checked = settings.reduceMotion,
                        onCheckedChange = { viewModel.setReduceMotion(it) }
                    )
                }
            }

            // Notifications Settings
            item {
                SettingsSection(title = "Notifications") {
                    SwitchSetting(
                        title = "Sound notifications",
                        subtitle = "Play sounds for notifications",
                        checked = settings.soundNotifications,
                        onCheckedChange = { viewModel.setSoundNotifications(it) }
                    )

                    SwitchSetting(
                        title = "Message previews",
                        subtitle = "Show message content in notifications",
                        checked = settings.messagePreviews,
                        onCheckedChange = { viewModel.setMessagePreviews(it) }
                    )
                }
            }

            // Server Settings
            item {
                SettingsSection(title = "Server") {
                    TextFieldSetting(
                        title = "Alicia Server URL",
                        value = settings.serverUrl,
                        onValueChange = { viewModel.setServerUrl(it) },
                        placeholder = "https://alicia.example.com"
                    )

                    ConnectionStatusSetting(
                        isConnected = settings.isConnected,
                        lastChecked = settings.lastConnectionCheck,
                        onTestConnection = { viewModel.testConnection() }
                    )

                    ButtonSetting(
                        title = "MCP Server Settings",
                        subtitle = "Configure Model Context Protocol servers",
                        buttonText = "Configure",
                        onClick = onNavigateToMCPSettings
                    )
                }
            }

            // Privacy Settings
            item {
                SettingsSection(title = "Privacy") {
                    SwitchSetting(
                        title = "Save Conversation History",
                        subtitle = "Store conversations locally on device",
                        checked = settings.saveHistory,
                        onCheckedChange = { viewModel.setSaveHistory(it) }
                    )

                    ButtonSetting(
                        title = "Clear All History",
                        subtitle = "Delete all saved conversations",
                        buttonText = "Clear",
                        onClick = { viewModel.clearHistory() },
                        destructive = true
                    )
                }
            }

            // About
            item {
                SettingsSection(title = "About") {
                    // TODO: Read version from BuildConfig.VERSION_NAME when build configuration is set up
                    InfoSetting(
                        title = "Version",
                        value = "1.0.0"
                    )

                    // TODO: Read build from BuildConfig.VERSION_CODE or BuildConfig.BUILD_TIME when available
                    InfoSetting(
                        title = "Build",
                        value = "2025.01.01"
                    )
                }
            }
        }
    }
}
