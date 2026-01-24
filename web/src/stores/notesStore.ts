import { create } from 'zustand';
import { api, NoteResponse } from '../services/api';

interface NotesState {
  notes: NoteResponse[];
  selectedNoteId: string | null;
  loading: boolean;
  error: string | null;

  fetchNotes: () => Promise<void>;
  createNote: (title: string, content: string) => Promise<NoteResponse | null>;
  updateNote: (id: string, data: { title?: string; content?: string }) => Promise<void>;
  deleteNote: (id: string) => Promise<void>;
  selectNote: (id: string | null) => void;
}

export const useNotesStore = create<NotesState>((set) => ({
  notes: [],
  selectedNoteId: null,
  loading: false,
  error: null,

  fetchNotes: async () => {
    set({ loading: true, error: null });
    try {
      const notes = await api.listNotes();
      set({ notes, loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : 'Failed to fetch notes', loading: false });
    }
  },

  createNote: async (title: string, content: string) => {
    set({ loading: true, error: null });
    try {
      const note = await api.createNote({ title, content });
      set((state) => ({ notes: [note, ...state.notes], loading: false }));
      return note;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : 'Failed to create note', loading: false });
      return null;
    }
  },

  updateNote: async (id: string, data: { title?: string; content?: string }) => {
    set({ error: null });
    try {
      const updated = await api.updateNote(id, data);
      set((state) => ({
        notes: state.notes.map((n) => (n.id === id ? updated : n)),
      }));
    } catch (err) {
      set({ error: err instanceof Error ? err.message : 'Failed to update note' });
    }
  },

  deleteNote: async (id: string) => {
    set({ loading: true, error: null });
    try {
      await api.deleteNote(id);
      set((state) => ({
        notes: state.notes.filter((n) => n.id !== id),
        selectedNoteId: state.selectedNoteId === id ? null : state.selectedNoteId,
        loading: false,
      }));
    } catch (err) {
      set({ error: err instanceof Error ? err.message : 'Failed to delete note', loading: false });
    }
  },

  selectNote: (id: string | null) => {
    set({ selectedNoteId: id });
  },
}));
