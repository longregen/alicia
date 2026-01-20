package org.localforge.alicia.feature.server

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import org.localforge.alicia.core.domain.model.ServerInfo
import org.localforge.alicia.core.domain.repository.ServerRepository
import javax.inject.Inject

/**
 * ViewModel for the ServerScreen.
 *
 * Manages:
 * - Server connection status
 * - Model information
 * - MCP server statuses
 * - Session statistics
 */
@HiltViewModel
class ServerViewModel @Inject constructor(
    private val serverRepository: ServerRepository
) : ViewModel() {

    private val _serverInfo = MutableStateFlow(ServerInfo())
    val serverInfo: StateFlow<ServerInfo> = _serverInfo.asStateFlow()

    private val _isLoading = MutableStateFlow(true)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error.asStateFlow()

    init {
        loadServerInfo()
    }

    private fun loadServerInfo() {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                serverRepository.getServerInfo()
                    .collect { info ->
                        _serverInfo.value = info
                        _isLoading.value = false
                    }
            } catch (e: Exception) {
                _error.value = e.message ?: "Failed to fetch server info"
                _isLoading.value = false
            }
        }
    }

    fun refresh() {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                serverRepository.refreshServerInfo()
            } catch (e: Exception) {
                _error.value = e.message ?: "Failed to refresh server info"
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun clearError() {
        _error.value = null
    }
}
