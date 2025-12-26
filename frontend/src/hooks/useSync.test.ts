import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useSync } from './useSync';
import { api } from '../services/api';
import { useMessageContext } from '../contexts/MessageContext';
import { useSSE } from './useSSE';
import { Message } from '../types/models';
import { SyncResponse } from '../types/sync';

// Mock dependencies
vi.mock('../services/api');
vi.mock('../contexts/MessageContext');
vi.mock('./useSSE');

const mockApi = api as ReturnType<typeof vi.mocked<typeof api>>;
const mockUseMessageContext = useMessageContext as ReturnType<typeof vi.mocked<typeof useMessageContext>>;
const mockUseSSE = useSSE as ReturnType<typeof vi.mocked<typeof useSSE>>;

describe('useSync', () => {
  const mockMessages: Message[] = [
    {
      id: 'msg1',
      conversation_id: 'conv1',
      sequence_number: 1,
      role: 'user',
      contents: 'Hello',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      local_id: 'msg1',
      sync_status: 'pending',
    },
  ];

  const mockSyncResponse: SyncResponse = {
    synced_messages: [
      {
        local_id: 'msg1',
        server_id: 'msg1',
        status: 'synced',
        message: mockMessages[0],
      },
    ],
    synced_at: '2024-01-01T00:00:00Z',
  };

  const mockMergeMessages = vi.fn();
  const mockUpdateMessage = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    mockUseMessageContext.mockReturnValue({
      messages: mockMessages,
      mergeMessages: mockMergeMessages,
      updateMessage: mockUpdateMessage,
    } as any);

    mockUseSSE.mockReturnValue({
      isConnected: false,
      error: null,
    } as any);

    mockApi.syncConversation.mockResolvedValue(mockSyncResponse);
  });

  it('should use sync protocol instead of getMessages', async () => {
    renderHook(() => useSync('conv1'));

    // Initial sync should happen
    await waitFor(() => {
      expect(mockApi.syncConversation).toHaveBeenCalledWith('conv1', {
        messages: expect.arrayContaining([
          expect.objectContaining({
            local_id: 'msg1',
            sequence_number: 1,
            role: 'user',
            contents: 'Hello',
          }),
        ]),
      });
    });

    // Should NOT call getMessages
    expect(mockApi.getMessages).not.toHaveBeenCalled();
  });

  it('should handle sync response and merge messages', async () => {
    renderHook(() => useSync('conv1'));

    await waitFor(() => {
      expect(mockMergeMessages).toHaveBeenCalledWith([
        expect.objectContaining({
          ...mockMessages[0],
          sync_status: 'synced',
        }),
      ]);
    });
  });

  it('should handle conflicts by using server version', async () => {
    const conflictResponse: SyncResponse = {
      synced_messages: [
        {
          local_id: 'msg1',
          server_id: 'msg1-server',
          status: 'conflict',
          conflict: {
            reason: 'Sequence mismatch',
            server_message: {
              ...mockMessages[0],
              id: 'msg1-server',
              contents: 'Server version',
            },
            resolution: 'server_wins',
          },
        },
      ],
      synced_at: '2024-01-01T00:00:00Z',
    };

    mockApi.syncConversation.mockResolvedValue(conflictResponse);

    renderHook(() => useSync('conv1'));

    await waitFor(() => {
      expect(mockMergeMessages).toHaveBeenCalledWith([
        expect.objectContaining({
          contents: 'Server version',
          sync_status: 'synced',
        }),
      ]);
    });
  });

  it('should implement exponential backoff when idle', async () => {
    vi.useFakeTimers();

    try {
      const { unmount } = renderHook(() => useSync('conv1'));

      // Initial sync happens immediately - wait for it to complete
      await vi.waitFor(() => {
        expect(mockApi.syncConversation).toHaveBeenCalledTimes(1);
      });

      // Advance 5 seconds (BASE_SYNC_INTERVAL_MS) - should sync again
      await vi.advanceTimersByTimeAsync(5000);
      await vi.waitFor(() => {
        expect(mockApi.syncConversation).toHaveBeenCalled();
      });

      const callCountBefore = mockApi.syncConversation.mock.calls.length;

      // Advance 30 seconds to trigger idle mode
      await vi.advanceTimersByTimeAsync(30000);

      // Verify that sync interval is increasing (fewer calls during idle period)
      const callCountAfterIdle = mockApi.syncConversation.mock.calls.length;

      // During 30s of idle, with exponential backoff, we should have fewer syncs
      // than we would with base interval (which would be 30s / 5s = 6 syncs)
      expect(callCountAfterIdle - callCountBefore).toBeLessThan(6);

      unmount();
    } finally {
      vi.useRealTimers();
    }
  });

  it('should reset backoff on error', async () => {
    vi.useFakeTimers();

    try {
      mockApi.syncConversation.mockRejectedValueOnce(new Error('Network error'));

      const { result, unmount } = renderHook(() => useSync('conv1'));

      // Initial sync with error
      await vi.runOnlyPendingTimersAsync();
      await vi.waitFor(() => {
        expect(result.current.syncError).toBe('Network error');
      });

      // Should reset and try again
      mockApi.syncConversation.mockResolvedValue(mockSyncResponse);

      await vi.advanceTimersByTimeAsync(5000);

      await vi.waitFor(() => {
        expect(result.current.syncError).toBe(null);
      });

      unmount();
    } finally {
      vi.useRealTimers();
    }
  });
});
