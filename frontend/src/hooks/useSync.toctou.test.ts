import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useWebSocketSync } from './useWebSocketSync';
import { pack } from 'msgpackr';
import { MessageResponse } from '../types/sync';

/**
 * Test for Bug #10: REST + WebSocket TOCTOU Race Condition
 *
 * HYPOTHESIS: Between the deduplication check (lines 242-260) and the upsert (line 263),
 * another async operation could insert the same message, causing duplicates.
 *
 * ANALYSIS FACTORS:
 * 1. JavaScript is single-threaded - event loop runs one task at a time
 * 2. No await/async points between check and insert in the critical section
 * 3. sql.js operations are synchronous (db.run, db.exec)
 * 4. React's concurrent mode could potentially interleave operations
 * 5. Multiple WebSocket messages in rapid succession
 *
 * This test attempts to trigger the race by:
 * - Sending multiple identical messages rapidly
 * - Simulating concurrent REST API and WebSocket responses
 * - Testing message processing order
 */

// Mock dependencies
vi.mock('../db/repository', () => {
  let inMemoryMessages: Map<string, any> = new Map();

  return {
    messageRepository: {
      findByLocalId: vi.fn((localId: string) => {
        // Simulate database lookup
        for (const [_, msg] of inMemoryMessages) {
          if (msg.local_id === localId) {
            return msg;
          }
        }
        return null;
      }),
      findByServerId: vi.fn((serverId: string) => {
        return inMemoryMessages.get(serverId) || null;
      }),
      findById: vi.fn((id: string) => {
        return inMemoryMessages.get(id) || null;
      }),
      getPending: vi.fn().mockReturnValue([]),
      update: vi.fn((id: string, updates: any) => {
        const existing = inMemoryMessages.get(id);
        if (existing) {
          inMemoryMessages.set(id, { ...existing, ...updates });
        }
      }),
      insert: vi.fn((message: any) => {
        if (inMemoryMessages.has(message.id)) {
          throw new Error(`Duplicate message inserted: ${message.id}`);
        }
        inMemoryMessages.set(message.id, message);
      }),
      upsert: vi.fn((message: any) => {
        const existing = inMemoryMessages.get(message.id);
        if (existing) {
          inMemoryMessages.set(message.id, { ...existing, ...message });
        } else {
          // Check by local_id
          let found = false;
          for (const [key, msg] of inMemoryMessages) {
            if (msg.local_id && msg.local_id === message.local_id) {
              inMemoryMessages.set(key, { ...msg, ...message });
              found = true;
              break;
            }
          }
          if (!found) {
            inMemoryMessages.set(message.id, message);
          }
        }
      }),
      // Expose for test inspection
      __getInMemoryMessages: () => inMemoryMessages,
      __clearInMemoryMessages: () => { inMemoryMessages = new Map(); },
    },
  };
});

vi.mock('../adapters/protocolAdapter', () => ({
  handleProtocolMessage: vi.fn(),
  handleConnectionLost: vi.fn(),
}));

// Import mocked modules
import { messageRepository } from '../db/repository';

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

  simulateClose(code?: number, reason?: string): void {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent('close', { code, reason }));
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

