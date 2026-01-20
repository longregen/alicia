package org.localforge.alicia.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.navigation.NavHostController
import androidx.navigation.compose.rememberNavController
import kotlinx.coroutines.launch
import org.localforge.alicia.core.common.ui.ToastHost
import org.localforge.alicia.navigation.AliciaNavigation
import org.localforge.alicia.navigation.Screen
import org.localforge.alicia.ui.components.AliciaSidebar

/**
 * Main Alicia App composable with drawer navigation.
 *
 * This provides the drawer scaffold that wraps the entire app,
 * matching the web frontend's sidebar navigation pattern.
 */
@Composable
fun AliciaApp(
    viewModel: AliciaAppViewModel = hiltViewModel(),
    navController: NavHostController = rememberNavController()
) {
    val drawerState = rememberDrawerState(DrawerValue.Closed)
    val scope = rememberCoroutineScope()

    val conversations by viewModel.conversations.collectAsState()
    val selectedConversationId by viewModel.selectedConversationId.collectAsState()
    val isConnected by viewModel.isConnected.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()

    // Track current conversation selection
    LaunchedEffect(navController) {
        navController.currentBackStackEntryFlow.collect { backStackEntry ->
            when {
                backStackEntry.destination.route == Screen.ConversationDetail.route -> {
                    val conversationId = backStackEntry.arguments?.getString("conversationId")
                    viewModel.setSelectedConversation(conversationId)
                }
                backStackEntry.destination.route == Screen.Assistant.route -> {
                    viewModel.setSelectedConversation(null)
                }
            }
        }
    }

    ModalNavigationDrawer(
        drawerState = drawerState,
        drawerContent = {
            ModalDrawerSheet {
                AliciaSidebar(
                    conversations = conversations,
                    selectedConversationId = selectedConversationId,
                    isConnected = isConnected,
                    isLoading = isLoading,
                    onNewConversation = {
                        scope.launch {
                            drawerState.close()
                            navController.navigate(Screen.Assistant.route) {
                                popUpTo(Screen.Welcome.route) { inclusive = false }
                            }
                        }
                    },
                    onSelectConversation = { conversationId ->
                        scope.launch {
                            drawerState.close()
                            navController.navigate(Screen.ConversationDetail.createRoute(conversationId)) {
                                popUpTo(Screen.Welcome.route) { inclusive = false }
                            }
                        }
                    },
                    onRenameConversation = { id, title ->
                        viewModel.renameConversation(id, title)
                    },
                    onArchiveConversation = { id ->
                        viewModel.archiveConversation(id)
                    },
                    onUnarchiveConversation = { id ->
                        viewModel.unarchiveConversation(id)
                    },
                    onDeleteConversation = { id ->
                        viewModel.deleteConversation(id)
                        // If we deleted the currently selected conversation, go back to welcome
                        if (id == selectedConversationId) {
                            scope.launch {
                                navController.navigate(Screen.Welcome.route) {
                                    popUpTo(0) { inclusive = true }
                                }
                            }
                        }
                    },
                    onNavigateToMemory = {
                        scope.launch {
                            drawerState.close()
                            navController.navigate(Screen.Memory.route)
                        }
                    },
                    onNavigateToServer = {
                        scope.launch {
                            drawerState.close()
                            navController.navigate(Screen.Server.route)
                        }
                    },
                    onNavigateToSettings = {
                        scope.launch {
                            drawerState.close()
                            navController.navigate(Screen.Settings.route)
                        }
                    },
                    onClose = {
                        scope.launch { drawerState.close() }
                    }
                )
            }
        },
        gesturesEnabled = true
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            AliciaNavigation(
                modifier = Modifier.fillMaxSize(),
                navController = navController,
                onOpenDrawer = {
                    scope.launch { drawerState.open() }
                }
            )

            // Toast notifications overlay
            ToastHost()
        }
    }
}
