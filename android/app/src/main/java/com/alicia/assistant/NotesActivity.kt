package com.alicia.assistant

import android.content.Intent
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.DiffUtil
import androidx.recyclerview.widget.ItemTouchHelper
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.ListAdapter
import androidx.recyclerview.widget.RecyclerView
import com.alicia.assistant.databinding.ActivityNotesBinding
import com.alicia.assistant.service.AliciaApiClient
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Locale
import java.util.TimeZone

class NotesActivity : ComponentActivity() {

    private lateinit var binding: ActivityNotesBinding
    private lateinit var apiClient: AliciaApiClient
    private val adapter = NoteAdapter { note ->
        openNoteEditor(note.id)
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityNotesBinding.inflate(layoutInflater)
        setContentView(binding.root)

        apiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)

        binding.toolbar.setNavigationOnClickListener { finish() }

        binding.notesRecyclerView.layoutManager = LinearLayoutManager(this)
        binding.notesRecyclerView.adapter = adapter

        // Swipe to delete
        val swipeHandler = object : ItemTouchHelper.SimpleCallback(0, ItemTouchHelper.LEFT or ItemTouchHelper.RIGHT) {
            override fun onMove(rv: RecyclerView, vh: RecyclerView.ViewHolder, target: RecyclerView.ViewHolder) = false

            override fun onSwiped(viewHolder: RecyclerView.ViewHolder, direction: Int) {
                val position = viewHolder.adapterPosition
                val note = adapter.currentList[position]
                confirmDelete(note, position)
            }
        }
        ItemTouchHelper(swipeHandler).attachToRecyclerView(binding.notesRecyclerView)

        binding.newNoteFab.setOnClickListener {
            openNoteEditor(null)
        }

        loadNotes()
    }

    override fun onResume() {
        super.onResume()
        loadNotes()
    }

    private fun loadNotes() {
        lifecycleScope.launch {
            try {
                val notes = apiClient.listNotes()
                binding.loadingState.visibility = View.GONE

                if (notes.isEmpty()) {
                    binding.emptyState.visibility = View.VISIBLE
                    binding.notesRecyclerView.visibility = View.GONE
                } else {
                    binding.emptyState.visibility = View.GONE
                    binding.notesRecyclerView.visibility = View.VISIBLE
                    adapter.submitList(notes)
                }
            } catch (e: Exception) {
                binding.loadingState.visibility = View.GONE
                binding.emptyState.visibility = View.VISIBLE
                Toast.makeText(this@NotesActivity, R.string.notes_load_failed, Toast.LENGTH_SHORT).show()
            }
        }
    }

    private fun confirmDelete(note: AliciaApiClient.Note, position: Int) {
        MaterialAlertDialogBuilder(this)
            .setTitle(R.string.delete_note)
            .setMessage(R.string.delete_note_confirm)
            .setPositiveButton(android.R.string.ok) { _, _ ->
                deleteNote(note)
            }
            .setNegativeButton(android.R.string.cancel) { _, _ ->
                // Restore the item by resubmitting the list
                adapter.notifyItemChanged(position)
            }
            .setOnCancelListener {
                adapter.notifyItemChanged(position)
            }
            .show()
    }

    private fun deleteNote(note: AliciaApiClient.Note) {
        lifecycleScope.launch {
            try {
                apiClient.deleteNote(note.id)
                loadNotes()
                Toast.makeText(this@NotesActivity, R.string.note_deleted, Toast.LENGTH_SHORT).show()
            } catch (e: Exception) {
                Toast.makeText(this@NotesActivity, R.string.note_delete_failed, Toast.LENGTH_SHORT).show()
                loadNotes()
            }
        }
    }

    private fun openNoteEditor(noteId: String?) {
        val intent = Intent(this, NoteEditorActivity::class.java)
        if (noteId != null) {
            intent.putExtra(NoteEditorActivity.EXTRA_NOTE_ID, noteId)
        }
        startActivity(intent)
    }

    private class NoteAdapter(
        private val onClick: (AliciaApiClient.Note) -> Unit
    ) : ListAdapter<AliciaApiClient.Note, NoteAdapter.ViewHolder>(DiffCallback) {

        private val dateParser = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }
        private val dateFormatter = SimpleDateFormat("MMM d, h:mm a", Locale.getDefault())

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
            val view = LayoutInflater.from(parent.context)
                .inflate(R.layout.item_note, parent, false)
            return ViewHolder(view)
        }

        override fun onBindViewHolder(holder: ViewHolder, position: Int) {
            holder.bind(getItem(position))
        }

        inner class ViewHolder(view: View) : RecyclerView.ViewHolder(view) {
            private val titleText: TextView = view.findViewById(R.id.noteTitle)
            private val previewText: TextView = view.findViewById(R.id.notePreview)
            private val dateText: TextView = view.findViewById(R.id.noteDate)

            fun bind(note: AliciaApiClient.Note) {
                titleText.text = note.title.ifBlank { "Untitled" }
                previewText.text = note.content.take(100)
                previewText.visibility = if (note.content.isBlank()) View.GONE else View.VISIBLE
                dateText.text = formatDate(note.updatedAt)
                itemView.setOnClickListener { onClick(note) }
            }

            private fun formatDate(isoDate: String): String {
                if (isoDate.isBlank()) return ""
                return try {
                    val date = dateParser.parse(isoDate) ?: return isoDate
                    dateFormatter.format(date)
                } catch (e: Exception) {
                    isoDate.take(10)
                }
            }
        }

        companion object DiffCallback : DiffUtil.ItemCallback<AliciaApiClient.Note>() {
            override fun areItemsTheSame(
                oldItem: AliciaApiClient.Note,
                newItem: AliciaApiClient.Note
            ): Boolean = oldItem.id == newItem.id

            override fun areContentsTheSame(
                oldItem: AliciaApiClient.Note,
                newItem: AliciaApiClient.Note
            ): Boolean = oldItem == newItem
        }
    }
}
