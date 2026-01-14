import { useEffect, useRef, useCallback } from 'react';
import { Message } from '../types/models';
import { Envelope } from '../types/protocol';
import {
  useWebSocket,
  registerMessageHandler,
  registerSyncHandler,
} from '../contexts/WebSocketContext';

export interface UseWebSocketSyncOptions {
  onMessage?: (message: Message) => void;
  onSync?: () => void;
  enabled?: boolean;
}

/**
 * Hook for WebSocket-based message synchronization.
 * Uses the multiplexed WebSocket connection from WebSocketContext.
 *
 * When the conversationId changes, it automatically:
 * - Unsubscribes from the previous conversation
 * - Subscribes to the new conversation
 *
 * The underlying WebSocket connection is maintained at the app level,
 * so switching conversations doesn't create new connections.
 */
export function useWebSocketSync(
  conversationId: string | null,
  options: UseWebSocketSyncOptions = {}
) {
  const { onMessage, onSync, enabled = true } = options;
  const {
    isConnected,
    connectionError,
    subscribe,
    unsubscribe,
    send,
    syncConversation,
  } = useWebSocket();

  // Keep track of the previous conversation to unsubscribe
  const previousConversationRef = useRef<string | null>(null);

  // Store callbacks in refs
  const onMessageRef = useRef(onMessage);
  const onSyncRef = useRef(onSync);

  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    onSyncRef.current = onSync;
  }, [onSync]);

  // Register message and sync handlers
  useEffect(() => {
    if (!conversationId || !enabled) return;

    const unregisterMessage = registerMessageHandler(conversationId, (message) => {
      onMessageRef.current?.(message);
    });

    const unregisterSync = registerSyncHandler(conversationId, () => {
      onSyncRef.current?.();
    });

    return () => {
      unregisterMessage();
      unregisterSync();
    };
  }, [conversationId, enabled]);

  // Handle subscription changes
  useEffect(() => {
    if (!enabled) return;

    const previousConversation = previousConversationRef.current;

    // Unsubscribe from previous conversation if it changed
    if (previousConversation && previousConversation !== conversationId) {
      unsubscribe(previousConversation);
    }

    // Subscribe to new conversation
    if (conversationId && isConnected) {
      subscribe(conversationId)
        .then(() => {
          console.log(`Subscribed to conversation ${conversationId}`);
          // Sync pending messages for this conversation
          syncConversation(conversationId);
        })
        .catch((err) => {
          console.error(`Failed to subscribe to conversation ${conversationId}:`, err);
        });
    }

    previousConversationRef.current = conversationId;

    return () => {
      // Unsubscribe on cleanup
      if (conversationId) {
        unsubscribe(conversationId);
      }
    };
  }, [conversationId, enabled, isConnected, subscribe, unsubscribe, syncConversation]);

  const sendMessage = useCallback(
    (envelope: Envelope) => {
      send(envelope);
    },
    [send]
  );

  const syncNow = useCallback(() => {
    if (conversationId) {
      syncConversation(conversationId);
    }
  }, [conversationId, syncConversation]);

  return {
    isConnected,
    error: connectionError,
    send: sendMessage,
    syncNow,
  };
}
