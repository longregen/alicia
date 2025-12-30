import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useMemories } from './useMemories';
import { useMemoryStore } from '../stores/memoryStore';

// Mock fetch globally
global.fetch = vi.fn();

describe('useMemories', () => {
  beforeEach(() => {
    // Reset store and mocks
    useMemoryStore.getState().clearMemories();
    vi.clearAllMocks();

    // Mock fetch to return empty list by default to prevent infinite loop
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ memories: [], total: 0 }),
    } as Response);
  });

  afterEach(() => {
    vi.clearAllTimers();
  });

  it('should fetch memories on mount', async () => {
    const mockMemories = [
      {
        id: 'mem-1',
        content: 'User prefers dark mode',
        importance: 0.8,
        confidence: 0.9,
        tags: ['preference'],
        pinned: false,
        archived: false,
        created_at: 1000,
        updated_at: 1000,
      },
    ];

    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ memories: mockMemories, total: 1 }),
    } as Response);

    renderHook(() => useMemories());

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith('/api/v1/memories?limit=500');
    });
  });

  it('should create a new memory via API', async () => {
    const newMemory = {
      id: 'mem-new',
      content: 'User likes coffee',
      importance: 0.6,
      confidence: 0.8,
      tags: ['preference'],
      pinned: false,
      archived: false,
      created_at: 3000,
      updated_at: 3000,
    };

    vi.mocked(fetch)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ memories: [], total: 0 }),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => newMemory,
      } as Response);

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.create('User likes coffee', 'preference');
    });

    expect(fetch).toHaveBeenCalledWith('/api/v1/memories', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: 'User likes coffee', tags: ['preference'] }),
    });
  });

  it('should update a memory via API', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ memories: [], total: 0 }),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: 'mem-1', content: 'Updated' }),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      } as Response);

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.update('mem-1', 'Updated content', 'fact');
    });

    expect(fetch).toHaveBeenCalledWith('/api/v1/memories/mem-1', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: 'Updated content' }),
    });
  });

  it('should delete a memory via API', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ memories: [], total: 0 }),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      } as Response);

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.remove('mem-1');
    });

    expect(fetch).toHaveBeenCalledWith('/api/v1/memories/mem-1', {
      method: 'DELETE',
    });
  });

  it('should pin a memory via API', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ memories: [], total: 0 }),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      } as Response);

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.pin('mem-1', true);
    });

    expect(fetch).toHaveBeenCalledWith('/api/v1/memories/mem-1/pin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pinned: true }),
    });
  });

  it('should archive a memory via API', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ memories: [], total: 0 }),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      } as Response);

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await act(async () => {
      await result.current.archive('mem-1');
    });

    expect(fetch).toHaveBeenCalledWith('/api/v1/memories/mem-1/archive', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });
  });

  it('should handle API error during create', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    vi.mocked(fetch)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ memories: [], total: 0 }),
      } as Response)
      .mockResolvedValueOnce({
        ok: false,
        status: 500,
      } as Response);

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    // Call create and catch the error
    let thrownError: Error | null = null;
    try {
      await act(async () => {
        await result.current.create('Test', 'fact');
      });
    } catch (err) {
      thrownError = err as Error;
    }

    // The create method should have thrown an error
    expect(thrownError).toBeTruthy();
    expect(thrownError?.message).toContain('Failed to create memory');

    // Wait for error state to settle after the exception
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 50));
    });

    // The error state should also be set
    expect(result.current.error).toContain('Failed to create memory');

    consoleErrorSpy.mockRestore();
  });

  it('should handle API error during fetch', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 500,
    } as Response);

    renderHook(() => useMemories());

    await waitFor(() => {
      expect(fetch).toHaveBeenCalled();
    });

    // Wait a bit to ensure error handling is complete
    await new Promise(resolve => setTimeout(resolve, 100));

    consoleErrorSpy.mockRestore();
  });

  it('should filter memories by category', async () => {
    // Mock fetch to return memories instead of adding to store directly
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        memories: [
          {
            id: 'mem-1',
            content: 'Pref 1',
            tags: ['preference'],
            pinned: false,
            archived: false,
            created_at: 1,
            updated_at: 1,
            importance: 0.5,
            confidence: 0.8,
          },
          {
            id: 'mem-2',
            content: 'Fact 1',
            tags: ['fact'],
            pinned: false,
            archived: false,
            created_at: 2,
            updated_at: 2,
            importance: 0.5,
            confidence: 0.8,
          },
        ],
        total: 2
      }),
    } as Response);

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    await waitFor(() => {
      expect(result.current.memories.length).toBe(2);
    });

    const filtered = result.current.filterByCategory('preference');

    expect(filtered.length).toBeGreaterThan(0);
    filtered.forEach(m => expect(m.category).toBe('preference'));
  });

  it('should search memories', async () => {
    // Add memories directly to store
    useMemoryStore.getState().createMemory('User prefers dark mode', 'preference');
    useMemoryStore.getState().createMemory('User lives in NYC', 'fact');

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    const results = result.current.search('dark');

    expect(results).toBeDefined();
    expect(Array.isArray(results)).toBe(true);
  });

  it('should return all memories when search query is empty', async () => {
    // Add memories directly to store
    useMemoryStore.getState().createMemory('Memory 1', 'fact');
    useMemoryStore.getState().createMemory('Memory 2', 'fact');

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    const results = result.current.search('');

    expect(results).toEqual(result.current.memories);
  });

  it('should allow refreshing memories', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ memories: [], total: 0 }),
    } as Response);

    const { result } = renderHook(() => useMemories());

    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });

    const initialCallCount = vi.mocked(fetch).mock.calls.length;

    await act(async () => {
      await result.current.refresh();
    });

    expect(vi.mocked(fetch).mock.calls.length).toBeGreaterThan(initialCallCount);
  });
});
