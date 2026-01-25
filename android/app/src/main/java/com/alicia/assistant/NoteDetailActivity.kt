package com.alicia.assistant

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Intent
import android.os.Bundle
import android.util.Log
import android.widget.EditText
import android.widget.ImageButton
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.model.VoiceNote
import com.alicia.assistant.service.AliciaApiClient
import com.alicia.assistant.service.VoiceAssistantService
import com.alicia.assistant.storage.NoteRepository
import com.google.android.material.appbar.MaterialToolbar
import kotlinx.coroutines.launch

class NoteDetailActivity : ComponentActivity() {

    private val apiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)
    private lateinit var noteRepository: NoteRepository
    private lateinit var titleEdit: EditText
    private lateinit var contentEdit: EditText
    private var noteId: String? = null
    private var originalNote: VoiceNote? = null

    companion object {
        private const val TAG = "NoteDetailActivity"
        const val EXTRA_NOTE_ID = "note_id"
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_note_detail)

        noteRepository = NoteRepository(this)
        noteId = intent.getStringExtra(EXTRA_NOTE_ID)

        val toolbar = findViewById<MaterialToolbar>(R.id.toolbar)
        toolbar.setNavigationOnClickListener { finish() }

        titleEdit = findViewById(R.id.titleEdit)
        contentEdit = findViewById(R.id.contentEdit)
        val copyButton = findViewById<ImageButton>(R.id.copyButton)
        val shareButton = findViewById<ImageButton>(R.id.shareButton)

        copyButton.setOnClickListener { copyToClipboard() }
        shareButton.setOnClickListener { shareNote() }

        loadNote()
    }

    private fun loadNote() {
        val id = noteId ?: return
        lifecycleScope.launch {
            val note = noteRepository.getNote(id) ?: run {
                finish()
                return@launch
            }
            originalNote = note
            titleEdit.setText(note.title)
            contentEdit.setText(note.content)
        }
    }

    private fun copyToClipboard() {
        val content = contentEdit.text.toString()
        val clipboard = getSystemService(ClipboardManager::class.java)
        clipboard.setPrimaryClip(ClipData.newPlainText("note", content))
        Toast.makeText(this, R.string.copied, Toast.LENGTH_SHORT).show()
    }

    private fun shareNote() {
        val title = titleEdit.text.toString()
        val content = contentEdit.text.toString()
        val shareText = if (title.isNotBlank()) "$title\n\n$content" else content
        val intent = Intent(Intent.ACTION_SEND).apply {
            type = "text/plain"
            putExtra(Intent.EXTRA_TEXT, shareText)
        }
        startActivity(Intent.createChooser(intent, getString(R.string.share)))
    }

    private fun saveIfChanged() {
        val note = originalNote ?: return
        val newTitle = titleEdit.text.toString().trim()
        val newContent = contentEdit.text.toString().trim()

        val effectiveTitle = if (newTitle.isBlank()) {
            if (newContent.length > 50) newContent.substring(0, 50) + "\u2026" else newContent
        } else {
            newTitle
        }

        if (effectiveTitle == note.title && newContent == note.content) return

        val updated = note.copy(title = effectiveTitle, content = newContent)
        lifecycleScope.launch {
            noteRepository.saveNote(updated)
            runCatching { apiClient.updateNote(note.id, effectiveTitle, newContent) }
                .onFailure { Log.w(TAG, "Failed to sync note update to server", it) }
        }
    }

    override fun onResume() {
        super.onResume()
        VoiceAssistantService.ensureRunning(this)
    }

    override fun onPause() {
        super.onPause()
        saveIfChanged()
    }
}
