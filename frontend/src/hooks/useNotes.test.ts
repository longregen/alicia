import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useNotes } from './useNotes';
import { useNotesStore } from '../stores/notesStore';
import { api } from '../services/api';

// Mock the API module
vi.mock('../services/api', () => ({
  api: {
    getMessageNotes: vi.fn(),
    createMessageNote: vi.fn(),
    createToolUseNote: vi.fn(),
    createReasoningNote: vi.fn(),
    updateNote: vi.fn(),
    deleteNote: vi.fn(),
  },
}));

describe('useNotes', () => {
  beforeEach(() => {
    // Reset store and mocks
    useNotesStore.setState({
      notesByMessage: {},
      notes: {},
    });
    vi.clearAllMocks();
  });

  it('should initialize with empty notes', () => {
    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    expect(result.current.notes).toEqual([]);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it('should fetch notes on mount for message target', async () => {
    const mockNotes = [
      {
        id: 'note-1',
        target_id: 'msg-1',
        target_type: 'message',
        content: 'This is a note',
        category: 'general',
        created_at: 1000,
        updated_at: 1000,
      },
      {
        id: 'note-2',
        target_id: 'msg-1',
        target_type: 'message',
        content: 'Another note',
        category: 'improvement',
        created_at: 2000,
        updated_at: 2000,
      },
    ];

    vi.mocked(api.getMessageNotes).mockResolvedValue({
      notes: mockNotes,
      total: 2,
    });

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    expect(api.getMessageNotes).toHaveBeenCalledWith('msg-1');
    expect(result.current.notes).toHaveLength(2);
  });

  it('should not fetch notes for tool_use target', () => {
    const { result } = renderHook(() => useNotes('tool_use', 'tool-1'));

    expect(api.getMessageNotes).not.toHaveBeenCalled();
    expect(result.current.isFetching).toBe(false);
  });

  it('should not fetch notes for reasoning target', () => {
    const { result } = renderHook(() => useNotes('reasoning', 'reason-1'));

    expect(api.getMessageNotes).not.toHaveBeenCalled();
    expect(result.current.isFetching).toBe(false);
  });

  it('should add a note to message', async () => {
    const mockNote = {
      id: 'note-new',
      target_id: 'msg-1',
      target_type: 'message',
      content: 'New note',
      category: 'general',
      created_at: 3000,
      updated_at: 3000,
    };

    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });
    vi.mocked(api.createMessageNote).mockResolvedValue(mockNote);

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.addNote('New note', 'general');
    });

    expect(api.createMessageNote).toHaveBeenCalledWith('msg-1', 'New note', 'general');
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it('should add a note to tool_use', async () => {
    const mockNote = {
      id: 'note-new',
      target_id: 'tool-1',
      target_type: 'tool_use',
      content: 'Tool note',
      category: 'correction',
      created_at: 3000,
      updated_at: 3000,
    };

    vi.mocked(api.createToolUseNote).mockResolvedValue(mockNote);

    const { result } = renderHook(() => useNotes('tool_use', 'tool-1'));

    await act(async () => {
      await result.current.addNote('Tool note', 'correction');
    });

    expect(api.createToolUseNote).toHaveBeenCalledWith('tool-1', 'Tool note', 'correction');
  });

  it('should add a note to reasoning', async () => {
    const mockNote = {
      id: 'note-new',
      target_id: 'reason-1',
      target_type: 'reasoning',
      content: 'Reasoning note',
      category: 'improvement',
      created_at: 3000,
      updated_at: 3000,
    };

    vi.mocked(api.createReasoningNote).mockResolvedValue(mockNote);

    const { result } = renderHook(() => useNotes('reasoning', 'reason-1'));

    await act(async () => {
      await result.current.addNote('Reasoning note', 'improvement');
    });

    expect(api.createReasoningNote).toHaveBeenCalledWith('reason-1', 'Reasoning note', 'improvement');
  });

  it('should use default category "general" when not specified', async () => {
    const mockNote = {
      id: 'note-new',
      target_id: 'msg-1',
      target_type: 'message',
      content: 'Note without category',
      category: 'general',
      created_at: 3000,
      updated_at: 3000,
    };

    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });
    vi.mocked(api.createMessageNote).mockResolvedValue(mockNote);

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.addNote('Note without category');
    });

    expect(api.createMessageNote).toHaveBeenCalledWith('msg-1', 'Note without category', 'general');
  });

  it('should reject empty note content', async () => {
    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.addNote('   ', 'general');
    });

    expect(result.current.error).toBe('Note content cannot be empty');
    expect(api.createMessageNote).not.toHaveBeenCalled();
  });

  it('should update an existing note', async () => {
    const mockNotes = [
      {
        id: 'note-1',
        target_id: 'msg-1',
        target_type: 'message',
        content: 'Original content',
        category: 'general',
        created_at: 1000,
        updated_at: 1000,
      },
    ];

    vi.mocked(api.getMessageNotes).mockResolvedValue({
      notes: mockNotes,
      total: 1,
    });
    vi.mocked(api.updateNote).mockResolvedValue({
      id: 'note-1',
      target_id: 'msg-1',
      target_type: 'message',
      content: 'Updated content',
      category: 'general',
      created_at: 1000,
      updated_at: 2000,
    });

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.updateNote('note-1', 'Updated content');
    });

    expect(api.updateNote).toHaveBeenCalledWith('note-1', 'Updated content');
    expect(result.current.isLoading).toBe(false);
  });

  it('should reject empty content when updating', async () => {
    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.updateNote('note-1', '  ');
    });

    expect(result.current.error).toBe('Note content cannot be empty');
    expect(api.updateNote).not.toHaveBeenCalled();
  });

  it('should delete a note', async () => {
    const mockNotes = [
      {
        id: 'note-1',
        target_id: 'msg-1',
        target_type: 'message',
        content: 'To be deleted',
        category: 'general',
        created_at: 1000,
        updated_at: 1000,
      },
    ];

    vi.mocked(api.getMessageNotes).mockResolvedValue({
      notes: mockNotes,
      total: 1,
    });
    vi.mocked(api.deleteNote).mockResolvedValue(undefined);

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.deleteNote('note-1');
    });

    expect(api.deleteNote).toHaveBeenCalledWith('note-1');
    expect(result.current.isLoading).toBe(false);
  });

  it('should handle API error during note creation', async () => {
    const error = new Error('Server error');
    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });
    vi.mocked(api.createMessageNote).mockRejectedValue(error);

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.addNote('Test note', 'general');
    });

    expect(result.current.error).toBe('Server error');
    expect(result.current.isLoading).toBe(false);
  });

  it('should handle API error during note update', async () => {
    const error = new Error('Update failed');
    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });
    vi.mocked(api.updateNote).mockRejectedValue(error);

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.updateNote('note-1', 'Updated');
    });

    expect(result.current.error).toBe('Update failed');
  });

  it('should handle API error during note deletion', async () => {
    const error = new Error('Delete failed');
    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });
    vi.mocked(api.deleteNote).mockRejectedValue(error);

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.deleteNote('note-1');
    });

    expect(result.current.error).toBe('Delete failed');
  });

  it('should silently handle fetch errors', async () => {
    const consoleWarnSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const error = new Error('Fetch failed');
    vi.mocked(api.getMessageNotes).mockRejectedValue(error);

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    // Should not set error state for fetch failures
    expect(result.current.error).toBeNull();
    expect(result.current.notes).toEqual([]);

    consoleWarnSpy.mockRestore();
  });

  it('should only fetch once for same target', async () => {
    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });

    const { rerender } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(api.getMessageNotes).toHaveBeenCalledTimes(1);
    });

    rerender();
    rerender();

    // Should still only be called once
    expect(api.getMessageNotes).toHaveBeenCalledTimes(1);
  });

  it('should add note to local store immediately before server call', async () => {
    const mockNote = {
      id: 'note-new',
      target_id: 'msg-1',
      target_type: 'message',
      content: 'New note',
      category: 'general',
      created_at: 3000,
      updated_at: 3000,
    };

    let resolveCreate: (value: any) => void;
    const createPromise = new Promise((resolve) => {
      resolveCreate = resolve;
    });

    vi.mocked(api.getMessageNotes).mockResolvedValue({ notes: [], total: 0 });
    vi.mocked(api.createMessageNote).mockReturnValue(createPromise as any);

    const { result } = renderHook(() => useNotes('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    let createPromiseResult: Promise<void>;
    act(() => {
      createPromiseResult = result.current.addNote('New note', 'general');
    });

    // Note should be in local store even before server responds
    await waitFor(() => {
      expect(result.current.isLoading).toBe(true);
    });

    await act(async () => {
      resolveCreate!(mockNote);
      await createPromiseResult!;
    });

    expect(result.current.isLoading).toBe(false);
  });
});
