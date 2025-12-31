import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useConversations } from './useConversations';
import { api } from '../services/api';
import type { Conversation } from '../types/models';

// Mock the api module
vi.mock('../services/api', () => ({
  api: {
    getConversations: vi.fn(),
    createConversation: vi.fn(),
    deleteConversation: vi.fn(),
    updateConversation: vi.fn(),
  },
}));

describe('useConversations', () => {
  const mockConversations: Conversation[] = [
    {
      id: 'conv-1',
      title: 'First Conversation',
      status: 'active',
      last_client_stanza_id: 0,
      last_server_stanza_id: 0,
      created_at: '2024-01-01T10:00:00Z',
      updated_at: '2024-01-01T10:00:00Z',
    },
    {
      id: 'conv-2',
      title: 'Second Conversation',
      status: 'active',
      last_client_stanza_id: 0,
      last_server_stanza_id: 0,
      created_at: '2024-01-02T10:00:00Z',
      updated_at: '2024-01-02T10:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getConversations).mockResolvedValue(mockConversations);
  });

  describe('Initial Fetch', () => {
    it('fetches conversations on mount', async () => {
      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(api.getConversations).toHaveBeenCalledTimes(1);
      expect(result.current.conversations).toEqual(mockConversations);
    });

    it('sets loading to true during fetch', async () => {
      // Don't resolve the promise immediately
      vi.mocked(api.getConversations).mockImplementation(
        () => new Promise(resolve => setTimeout(() => resolve(mockConversations), 100))
      );

      const { result } = renderHook(() => useConversations());

      expect(result.current.loading).toBe(true);

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });
    });

    it('sets error when fetch fails', async () => {
      vi.mocked(api.getConversations).mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Failed to fetch conversations');
    });

    it('starts with empty conversations array', () => {
      vi.mocked(api.getConversations).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      const { result } = renderHook(() => useConversations());

      expect(result.current.conversations).toEqual([]);
    });
  });

  describe('createConversation', () => {
    it('creates a new conversation', async () => {
      const newConversation: Conversation = {
        id: 'conv-3',
        title: 'New Conversation',
        status: 'active',
        last_client_stanza_id: 0,
        last_server_stanza_id: 0,
        created_at: '2024-01-03T10:00:00Z',
        updated_at: '2024-01-03T10:00:00Z',
      };

      vi.mocked(api.createConversation).mockResolvedValue(newConversation);

      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.createConversation('New Conversation');
      });

      expect(api.createConversation).toHaveBeenCalledWith({ title: 'New Conversation' });
      expect(result.current.conversations).toContainEqual(newConversation);
    });

    it('prepends new conversation to the list', async () => {
      const newConversation: Conversation = {
        id: 'conv-3',
        title: 'New Conversation',
        status: 'active',
        last_client_stanza_id: 0,
        last_server_stanza_id: 0,
        created_at: '2024-01-03T10:00:00Z',
        updated_at: '2024-01-03T10:00:00Z',
      };

      vi.mocked(api.createConversation).mockResolvedValue(newConversation);

      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.createConversation('New Conversation');
      });

      expect(result.current.conversations[0]).toEqual(newConversation);
    });

    it('creates conversation without title', async () => {
      const newConversation: Conversation = {
        id: 'conv-3',
        title: '',
        status: 'active',
        last_client_stanza_id: 0,
        last_server_stanza_id: 0,
        created_at: '2024-01-03T10:00:00Z',
        updated_at: '2024-01-03T10:00:00Z',
      };

      vi.mocked(api.createConversation).mockResolvedValue(newConversation);

      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.createConversation();
      });

      expect(api.createConversation).toHaveBeenCalledWith({ title: undefined });
    });
  });

  describe('deleteConversation', () => {
    it('deletes a conversation', async () => {
      vi.mocked(api.deleteConversation).mockResolvedValue(undefined);

      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.conversations).toHaveLength(2);

      await act(async () => {
        await result.current.deleteConversation('conv-1');
      });

      expect(api.deleteConversation).toHaveBeenCalledWith('conv-1');
      expect(result.current.conversations).toHaveLength(1);
      expect(result.current.conversations.find(c => c.id === 'conv-1')).toBeUndefined();
    });

    it('does not remove conversation from list if delete fails', async () => {
      vi.mocked(api.deleteConversation).mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.deleteConversation('conv-1');
      });

      // Conversation should still be in the list
      expect(result.current.conversations).toHaveLength(2);
      expect(result.current.conversations.find(c => c.id === 'conv-1')).toBeDefined();
    });
  });

  describe('updateConversation', () => {
    it('updates a conversation in local state', async () => {
      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const updatedConversation: Conversation = {
        ...mockConversations[0],
        title: 'Updated Title',
      };

      vi.mocked(api.updateConversation).mockResolvedValue(updatedConversation);

      await act(async () => {
        await result.current.updateConversation(updatedConversation.id, { title: 'Updated Title' });
      });

      expect(api.updateConversation).toHaveBeenCalledWith('conv-1', { title: 'Updated Title' });
      const found = result.current.conversations.find(c => c.id === 'conv-1');
      expect(found?.title).toBe('Updated Title');
    });

    it('does not add new conversation if id does not exist', async () => {
      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const newConversation: Conversation = {
        id: 'non-existent',
        title: 'New',
        status: 'active',
        last_client_stanza_id: 0,
        last_server_stanza_id: 0,
        created_at: '2024-01-03T10:00:00Z',
        updated_at: '2024-01-03T10:00:00Z',
      };

      vi.mocked(api.updateConversation).mockResolvedValue(newConversation);

      await act(async () => {
        await result.current.updateConversation(newConversation.id, { title: newConversation.title });
      });

      // Length should stay the same (doesn't add, just maps)
      expect(result.current.conversations).toHaveLength(2);
    });
  });

  describe('refetch', () => {
    it('refetches conversations', async () => {
      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(api.getConversations).toHaveBeenCalledTimes(1);

      const newConversations: Conversation[] = [
        {
          id: 'conv-new',
          title: 'New Conversation',
          status: 'active',
          last_client_stanza_id: 0,
          last_server_stanza_id: 0,
          created_at: '2024-01-05T10:00:00Z',
          updated_at: '2024-01-05T10:00:00Z',
        },
      ];

      vi.mocked(api.getConversations).mockResolvedValue(newConversations);

      await act(async () => {
        await result.current.refetch();
      });

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(api.getConversations).toHaveBeenCalledTimes(2);
      expect(result.current.conversations).toEqual(newConversations);
    });
  });

  describe('State Management', () => {
    it('clears error on successful fetch', async () => {
      // First, make it fail
      vi.mocked(api.getConversations).mockRejectedValueOnce(new Error('Network error'));

      const { result } = renderHook(() => useConversations());

      await waitFor(() => {
        expect(result.current.error).toBe('Failed to fetch conversations');
      });

      // Then make it succeed
      vi.mocked(api.getConversations).mockResolvedValue(mockConversations);

      await act(async () => {
        await result.current.refetch();
      });

      await waitFor(() => {
        expect(result.current.error).toBeNull();
      });
    });
  });
});
