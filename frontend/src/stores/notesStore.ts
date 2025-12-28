import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';

export type NoteCategory = 'improvement' | 'correction' | 'context' | 'general';

export interface UserNote {
  id: string;
  messageId: string;
  content: string;
  category: NoteCategory;
  createdAt: number;
  updatedAt: number;
}

interface NotesStoreState {
  notes: Record<string, UserNote>;
  notesByMessage: Record<string, string[]>; // messageId -> noteIds
}

interface NotesStoreActions {
  // Note actions
  addNote: (
    messageId: string,
    content: string,
    category: NoteCategory
  ) => void;
  updateNote: (noteId: string, content: string) => void;
  deleteNote: (noteId: string) => void;

  // Server sync actions
  setNotesFromServer: (messageId: string, notes: Array<{
    id: string;
    content: string;
    category: NoteCategory;
    created_at?: number;
    updated_at?: number;
  }>) => void;

  // Query actions
  getNotesForMessage: (messageId: string) => UserNote[];
  getNotesByCategory: (category: NoteCategory) => UserNote[];

  // Bulk operations
  clearNotes: () => void;
  deleteNotesForMessage: (messageId: string) => void;
}

type NotesStore = NotesStoreState & NotesStoreActions;

const initialState: NotesStoreState = {
  notes: {},
  notesByMessage: {},
};

export const useNotesStore = create<NotesStore>()(
  immer((set, get) => ({
    ...initialState,

    // Note actions
    addNote: (messageId, content, category) =>
      set((state) => {
        const id = crypto.randomUUID();
        const timestamp = Date.now();

        state.notes[id] = {
          id,
          messageId,
          content,
          category,
          createdAt: timestamp,
          updatedAt: timestamp,
        };

        // Update index
        if (!state.notesByMessage[messageId]) {
          state.notesByMessage[messageId] = [];
        }
        state.notesByMessage[messageId].push(id);
      }),

    setNotesFromServer: (messageId, notes) =>
      set((state) => {
        // Clear existing notes for this message first
        const existingIds = state.notesByMessage[messageId] || [];
        existingIds.forEach((id) => {
          delete state.notes[id];
        });

        // Add new notes from server
        const newNoteIds: string[] = [];
        for (const note of notes) {
          const createdAt = note.created_at ?? Date.now();
          const updatedAt = note.updated_at ?? createdAt;

          state.notes[note.id] = {
            id: note.id,
            messageId,
            content: note.content,
            category: note.category,
            createdAt,
            updatedAt,
          };
          newNoteIds.push(note.id);
        }

        // Update index
        if (newNoteIds.length > 0) {
          state.notesByMessage[messageId] = newNoteIds;
        } else {
          delete state.notesByMessage[messageId];
        }
      }),

    updateNote: (noteId, content) =>
      set((state) => {
        if (state.notes[noteId]) {
          state.notes[noteId].content = content;
          state.notes[noteId].updatedAt = Date.now();
        }
      }),

    deleteNote: (noteId) =>
      set((state) => {
        const note = state.notes[noteId];
        if (note) {
          // Remove from index
          const messageNotes = state.notesByMessage[note.messageId];
          if (messageNotes) {
            const index = messageNotes.indexOf(noteId);
            if (index > -1) {
              messageNotes.splice(index, 1);
            }
            // Clean up empty arrays
            if (messageNotes.length === 0) {
              delete state.notesByMessage[note.messageId];
            }
          }
          // Remove note
          delete state.notes[noteId];
        }
      }),

    // Query actions
    getNotesForMessage: (messageId) => {
      const state = get();
      const noteIds = state.notesByMessage[messageId] || [];
      return noteIds
        .map((id) => state.notes[id])
        .filter(Boolean)
        .sort((a, b) => a.createdAt - b.createdAt);
    },

    getNotesByCategory: (category) => {
      return Object.values(get().notes)
        .filter((note) => note.category === category)
        .sort((a, b) => b.updatedAt - a.updatedAt);
    },

    // Bulk operations
    clearNotes: () =>
      set((state) => {
        Object.assign(state, initialState);
      }),

    deleteNotesForMessage: (messageId) =>
      set((state) => {
        const noteIds = state.notesByMessage[messageId] || [];
        noteIds.forEach((noteId) => {
          delete state.notes[noteId];
        });
        delete state.notesByMessage[messageId];
      }),
  }))
);

// Utility selectors
export const selectAllNotes = (state: NotesStore) =>
  Object.values(state.notes).sort((a, b) => b.updatedAt - a.updatedAt);

export const selectRecentNotes = (state: NotesStore, limit: number = 10) =>
  Object.values(state.notes)
    .sort((a, b) => b.updatedAt - a.updatedAt)
    .slice(0, limit);

export const selectNotesByCategory = (
  state: NotesStore,
  category: NoteCategory
) =>
  Object.values(state.notes)
    .filter((note) => note.category === category)
    .sort((a, b) => b.updatedAt - a.updatedAt);

export const selectMessageNoteCount = (state: NotesStore, messageId: string) =>
  state.notesByMessage[messageId]?.length || 0;
