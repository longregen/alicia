import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useSSE } from './useSSE';

// Mock EventSource
class MockEventSource {
  static instances: MockEventSource[] = [];

  url: string;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  readyState: number = 0;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  close() {
    this.readyState = 2;
  }

  // Helper to simulate events
  simulateOpen() {
    this.readyState = 1;
    if (this.onopen) {
      this.onopen(new Event('open'));
    }
  }

  simulateMessage(data: unknown) {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', { data: JSON.stringify(data) }));
    }
  }

  simulateError() {
    if (this.onerror) {
      this.onerror(new Event('error'));
    }
  }
}

describe('useSSE', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    MockEventSource.instances = [];
    // @ts-expect-error - mocking global
    global.EventSource = MockEventSource;
    vi.useFakeTimers();

    // Suppress console output during tests
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('Connection Management', () => {
    it('does not connect when conversationId is null', () => {
      renderHook(() => useSSE(null));

      expect(MockEventSource.instances).toHaveLength(0);
    });

    it('does not connect when enabled is false', () => {
      renderHook(() => useSSE('conv-1', { enabled: false }));

      expect(MockEventSource.instances).toHaveLength(0);
    });

    it('connects when conversationId is provided and enabled', () => {
      renderHook(() => useSSE('conv-1'));

      expect(MockEventSource.instances).toHaveLength(1);
      expect(MockEventSource.instances[0].url).toBe('/api/v1/conversations/conv-1/events');
    });

    it('sets isConnected to true on successful connection', async () => {
      const { result } = renderHook(() => useSSE('conv-1'));

      expect(result.current.isConnected).toBe(false);

      act(() => {
        MockEventSource.instances[0].simulateOpen();
      });

      expect(result.current.isConnected).toBe(true);
    });

    it('closes connection on unmount', () => {
      const { unmount } = renderHook(() => useSSE('conv-1'));

      const eventSource = MockEventSource.instances[0];
      const closeSpy = vi.spyOn(eventSource, 'close');

      unmount();

      expect(closeSpy).toHaveBeenCalled();
    });

    it('reconnects when conversationId changes', () => {
      const { rerender } = renderHook(
        ({ conversationId }) => useSSE(conversationId),
        { initialProps: { conversationId: 'conv-1' } }
      );

      expect(MockEventSource.instances).toHaveLength(1);
      expect(MockEventSource.instances[0].url).toContain('conv-1');

      rerender({ conversationId: 'conv-2' });

      expect(MockEventSource.instances).toHaveLength(2);
      expect(MockEventSource.instances[1].url).toContain('conv-2');
    });
  });

  describe('Message Handling', () => {
    it('calls onMessage when message event is received', () => {
      const onMessage = vi.fn();
      renderHook(() => useSSE('conv-1', { onMessage }));

      act(() => {
        MockEventSource.instances[0].simulateOpen();
      });

      const mockMessage = { id: 'msg-1', content: 'Hello' };
      act(() => {
        MockEventSource.instances[0].simulateMessage({
          type: 'message',
          message: mockMessage,
        });
      });

      expect(onMessage).toHaveBeenCalledWith(mockMessage);
    });

    it('calls onSync when sync event is received', () => {
      const onSync = vi.fn();
      renderHook(() => useSSE('conv-1', { onSync }));

      act(() => {
        MockEventSource.instances[0].simulateOpen();
      });

      act(() => {
        MockEventSource.instances[0].simulateMessage({ type: 'sync' });
      });

      expect(onSync).toHaveBeenCalled();
    });

    it('handles connected event without error', () => {
      renderHook(() => useSSE('conv-1'));

      act(() => {
        MockEventSource.instances[0].simulateOpen();
      });

      act(() => {
        MockEventSource.instances[0].simulateMessage({ type: 'connected' });
      });

      expect(console.log).toHaveBeenCalledWith('SSE: Connection confirmed');
    });

    it('logs warning for unknown event types', () => {
      renderHook(() => useSSE('conv-1'));

      act(() => {
        MockEventSource.instances[0].simulateOpen();
      });

      act(() => {
        MockEventSource.instances[0].simulateMessage({ type: 'unknown' });
      });

      expect(console.warn).toHaveBeenCalledWith('SSE: Unknown event type:', 'unknown');
    });
  });

  describe('Error Handling', () => {
    it('sets error state on connection error', () => {
      const { result } = renderHook(() => useSSE('conv-1'));

      act(() => {
        MockEventSource.instances[0].simulateError();
      });

      expect(result.current.isConnected).toBe(false);
      expect(result.current.error).toBeInstanceOf(Error);
      expect(result.current.error?.message).toBe('SSE connection failed');
    });

    it('calls onError callback on error', () => {
      const onError = vi.fn();
      renderHook(() => useSSE('conv-1', { onError }));

      act(() => {
        MockEventSource.instances[0].simulateError();
      });

      expect(onError).toHaveBeenCalledWith(expect.any(Error));
    });

    it('attempts reconnection with exponential backoff', async () => {
      renderHook(() => useSSE('conv-1'));

      // First connection
      expect(MockEventSource.instances).toHaveLength(1);

      // Simulate error
      act(() => {
        MockEventSource.instances[0].simulateError();
      });

      // Should schedule reconnect
      expect(MockEventSource.instances).toHaveLength(1);

      // Advance timer for first reconnect (3 seconds)
      await act(async () => {
        vi.advanceTimersByTime(3000);
      });

      expect(MockEventSource.instances).toHaveLength(2);
    });

    it('clears error on successful reconnection', async () => {
      const { result } = renderHook(() => useSSE('conv-1'));

      // Simulate error
      act(() => {
        MockEventSource.instances[0].simulateError();
      });

      expect(result.current.error).not.toBeNull();

      // Advance timer for reconnect
      await act(async () => {
        vi.advanceTimersByTime(3000);
      });

      // Simulate successful connection on reconnect
      act(() => {
        MockEventSource.instances[1].simulateOpen();
      });

      expect(result.current.isConnected).toBe(true);
      expect(result.current.error).toBeNull();
    });
  });

  describe('Manual Reconnect', () => {
    it('provides reconnect function', () => {
      const { result } = renderHook(() => useSSE('conv-1'));

      expect(typeof result.current.reconnect).toBe('function');
    });

    it('reconnect creates new connection', () => {
      const { result } = renderHook(() => useSSE('conv-1'));

      expect(MockEventSource.instances).toHaveLength(1);

      act(() => {
        result.current.reconnect();
      });

      expect(MockEventSource.instances).toHaveLength(2);
    });

    it('reconnect resets backoff delay', async () => {
      const { result } = renderHook(() => useSSE('conv-1'));

      // Simulate multiple errors to increase backoff
      act(() => {
        MockEventSource.instances[0].simulateError();
      });

      await act(async () => {
        vi.advanceTimersByTime(3000);
      });

      act(() => {
        MockEventSource.instances[1].simulateError();
      });

      // Manual reconnect should reset the delay
      act(() => {
        result.current.reconnect();
      });

      // New connection should be created immediately
      expect(MockEventSource.instances.length).toBeGreaterThan(2);
    });
  });

  describe('Cleanup', () => {
    it('cleans up connection when disabled changes to false', () => {
      const { result, rerender } = renderHook(
        ({ enabled }) => useSSE('conv-1', { enabled }),
        { initialProps: { enabled: true } }
      );

      act(() => {
        MockEventSource.instances[0].simulateOpen();
      });

      expect(result.current.isConnected).toBe(true);

      rerender({ enabled: false });

      expect(result.current.isConnected).toBe(false);
    });

    it('cleans up connection when conversationId becomes null', () => {
      const { result, rerender } = renderHook(
        ({ conversationId }) => useSSE(conversationId),
        { initialProps: { conversationId: 'conv-1' as string | null } }
      );

      act(() => {
        MockEventSource.instances[0].simulateOpen();
      });

      expect(result.current.isConnected).toBe(true);

      rerender({ conversationId: null });

      expect(result.current.isConnected).toBe(false);
      expect(result.current.error).toBeNull();
    });

    it('clears reconnect timeout on cleanup', () => {
      const { unmount } = renderHook(() => useSSE('conv-1'));

      // Simulate error to trigger reconnect timeout
      act(() => {
        MockEventSource.instances[0].simulateError();
      });

      // Unmount before reconnect
      unmount();

      // Advance timer - should not create new connection
      const instanceCount = MockEventSource.instances.length;
      act(() => {
        vi.advanceTimersByTime(3000);
      });

      expect(MockEventSource.instances.length).toBe(instanceCount);
    });
  });

  describe('Callback Stability', () => {
    it('does not reconnect when callbacks change', () => {
      const { rerender } = renderHook(
        ({ onMessage }) => useSSE('conv-1', { onMessage }),
        { initialProps: { onMessage: vi.fn() } }
      );

      expect(MockEventSource.instances).toHaveLength(1);

      // Change callback
      rerender({ onMessage: vi.fn() });

      // Should not create new connection
      expect(MockEventSource.instances).toHaveLength(1);
    });

    it('uses latest callback when event is received', () => {
      const firstCallback = vi.fn();
      const secondCallback = vi.fn();

      const { rerender } = renderHook(
        ({ onMessage }) => useSSE('conv-1', { onMessage }),
        { initialProps: { onMessage: firstCallback } }
      );

      act(() => {
        MockEventSource.instances[0].simulateOpen();
      });

      // Update callback
      rerender({ onMessage: secondCallback });

      // Simulate message
      const mockMessage = { id: 'msg-1', content: 'Hello' };
      act(() => {
        MockEventSource.instances[0].simulateMessage({
          type: 'message',
          message: mockMessage,
        });
      });

      expect(firstCallback).not.toHaveBeenCalled();
      expect(secondCallback).toHaveBeenCalledWith(mockMessage);
    });
  });
});
