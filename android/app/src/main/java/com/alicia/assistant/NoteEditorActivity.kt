package com.alicia.assistant

import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.OnBackPressedCallback
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.databinding.ActivityNoteEditorBinding
import com.alicia.assistant.service.AliciaApiClient
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import kotlinx.coroutines.launch
import java.util.UUID

class NoteEditorActivity : ComponentActivity() {

    companion object {
        const val EXTRA_NOTE_ID = "note_id"
    }

    private lateinit var binding: ActivityNoteEditorBinding
    private lateinit var apiClient: AliciaApiClient
    private var noteId: String? = null
    private var isNewNote = true
    private var isSaving = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityNoteEditorBinding.inflate(layoutInflater)
        setContentView(binding.root)

        apiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)

        noteId = intent.getStringExtra(EXTRA_NOTE_ID)
        isNewNote = noteId == null

        if (isNewNote) {
            noteId = UUID.randomUUID().toString()
            binding.toolbar.title = getString(R.string.new_note)
        } else {
            binding.toolbar.title = getString(R.string.edit_note)
            loadNote()
        }

        binding.toolbar.setNavigationOnClickListener {
            saveAndFinish()
        }

        binding.toolbar.setOnMenuItemClickListener { menuItem ->
            when (menuItem.itemId) {
                R.id.action_save -> {
                    saveNote()
                    true
                }
                R.id.action_delete -> {
                    confirmDelete()
                    true
                }
                else -> false
            }
        }

        // Hide delete option for new notes
        if (isNewNote) {
            binding.toolbar.menu.findItem(R.id.action_delete)?.isVisible = false
        }

        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                saveAndFinish()
            }
        })
    }

    private fun loadNote() {
        lifecycleScope.launch {
            try {
                val note = apiClient.getNote(noteId!!)
                binding.titleEditText.setText(note.title)
                binding.contentEditText.setText(note.content)
            } catch (e: Exception) {
                Toast.makeText(this@NoteEditorActivity, R.string.notes_load_failed, Toast.LENGTH_SHORT).show()
                finish()
            }
        }
    }

    private fun saveNote() {
        if (isSaving) return
        val title = binding.titleEditText.text.toString().trim()
        val content = binding.contentEditText.text.toString().trim()

        if (title.isBlank() && content.isBlank()) {
            Toast.makeText(this, R.string.note_empty, Toast.LENGTH_SHORT).show()
            return
        }

        isSaving = true
        lifecycleScope.launch {
            try {
                if (isNewNote) {
                    apiClient.createNote(noteId!!, title, content)
                    isNewNote = false
                    binding.toolbar.menu.findItem(R.id.action_delete)?.isVisible = true
                } else {
                    apiClient.updateNote(noteId!!, title, content)
                }
                Toast.makeText(this@NoteEditorActivity, R.string.note_saved_success, Toast.LENGTH_SHORT).show()
            } catch (e: Exception) {
                Toast.makeText(this@NoteEditorActivity, R.string.note_save_failed, Toast.LENGTH_SHORT).show()
            } finally {
                isSaving = false
            }
        }
    }

    private fun saveAndFinish() {
        val title = binding.titleEditText.text.toString().trim()
        val content = binding.contentEditText.text.toString().trim()

        if (title.isBlank() && content.isBlank()) {
            finish()
            return
        }

        lifecycleScope.launch {
            try {
                if (isNewNote) {
                    apiClient.createNote(noteId!!, title, content)
                } else {
                    apiClient.updateNote(noteId!!, title, content)
                }
            } catch (e: Exception) {
                Toast.makeText(this@NoteEditorActivity, R.string.note_save_failed, Toast.LENGTH_SHORT).show()
            }
            finish()
        }
    }

    private fun confirmDelete() {
        MaterialAlertDialogBuilder(this)
            .setTitle(R.string.delete_note)
            .setMessage(R.string.delete_note_confirm)
            .setPositiveButton(android.R.string.ok) { _, _ ->
                deleteNote()
            }
            .setNegativeButton(android.R.string.cancel, null)
            .show()
    }

    private fun deleteNote() {
        lifecycleScope.launch {
            try {
                apiClient.deleteNote(noteId!!)
                Toast.makeText(this@NoteEditorActivity, R.string.note_deleted, Toast.LENGTH_SHORT).show()
                finish()
            } catch (e: Exception) {
                Toast.makeText(this@NoteEditorActivity, R.string.note_delete_failed, Toast.LENGTH_SHORT).show()
            }
        }
    }

}
