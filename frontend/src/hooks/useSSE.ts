import { useEffect, useRef, useCallback, useState } from 'react';
import { Message } from '../types/models';

interface SSEEvent {
  type: 'connected' | 'message' | 'sync';
  conversation_id?: string;
  message?: Message;
}

interface UseSSEOptions {
  onMessage?: (message: Message) => void;
  onSync?: () => void;
  onError?: (error: Error) => void;
  enabled?: boolean;
}

interface UseSSEResult {
  isConnected: boolean;
  error: Error | null;
  reconnect: () => void;
}

const RECONNECT_DELAY_MS = 3000; // 3 seconds
const MAX_RECONNECT_DELAY_MS = 30000; // 30 seconds max backoff

export function useSSE(
  conversationId: string | null,
  options: UseSSEOptions = {}
): UseSSEResult {
  const {
    onMessage,
    onSync,
    onError,
    enabled = true,
  } = options;

  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectDelayRef = useRef(RECONNECT_DELAY_MS);
  const isMountedRef = useRef(true);
  const conversationIdRef = useRef<string | null>(null);

  // Update refs when values change
  useEffect(() => {
    conversationIdRef.current = conversationId;
  }, [conversationId]);

  // Cleanup function
  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  // Connect to SSE endpoint
  const connect = useCallback(() => {
    const currentConversationId = conversationIdRef.current;

    if (!currentConversationId || !enabled) {
      return;
    }

    // Close existing connection
    cleanup();

    try {
      const url = `/api/v1/conversations/${currentConversationId}/events`;
      const eventSource = new EventSource(url);

      eventSource.onopen = () => {
        if (!isMountedRef.current) return;

        setIsConnected(true);
        setError(null);
        reconnectDelayRef.current = RECONNECT_DELAY_MS; // Reset backoff on success
        console.log(`SSE: Connected to conversation ${currentConversationId}`);
      };

      eventSource.onmessage = (event) => {
        if (!isMountedRef.current) return;

        try {
          const data: SSEEvent = JSON.parse(event.data);

          switch (data.type) {
            case 'connected':
              console.log('SSE: Connection confirmed');
              break;

            case 'message':
              if (data.message && onMessage) {
                onMessage(data.message);
              }
              break;

            case 'sync':
              if (onSync) {
                onSync();
              }
              break;

            default:
              console.warn('SSE: Unknown event type:', data.type);
          }
        } catch (err) {
          console.error('SSE: Failed to parse event data:', err);
        }
      };

      eventSource.onerror = (err) => {
        if (!isMountedRef.current) return;

        console.error('SSE: Connection error', err);

        setIsConnected(false);
        const errorObj = new Error('SSE connection failed');
        setError(errorObj);

        if (onError) {
          onError(errorObj);
        }

        // Close and attempt reconnect with exponential backoff
        eventSource.close();
        eventSourceRef.current = null;

        const delay = reconnectDelayRef.current;
        console.log(`SSE: Reconnecting in ${delay}ms...`);

        reconnectTimeoutRef.current = setTimeout(() => {
          if (isMountedRef.current && conversationIdRef.current === currentConversationId) {
            reconnectDelayRef.current = Math.min(
              reconnectDelayRef.current * 2,
              MAX_RECONNECT_DELAY_MS
            );
            connect();
          }
        }, delay);
      };

      eventSourceRef.current = eventSource;
    } catch (err) {
      console.error('SSE: Failed to create EventSource:', err);
      const errorObj = err instanceof Error ? err : new Error('Failed to create SSE connection');
      setError(errorObj);
      if (onError) {
        onError(errorObj);
      }
    }
  }, [enabled, cleanup, onMessage, onSync, onError]);

  // Manual reconnect function
  const reconnect = useCallback(() => {
    cleanup();
    reconnectDelayRef.current = RECONNECT_DELAY_MS; // Reset backoff
    connect();
  }, [cleanup, connect]);

  // Connect when conversation changes or enabled state changes
  useEffect(() => {
    if (conversationId && enabled) {
      connect();
    } else {
      cleanup();
      setIsConnected(false);
      setError(null);
    }

    return cleanup;
  }, [conversationId, enabled, connect, cleanup]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      isMountedRef.current = false;
      cleanup();
    };
  }, [cleanup]);

  return {
    isConnected,
    error,
    reconnect,
  };
}
