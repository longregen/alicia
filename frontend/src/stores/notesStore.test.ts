import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  useNotesStore,
  selectAllNotes,
  selectRecentNotes,
  selectNotesByCategory,
  selectMessageNoteCount,
  type NoteCategory,
} from './notesStore';

describe('notesStore', () => {
  beforeEach(() => {
    useNotesStore.getState().clearNotes();
    vi.stubGlobal('crypto', {
      randomUUID: () => 'test-uuid-' + Math.random(),
    });
  });

  describe('addNote', () => {
    it('should add a note to the store', () => {
      useNotesStore.getState().addNote('msg-1', 'This is helpful', 'improvement');

      const state = useNotesStore.getState();
      const note = Object.values(state.notes)[0];
      expect(note).toBeDefined();
      expect(note.messageId).toBe('msg-1');
      expect(note.content).toBe('This is helpful');
      expect(note.category).toBe('improvement');
    });

    it('should create timestamps on note creation', () => {
      const before = Date.now();
      useNotesStore.getState().addNote('msg-1', 'Test note', 'general');
      const after = Date.now();

      const state = useNotesStore.getState();
      const note = Object.values(state.notes)[0];
      expect(note.createdAt).toBeGreaterThanOrEqual(before);
      expect(note.createdAt).toBeLessThanOrEqual(after);
      expect(note.updatedAt).toBe(note.createdAt);
    });

    it('should update notesByMessage index', () => {
      useNotesStore.getState().addNote('msg-1', 'Note 1', 'improvement');
      useNotesStore.getState().addNote('msg-1', 'Note 2', 'correction');

      const state = useNotesStore.getState();
      expect(state.notesByMessage['msg-1']).toHaveLength(2);
    });

    it('should handle all note categories', () => {
      const categories: NoteCategory[] = ['improvement', 'correction', 'context', 'general'];

      categories.forEach((category) => {
        useNotesStore.getState().addNote('msg-1', `Note for ${category}`, category);
      });

      const state = useNotesStore.getState();
      expect(Object.keys(state.notes)).toHaveLength(4);
    });

    it('should create notesByMessage entry if it does not exist', () => {
      useNotesStore.getState().addNote('msg-1', 'First note', 'general');

      const state = useNotesStore.getState();
      expect(state.notesByMessage['msg-1']).toBeDefined();
      expect(state.notesByMessage['msg-1']).toHaveLength(1);
    });
  });

  describe('updateNote', () => {
    it('should update note content', () => {
      useNotesStore.getState().addNote('msg-1', 'Original content', 'general');
      const state = useNotesStore.getState();
      const noteId = Object.keys(state.notes)[0];

      useNotesStore.getState().updateNote(noteId, 'Updated content');

      const updatedNote = useNotesStore.getState().notes[noteId];
      expect(updatedNote.content).toBe('Updated content');
    });

    it('should update timestamp when updating note', () => {
      vi.useFakeTimers();

      useNotesStore.getState().addNote('msg-1', 'Original content', 'general');
      const state = useNotesStore.getState();
      const noteId = Object.keys(state.notes)[0];
      const originalTimestamp = state.notes[noteId].updatedAt;

      vi.advanceTimersByTime(1000);

      useNotesStore.getState().updateNote(noteId, 'Updated content');

      const updatedNote = useNotesStore.getState().notes[noteId];
      expect(updatedNote.updatedAt).toBeGreaterThan(originalTimestamp);

      vi.useRealTimers();
    });

    it('should not update non-existent note', () => {
      expect(() => {
        useNotesStore.getState().updateNote('non-existent', 'Updated content');
      }).not.toThrow();

      const state = useNotesStore.getState();
      expect(Object.keys(state.notes)).toHaveLength(0);
    });
  });

  describe('deleteNote', () => {
    it('should delete a note from the store', () => {
      useNotesStore.getState().addNote('msg-1', 'Test note', 'general');
      const state = useNotesStore.getState();
      const noteId = Object.keys(state.notes)[0];

      useNotesStore.getState().deleteNote(noteId);

      const updatedState = useNotesStore.getState();
      expect(Object.keys(updatedState.notes)).toHaveLength(0);
    });

    it('should remove note from notesByMessage index', () => {
      useNotesStore.getState().addNote('msg-1', 'Note 1', 'general');
      useNotesStore.getState().addNote('msg-1', 'Note 2', 'general');
      const state = useNotesStore.getState();
      const noteId = Object.keys(state.notes)[0];

      useNotesStore.getState().deleteNote(noteId);

      const updatedState = useNotesStore.getState();
      expect(updatedState.notesByMessage['msg-1']).toHaveLength(1);
    });

    it('should clean up empty notesByMessage arrays', () => {
      useNotesStore.getState().addNote('msg-1', 'Only note', 'general');
      const state = useNotesStore.getState();
      const noteId = Object.keys(state.notes)[0];

      useNotesStore.getState().deleteNote(noteId);

      const updatedState = useNotesStore.getState();
      expect(updatedState.notesByMessage['msg-1']).toBeUndefined();
    });

    it('should not throw when deleting non-existent note', () => {
      expect(() => {
        useNotesStore.getState().deleteNote('non-existent');
      }).not.toThrow();
    });
  });

  describe('setNotesFromServer', () => {
    it('should set notes from server data', () => {
      const serverNotes = [
        {
          id: 'note-1',
          content: 'Server note 1',
          category: 'improvement' as NoteCategory,
          created_at: 1000,
          updated_at: 2000,
        },
        {
          id: 'note-2',
          content: 'Server note 2',
          category: 'correction' as NoteCategory,
          created_at: 1500,
          updated_at: 2500,
        },
      ];

      useNotesStore.getState().setNotesFromServer('msg-1', serverNotes);

      const state = useNotesStore.getState();
      expect(Object.keys(state.notes)).toHaveLength(2);
      expect(state.notes['note-1']).toBeDefined();
      expect(state.notes['note-2']).toBeDefined();
      expect(state.notes['note-1'].content).toBe('Server note 1');
    });

    it('should clear existing notes for message before setting new ones', () => {
      useNotesStore.getState().addNote('msg-1', 'Local note', 'general');

      const serverNotes = [
        {
          id: 'note-server',
          content: 'Server note',
          category: 'improvement' as NoteCategory,
        },
      ];

      useNotesStore.getState().setNotesFromServer('msg-1', serverNotes);

      const state = useNotesStore.getState();
      expect(Object.keys(state.notes)).toHaveLength(1);
      expect(state.notes['note-server']).toBeDefined();
    });

    it('should use current timestamp if timestamps not provided', () => {
      const before = Date.now();

      const serverNotes = [
        {
          id: 'note-1',
          content: 'Server note',
          category: 'general' as NoteCategory,
        },
      ];

      useNotesStore.getState().setNotesFromServer('msg-1', serverNotes);

      const after = Date.now();

      const state = useNotesStore.getState();
      const note = state.notes['note-1'];
      expect(note.createdAt).toBeGreaterThanOrEqual(before);
      expect(note.createdAt).toBeLessThanOrEqual(after);
    });

    it('should update notesByMessage index', () => {
      const serverNotes = [
        {
          id: 'note-1',
          content: 'Note 1',
          category: 'general' as NoteCategory,
        },
        {
          id: 'note-2',
          content: 'Note 2',
          category: 'general' as NoteCategory,
        },
      ];

      useNotesStore.getState().setNotesFromServer('msg-1', serverNotes);

      const state = useNotesStore.getState();
      expect(state.notesByMessage['msg-1']).toEqual(['note-1', 'note-2']);
    });

    it('should clear notesByMessage entry when setting empty notes', () => {
      useNotesStore.getState().addNote('msg-1', 'Local note', 'general');

      useNotesStore.getState().setNotesFromServer('msg-1', []);

      const state = useNotesStore.getState();
      expect(state.notesByMessage['msg-1']).toBeUndefined();
    });
  });

  describe('getNotesForMessage', () => {
    it('should return notes for a message', () => {
      useNotesStore.getState().addNote('msg-1', 'Note 1', 'general');
      useNotesStore.getState().addNote('msg-1', 'Note 2', 'improvement');
      useNotesStore.getState().addNote('msg-2', 'Note 3', 'correction');

      const notes = useNotesStore.getState().getNotesForMessage('msg-1');
      expect(notes).toHaveLength(2);
      expect(notes.every((n) => n.messageId === 'msg-1')).toBe(true);
    });

    it('should return empty array for message with no notes', () => {
      const notes = useNotesStore.getState().getNotesForMessage('msg-nonexistent');
      expect(notes).toEqual([]);
    });

    it('should sort notes by createdAt ascending', () => {
      vi.useFakeTimers();

      useNotesStore.getState().addNote('msg-1', 'Old note', 'general');
      vi.advanceTimersByTime(1000);

      useNotesStore.getState().addNote('msg-1', 'New note', 'general');

      const notes = useNotesStore.getState().getNotesForMessage('msg-1');
      expect(notes[0].content).toBe('Old note');
      expect(notes[1].content).toBe('New note');

      vi.useRealTimers();
    });
  });

  describe('getNotesByCategory', () => {
    it('should return notes filtered by category', () => {
      useNotesStore.getState().addNote('msg-1', 'Improvement 1', 'improvement');
      useNotesStore.getState().addNote('msg-2', 'Correction 1', 'correction');
      useNotesStore.getState().addNote('msg-3', 'Improvement 2', 'improvement');

      const improvements = useNotesStore.getState().getNotesByCategory('improvement');
      expect(improvements).toHaveLength(2);
      expect(improvements.every((n) => n.category === 'improvement')).toBe(true);
    });

    it('should return empty array when no notes match category', () => {
      useNotesStore.getState().addNote('msg-1', 'General note', 'general');

      const corrections = useNotesStore.getState().getNotesByCategory('correction');
      expect(corrections).toEqual([]);
    });

    it('should sort by updated timestamp descending', () => {
      vi.useFakeTimers();

      useNotesStore.getState().addNote('msg-1', 'Old note', 'general');
      vi.advanceTimersByTime(1000);

      useNotesStore.getState().addNote('msg-2', 'New note', 'general');

      const notes = useNotesStore.getState().getNotesByCategory('general');
      expect(notes[0].content).toBe('New note');
      expect(notes[1].content).toBe('Old note');

      vi.useRealTimers();
    });
  });

  describe('clearNotes', () => {
    it('should reset all state to initial values', () => {
      useNotesStore.getState().addNote('msg-1', 'Note 1', 'general');
      useNotesStore.getState().addNote('msg-2', 'Note 2', 'improvement');

      useNotesStore.getState().clearNotes();

      const state = useNotesStore.getState();
      expect(Object.keys(state.notes)).toHaveLength(0);
      expect(Object.keys(state.notesByMessage)).toHaveLength(0);
    });
  });

  describe('deleteNotesForMessage', () => {
    it('should delete all notes for a message', () => {
      useNotesStore.getState().addNote('msg-1', 'Note 1', 'general');
      useNotesStore.getState().addNote('msg-1', 'Note 2', 'improvement');
      useNotesStore.getState().addNote('msg-2', 'Note 3', 'correction');

      useNotesStore.getState().deleteNotesForMessage('msg-1');

      const state = useNotesStore.getState();
      expect(Object.keys(state.notes)).toHaveLength(1);
      expect(state.notesByMessage['msg-1']).toBeUndefined();
      expect(state.notesByMessage['msg-2']).toBeDefined();
    });

    it('should handle deleting notes for non-existent message', () => {
      expect(() => {
        useNotesStore.getState().deleteNotesForMessage('non-existent');
      }).not.toThrow();
    });
  });

  describe('selectAllNotes', () => {
    it('should return all notes sorted by updated timestamp descending', () => {
      vi.useFakeTimers();

      useNotesStore.getState().addNote('msg-1', 'Old note', 'general');
      vi.advanceTimersByTime(1000);

      useNotesStore.getState().addNote('msg-2', 'New note', 'improvement');

      const notes = selectAllNotes(useNotesStore.getState());
      expect(notes).toHaveLength(2);
      expect(notes[0].content).toBe('New note');
      expect(notes[1].content).toBe('Old note');

      vi.useRealTimers();
    });

    it('should return empty array when no notes exist', () => {
      const notes = selectAllNotes(useNotesStore.getState());
      expect(notes).toEqual([]);
    });
  });

  describe('selectRecentNotes', () => {
    it('should return most recent notes up to limit', () => {
      for (let i = 0; i < 15; i++) {
        useNotesStore.getState().addNote(`msg-${i}`, `Note ${i}`, 'general');
      }

      const recent = selectRecentNotes(useNotesStore.getState(), 10);
      expect(recent).toHaveLength(10);
    });

    it('should default to 10 notes', () => {
      for (let i = 0; i < 15; i++) {
        useNotesStore.getState().addNote(`msg-${i}`, `Note ${i}`, 'general');
      }

      const recent = selectRecentNotes(useNotesStore.getState());
      expect(recent).toHaveLength(10);
    });

    it('should return all notes when less than limit', () => {
      useNotesStore.getState().addNote('msg-1', 'Note 1', 'general');
      useNotesStore.getState().addNote('msg-2', 'Note 2', 'general');

      const recent = selectRecentNotes(useNotesStore.getState(), 10);
      expect(recent).toHaveLength(2);
    });

    it('should sort by updated timestamp descending', () => {
      vi.useFakeTimers();

      useNotesStore.getState().addNote('msg-1', 'Old note', 'general');
      vi.advanceTimersByTime(1000);

      useNotesStore.getState().addNote('msg-2', 'New note', 'general');

      const recent = selectRecentNotes(useNotesStore.getState());
      expect(recent[0].content).toBe('New note');

      vi.useRealTimers();
    });
  });

  describe('selectNotesByCategory', () => {
    it('should filter and sort by category', () => {
      vi.useFakeTimers();

      useNotesStore.getState().addNote('msg-1', 'Old improvement', 'improvement');
      vi.advanceTimersByTime(1000);

      useNotesStore.getState().addNote('msg-2', 'New improvement', 'improvement');
      useNotesStore.getState().addNote('msg-3', 'Correction', 'correction');

      const improvements = selectNotesByCategory(useNotesStore.getState(), 'improvement');
      expect(improvements).toHaveLength(2);
      expect(improvements[0].content).toBe('New improvement');

      vi.useRealTimers();
    });
  });

  describe('selectMessageNoteCount', () => {
    it('should return count of notes for message', () => {
      useNotesStore.getState().addNote('msg-1', 'Note 1', 'general');
      useNotesStore.getState().addNote('msg-1', 'Note 2', 'improvement');
      useNotesStore.getState().addNote('msg-2', 'Note 3', 'correction');

      const count = selectMessageNoteCount(useNotesStore.getState(), 'msg-1');
      expect(count).toBe(2);
    });

    it('should return 0 for message with no notes', () => {
      const count = selectMessageNoteCount(useNotesStore.getState(), 'non-existent');
      expect(count).toBe(0);
    });
  });
});
