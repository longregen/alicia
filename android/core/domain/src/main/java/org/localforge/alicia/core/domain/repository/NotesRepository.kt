package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.Note
import org.localforge.alicia.core.domain.model.NoteCategory

interface NotesRepository {
    suspend fun createMessageNote(
        messageId: String,
        content: String,
        category: NoteCategory = NoteCategory.GENERAL
    ): Result<Note>

    suspend fun getMessageNotes(messageId: String): Result<List<Note>>

    suspend fun createToolUseNote(
        toolUseId: String,
        content: String,
        category: NoteCategory = NoteCategory.GENERAL
    ): Result<Note>

    suspend fun createReasoningNote(
        reasoningId: String,
        content: String,
        category: NoteCategory = NoteCategory.GENERAL
    ): Result<Note>

    suspend fun updateNote(noteId: String, content: String): Result<Note>

    suspend fun deleteNote(noteId: String): Result<Unit>
}
