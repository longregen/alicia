import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useWebSocketSync } from './useWebSocketSync';
import { pack } from 'msgpackr';
import { MessageType } from '../types/protocol';

// Mock dependencies
vi.mock('../db/repository', () => ({
  messageRepository: {
    findByLocalId: vi.fn(),
    findByServerId: vi.fn(),
    findById: vi.fn(),
    getPending: vi.fn().mockReturnValue([]),
    update: vi.fn(),
    insert: vi.fn(),
    upsert: vi.fn(),
  },
}));

vi.mock('../adapters/protocolAdapter', () => ({
  handleProtocolMessage: vi.fn(),
}));

// Import mocked modules
import { messageRepository } from '../db/repository';
import { handleProtocolMessage } from '../adapters/protocolAdapter';

// MockWebSocket class
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  url: string;
  binaryType: BinaryType = 'blob';
  readyState: number = MockWebSocket.CONNECTING;

  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;

  sentMessages: Uint8Array[] = [];

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send(data: ArrayBuffer | SharedArrayBuffer | Uint8Array): void {
    if (this.readyState !== MockWebSocket.OPEN) {
      throw new Error('WebSocket is not open');
    }
    const uint8 = (data instanceof ArrayBuffer || data instanceof SharedArrayBuffer) ? new Uint8Array(data) : data;
    this.sentMessages.push(uint8);
  }

  close(): void {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent('close'));
  }

  // Test helpers
  simulateOpen(): void {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event('open'));
  }

  simulateClose(): void {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent('close'));
  }

  simulateError(): void {
    this.onerror?.(new Event('error'));
  }

  simulateMessage(data: ArrayBuffer): void {
    this.onmessage?.(new MessageEvent('message', { data }));
  }

  simulateBinaryMessage(dto: unknown): void {
    const packed = pack(dto);
    // Convert to ArrayBuffer (not SharedArrayBuffer)
    const arrayBuffer = packed.buffer instanceof SharedArrayBuffer
      ? new ArrayBuffer(packed.byteLength)
      : packed.buffer.slice(packed.byteOffset, packed.byteOffset + packed.byteLength);

    if (packed.buffer instanceof SharedArrayBuffer) {
      const view = new Uint8Array(arrayBuffer);
      const sourceView = new Uint8Array(packed.buffer, packed.byteOffset, packed.byteLength);
      view.set(sourceView);
    }

    this.simulateMessage(arrayBuffer);
  }
}

