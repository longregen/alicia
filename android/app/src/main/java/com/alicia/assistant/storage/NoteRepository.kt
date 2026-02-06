package com.alicia.assistant.storage

import android.content.Context
import android.util.Log
import com.alicia.assistant.model.VoiceNote
import com.google.gson.Gson
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import java.io.File

class NoteRepository(private val context: Context) {
    private val gson = Gson()
    private val metaDir: File by lazy {
        File(context.filesDir, "voice_notes_meta").also { it.mkdirs() }
    }

    companion object {
        private const val TAG = "NoteRepository"
        // Shared across instances to serialize filesystem access
        private val mutex = Mutex()
    }

    suspend fun saveNote(note: VoiceNote) = withContext(Dispatchers.IO) {
        mutex.withLock {
            val file = File(metaDir, "${note.id}.json")
            file.writeText(gson.toJson(note))
        }
    }

    suspend fun getNotes(): List<VoiceNote> = withContext(Dispatchers.IO) {
        mutex.withLock {
            metaDir.listFiles { f -> f.extension == "json" }
                ?.mapNotNull { file ->
                    try { gson.fromJson(file.readText(), VoiceNote::class.java) }
                    catch (e: Exception) {
                        Log.w(TAG, "Failed to parse note: ${file.name}", e)
                        null
                    }
                }
                ?.sortedByDescending { it.timestamp }
                ?: emptyList()
        }
    }

    suspend fun getNote(noteId: String): VoiceNote? = withContext(Dispatchers.IO) {
        mutex.withLock {
            val file = File(metaDir, "$noteId.json")
            if (file.exists()) {
                try { gson.fromJson(file.readText(), VoiceNote::class.java) }
                catch (e: Exception) {
                    Log.w(TAG, "Failed to parse note: ${file.name}", e)
                    null
                }
            } else null
        }
    }

    suspend fun deleteNote(noteId: String) = withContext(Dispatchers.IO) {
        mutex.withLock {
            File(metaDir, "$noteId.json").delete()
        }
    }

    suspend fun clearNotes() = withContext(Dispatchers.IO) {
        mutex.withLock {
            val notes = metaDir.listFiles { f -> f.extension == "json" }
                ?.mapNotNull { file ->
                    try { gson.fromJson(file.readText(), VoiceNote::class.java) }
                    catch (e: Exception) {
                        Log.w(TAG, "Failed to parse note: ${file.name}", e)
                        null
                    }
                } ?: emptyList()
            notes.forEach { note ->
                note.audioPath?.let { File(it).delete() }
            }
            metaDir.listFiles()?.forEach { it.delete() }
        }
    }

    suspend fun migrateFromPreferences(preferencesManager: PreferencesManager) =
        withContext(Dispatchers.IO) {
            mutex.withLock {
                if (metaDir.listFiles()?.any { it.extension == "json" } == true) return@withContext
                val oldNotes = preferencesManager.getLegacyNotes()
                for (note in oldNotes) {
                    val file = File(metaDir, "${note.id}.json")
                    file.writeText(gson.toJson(note))
                }
                if (oldNotes.isNotEmpty()) {
                    preferencesManager.clearLegacyNotes()
                }
            }
        }
}
