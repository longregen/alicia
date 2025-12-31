/**
 * Bug #3 Verification: WebSocket reconnection loses streaming state
 *
 * This test file validates the hypothesis that when WebSocket disconnects
 * during streaming, there is no mechanism to resume or recover the streaming state.
 */

import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useWebSocketSync } from './useWebSocketSync';
import { useConversationStore } from '../stores/conversationStore';
import { createMessageId, MessageStatus } from '../types/streaming';

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

vi.mock('../adapters/protocolAdapter', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../adapters/protocolAdapter')>();
  return {
    ...actual,
    handleProtocolMessage: vi.fn(),
  };
});

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

  simulateOpen(): void {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event('open'));
  }

  simulateClose(): void {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent('close'));
  }
}

describe('Bug #3: WebSocket Reconnection Loses Streaming State', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    MockWebSocket.instances = [];
    // @ts-expect-error - mocking global
    global.WebSocket = MockWebSocket;
    vi.useFakeTimers();

    // Clear conversation store state between tests
    useConversationStore.getState().clearConversation();

    // Suppress console
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});

    Object.defineProperty(window, 'location', {
      value: {
        protocol: 'http:',
        host: 'localhost:3000',
      },
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('streaming state is cleaned up when WebSocket disconnects', async () => {
    // Setup: Connect WebSocket
    renderHook(() => useWebSocketSync('conv-1'));

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
    });

    // Simulate a message starting to stream (would be set by protocolAdapter)
    const messageId = createMessageId('msg-streaming');
    useConversationStore.getState().setCurrentStreamingMessageId(messageId);
    useConversationStore.getState().addMessage({
      id: messageId,
      conversationId: 'conv-1' as any,
      role: 'assistant',
      content: 'This is partial content that was being streamed...',
      status: MessageStatus.Streaming,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    });

    // BEFORE disconnect: Verify streaming state exists
    const beforeDisconnect = {
      currentStreamingMessageId: useConversationStore.getState().currentStreamingMessageId,
      messageStatus: useConversationStore.getState().messages[messageId]?.status,
    };

    expect(beforeDisconnect.currentStreamingMessageId).toBe(messageId);
    expect(beforeDisconnect.messageStatus).toBe(MessageStatus.Streaming);

    // ACTION: Disconnect during streaming
    act(() => {
      MockWebSocket.instances[0].simulateClose();
    });

    // AFTER disconnect: Check what happened to streaming state
    const afterDisconnect = {
      currentStreamingMessageId: useConversationStore.getState().currentStreamingMessageId,
      messageStatus: useConversationStore.getState().messages[messageId]?.status,
      wsConnectionCount: MockWebSocket.instances.length,
    };

    // FIX VERIFICATION 1: Streaming state is cleared and message marked as error
    expect(afterDisconnect.currentStreamingMessageId).toBeNull();
    expect(afterDisconnect.messageStatus).toBe(MessageStatus.Error);

    // Wait for reconnection
    await act(async () => {
      vi.advanceTimersByTime(1000);
    });

    act(() => {
      MockWebSocket.instances[1].simulateOpen();
    });

    // AFTER reconnect: Check if streaming state remains clean
    const afterReconnect = {
      currentStreamingMessageId: useConversationStore.getState().currentStreamingMessageId,
      messageStatus: useConversationStore.getState().messages[messageId]?.status,
      wsConnectionCount: MockWebSocket.instances.length,
    };

    // FIX VERIFICATION 2: WebSocket reconnected and streaming state is clean
    expect(afterReconnect.wsConnectionCount).toBe(2); // New connection created
    expect(afterReconnect.currentStreamingMessageId).toBeNull(); // Cleared
    expect(afterReconnect.messageStatus).toBe(MessageStatus.Error); // Marked as error

    // FIX VERIFICATION 3: The onclose handler now:
    // - Detects orphaned streaming messages
    // - Marks the message as error (stream was interrupted)
    // - Clears currentStreamingMessageId
    // - Prevents UI from showing perpetual loading state
  });

  it('onclose handler now cleans up streaming state', async () => {
    renderHook(() => useWebSocketSync('conv-1'));

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
    });

    // Set up streaming message
    const messageId = createMessageId('msg-orphaned');
    useConversationStore.getState().setCurrentStreamingMessageId(messageId);
    useConversationStore.getState().addMessage({
      id: messageId,
      conversationId: 'conv-1' as any,
      role: 'assistant',
      content: '',
      status: MessageStatus.Streaming,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    });

    // Disconnect
    act(() => {
      MockWebSocket.instances[0].simulateClose();
    });

    // FIX: The onclose handler now:
    // 1. Checks if there's a currentStreamingMessageId
    // 2. Clears it and marks the message as error
    // 3. Prevents orphaned streaming state

    // Verification: Streaming state is cleaned up after close
    expect(useConversationStore.getState().currentStreamingMessageId).toBeNull();
    expect(useConversationStore.getState().messages[messageId].status).toBe(MessageStatus.Error);

    // The onclose handler now handles BOTH reconnection logic AND streaming state cleanup
  });

});