describe('useWebSocketSync', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    MockWebSocket.instances = [];
    // @ts-expect-error - mocking global
    global.WebSocket = MockWebSocket;
    vi.useFakeTimers();

    // Suppress console output during tests
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});

    // Mock location for URL construction
    Object.defineProperty(window, 'location', {
      value: {
        protocol: 'http:',
        host: 'localhost:3000',
      },
      writable: true,
      configurable: true,
    });

    // Reset repository mocks
    vi.mocked(messageRepository.getPending).mockReturnValue([]);
    vi.mocked(messageRepository.findByLocalId).mockReturnValue(null);
    vi.mocked(messageRepository.findByServerId).mockReturnValue(null);
    vi.mocked(messageRepository.findById).mockReturnValue(null);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('Connection Management', () => {
    it('does not connect when conversationId is null', () => {
      renderHook(() => useWebSocketSync(null));

      expect(MockWebSocket.instances).toHaveLength(0);
    });

    it('does not connect when enabled is false', () => {
      renderHook(() => useWebSocketSync('conv-1', { enabled: false }));

      expect(MockWebSocket.instances).toHaveLength(0);
    });

    it('connects when conversationId is provided and enabled', () => {
      renderHook(() => useWebSocketSync('conv-1'));

      expect(MockWebSocket.instances).toHaveLength(1);
      expect(MockWebSocket.instances[0].url).toBe(
        'ws://localhost:3000/api/v1/conversations/conv-1/sync/ws'
      );
    });

    it('sets binaryType to arraybuffer', () => {
      renderHook(() => useWebSocketSync('conv-1'));

      expect(MockWebSocket.instances[0].binaryType).toBe('arraybuffer');
    });

    it('sets isConnected to true on successful connection', () => {
      const { result } = renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      expect(result.current.isConnected).toBe(true);
    });

    it('sets isConnected to false on disconnect', () => {
      const { result } = renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      expect(result.current.isConnected).toBe(true);

      act(() => {
        MockWebSocket.instances[0].simulateClose();
      });

      expect(result.current.isConnected).toBe(false);
    });

    it('clears error on successful connection', () => {
      const { result } = renderHook(() => useWebSocketSync('conv-1'));

      // First connect
      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      // Trigger error
      act(() => {
        MockWebSocket.instances[0].simulateError();
      });

      expect(result.current.error).not.toBeNull();

      // Trigger close which schedules reconnection
      act(() => {
        MockWebSocket.instances[0].simulateClose();
      });

      // Advance timer to trigger reconnect
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Should have created a second instance
      expect(MockWebSocket.instances).toHaveLength(2);

      // Open the new connection
      act(() => {
        MockWebSocket.instances[1].simulateOpen();
      });

      expect(result.current.error).toBeNull();
    });

    it('closes connection on unmount', () => {
      const { unmount } = renderHook(() => useWebSocketSync('conv-1'));

      const ws = MockWebSocket.instances[0];
      const closeSpy = vi.spyOn(ws, 'close');

      act(() => {
        ws.simulateOpen();
      });

      unmount();

      expect(closeSpy).toHaveBeenCalled();
    });

    it('reconnects when conversationId changes', () => {
      const { rerender } = renderHook(
        ({ conversationId }) => useWebSocketSync(conversationId),
        { initialProps: { conversationId: 'conv-1' } }
      );

      expect(MockWebSocket.instances).toHaveLength(1);
      expect(MockWebSocket.instances[0].url).toContain('conv-1');

      rerender({ conversationId: 'conv-2' });

      expect(MockWebSocket.instances).toHaveLength(2);
      expect(MockWebSocket.instances[1].url).toContain('conv-2');
    });
  });

  describe('Initial Sync', () => {
    it('sends sync request with pending messages on connect', () => {
      const pendingMessages = [
        {
          id: 'local-1',
          local_id: 'local-1',
          conversation_id: 'conv-1',
          sequence_number: 1,
          previous_id: undefined,
          role: 'user' as const,
          contents: 'Hello',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          sync_status: 'pending' as const,
        },
      ];

      vi.mocked(messageRepository.getPending).mockReturnValue(pendingMessages);

      renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      expect(MockWebSocket.instances[0].sentMessages).toHaveLength(1);
    });

    it('does not send sync request when no pending messages', () => {
      vi.mocked(messageRepository.getPending).mockReturnValue([]);

      renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      expect(MockWebSocket.instances[0].sentMessages).toHaveLength(0);
    });
  });

  describe('SyncResponse Handling', () => {
    it('updates existing message with server data on sync', () => {
      const existingMessage = {
        id: 'local-1',
        local_id: 'local-1',
        sync_status: 'pending',
      };

      vi.mocked(messageRepository.findByLocalId).mockReturnValue(existingMessage as any);

      renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      const syncResponse = {
        syncedMessages: [
          {
            localId: 'local-1',
            serverId: 'server-1',
            status: 'synced',
            message: {
              id: 'server-1',
              conversationId: 'conv-1',
              sequenceNumber: 1,
              role: 'user',
              contents: 'Hello',
              createdAt: '2024-01-01T00:00:00Z',
              updatedAt: '2024-01-01T00:00:00Z',
            },
          },
        ],
        syncedAt: '2024-01-01T00:00:00Z',
      };

      act(() => {
        MockWebSocket.instances[0].simulateBinaryMessage(syncResponse);
      });

      expect(messageRepository.update).toHaveBeenCalledWith(
        'local-1',
        expect.objectContaining({
          server_id: 'server-1',
          sync_status: 'synced',
        })
      );
    });

    it('inserts new message when no local match exists', () => {
      vi.mocked(messageRepository.findByLocalId).mockReturnValue(null);

      renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      const syncResponse = {
        syncedMessages: [
          {
            localId: 'local-1',
            serverId: 'server-1',
            status: 'synced',
            message: {
              id: 'server-1',
              conversationId: 'conv-1',
              sequenceNumber: 1,
              role: 'user',
              contents: 'Hello',
              createdAt: '2024-01-01T00:00:00Z',
              updatedAt: '2024-01-01T00:00:00Z',
            },
          },
        ],
      };

      act(() => {
        MockWebSocket.instances[0].simulateBinaryMessage(syncResponse);
      });

      expect(messageRepository.insert).toHaveBeenCalled();
    });

    it('calls onSync callback after processing', () => {
      const onSync = vi.fn();
      renderHook(() => useWebSocketSync('conv-1', { onSync }));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      const syncResponse = {
        syncedMessages: [],
      };

      act(() => {
        MockWebSocket.instances[0].simulateBinaryMessage(syncResponse);
      });

      expect(onSync).toHaveBeenCalled();
    });
  });

  describe('Broadcast Message Handling', () => {
    it('inserts new message from broadcast', () => {
      vi.mocked(messageRepository.findByServerId).mockReturnValue(null);
      vi.mocked(messageRepository.findByLocalId).mockReturnValue(null);
      vi.mocked(messageRepository.findById).mockReturnValue(null);

      const onMessage = vi.fn();
      renderHook(() => useWebSocketSync('conv-1', { onMessage }));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      const broadcastMessage = {
        id: 'msg-1',
        conversationId: 'conv-1',
        sequenceNumber: 1,
        role: 'assistant',
        contents: 'Hello!',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      };

      act(() => {
        MockWebSocket.instances[0].simulateBinaryMessage(broadcastMessage);
      });

      expect(messageRepository.upsert).toHaveBeenCalled();
      expect(onMessage).toHaveBeenCalled();
    });

    it('skips duplicate when server_id already exists', () => {
      vi.mocked(messageRepository.findByServerId).mockReturnValue({ id: 'existing' } as any);

      const onMessage = vi.fn();
      renderHook(() => useWebSocketSync('conv-1', { onMessage }));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      const broadcastMessage = {
        id: 'msg-1',
        conversationId: 'conv-1',
        contents: 'Hello!',
      };

      act(() => {
        MockWebSocket.instances[0].simulateBinaryMessage(broadcastMessage);
      });

      expect(messageRepository.upsert).not.toHaveBeenCalled();
      expect(onMessage).not.toHaveBeenCalled();
    });
  });

  describe('Protocol Message Routing', () => {
    const testProtocolMessage = (dto: object, expectedType: MessageType) => {
      renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      act(() => {
        MockWebSocket.instances[0].simulateBinaryMessage(dto);
      });

      expect(handleProtocolMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          type: expectedType,
        })
      );
    };

    it('routes StartAnswer to handleProtocolMessage', () => {
      testProtocolMessage(
        {
          id: 'msg-1',
          conversationId: 'conv-1',
          previousId: 'prev-1',
          answerType: 'standard',
        },
        MessageType.StartAnswer
      );
    });

    it('routes AssistantSentence to handleProtocolMessage', () => {
      testProtocolMessage(
        {
          conversationId: 'conv-1',
          sequence: 1,
          text: 'Hello',
          previousId: 'prev-1',
        },
        MessageType.AssistantSentence
      );
    });

    it('routes ToolUseRequest to handleProtocolMessage', () => {
      testProtocolMessage(
        {
          id: 'tool-1',
          messageId: 'msg-1',
          toolName: 'search',
          parameters: { query: 'test' },
        },
        MessageType.ToolUseRequest
      );
    });

    it('routes ToolUseResult to handleProtocolMessage', () => {
      testProtocolMessage(
        {
          requestId: 'tool-1',
          success: true,
        },
        MessageType.ToolUseResult
      );
    });

    it('routes MemoryTrace to handleProtocolMessage', () => {
      testProtocolMessage(
        {
          memoryId: 'mem-1',
          messageId: 'msg-1',
          content: 'Memory content',
          relevance: 0.9,
        },
        MessageType.MemoryTrace
      );
    });
  });

  describe('Send Functionality', () => {
    it('sends packed envelope body when connected', () => {
      const { result } = renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      const envelope = {
        stanzaId: 1,
        conversationId: 'conv-1',
        type: MessageType.SyncRequest,
        body: { messages: [] },
      };

      act(() => {
        result.current.send(envelope);
      });

      expect(MockWebSocket.instances[0].sentMessages).toHaveLength(1);
    });

    it('logs warning when not connected', () => {
      const { result } = renderHook(() => useWebSocketSync('conv-1'));

      // Don't open the connection

      const envelope = {
        stanzaId: 1,
        conversationId: 'conv-1',
        type: MessageType.SyncRequest,
        body: { messages: [] },
      };

      act(() => {
        result.current.send(envelope);
      });

      expect(console.warn).toHaveBeenCalledWith(
        'WebSocket not connected, cannot send message'
      );
    });
  });

  describe('syncNow Functionality', () => {
    it('sends sync request for pending messages', () => {
      const pendingMessages = [
        {
          id: 'local-1',
          local_id: 'local-1',
          conversation_id: 'conv-1',
          sequence_number: 1,
          previous_id: undefined,
          role: 'user' as const,
          contents: 'Hello',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ];

      vi.mocked(messageRepository.getPending).mockReturnValue(pendingMessages);

      const { result } = renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      // Clear initial sync message
      MockWebSocket.instances[0].sentMessages = [];

      act(() => {
        result.current.syncNow();
      });

      expect(MockWebSocket.instances[0].sentMessages).toHaveLength(1);
    });

    it('does nothing when no pending messages', () => {
      vi.mocked(messageRepository.getPending).mockReturnValue([]);

      const { result } = renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      // Clear any initial messages
      MockWebSocket.instances[0].sentMessages = [];

      act(() => {
        result.current.syncNow();
      });

      expect(MockWebSocket.instances[0].sentMessages).toHaveLength(0);
    });
  });

  describe('Error Handling', () => {
    it('sets error state on WebSocket error', () => {
      const { result } = renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateError();
      });

      expect(result.current.error).not.toBeNull();
      expect(result.current.error?.message).toBe('WebSocket connection error');
    });

    it('handles parse errors in onmessage gracefully', () => {
      renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      // Send invalid data
      act(() => {
        MockWebSocket.instances[0].simulateMessage(new ArrayBuffer(0));
      });

      expect(console.error).toHaveBeenCalledWith(
        'Failed to parse WebSocket message:',
        expect.any(Error)
      );
    });
  });

  describe('Reconnection with Exponential Backoff', () => {
    it('attempts reconnect after unexpected close', async () => {
      renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      act(() => {
        MockWebSocket.instances[0].simulateClose();
      });

      expect(MockWebSocket.instances).toHaveLength(1);

      // Advance timer for first reconnect (1s)
      await act(async () => {
        vi.advanceTimersByTime(1000);
      });

      expect(MockWebSocket.instances).toHaveLength(2);
    });

    it('uses exponential backoff with increasing delays', async () => {
      renderHook(() => useWebSocketSync('conv-1'));

      // First connection + close
      act(() => {
        MockWebSocket.instances[0].simulateOpen();
        MockWebSocket.instances[0].simulateClose();
      });

      // First reconnect at 1s
      await act(async () => {
        vi.advanceTimersByTime(1000);
      });
      expect(MockWebSocket.instances).toHaveLength(2);

      // Second close
      act(() => {
        MockWebSocket.instances[1].simulateClose();
      });

      // Second reconnect at 2s (2^1 * 1000)
      await act(async () => {
        vi.advanceTimersByTime(1000);
      });
      expect(MockWebSocket.instances).toHaveLength(2); // Not yet

      await act(async () => {
        vi.advanceTimersByTime(1000);
      });
      expect(MockWebSocket.instances).toHaveLength(3);
    });

    it('resets backoff counter on successful connect', async () => {
      const { result } = renderHook(() => useWebSocketSync('conv-1'));

      // Connect and close multiple times to increase backoff
      act(() => {
        MockWebSocket.instances[0].simulateOpen();
        MockWebSocket.instances[0].simulateClose();
      });

      await act(async () => {
        vi.advanceTimersByTime(1000);
      });

      // Successful connect resets counter
      act(() => {
        MockWebSocket.instances[1].simulateOpen();
      });

      expect(result.current.isConnected).toBe(true);
    });

    it('does not reconnect during intentional cleanup', async () => {
      const { unmount } = renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      // Unmount triggers cleanup
      unmount();

      // Advance timers - should not create new connection
      await act(async () => {
        vi.advanceTimersByTime(30000);
      });

      expect(MockWebSocket.instances).toHaveLength(1);
    });

    it('clears reconnect timeout on unmount', async () => {
      const { unmount } = renderHook(() => useWebSocketSync('conv-1'));

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
        MockWebSocket.instances[0].simulateClose();
      });

      // Unmount before reconnect
      unmount();

      const instanceCount = MockWebSocket.instances.length;

      // Advance timer - should not create new connection
      await act(async () => {
        vi.advanceTimersByTime(5000);
      });

      expect(MockWebSocket.instances.length).toBe(instanceCount);
    });
  });

  describe('Callback Stability', () => {
    it('does not reconnect when callbacks change', () => {
      const { rerender } = renderHook(
        ({ onMessage }) => useWebSocketSync('conv-1', { onMessage }),
        { initialProps: { onMessage: vi.fn() } }
      );

      expect(MockWebSocket.instances).toHaveLength(1);

      // Change callback
      rerender({ onMessage: vi.fn() });

      // Should not create new connection
      expect(MockWebSocket.instances).toHaveLength(1);
    });

    it('uses latest onMessage callback', () => {
      const firstCallback = vi.fn();
      const secondCallback = vi.fn();

      vi.mocked(messageRepository.findByServerId).mockReturnValue(null);
      vi.mocked(messageRepository.findByLocalId).mockReturnValue(null);
      vi.mocked(messageRepository.findById).mockReturnValue(null);

      const { rerender } = renderHook(
        ({ onMessage }) => useWebSocketSync('conv-1', { onMessage }),
        { initialProps: { onMessage: firstCallback } }
      );

      act(() => {
        MockWebSocket.instances[0].simulateOpen();
      });

      // Update callback
      rerender({ onMessage: secondCallback });

      // Simulate message
      const broadcastMessage = {
        id: 'msg-1',
        conversationId: 'conv-1',
        contents: 'Hello!',
      };

      act(() => {
        MockWebSocket.instances[0].simulateBinaryMessage(broadcastMessage);
      });

      expect(firstCallback).not.toHaveBeenCalled();
      expect(secondCallback).toHaveBeenCalled();
    });
  });
});