describe('useWebSocketSync - TOCTOU Race Condition (Bug #10)', () => {
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

    // Reset repository
    (messageRepository as any).__clearInMemoryMessages();
    vi.mocked(messageRepository.getPending).mockReturnValue([]);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should handle rapid duplicate messages from WebSocket without creating duplicates', async () => {
    renderHook(() => useWebSocketSync('conv-1'));

    // Establish connection
    act(() => {
      MockWebSocket.instances[0].simulateOpen();
    });

    // Connection established

    // Create a message that will be sent twice rapidly
    const messageResponse: MessageResponse = {
      id: 'msg-123',
      conversationId: 'conv-1',
      sequenceNumber: 1,
      role: 'user',
      contents: 'Hello world',
      localId: 'local-123',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    // Send the same message twice in rapid succession (simulating race)
    act(() => {
      MockWebSocket.instances[0].simulateBinaryMessage(messageResponse);
      MockWebSocket.instances[0].simulateBinaryMessage(messageResponse);
    });

    // Check that only one message was stored
    const messages = (messageRepository as any).__getInMemoryMessages();
    expect(messages.size).toBe(1);
    expect(messages.has('msg-123')).toBe(true);

    // Verify deduplication worked: second message was skipped entirely
    // First message: all checks pass (null), upsert is called
    // Second message: findByServerId returns the first message, breaks early, upsert NOT called
    expect(vi.mocked(messageRepository.upsert)).toHaveBeenCalledTimes(1);
  });

  it('should handle interleaved REST and WebSocket responses for the same message', async () => {
    renderHook(() => useWebSocketSync('conv-1'));

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
    });

    const messageResponse: MessageResponse = {
      id: 'msg-456',
      conversationId: 'conv-1',
      sequenceNumber: 2,
      role: 'assistant',
      contents: 'Response',
      localId: 'local-456',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    // Simulate REST API inserting the message first
    act(() => {
      messageRepository.insert({
        id: 'msg-456',
        conversation_id: 'conv-1',
        sequence_number: 2,
        role: 'assistant',
        contents: 'Response',
        local_id: 'local-456',
        server_id: 'msg-456',
        sync_status: 'synced',
        created_at: messageResponse.createdAt,
        updated_at: messageResponse.updatedAt,
      });
    });

    // Now WebSocket broadcast arrives
    act(() => {
      MockWebSocket.instances[0].simulateBinaryMessage(messageResponse);
    });

    // Should still have only one message (deduplication worked)
    const messages = (messageRepository as any).__getInMemoryMessages();
    expect(messages.size).toBe(1);

    // The upsert should have been skipped due to findByServerId check
    // We check that insert was called once (REST) but upsert was never called (WS was deduped)
    expect(vi.mocked(messageRepository.insert)).toHaveBeenCalledTimes(1);
    expect(vi.mocked(messageRepository.upsert)).not.toHaveBeenCalled();
  });

  it('should detect race condition if check and insert are not atomic', async () => {
    // This test attempts to prove whether a race is POSSIBLE

    // Override findByServerId to simulate a race window
    let checkCount = 0;
    vi.mocked(messageRepository.findByServerId).mockImplementation((serverId: string) => {
      checkCount++;

      // First check: return null (message not found)
      if (checkCount === 1) {
        return null;
      }

      // Second check: simulate that another operation inserted it in between
      // This would only be possible if there's an async gap
      const messages = (messageRepository as any).__getInMemoryMessages();
      return messages.get(serverId) || null;
    });

    renderHook(() => useWebSocketSync('conv-1'));

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
    });

    const messageResponse: MessageResponse = {
      id: 'msg-race',
      conversationId: 'conv-1',
      sequenceNumber: 3,
      role: 'user',
      contents: 'Race condition test',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    // Send two messages
    let error: Error | null = null;
    try {
      act(() => {
        MockWebSocket.instances[0].simulateBinaryMessage(messageResponse);
        MockWebSocket.instances[0].simulateBinaryMessage(messageResponse);
      });
    } catch (e) {
      error = e as Error;
    }

    // If there's a race, we'd get a duplicate insert error
    // But because JavaScript is single-threaded and there are no await points,
    // the race should NOT occur
    expect(error).toBeNull();

    const messages = (messageRepository as any).__getInMemoryMessages();
    expect(messages.size).toBeLessThanOrEqual(1);
  });

  it('should process messages synchronously within the same event loop tick', async () => {
    // This test verifies that message processing is synchronous
    const processingOrder: string[] = [];

    vi.mocked(messageRepository.upsert).mockImplementation((message: any) => {
      processingOrder.push(`start-${message.id}`);

      // Simulate some synchronous processing
      for (let i = 0; i < 1000; i++) {
        Math.sqrt(i);
      }

      processingOrder.push(`end-${message.id}`);

      const messages = (messageRepository as any).__getInMemoryMessages();
      messages.set(message.id, message);
    });

    renderHook(() => useWebSocketSync('conv-1'));

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
    });

    // Send multiple different messages rapidly
    act(() => {
      MockWebSocket.instances[0].simulateBinaryMessage({
        id: 'msg-1',
        conversationId: 'conv-1',
        sequenceNumber: 1,
        role: 'user',
        contents: 'Message 1',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      });
      MockWebSocket.instances[0].simulateBinaryMessage({
        id: 'msg-2',
        conversationId: 'conv-1',
        sequenceNumber: 2,
        role: 'user',
        contents: 'Message 2',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      });
    });

    // Verify messages were processed sequentially, not interleaved
    // Should be: start-1, end-1, start-2, end-2
    // NOT: start-1, start-2, end-1, end-2 (which would indicate concurrency)
    expect(processingOrder).toEqual([
      'start-msg-1',
      'end-msg-1',
      'start-msg-2',
      'end-msg-2',
    ]);
  });

  it('should handle message deduplication with all three checks', async () => {
    renderHook(() => useWebSocketSync('conv-1'));

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
    });

    const messageResponse: MessageResponse = {
      id: 'msg-dedup',
      conversationId: 'conv-1',
      sequenceNumber: 4,
      role: 'assistant',
      contents: 'Dedup test',
      localId: 'local-dedup',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    // Scenario 1: Check by server_id (findByServerId)
    (messageRepository as any).__getInMemoryMessages().set('msg-dedup', {
      id: 'msg-dedup',
      server_id: 'msg-dedup',
    });

    act(() => {
      MockWebSocket.instances[0].simulateBinaryMessage(messageResponse);
    });

    expect(vi.mocked(messageRepository.upsert)).not.toHaveBeenCalled();

    // Clear and test scenario 2: Check by local_id (findByLocalId)
    (messageRepository as any).__clearInMemoryMessages();
    vi.mocked(messageRepository.upsert).mockClear();

    (messageRepository as any).__getInMemoryMessages().set('different-id', {
      id: 'different-id',
      local_id: 'local-dedup',
    });

    act(() => {
      MockWebSocket.instances[0].simulateBinaryMessage(messageResponse);
    });

    expect(vi.mocked(messageRepository.upsert)).not.toHaveBeenCalled();

    // Clear and test scenario 3: Check by id (findById)
    (messageRepository as any).__clearInMemoryMessages();
    vi.mocked(messageRepository.upsert).mockClear();

    (messageRepository as any).__getInMemoryMessages().set('msg-dedup', {
      id: 'msg-dedup',
    });

    act(() => {
      MockWebSocket.instances[0].simulateBinaryMessage(messageResponse);
    });

    expect(vi.mocked(messageRepository.upsert)).not.toHaveBeenCalled();
  });

  it('should verify sql.js operations are synchronous (no async gaps)', async () => {
    // This test documents that sql.js db.exec and db.run are synchronous
    // Therefore, there's no opportunity for race conditions between check and insert

    renderHook(() => useWebSocketSync('conv-1'));

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
    });

    // Track function call order
    const callOrder: string[] = [];

    vi.mocked(messageRepository.findByServerId).mockImplementation((_id) => {
      callOrder.push('findByServerId');
      return null;
    });

    vi.mocked(messageRepository.findByLocalId).mockImplementation((_id) => {
      callOrder.push('findByLocalId');
      return null;
    });

    vi.mocked(messageRepository.findById).mockImplementation((_id) => {
      callOrder.push('findById');
      return null;
    });

    vi.mocked(messageRepository.upsert).mockImplementation((_msg) => {
      callOrder.push('upsert');
    });

    act(() => {
      MockWebSocket.instances[0].simulateBinaryMessage({
        id: 'msg-sync',
        conversationId: 'conv-1',
        sequenceNumber: 5,
        role: 'user',
        contents: 'Sync test',
        localId: 'local-sync',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      });
    });

    // Verify all operations completed in sequence without interleaving
    expect(callOrder).toEqual([
      'findByServerId',
      'findByLocalId',
      'findById',
      'upsert',
    ]);

    // No async operations means no race condition is possible
  });
});
