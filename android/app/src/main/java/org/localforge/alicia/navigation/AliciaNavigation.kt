package org.localforge.alicia.navigation

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import org.localforge.alicia.feature.assistant.AssistantScreen
import org.localforge.alicia.feature.conversations.ConversationsScreen
import org.localforge.alicia.feature.memory.MemoryScreen
import org.localforge.alicia.feature.memory.MemoryDetailScreen
import org.localforge.alicia.feature.server.ServerScreen
import org.localforge.alicia.feature.settings.SettingsScreen
import org.localforge.alicia.feature.settings.mcp.MCPSettingsScreen
import org.localforge.alicia.feature.welcome.WelcomeScreen

/**
 * Navigation routes for the Alicia app
 */
sealed class Screen(val route: String) {
    object Welcome : Screen("welcome")
    object Assistant : Screen("assistant")
    object Conversations : Screen("conversations")
    object Settings : Screen("settings")
    object MCPSettings : Screen("settings/mcp")
    object Memory : Screen("memory")
    object MemoryDetail : Screen("memory/{memoryId}") {
        fun createRoute(memoryId: String) = "memory/$memoryId"
    }
    object Server : Screen("server")
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
    startDestination: String = Screen.Welcome.route,
    onOpenDrawer: () -> Unit = {}
) {
    NavHost(
        navController = navController,
        startDestination = startDestination,
        modifier = modifier
    ) {
        // Welcome Screen - Home/Landing page
        composable(Screen.Welcome.route) {
            WelcomeScreen(
                onNewConversation = {
                    navController.navigate(Screen.Assistant.route)
                },
                onSelectConversation = { conversationId ->
                    navController.navigate(Screen.ConversationDetail.createRoute(conversationId))
                },
                onOpenDrawer = onOpenDrawer
            )
        }

        // Main Assistant Screen
        composable(Screen.Assistant.route) {
            AssistantScreen(
                onNavigateToConversations = {
                    navController.navigate(Screen.Conversations.route)
                },
                onNavigateToSettings = {
                    navController.navigate(Screen.Settings.route)
                },
                onOpenDrawer = onOpenDrawer
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
                },
                onOpenDrawer = onOpenDrawer
            )
        }

        // Memory Screen
        composable(Screen.Memory.route) {
            MemoryScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onMemoryClick = { memoryId ->
                    navController.navigate(Screen.MemoryDetail.createRoute(memoryId))
                }
            )
        }

        // Memory Detail Screen
        composable(Screen.MemoryDetail.route) { backStackEntry ->
            val memoryId = backStackEntry.arguments?.getString("memoryId") ?: ""
            MemoryDetailScreen(
                memoryId = memoryId,
                onNavigateBack = {
                    navController.popBackStack()
                }
            )
        }

        // Server Info Screen
        composable(Screen.Server.route) {
            ServerScreen(
                onNavigateBack = {
                    navController.popBackStack()
                }
            )
        }
    }
}

