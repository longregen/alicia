package org.localforge.alicia.feature.settings.mcp

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import org.localforge.alicia.core.common.Logger
import org.localforge.alicia.core.domain.model.MCPServer
import org.localforge.alicia.core.domain.model.MCPServerConfig
import org.localforge.alicia.core.domain.model.MCPTool
import org.localforge.alicia.core.domain.repository.MCPRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * UI state for MCP settings screen.
 */
data class MCPSettingsUiState(
    val servers: List<MCPServer> = emptyList(),
    val tools: List<MCPTool> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
    val successMessage: String? = null
)

/**
 * ViewModel for MCP settings screen.
 */
@HiltViewModel
class MCPSettingsViewModel @Inject constructor(
    private val mcpRepository: MCPRepository
) : ViewModel() {

    private val logger = Logger.forClass(MCPSettingsViewModel::class.java)
    private val _uiState = MutableStateFlow(MCPSettingsUiState())
    val uiState: StateFlow<MCPSettingsUiState> = _uiState.asStateFlow()

    init {
        loadServers()
        loadTools()
    }

    /**
     * Load all MCP servers from the repository.
     */
    fun loadServers() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            mcpRepository.getServers()
                .onSuccess { servers ->
                    _uiState.update {
                        it.copy(
                            servers = servers,
                            isLoading = false,
                            error = null
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = error.message ?: "Failed to load servers"
                        )
                    }
                }
        }
    }

    /**
     * Load all MCP tools from the repository.
     */
    private fun loadTools() {
        viewModelScope.launch {
            mcpRepository.getTools()
                .onSuccess { tools ->
                    _uiState.update { it.copy(tools = tools) }
                }
                .onFailure { error ->
                    // Tools loading failure is non-critical, just log it
                    logger.w("Failed to load MCP tools: ${error.message}", error)
                }
        }
    }

    /**
     * Add a new MCP server.
     */
    fun addServer(config: MCPServerConfig) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            mcpRepository.addServer(config)
                .onSuccess { newServer ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            successMessage = "Server '${newServer.name}' added successfully"
                        )
                    }
                    // Reload servers and tools to get updated state
                    loadServers()
                    loadTools()
                }
                .onFailure { error ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = error.message ?: "Failed to add server"
                        )
                    }
                }
        }
    }

    /**
     * Delete an MCP server by name.
     */
    fun deleteServer(name: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(error = null) }

            mcpRepository.deleteServer(name)
                .onSuccess {
                    // Reload servers and tools to get updated state
                    loadServers()
                    loadTools()

                    _uiState.update {
                        it.copy(
                            successMessage = "Server '$name' removed successfully"
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update {
                        it.copy(
                            error = error.message ?: "Failed to delete server"
                        )
                    }
                }
        }
    }

    /**
     * Clear success message after it has been shown.
     */
    fun clearSuccessMessage() {
        _uiState.update { it.copy(successMessage = null) }
    }
}
