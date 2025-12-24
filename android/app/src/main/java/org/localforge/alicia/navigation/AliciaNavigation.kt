package org.localforge.alicia.navigation

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import org.localforge.alicia.feature.assistant.AssistantScreen
import org.localforge.alicia.feature.conversations.ConversationsScreen
import org.localforge.alicia.feature.settings.SettingsScreen
import org.localforge.alicia.feature.settings.mcp.MCPSettingsScreen

/**
 * Navigation routes for the Alicia app
 */
sealed class Screen(val route: String) {
    object Assistant : Screen("assistant")
    object Conversations : Screen("conversations")
    object Settings : Screen("settings")
    object MCPSettings : Screen("settings/mcp")
    object ConversationDetail : Screen("conversation/{conversationId}") {
        fun createRoute(conversationId: String) = "conversation/$conversationId"
    }
}

/**
 * Main navigation host for the Alicia app
 */
@Composable
fun AliciaNavigation(
    modifier: Modifier = Modifier,
    navController: NavHostController = rememberNavController(),
    startDestination: String = Screen.Assistant.route
) {
    NavHost(
        navController = navController,
        startDestination = startDestination,
        modifier = modifier
    ) {
        // Main Assistant Screen
        composable(Screen.Assistant.route) {
            AssistantScreen(
                onNavigateToConversations = {
                    navController.navigate(Screen.Conversations.route)
                },
                onNavigateToSettings = {
                    navController.navigate(Screen.Settings.route)
                }
            )
        }

        // Conversations History Screen
        composable(Screen.Conversations.route) {
            ConversationsScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onConversationClick = { conversationId ->
                    navController.navigate(Screen.ConversationDetail.createRoute(conversationId))
                }
            )
        }

        // Settings Screen
        composable(Screen.Settings.route) {
            SettingsScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onNavigateToMCPSettings = {
                    navController.navigate(Screen.MCPSettings.route)
                }
            )
        }

        // MCP Settings Screen
        composable(Screen.MCPSettings.route) {
            MCPSettingsScreen(
                onNavigateBack = {
                    navController.popBackStack()
                }
            )
        }

        // Conversation Detail - Opens existing conversation in AssistantScreen
        composable(Screen.ConversationDetail.route) { backStackEntry ->
            val conversationId = backStackEntry.arguments?.getString("conversationId")

            AssistantScreen(
                conversationId = conversationId,
                onNavigateToConversations = {
                    navController.navigate(Screen.Conversations.route)
                },
                onNavigateToSettings = {
                    navController.navigate(Screen.Settings.route)
                }
            )
        }
    }
}

