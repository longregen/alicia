package org.localforge.alicia

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.ui.Modifier
import dagger.hilt.android.AndroidEntryPoint
import org.localforge.alicia.navigation.AliciaNavigation
import org.localforge.alicia.ui.theme.AliciaTheme

/**
 * Main Activity for Alicia Android App
 *
 * Entry point activity that hosts the main Compose UI.
 * Configured with Hilt for dependency injection.
 */
@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        setContent {
            AliciaTheme {
                AliciaNavigation(
                    modifier = Modifier.fillMaxSize()
                )
            }
        }
    }
}
