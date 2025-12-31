import { useCallback, useEffect, useState, useRef } from 'react';
import {
  useNotesStore,
  type NoteCategory,
} from '../stores/notesStore';
import { api, type NoteListResponse } from '../services/api';

/**
 * Hook for note operations on a specific target.
 * Wraps the notesStore with API integration for persisting notes.
 *
 * @param targetType - Type of target element (message, tool_use, reasoning)
 * @param targetId - Unique ID of the target element
 * @returns Object with note state, handlers, and notes list
 *
 * @example
 * ```tsx
 * function MessageNotes({ messageId }) {
 *   const { notes, addNote, updateNote, deleteNote, isLoading } = useNotes('message', messageId);
 *
 *   return (
 *     <div>
 *       {notes.map(note => (
 *         <div key={note.id}>
 *           <p>{note.content}</p>
 *           <button onClick={() => deleteNote(note.id)}>Delete</button>
 *         </div>
 *       ))}
 *       <button onClick={() => addNote('New note', 'general')}>Add Note</button>
 *     </div>
 *   );
 * }
 * ```
 */
export function useNotes(targetType: 'message' | 'tool_use' | 'reasoning', targetId: string) {
  const [isLoading, setIsLoading] = useState(false);
  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const hasFetched = useRef(false);

  // Get store actions and selectors
  const addNoteToStore = useNotesStore((state) => state.addNote);
  const updateNoteInStore = useNotesStore((state) => state.updateNote);
  const deleteNoteFromStore = useNotesStore((state) => state.deleteNote);
  const setNotesFromServer = useNotesStore((state) => state.setNotesFromServer);
  const getNotesForMessage = useNotesStore((state) => state.getNotesForMessage);

  // Get notes for this target
  const notes = getNotesForMessage(targetId);

  // Fetch notes from server on mount
  useEffect(() => {
    // Only fetch for message targets (tool_use and reasoning notes would be fetched with parent message)
    // Also skip if we've already fetched for this target
    if (targetType !== 'message' || !targetId || hasFetched.current) {
      return;
    }

    const fetchNotes = async () => {
      setIsFetching(true);
      try {
        const response: NoteListResponse = await api.getMessageNotes(targetId);
        // Populate store with fetched notes (preserving server IDs)
        setNotesFromServer(
          targetId,
          response.notes.map((note) => ({
            id: note.id,
            content: note.content,
            category: (note.category || 'general') as NoteCategory,
            created_at: note.created_at,
            updated_at: note.updated_at,
          }))
        );
        hasFetched.current = true;
      } catch (err) {
        console.error('Failed to fetch notes:', err);
        // Don't set error state for fetch failures - notes will just show empty
      } finally {
        setIsFetching(false);
      }
    };

    fetchNotes();
  }, [targetType, targetId, setNotesFromServer]);

  // API call based on target type
  const createNoteOnServer = useCallback(async (content: string, category: NoteCategory) => {
    switch (targetType) {
      case 'message':
        return api.createMessageNote(targetId, content, category);
      case 'tool_use':
        return api.createToolUseNote(targetId, content, category);
      case 'reasoning':
        return api.createReasoningNote(targetId, content, category);
    }
  }, [targetType, targetId]);

  // Add note handler
  const handleAddNote = useCallback(async (content: string, category: NoteCategory = 'general') => {
    if (!content.trim()) {
      setError('Note content cannot be empty');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      // First add to local store for immediate UI update
      addNoteToStore(targetId, content, category);

      // Then persist to server
      await createNoteOnServer(content, category);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create note');
      console.error('Create note error:', err);
      // Note: We don't remove from store on error as it may still be useful locally
    } finally {
      setIsLoading(false);
    }
  }, [targetId, addNoteToStore, createNoteOnServer]);

  // Update note handler
  const handleUpdateNote = useCallback(async (noteId: string, content: string) => {
    if (!content.trim()) {
      setError('Note content cannot be empty');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      // First update in local store
      updateNoteInStore(noteId, content);

      // Then persist to server
      await api.updateNote(noteId, content);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update note');
      console.error('Update note error:', err);
    } finally {
      setIsLoading(false);
    }
  }, [updateNoteInStore]);

  // Delete note handler
  const handleDeleteNote = useCallback(async (noteId: string) => {
    setIsLoading(true);
    setError(null);

    try {
      // First remove from local store
      deleteNoteFromStore(noteId);

      // Then remove from server
      await api.deleteNote(noteId);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete note');
      console.error('Delete note error:', err);
    } finally {
      setIsLoading(false);
    }
  }, [deleteNoteFromStore]);

  return {
    // Current state
    notes,

    // Loading and error state
    isLoading,
    isFetching,
    error,

    // Handlers
    addNote: handleAddNote,
    updateNote: handleUpdateNote,
    deleteNote: handleDeleteNote,
  };
}
