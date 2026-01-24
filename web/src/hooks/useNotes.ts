import { useEffect, useCallback } from 'react';
import { useNotesStore } from '../stores/notesStore';

export function useNotes() {
  const notes = useNotesStore((s) => s.notes);
  const selectedNoteId = useNotesStore((s) => s.selectedNoteId);
  const loading = useNotesStore((s) => s.loading);
  const error = useNotesStore((s) => s.error);
  const fetchNotes = useNotesStore((s) => s.fetchNotes);
  const createNote = useNotesStore((s) => s.createNote);
  const updateNote = useNotesStore((s) => s.updateNote);
  const deleteNote = useNotesStore((s) => s.deleteNote);
  const selectNote = useNotesStore((s) => s.selectNote);

  useEffect(() => {
    fetchNotes();
  }, [fetchNotes]);

  const selectedNote = notes.find((n) => n.id === selectedNoteId) ?? null;

  const handleCreate = useCallback(async () => {
    const note = await createNote('Untitled', '');
    if (note) {
      selectNote(note.id);
    }
    return note;
  }, [createNote, selectNote]);

  return {
    notes,
    selectedNote,
    selectedNoteId,
    loading,
    error,
    createNote: handleCreate,
    updateNote,
    deleteNote,
    selectNote,
  };
}
