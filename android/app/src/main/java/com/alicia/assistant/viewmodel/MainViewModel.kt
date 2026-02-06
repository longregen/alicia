package com.alicia.assistant.viewmodel

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.alicia.assistant.model.AppSettings
import com.alicia.assistant.storage.NoteRepository
import com.alicia.assistant.storage.PreferencesManager
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class MainViewModel(application: Application) : AndroidViewModel(application) {
    private val preferencesManager = PreferencesManager(application)
    val noteRepository = NoteRepository(application)

    private val _settings = MutableStateFlow(AppSettings())
    val settings: StateFlow<AppSettings> = _settings.asStateFlow()

    init {
        refreshSettings()
    }

    fun refreshSettings() {
        viewModelScope.launch {
            _settings.value = preferencesManager.getSettings()
        }
    }
}
