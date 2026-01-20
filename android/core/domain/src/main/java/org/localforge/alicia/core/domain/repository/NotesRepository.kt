package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.Note
import org.localforge.alicia.core.domain.model.NoteCategory

/**
 * Repository interface for notes management.
 * Matches the web frontend's notes API functionality.
 */
interface NotesRepository {
    /**
     * Create a note on a message
     */
    suspend fun createMessageNote(
        messageId: String,
        content: String,
        category: NoteCategory = NoteCategory.GENERAL
    ): Result<Note>

    /**
     * Get all notes for a message
     */
    suspend fun getMessageNotes(messageId: String): Result<List<Note>>

    /**
     * Create a note on a tool use
     */
    suspend fun createToolUseNote(
        toolUseId: String,
        content: String,
        category: NoteCategory = NoteCategory.GENERAL
    ): Result<Note>

    /**
     * Create a note on a reasoning step
     */
    suspend fun createReasoningNote(
        reasoningId: String,
        content: String,
        category: NoteCategory = NoteCategory.GENERAL
    ): Result<Note>

    /**
     * Update an existing note
     */
    suspend fun updateNote(noteId: String, content: String): Result<Note>

    /**
     * Delete a note
     */
    suspend fun deleteNote(noteId: String): Result<Unit>
}
