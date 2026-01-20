package org.localforge.alicia

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import dagger.hilt.android.AndroidEntryPoint
import org.localforge.alicia.core.domain.repository.SettingsRepository
import org.localforge.alicia.ui.AliciaApp
import org.localforge.alicia.ui.theme.AliciaTheme
import javax.inject.Inject

/**
 * Main Activity for Alicia Android App
 *
 * Entry point activity that hosts the main Compose UI.
 * Configured with Hilt for dependency injection.
 *
 * Uses AliciaApp which provides:
 * - Drawer-based sidebar navigation (matching web frontend)
 * - Conversation management (create, rename, archive, delete)
 * - Navigation to Memory, Server, and Settings
 *
 * Theme preference is read from DataStore and applied to AliciaTheme,
 * matching the web frontend's localStorage-based theme persistence.
 */
@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject
    lateinit var settingsRepository: SettingsRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        setContent {
            // Collect theme preference from DataStore
            val themePreference by settingsRepository.theme.collectAsState(initial = "system")
            val systemDarkTheme = isSystemInDarkTheme()

            // Determine if dark theme should be used based on preference
            val useDarkTheme = when (themePreference) {
                "light" -> false
                "dark" -> true
                else -> systemDarkTheme // "system" or default
            }

            AliciaTheme(darkTheme = useDarkTheme) {
                AliciaApp()
            }
        }
    }
}
