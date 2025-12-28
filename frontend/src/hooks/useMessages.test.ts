import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useMessages } from './useMessages';
import { api } from '../services/api';
import { messageRepository } from '../db/repository';
import { useMessageContext } from '../contexts/MessageContext';
import { useSync } from './useSync';
import { Message } from '../types/models';

// Mock dependencies
vi.mock('../services/api', () => ({
  api: {
    getMessages: vi.fn(),
    sendMessage: vi.fn(),
  },
}));

vi.mock('../db/repository', () => ({
  messageRepository: {
    findByConversation: vi.fn(),
    insert: vi.fn(),
    update: vi.fn(),
    upsert: vi.fn(),
    incrementRetryCount: vi.fn(),
  },
}));

vi.mock('../contexts/MessageContext', () => ({
  useMessageContext: vi.fn(),
}));

vi.mock('./useSync', () => ({
  useSync: vi.fn(),
}));

describe('useMessages', () => {
  const mockClearMessages = vi.fn();
  const mockSyncNow = vi.fn();

  // Sample messages for testing
  const createMessage = (overrides: Partial<Message> = {}): Message => ({
    id: 'msg-1',
    conversation_id: 'conv-1',
    sequence_number: 1,
    role: 'user',
    contents: 'Hello',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  });

  beforeEach(() => {
    vi.clearAllMocks();

    // Setup default mock implementations
    vi.mocked(useMessageContext).mockReturnValue({
      clearMessages: mockClearMessages,
    } as unknown as ReturnType<typeof useMessageContext>);

    vi.mocked(useSync).mockReturnValue({
      isSyncing: false,
      lastSyncTime: null,
      syncError: null,
      syncNow: mockSyncNow,
      isSSEConnected: false,
    });

    vi.mocked(messageRepository.findByConversation).mockReturnValue([]);
    vi.mocked(api.getMessages).mockResolvedValue([]);
  });

  describe('Initialization', () => {
    it('should initialize with empty messages array', () => {
      const { result } = renderHook(() => useMessages(null));

      expect(result.current.messages).toEqual([]);
    });

    it('should initialize with loading false when conversationId is null', () => {
      const { result } = renderHook(() => useMessages(null));

      expect(result.current.loading).toBe(false);
    });

    it('should initialize sending as false', () => {
      const { result } = renderHook(() => useMessages(null));

      expect(result.current.sending).toBe(false);
    });

    it('should expose sync state from useSync', () => {
      vi.mocked(useSync).mockReturnValue({
        isSyncing: true,
        lastSyncTime: new Date('2024-01-01'),
        syncError: 'Connection failed',
        syncNow: mockSyncNow,
        isSSEConnected: true,
      });

      const { result } = renderHook(() => useMessages('conv-1'));

      expect(result.current.isSyncing).toBe(true);
      expect(result.current.lastSyncTime).toEqual(new Date('2024-01-01'));
      expect(result.current.syncError).toBe('Connection failed');
      expect(result.current.syncNow).toBe(mockSyncNow);
    });
  });

  describe('Null ConversationId Handling', () => {
    it('should clear messages when conversationId is null', () => {
      const { result } = renderHook(() => useMessages(null));

      expect(result.current.messages).toEqual([]);
    });

    it('should call clearMessages from context when conversationId is null', () => {
      renderHook(() => useMessages(null));

      expect(mockClearMessages).toHaveBeenCalled();
    });

    it('should not fetch messages when conversationId is null', () => {
      renderHook(() => useMessages(null));

      expect(api.getMessages).not.toHaveBeenCalled();
    });

    it('should not query SQLite when conversationId is null', () => {
      renderHook(() => useMessages(null));

      expect(messageRepository.findByConversation).not.toHaveBeenCalled();
    });
  });

  describe('Message Loading', () => {
    it('should load messages from SQLite first', async () => {
      const sqliteMessages = [createMessage({ id: 'local-1' })];
      vi.mocked(messageRepository.findByConversation).mockReturnValue(sqliteMessages);

      const { result } = renderHook(() => useMessages('conv-1'));

      // SQLite should be called immediately
      expect(messageRepository.findByConversation).toHaveBeenCalledWith('conv-1');
      expect(result.current.messages).toEqual(sqliteMessages);
    });

    it('should fetch messages from server after SQLite', async () => {
      renderHook(() => useMessages('conv-1'));

      await waitFor(() => {
        expect(api.getMessages).toHaveBeenCalledWith('conv-1');
      });
    });

    it('should upsert server messages into SQLite', async () => {
      const serverMessages = [
        createMessage({ id: 'server-1', contents: 'From server' }),
        createMessage({ id: 'server-2', contents: 'Also from server' }),
      ];
      vi.mocked(api.getMessages).mockResolvedValue(serverMessages);

      renderHook(() => useMessages('conv-1'));

      await waitFor(() => {
        expect(messageRepository.upsert).toHaveBeenCalledTimes(2);
        expect(messageRepository.upsert).toHaveBeenCalledWith({
          ...serverMessages[0],
          sync_status: 'synced',
        });
        expect(messageRepository.upsert).toHaveBeenCalledWith({
          ...serverMessages[1],
          sync_status: 'synced',
        });
      });
    });

    it('should refresh from SQLite after server fetch completes', async () => {
      const serverMessages = [createMessage({ id: 'server-1' })];
      vi.mocked(api.getMessages).mockResolvedValue(serverMessages);

      renderHook(() => useMessages('conv-1'));

      await waitFor(() => {
        // findByConversation should be called again after upsert
        expect(messageRepository.findByConversation).toHaveBeenCalledTimes(2);
      });
    });

    it('should set loading true during fetch', async () => {
      let resolvePromise: (value: Message[]) => void;
      vi.mocked(api.getMessages).mockImplementation(
        () =>
          new Promise((resolve) => {
            resolvePromise = resolve;
          })
      );

      const { result } = renderHook(() => useMessages('conv-1'));

      // Loading should be true while waiting
      await waitFor(() => {
        expect(result.current.loading).toBe(true);
      });

      // Resolve the promise
      await act(async () => {
        resolvePromise!([]);
      });

      expect(result.current.loading).toBe(false);
    });

    it('should handle fetch error', async () => {
      vi.mocked(api.getMessages).mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useMessages('conv-1'));

      await waitFor(() => {
        expect(result.current.error).toBe('Failed to fetch messages');
        expect(result.current.loading).toBe(false);
      });
    });
  });

  describe('ConversationId Changes', () => {
    it('should reload messages when conversationId changes', async () => {
      const { rerender } = renderHook(({ convId }) => useMessages(convId), {
        initialProps: { convId: 'conv-1' },
      });

      await waitFor(() => {
        expect(api.getMessages).toHaveBeenCalledWith('conv-1');
      });

      // Change conversation
      rerender({ convId: 'conv-2' });

      await waitFor(() => {
        expect(api.getMessages).toHaveBeenCalledWith('conv-2');
        expect(messageRepository.findByConversation).toHaveBeenCalledWith('conv-2');
      });
    });

    it('should clear messages and call clearMessages when conversationId becomes null', async () => {
      const { result, rerender } = renderHook(({ convId }) => useMessages(convId), {
        initialProps: { convId: 'conv-1' as string | null },
      });

      await waitFor(() => {
        expect(api.getMessages).toHaveBeenCalled();
      });

      // Change to null
      rerender({ convId: null });

      expect(result.current.messages).toEqual([]);
      expect(mockClearMessages).toHaveBeenCalled();
    });
  });

  describe('sendMessage - Optimistic Updates', () => {
    beforeEach(() => {
      vi.mocked(api.sendMessage).mockResolvedValue(
        createMessage({ id: 'server-msg-1', contents: 'Test message' })
      );
    });

    it('should return false and not send when conversationId is null', async () => {
      const { result } = renderHook(() => useMessages(null));

      const success = await result.current.sendMessage('Hello');

      expect(success).toBe(false);
      expect(api.sendMessage).not.toHaveBeenCalled();
    });

    it('should return false for empty content', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      const success = await result.current.sendMessage('');

      expect(success).toBe(false);
      expect(api.sendMessage).not.toHaveBeenCalled();
    });

    it('should return false for whitespace-only content', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      const success = await result.current.sendMessage('   ');

      expect(success).toBe(false);
      expect(api.sendMessage).not.toHaveBeenCalled();
    });

    it('should insert optimistic message into SQLite immediately', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      await act(async () => {
        await result.current.sendMessage('Hello world');
      });

      expect(messageRepository.insert).toHaveBeenCalledWith(
        expect.objectContaining({
          conversation_id: 'conv-1',
          role: 'user',
          contents: 'Hello world',
          sync_status: 'pending',
          sequence_number: -1,
        })
      );
    });

    it('should generate unique local_id for optimistic message', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      await act(async () => {
        await result.current.sendMessage('Message 1');
      });

      const firstCall = vi.mocked(messageRepository.insert).mock.calls[0][0];
      expect(firstCall.local_id).toMatch(/^local_\d+_[a-z0-9]+$/);
      expect(firstCall.id).toBe(firstCall.local_id);
    });

    it('should trim content before sending', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      await act(async () => {
        await result.current.sendMessage('  Hello world  ');
      });

      expect(messageRepository.insert).toHaveBeenCalledWith(
        expect.objectContaining({
          contents: 'Hello world',
        })
      );
    });

    it('should set sending to true during send', async () => {
      let resolvePromise: (value: Message) => void;
      vi.mocked(api.sendMessage).mockImplementation(
        () =>
          new Promise((resolve) => {
            resolvePromise = resolve;
          })
      );

      const { result } = renderHook(() => useMessages('conv-1'));

      // Start sending
      let sendPromise: Promise<boolean>;
      act(() => {
        sendPromise = result.current.sendMessage('Hello');
      });

      // Should be sending
      expect(result.current.sending).toBe(true);

      // Resolve
      await act(async () => {
        resolvePromise!(createMessage());
        await sendPromise;
      });

      expect(result.current.sending).toBe(false);
    });
  });

  describe('sendMessage - Success Flow', () => {
    it('should call api.sendMessage with correct parameters', async () => {
      const serverResponse = createMessage({ id: 'server-id', contents: 'Test' });
      vi.mocked(api.sendMessage).mockResolvedValue(serverResponse);

      const { result } = renderHook(() => useMessages('conv-1'));

      await act(async () => {
        await result.current.sendMessage('Test');
      });

      expect(api.sendMessage).toHaveBeenCalledWith(
        'conv-1',
        expect.objectContaining({
          contents: 'Test',
          local_id: expect.stringMatching(/^local_/),
        })
      );
    });

    it('should update SQLite with server response on success', async () => {
      const serverResponse = createMessage({
        id: 'server-id',
        sequence_number: 42,
        contents: 'Test',
      });
      vi.mocked(api.sendMessage).mockResolvedValue(serverResponse);

      const { result } = renderHook(() => useMessages('conv-1'));

      await act(async () => {
        await result.current.sendMessage('Test');
      });

      expect(messageRepository.update).toHaveBeenCalledWith(
        expect.stringMatching(/^local_/),
        expect.objectContaining({
          ...serverResponse,
          sync_status: 'synced',
          server_id: 'server-id',
        })
      );
    });

    it('should return true on success', async () => {
      vi.mocked(api.sendMessage).mockResolvedValue(createMessage());

      const { result } = renderHook(() => useMessages('conv-1'));

      let success: boolean;
      await act(async () => {
        success = await result.current.sendMessage('Test');
      });

      expect(success!).toBe(true);
    });

    it('should refresh messages after successful send', async () => {
      vi.mocked(api.sendMessage).mockResolvedValue(createMessage());

      const { result } = renderHook(() => useMessages('conv-1'));

      // Wait for initial load
      await waitFor(() => {
        expect(api.getMessages).toHaveBeenCalled();
      });

      // Clear initial calls
      vi.mocked(messageRepository.findByConversation).mockClear();

      // Send a message
      await act(async () => {
        await result.current.sendMessage('Test');
      });

      // Should have refreshed (findByConversation called again)
      expect(messageRepository.findByConversation).toHaveBeenCalled();
    });
  });

  describe('sendMessage - Failure Flow', () => {
    beforeEach(() => {
      vi.mocked(api.sendMessage).mockRejectedValue(new Error('Network error'));
    });

    it('should return false on failure', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      let success: boolean;
      await act(async () => {
        success = await result.current.sendMessage('Test');
      });

      expect(success!).toBe(false);
    });

    it('should keep message as pending status on failure', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      await act(async () => {
        await result.current.sendMessage('Test');
      });

      expect(messageRepository.update).toHaveBeenCalledWith(
        expect.stringMatching(/^local_/),
        expect.objectContaining({
          sync_status: 'pending',
        })
      );
    });

    it('should increment retry count on failure', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      await act(async () => {
        await result.current.sendMessage('Test');
      });

      expect(messageRepository.incrementRetryCount).toHaveBeenCalledWith(
        expect.stringMatching(/^local_/)
      );
    });

    it('should set sending to false after failure', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      await act(async () => {
        await result.current.sendMessage('Test');
      });

      expect(result.current.sending).toBe(false);
    });

    it('should refresh messages after failure', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      // Wait for initial load
      await waitFor(() => {
        expect(api.getMessages).toHaveBeenCalled();
      });

      // Clear initial calls
      vi.mocked(messageRepository.findByConversation).mockClear();

      // Send a message (which will fail)
      await act(async () => {
        await result.current.sendMessage('Test');
      });

      // Should have refreshed
      expect(messageRepository.findByConversation).toHaveBeenCalled();
    });
  });

  describe('Sync Integration', () => {
    it('should pass callbacks to useSync', () => {
      renderHook(() => useMessages('conv-1'));

      expect(useSync).toHaveBeenCalledWith(
        'conv-1',
        expect.objectContaining({
          onSync: expect.any(Function),
          onMessage: expect.any(Function),
        })
      );
    });

    it('should refresh messages when onSync callback is triggered', async () => {
      let capturedOnSync: (() => void) | undefined;
      vi.mocked(useSync).mockImplementation((_convId, options) => {
        capturedOnSync = options?.onSync;
        return {
          isSyncing: false,
          lastSyncTime: null,
          syncError: null,
          syncNow: mockSyncNow,
          isSSEConnected: false,
        };
      });

      renderHook(() => useMessages('conv-1'));

      // Clear initial calls
      vi.mocked(messageRepository.findByConversation).mockClear();

      // Trigger sync callback
      act(() => {
        capturedOnSync?.();
      });

      await waitFor(() => {
        expect(messageRepository.findByConversation).toHaveBeenCalledWith('conv-1');
      });
    });

    it('should refresh messages when onMessage callback is triggered', async () => {
      let capturedOnMessage: (() => void) | undefined;
      vi.mocked(useSync).mockImplementation((_convId, options) => {
        capturedOnMessage = options?.onMessage as () => void;
        return {
          isSyncing: false,
          lastSyncTime: null,
          syncError: null,
          syncNow: mockSyncNow,
          isSSEConnected: false,
        };
      });

      renderHook(() => useMessages('conv-1'));

      // Clear initial calls
      vi.mocked(messageRepository.findByConversation).mockClear();

      // Trigger message callback
      act(() => {
        capturedOnMessage?.();
      });

      await waitFor(() => {
        expect(messageRepository.findByConversation).toHaveBeenCalledWith('conv-1');
      });
    });
  });

  describe('Manual Refresh', () => {
    it('should expose refresh function', () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      expect(typeof result.current.refresh).toBe('function');
    });

    it('should refresh messages from SQLite when refresh is called', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      // Clear initial calls
      vi.mocked(messageRepository.findByConversation).mockClear();

      // Call refresh
      act(() => {
        result.current.refresh();
      });

      await waitFor(() => {
        expect(messageRepository.findByConversation).toHaveBeenCalledWith('conv-1');
      });
    });
  });

  describe('Error State', () => {
    it('should expose error from fetch', async () => {
      vi.mocked(api.getMessages).mockRejectedValue(new Error('Server error'));

      const { result } = renderHook(() => useMessages('conv-1'));

      await waitFor(() => {
        expect(result.current.error).toBe('Failed to fetch messages');
      });
    });

    it('should clear error on successful fetch', async () => {
      vi.mocked(api.getMessages).mockRejectedValueOnce(new Error('Error'));

      const { result, rerender } = renderHook(({ convId }) => useMessages(convId), {
        initialProps: { convId: 'conv-1' },
      });

      await waitFor(() => {
        expect(result.current.error).toBe('Failed to fetch messages');
      });

      // Fix the mock and refetch
      vi.mocked(api.getMessages).mockResolvedValue([]);
      rerender({ convId: 'conv-2' });

      await waitFor(() => {
        expect(result.current.error).toBe(null);
      });
    });
  });

  describe('Multiple Messages', () => {
    it('should handle multiple server messages correctly', async () => {
      const serverMessages = [
        createMessage({ id: 'msg-1', sequence_number: 1 }),
        createMessage({ id: 'msg-2', sequence_number: 2 }),
        createMessage({ id: 'msg-3', sequence_number: 3 }),
      ];
      vi.mocked(api.getMessages).mockResolvedValue(serverMessages);

      renderHook(() => useMessages('conv-1'));

      await waitFor(() => {
        expect(messageRepository.upsert).toHaveBeenCalledTimes(3);
      });
    });

    it('should preserve local pending messages in SQLite', async () => {
      const localMessages = [
        createMessage({ id: 'local-1', sync_status: 'pending' }),
        createMessage({ id: 'server-1', sync_status: 'synced' }),
      ];
      vi.mocked(messageRepository.findByConversation).mockReturnValue(localMessages);

      const { result } = renderHook(() => useMessages('conv-1'));

      expect(result.current.messages).toEqual(localMessages);
    });
  });

  describe('Callback Stability', () => {
    it('should have stable sendMessage reference', async () => {
      const { result, rerender } = renderHook(() => useMessages('conv-1'));

      const firstSendMessage = result.current.sendMessage;

      // Force rerender
      rerender();

      expect(result.current.sendMessage).toBe(firstSendMessage);
    });

    it('should expose working refresh function', async () => {
      const { result } = renderHook(() => useMessages('conv-1'));

      // Wait for initial load
      await waitFor(() => {
        expect(api.getMessages).toHaveBeenCalled();
      });

      // Clear calls
      vi.mocked(messageRepository.findByConversation).mockClear();

      // Call refresh
      act(() => {
        result.current.refresh();
      });

      // Should trigger a refresh
      await waitFor(() => {
        expect(messageRepository.findByConversation).toHaveBeenCalledWith('conv-1');
      });
    });
  });
});
