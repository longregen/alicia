package org.localforge.alicia.feature.memory

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import org.localforge.alicia.core.domain.model.Memory
import org.localforge.alicia.core.domain.model.MemoryCategory
import org.localforge.alicia.core.domain.repository.MemoryRepository
import javax.inject.Inject

/**
 * ViewModel for the MemoryScreen.
 *
 * Manages:
 * - Memory list with filtering and search
 * - Memory CRUD operations
 * - Editor state
 */
@HiltViewModel
class MemoryViewModel @Inject constructor(
    private val memoryRepository: MemoryRepository
) : ViewModel() {

    private val _memories = MutableStateFlow<List<Memory>>(emptyList())

    private val _searchQuery = MutableStateFlow("")
    val searchQuery: StateFlow<String> = _searchQuery.asStateFlow()

    private val _selectedCategory = MutableStateFlow<MemoryCategory?>(null)
    val selectedCategory: StateFlow<MemoryCategory?> = _selectedCategory.asStateFlow()

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    private val _errorMessage = MutableStateFlow<String?>(null)
    val errorMessage: StateFlow<String?> = _errorMessage.asStateFlow()

    private val _editingMemory = MutableStateFlow<Memory?>(null)
    val editingMemory: StateFlow<Memory?> = _editingMemory.asStateFlow()

    // Expose memories for detail screen access
    val memories: StateFlow<List<Memory>> = _memories.asStateFlow()

    private val _isEditorOpen = MutableStateFlow(false)
    val isEditorOpen: StateFlow<Boolean> = _isEditorOpen.asStateFlow()

    // Combined filtered memories
    val filteredMemories: StateFlow<List<Memory>> = combine(
        _memories,
        _searchQuery,
        _selectedCategory
    ) { memories, query, category ->
        memories
            .filter { memory ->
                // Filter by category
                (category == null || memory.category == category) &&
                // Filter by search query
                (query.isEmpty() || memory.content.contains(query, ignoreCase = true))
            }
            .sortedWith(
                compareByDescending<Memory> { it.pinned }
                    .thenByDescending { it.updatedAt }
            )
    }.stateIn(
        scope = viewModelScope,
        started = SharingStarted.WhileSubscribed(5000),
        initialValue = emptyList()
    )

    init {
        loadMemories()
    }

    private fun loadMemories() {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                memoryRepository.getAllMemories()
                    .collect { memories ->
                        _memories.value = memories
                        _isLoading.value = false
                    }
            } catch (e: Exception) {
                _isLoading.value = false
                // Handle error
            }
        }
    }

    fun setSearchQuery(query: String) {
        _searchQuery.value = query
    }

    fun setSelectedCategory(category: MemoryCategory?) {
        _selectedCategory.value = category
    }

    fun openEditor(memory: Memory?) {
        _editingMemory.value = memory
        _isEditorOpen.value = true
    }

    fun closeEditor() {
        _isEditorOpen.value = false
        _editingMemory.value = null
    }

    fun saveMemory(content: String, category: MemoryCategory) {
        viewModelScope.launch {
            try {
                val editing = _editingMemory.value
                if (editing != null) {
                    memoryRepository.updateMemory(editing.id, content, category)
                } else {
                    memoryRepository.createMemory(content, category)
                }
                closeEditor()
            } catch (e: Exception) {
                // Handle error
            }
        }
    }

    fun togglePin(memoryId: String) {
        viewModelScope.launch {
            try {
                val memory = _memories.value.find { it.id == memoryId }
                if (memory != null) {
                    memoryRepository.pinMemory(memoryId, !memory.pinned)
                }
            } catch (e: Exception) {
                // Handle error
            }
        }
    }

    fun archiveMemory(memoryId: String) {
        viewModelScope.launch {
            try {
                memoryRepository.archiveMemory(memoryId)
            } catch (e: Exception) {
                // Handle error
            }
        }
    }

    fun deleteMemory(memoryId: String) {
        viewModelScope.launch {
            try {
                memoryRepository.deleteMemory(memoryId)
            } catch (e: Exception) {
                _errorMessage.value = e.message ?: "Failed to delete memory"
            }
        }
    }

    /**
     * Load a specific memory by ID (for detail screen)
     */
    fun loadMemory(memoryId: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _errorMessage.value = null
            try {
                val memory = memoryRepository.getMemory(memoryId)
                if (memory != null) {
                    // Add to memories list if not present
                    _memories.update { current ->
                        if (current.none { it.id == memoryId }) {
                            current + memory
                        } else {
                            current.map { if (it.id == memoryId) memory else it }
                        }
                    }
                } else {
                    _errorMessage.value = "Memory not found"
                }
            } catch (e: Exception) {
                _errorMessage.value = e.message ?: "Failed to load memory"
            } finally {
                _isLoading.value = false
            }
        }
    }

    /**
     * Toggle pin status for a memory
     */
    fun togglePinMemory(memoryId: String) {
        viewModelScope.launch {
            try {
                val memory = _memories.value.find { it.id == memoryId }
                if (memory != null) {
                    memoryRepository.pinMemory(memoryId, !memory.pinned)
                }
            } catch (e: Exception) {
                _errorMessage.value = e.message ?: "Failed to update memory"
            }
        }
    }

    /**
     * Clear any error message
     */
    fun clearError() {
        _errorMessage.value = null
    }

    // ========== Tag Operations (matching web frontend) ==========

    /**
     * Add tags to a memory
     */
    fun addTags(memoryId: String, tags: List<String>) {
        viewModelScope.launch {
            try {
                memoryRepository.addTags(memoryId, tags)
                    .onSuccess { updatedMemory ->
                        _memories.update { current ->
                            current.map { if (it.id == memoryId) updatedMemory else it }
                        }
                    }
                    .onFailure { e ->
                        _errorMessage.value = e.message ?: "Failed to add tags"
                    }
            } catch (e: Exception) {
                _errorMessage.value = e.message ?: "Failed to add tags"
            }
        }
    }

    /**
     * Remove a tag from a memory
     */
    fun removeTag(memoryId: String, tag: String) {
        viewModelScope.launch {
            try {
                memoryRepository.removeTag(memoryId, tag)
                    .onSuccess { updatedMemory ->
                        _memories.update { current ->
                            current.map { if (it.id == memoryId) updatedMemory else it }
                        }
                    }
                    .onFailure { e ->
                        _errorMessage.value = e.message ?: "Failed to remove tag"
                    }
            } catch (e: Exception) {
                _errorMessage.value = e.message ?: "Failed to remove tag"
            }
        }
    }

    // ========== Importance Operations (matching web frontend) ==========

    /**
     * Set memory importance (0.0 - 1.0)
     */
    fun setImportance(memoryId: String, importance: Double) {
        viewModelScope.launch {
            try {
                memoryRepository.setImportance(memoryId, importance)
                    .onSuccess { updatedMemory ->
                        _memories.update { current ->
                            current.map { if (it.id == memoryId) updatedMemory else it }
                        }
                    }
                    .onFailure { e ->
                        _errorMessage.value = e.message ?: "Failed to set importance"
                    }
            } catch (e: Exception) {
                _errorMessage.value = e.message ?: "Failed to set importance"
            }
        }
    }

    // ========== Server-side Search (matching web frontend) ==========

    /**
     * Search memories on server with semantic search
     */
    fun searchOnServer(query: String, limit: Int = 10) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                memoryRepository.searchMemoriesOnServer(query, limit)
                    .onSuccess { searchResults ->
                        _memories.value = searchResults
                        _isLoading.value = false
                    }
                    .onFailure { e ->
                        _errorMessage.value = e.message ?: "Search failed"
                        _isLoading.value = false
                    }
            } catch (e: Exception) {
                _errorMessage.value = e.message ?: "Search failed"
                _isLoading.value = false
            }
        }
    }
}
