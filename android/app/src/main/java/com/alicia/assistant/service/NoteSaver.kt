package com.alicia.assistant.service

import android.util.Log
import com.alicia.assistant.model.VoiceNote
import com.alicia.assistant.storage.NoteRepository
import java.io.File
import java.util.UUID

private const val TAG = "NoteSaver"

sealed class SaveNoteResult {
    data class Success(val note: VoiceNote) : SaveNoteResult()
    data object NoSpeechDetected : SaveNoteResult()
}

suspend fun saveRecordedNote(
    tempFile: File,
    notesDir: File,
    voiceRecognitionManager: VoiceRecognitionManager,
    noteRepository: NoteRepository,
    apiClient: AliciaApiClient
): SaveNoteResult {
    val noteId = UUID.randomUUID().toString()
    notesDir.mkdirs()
    val permanentFile = File(notesDir, "$noteId.m4a")
    tempFile.copyTo(permanentFile, overwrite = true)
    tempFile.delete()

    val transcription = voiceRecognitionManager.transcribeVerbose(permanentFile)
    val text = transcription?.text ?: ""

    if (text.isBlank()) {
        permanentFile.delete()
        return SaveNoteResult.NoSpeechDetected
    }

    val title = if (text.length > 50) text.substring(0, 50) + "\u2026" else text
    val note = VoiceNote(
        id = noteId,
        title = title,
        content = text,
        timestamp = System.currentTimeMillis(),
        duration = transcription?.durationMs ?: 0,
        audioPath = permanentFile.absolutePath,
        words = transcription?.words ?: emptyList()
    )
    noteRepository.saveNote(note)
    runCatching { apiClient.createNote(noteId, title, text) }
        .onFailure { Log.w(TAG, "Failed to sync note to server", it) }
    return SaveNoteResult.Success(note)
}
