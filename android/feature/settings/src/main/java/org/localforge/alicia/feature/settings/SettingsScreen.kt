package org.localforge.alicia.feature.settings

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import org.localforge.alicia.feature.settings.components.*

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    viewModel: SettingsViewModel = hiltViewModel(),
    onNavigateBack: () -> Unit = {},
    onNavigateToMCPSettings: () -> Unit = {}
) {
    val settings by viewModel.settings.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Settings") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
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
