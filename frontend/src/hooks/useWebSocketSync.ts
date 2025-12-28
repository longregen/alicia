import { useEffect, useRef, useState, useCallback } from 'react';
import { pack, unpack } from 'msgpackr';
import { Message } from '../types/models';
import { messageRepository } from '../db/repository';

interface SyncEnvelope {
  type: 'sync_request' | 'sync_response' | 'message' | 'ack';
  payload: unknown;
}

interface SyncResponse {
  messages: Message[];
}

interface MessagePayload {
  message: Message;
}

export interface UseWebSocketSyncOptions {
  onMessage?: (message: Message) => void;
  onSync?: () => void;
  enabled?: boolean;
}

export function useWebSocketSync(
  conversationId: string | null,
  options: UseWebSocketSyncOptions = {}
) {
  const { onMessage, onSync, enabled = true } = options;
  const wsRef = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  // Track intentional closure to prevent reconnect on cleanup
  const isCleaningUpRef = useRef(false);

  // Store callbacks in refs to avoid recreating handleEnvelope/connect on every render
  const onMessageRef = useRef(onMessage);
  const onSyncRef = useRef(onSync);

  // Keep refs up to date with latest callbacks
  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    onSyncRef.current = onSync;
  }, [onSync]);

  const handleEnvelope = useCallback((envelope: SyncEnvelope) => {
    switch (envelope.type) {
      case 'sync_response': {
        const response = envelope.payload as SyncResponse;
        // Update local database with synced messages
        response.messages.forEach(msg => {
          messageRepository.upsert({
            ...msg,
            sync_status: 'synced',
          });
        });
        onSyncRef.current?.();
        break;
      }

      case 'message': {
        const { message } = envelope.payload as MessagePayload;
        // Save incoming message to database
        messageRepository.upsert({
          ...message,
          sync_status: 'synced',
        });
        onMessageRef.current?.(message);
        break;
      }

      case 'ack': {
        // Message acknowledged by server
        console.log('Message acknowledged:', envelope.payload);
        break;
      }

      default:
        console.warn('Unknown envelope type:', envelope.type);
    }
  }, []);

  const connect = useCallback(() => {
    if (!conversationId || !enabled) return;

    // Prevent creating duplicate connections
    if (wsRef.current && wsRef.current.readyState !== WebSocket.CLOSED) {
      console.log('WebSocket already connected or connecting, skipping');
      return;
    }

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${location.host}/api/v1/conversations/${conversationId}/sync/ws`;

    try {
      const ws = new WebSocket(wsUrl);
      ws.binaryType = 'arraybuffer';

      ws.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        setError(null);
        reconnectAttemptsRef.current = 0;

        // Send initial sync request to get pending messages
        const pendingMessages = messageRepository.getPending();
        if (pendingMessages.length > 0) {
          const envelope: SyncEnvelope = {
            type: 'sync_request',
            payload: { messages: pendingMessages },
          };
          ws.send(pack(envelope));
        }
      };

      ws.onclose = () => {
        console.log('WebSocket disconnected');
        setIsConnected(false);
        wsRef.current = null;

        // Don't reconnect if we're intentionally cleaning up (effect cleanup or unmount)
        if (isCleaningUpRef.current) {
          return;
        }

        // Exponential backoff for reconnection
        if (enabled && conversationId) {
          const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
          reconnectAttemptsRef.current++;

          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, delay);
        }
      };

      ws.onerror = (event) => {
        console.error('WebSocket error:', event);
        setError(new Error('WebSocket connection error'));
      };

      ws.onmessage = (event) => {
        try {
          const envelope = unpack(new Uint8Array(event.data)) as SyncEnvelope;
          handleEnvelope(envelope);
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      wsRef.current = ws;
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to create WebSocket'));
    }
  }, [conversationId, enabled, handleEnvelope]);

  const send = useCallback((envelope: SyncEnvelope) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(pack(envelope));
    } else {
      console.warn('WebSocket not connected, cannot send message');
    }
  }, []);

  const syncNow = useCallback(() => {
    const pendingMessages = messageRepository.getPending();
    if (pendingMessages.length > 0) {
      const envelope: SyncEnvelope = {
        type: 'sync_request',
        payload: { messages: pendingMessages },
      };
      send(envelope);
    }
  }, [send]);

  useEffect(() => {
    // Reset cleanup flag when starting a new connection
    isCleaningUpRef.current = false;

    if (conversationId && enabled) {
      connect();
    }

    return () => {
      // Mark as intentional cleanup to prevent reconnect in onclose handler
      isCleaningUpRef.current = true;

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [conversationId, enabled, connect]);

  return { isConnected, error, send, syncNow };
}
