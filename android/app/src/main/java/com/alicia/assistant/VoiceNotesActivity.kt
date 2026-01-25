package com.alicia.assistant

import android.Manifest
import android.content.pm.PackageManager
import android.media.MediaPlayer
import android.os.Bundle
import android.text.SpannableString
import android.text.style.BackgroundColorSpan
import android.util.Log
import android.view.View
import android.widget.ImageButton
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.alicia.assistant.model.RecognitionResult
import com.alicia.assistant.model.VoiceNote
import com.alicia.assistant.service.AliciaApiClient
import com.alicia.assistant.service.SaveNoteResult
import com.alicia.assistant.service.VoiceAssistantService
import com.alicia.assistant.service.VoiceRecognitionManager
import com.alicia.assistant.service.saveRecordedNote
import com.alicia.assistant.storage.NoteRepository
import com.alicia.assistant.telemetry.AliciaTelemetry
import io.opentelemetry.api.common.Attributes
import io.opentelemetry.api.trace.Span
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.color.MaterialColors
import android.content.Intent
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.floatingactionbutton.FloatingActionButton
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.io.File
import java.text.SimpleDateFormat
import java.util.*

class VoiceNotesActivity : ComponentActivity() {

    private val apiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)
    private lateinit var noteRepository: NoteRepository
    private lateinit var voiceRecognitionManager: VoiceRecognitionManager
    private lateinit var notesRecyclerView: RecyclerView
    private lateinit var emptyState: View
    private lateinit var recordingOverlay: View
    private lateinit var recordFab: FloatingActionButton
    private lateinit var stopRecordButton: FloatingActionButton
    private lateinit var partialText: TextView
    private lateinit var adapter: VoiceNoteAdapter
    private var isRecording = false

    private var mediaPlayer: MediaPlayer? = null
    private var playingNoteId: String? = null
    private var highlightJob: Job? = null
    private var playbackFullText: String? = null
    private var recordingSpan: Span? = null
    private var playbackSpan: Span? = null

    companion object {
        private const val REQUEST_RECORD_AUDIO = 2001
        private const val TAG = "VoiceNotesActivity"
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_voice_notes)

        noteRepository = NoteRepository(this)
        voiceRecognitionManager = VoiceRecognitionManager(this, lifecycleScope)

        val toolbar = findViewById<MaterialToolbar>(R.id.toolbar)
        toolbar.setNavigationOnClickListener { finish() }
        toolbar.inflateMenu(R.menu.menu_notes)
        toolbar.setOnMenuItemClickListener { menuItem ->
            when (menuItem.itemId) {
                R.id.action_add_text_note -> {
                    createTextNote()
                    true
                }
                else -> false
            }
        }

        emptyState = findViewById(R.id.emptyState)
        notesRecyclerView = findViewById(R.id.notesRecyclerView)
        notesRecyclerView.layoutManager = LinearLayoutManager(this)
        recordingOverlay = findViewById(R.id.recordingOverlay)
        recordFab = findViewById(R.id.recordFab)
        stopRecordButton = findViewById(R.id.stopRecordButton)
        partialText = findViewById(R.id.partialText)

        adapter = VoiceNoteAdapter(
            onDeleteClick = { note -> confirmDelete(note) },
            onPlayClick = { note -> togglePlayback(note) },
            onItemClick = { note -> openNoteDetail(note) },
            playingNoteId = { playingNoteId }
        )
        notesRecyclerView.adapter = adapter

        recordFab.setOnClickListener { startRecording() }
        stopRecordButton.setOnClickListener { stopRecording() }

        loadNotes()
    }

    private fun togglePlayback(note: VoiceNote) {
        if (playingNoteId == note.id) {
            stopPlayback()
        } else {
            startPlayback(note)
        }
    }

    private fun startPlayback(note: VoiceNote) {
        stopPlayback()

        val audioPath = note.audioPath ?: return
        val file = File(audioPath)
        if (!file.exists()) {
            Toast.makeText(this, "Audio file not found", Toast.LENGTH_SHORT).show()
            return
        }

        playbackSpan = AliciaTelemetry.startSpan("note.playback", Attributes.builder()
            .put("note.id", note.id)
            .build())

        try {
            mediaPlayer = MediaPlayer().apply {
                setDataSource(audioPath)
                prepare()
                start()
                setOnCompletionListener { stopPlayback() }
            }
            val previousNoteId = playingNoteId
            playingNoteId = note.id
            val newPosition = adapter.getPositionForId(note.id)
            if (newPosition != -1) adapter.notifyItemChanged(newPosition)
            if (previousNoteId != null) {
                val oldPosition = adapter.getPositionForId(previousNoteId)
                if (oldPosition != -1) adapter.notifyItemChanged(oldPosition)
            }

            if (note.words.isNotEmpty()) {
                playbackFullText = note.words.joinToString(" ") { it.word }
                highlightJob = lifecycleScope.launch {
                    while (isActive) {
                        val posMs = mediaPlayer?.currentPosition ?: break
                        val posSec = posMs / 1000f
                        updateHighlight(note, posSec)
                        delay(50)
                    }
                }
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to start playback", e)
            Toast.makeText(this, "Playback failed", Toast.LENGTH_SHORT).show()
        }
    }

    private fun stopPlayback() {
        highlightJob?.cancel()
        highlightJob = null
        playbackFullText = null
        mediaPlayer?.release()
        mediaPlayer = null
        playbackSpan?.let { span ->
            span.end()
            playbackSpan = null
        }
        val wasPlaying = playingNoteId
        playingNoteId = null
        if (wasPlaying != null) {
            val oldPosition = adapter.getPositionForId(wasPlaying)
            if (oldPosition != -1) adapter.notifyItemChanged(oldPosition)
        }
    }

    private fun updateHighlight(note: VoiceNote, positionSec: Float) {
        val holder = findViewHolderForNote(note.id) ?: return
        val contentView = holder.itemView.findViewById<TextView>(R.id.noteContent)

        val fullText = playbackFullText ?: return
        val spannable = SpannableString(fullText)
        val highlightColor = MaterialColors.getColor(contentView, com.google.android.material.R.attr.colorPrimaryContainer)

        var offset = 0
        for (word in note.words) {
            val wordStart = offset
            val wordEnd = offset + word.word.length
            if (positionSec >= word.start && positionSec < word.end) {
                spannable.setSpan(
                    BackgroundColorSpan(highlightColor),
                    wordStart, wordEnd,
                    SpannableString.SPAN_EXCLUSIVE_EXCLUSIVE
                )
            }
            offset = wordEnd + 1 // +1 for space
        }

        contentView.text = spannable
    }

    private fun findViewHolderForNote(noteId: String): RecyclerView.ViewHolder? {
        val position = adapter.getPositionForId(noteId)
        if (position == -1) return null
        return notesRecyclerView.findViewHolderForAdapterPosition(position)
    }

    private fun startRecording() {
        if (checkSelfPermission(Manifest.permission.RECORD_AUDIO)
            != PackageManager.PERMISSION_GRANTED) {
            requestPermissions(
                arrayOf(Manifest.permission.RECORD_AUDIO),
                REQUEST_RECORD_AUDIO
            )
            return
        }

        isRecording = true
        recordingSpan = AliciaTelemetry.startSpan("note.record")
        recordFab.visibility = View.GONE
        emptyState.visibility = View.GONE
        notesRecyclerView.visibility = View.GONE
        recordingOverlay.visibility = View.VISIBLE
        partialText.text = getString(R.string.listening)

        voiceRecognitionManager.startListening { result ->
            if (result is RecognitionResult.Error) {
                lifecycleScope.launch {
                    isRecording = false
                    recordingSpan?.let { span ->
                        AliciaTelemetry.addSpanEvent(span, "recording.error")
                        span.end()
                        recordingSpan = null
                    }
                    recordingOverlay.visibility = View.GONE
                    recordFab.visibility = View.VISIBLE
                    stopRecordButton.isEnabled = true
                    Toast.makeText(this@VoiceNotesActivity, "Recording failed", Toast.LENGTH_SHORT).show()
                    loadNotes()
                }
            }
        }
    }

    private fun stopRecording() {
        if (!isRecording) return
        isRecording = false
        partialText.text = getString(R.string.processing)
        stopRecordButton.isEnabled = false

        val tempFile = voiceRecognitionManager.stopAndGetFile()
        if (tempFile == null) {
            recordingSpan?.let { span ->
                AliciaTelemetry.addSpanEvent(span, "recording.failed_no_file")
                span.end()
                recordingSpan = null
            }
            recordingOverlay.visibility = View.GONE
            recordFab.visibility = View.VISIBLE
            stopRecordButton.isEnabled = true
            Toast.makeText(this, "Recording failed", Toast.LENGTH_SHORT).show()
            loadNotes()
            return
        }

        lifecycleScope.launch {
            val notesDir = File(filesDir, "voice_notes")
            val result = saveRecordedNote(tempFile, notesDir, voiceRecognitionManager, noteRepository, apiClient)
            recordingOverlay.visibility = View.GONE
            stopRecordButton.isEnabled = true
            recordFab.visibility = View.VISIBLE

            if (result is SaveNoteResult.NoSpeechDetected) {
                Toast.makeText(this@VoiceNotesActivity, "No speech detected", Toast.LENGTH_SHORT).show()
            }
            recordingSpan?.let { span ->
                span.end()
                recordingSpan = null
            }
            loadNotes()
        }
    }

    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == REQUEST_RECORD_AUDIO) {
            if (grantResults.isNotEmpty() && grantResults[0] == PackageManager.PERMISSION_GRANTED) {
                startRecording()
            } else {
                Toast.makeText(this, "Microphone permission is required to record notes", Toast.LENGTH_SHORT).show()
            }
        }
    }

    private fun loadNotes() {
        lifecycleScope.launch {
            val notes = noteRepository.getNotes()
            if (notes.isEmpty()) {
                emptyState.visibility = View.VISIBLE
                notesRecyclerView.visibility = View.GONE
            } else {
                emptyState.visibility = View.GONE
                notesRecyclerView.visibility = View.VISIBLE
                adapter.submitList(notes)
            }
        }
    }

    private fun openNoteDetail(note: VoiceNote) {
        val intent = Intent(this, NoteDetailActivity::class.java).apply {
            putExtra(NoteDetailActivity.EXTRA_NOTE_ID, note.id)
        }
        startActivity(intent)
    }

    private fun createTextNote() {
        val noteId = UUID.randomUUID().toString()
        val note = VoiceNote(
            id = noteId,
            title = "",
            content = "",
            timestamp = System.currentTimeMillis(),
            duration = 0,
            audioPath = null,
            words = emptyList()
        )
        lifecycleScope.launch {
            noteRepository.saveNote(note)
            runCatching { apiClient.createNote(noteId, "", "") }
                .onFailure { Log.w(TAG, "Failed to sync note to server", it) }
            val intent = Intent(this@VoiceNotesActivity, NoteDetailActivity::class.java).apply {
                putExtra(NoteDetailActivity.EXTRA_NOTE_ID, noteId)
            }
            startActivity(intent)
        }
    }

    private fun confirmDelete(note: VoiceNote) {
        MaterialAlertDialogBuilder(this)
            .setTitle(R.string.delete_note)
            .setMessage(R.string.delete_note_confirm)
            .setNegativeButton(android.R.string.cancel, null)
            .setPositiveButton(android.R.string.ok) { _, _ ->
                lifecycleScope.launch {
                    AliciaTelemetry.withSpan("note.delete", Attributes.builder()
                        .put("note.id", note.id)
                        .build()
                    ) {
                        if (playingNoteId == note.id) stopPlayback()
                        note.audioPath?.let { File(it).delete() }
                        noteRepository.deleteNote(note.id)
                        runCatching { apiClient.deleteNote(note.id) }
                            .onFailure { Log.w(TAG, "Failed to delete note from server", it) }
                        loadNotes()
                    }
                }
            }
            .show()
    }

    override fun onResume() {
        super.onResume()
        VoiceAssistantService.ensureRunning(this)
        loadNotes()
    }

    override fun onPause() {
        super.onPause()
        stopPlayback()
    }

    override fun onDestroy() {
        super.onDestroy()
        stopPlayback()
        voiceRecognitionManager.destroy()
    }

    private class VoiceNoteAdapter(
        private val onDeleteClick: (VoiceNote) -> Unit,
        private val onPlayClick: (VoiceNote) -> Unit,
        private val onItemClick: (VoiceNote) -> Unit,
        private val playingNoteId: () -> String?
    ) : RecyclerView.Adapter<VoiceNoteAdapter.ViewHolder>() {

        private var notes: List<VoiceNote> = emptyList()
        private val dateFormat = SimpleDateFormat("MMM d, yyyy 'at' h:mm a", Locale.getDefault())

        fun submitList(newNotes: List<VoiceNote>) {
            notes = newNotes
            notifyDataSetChanged()
        }

        fun getPositionForId(noteId: String): Int = notes.indexOfFirst { it.id == noteId }

        override fun onCreateViewHolder(parent: android.view.ViewGroup, viewType: Int): ViewHolder {
            val view = android.view.LayoutInflater.from(parent.context)
                .inflate(R.layout.item_voice_note, parent, false)
            return ViewHolder(view)
        }

        override fun onBindViewHolder(holder: ViewHolder, position: Int) {
            holder.bind(notes[position])
        }

        override fun getItemCount() = notes.size

        inner class ViewHolder(itemView: View) : RecyclerView.ViewHolder(itemView) {
            private val titleText: TextView = itemView.findViewById(R.id.noteTitle)
            private val contentText: TextView = itemView.findViewById(R.id.noteContent)
            private val timestampText: TextView = itemView.findViewById(R.id.noteTimestamp)
            private val playButton: ImageButton = itemView.findViewById(R.id.playButton)
            private val deleteButton: ImageButton = itemView.findViewById(R.id.deleteButton)

            fun bind(note: VoiceNote) {
                titleText.text = note.title
                contentText.text = note.content
                timestampText.text = dateFormat.format(Date(note.timestamp))
                itemView.setOnClickListener { onItemClick(note) }
                deleteButton.setOnClickListener { onDeleteClick(note) }

                val hasAudio = note.audioPath != null && File(note.audioPath).exists()
                playButton.visibility = if (hasAudio) View.VISIBLE else View.GONE

                val isPlaying = playingNoteId() == note.id
                playButton.setImageResource(if (isPlaying) R.drawable.ic_stop else R.drawable.ic_play)
                contentText.maxLines = if (isPlaying) Int.MAX_VALUE else 3

                playButton.setOnClickListener { onPlayClick(note) }
            }
        }
    }
}
