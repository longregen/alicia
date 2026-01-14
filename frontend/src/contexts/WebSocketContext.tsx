import {
  createContext,
  useContext,
  useState,
  useEffect,
  useRef,
  useCallback,
  ReactNode,
} from 'react';
import { pack, unpack } from 'msgpackr';
import {
  Envelope,
  MessageType,
  SubscribeRequest,
  UnsubscribeRequest,
  SubscribeAck,
  UnsubscribeAck,
  ConversationUpdate,
} from '../types/protocol';
import { SyncRequest, SyncResponse, MessageResponse, messageResponseToMessage } from '../types/sync';
import { Message } from '../types/models';
import { messageRepository } from '../db/repository';
import { handleProtocolMessage, handleConnectionLost } from '../adapters/protocolAdapter';
import { useConnectionStore, ConnectionStatus } from '../stores/connectionStore';

interface PendingSubscription {
  resolve: (ack: SubscribeAck) => void;
  reject: (error: Error) => void;
}

interface WebSocketContextType {
  isConnected: boolean;
  connectionError: Error | null;
  subscribe: (conversationId: string) => Promise<SubscribeAck>;
  unsubscribe: (conversationId: string) => void;
  isSubscribed: (conversationId: string) => boolean;
  activeSubscriptions: Set<string>;
  send: (envelope: Envelope) => void;
  syncConversation: (conversationId: string) => void;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

// Message handlers that can be registered per conversation
type MessageHandler = (message: Message) => void;
type SyncHandler = () => void;
type ConversationUpdateHandler = (update: ConversationUpdate) => void;

const messageHandlers = new Map<string, Set<MessageHandler>>();
const syncHandlers = new Map<string, Set<SyncHandler>>();
const conversationUpdateHandlers = new Set<ConversationUpdateHandler>();

export function registerMessageHandler(conversationId: string, handler: MessageHandler) {
  if (!messageHandlers.has(conversationId)) {
    messageHandlers.set(conversationId, new Set());
  }
  messageHandlers.get(conversationId)!.add(handler);
  return () => {
    messageHandlers.get(conversationId)?.delete(handler);
  };
}

export function registerSyncHandler(conversationId: string, handler: SyncHandler) {
  if (!syncHandlers.has(conversationId)) {
    syncHandlers.set(conversationId, new Set());
  }
  syncHandlers.get(conversationId)!.add(handler);
  return () => {
    syncHandlers.get(conversationId)?.delete(handler);
  };
}

export function registerConversationUpdateHandler(handler: ConversationUpdateHandler) {
  conversationUpdateHandlers.add(handler);
  return () => {
    conversationUpdateHandlers.delete(handler);
  };
}

function notifyMessageHandlers(conversationId: string, message: Message) {
  messageHandlers.get(conversationId)?.forEach((handler) => handler(message));
}

function notifySyncHandlers(conversationId: string) {
  syncHandlers.get(conversationId)?.forEach((handler) => handler());
}

function notifyConversationUpdateHandlers(update: ConversationUpdate) {
  conversationUpdateHandlers.forEach((handler) => handler(update));
}

/**
 * Type guard to check if a message is already an Envelope.
 */
function isEnvelope(message: unknown): message is Envelope {
  if (!message || typeof message !== 'object') return false;
  const obj = message as Record<string, unknown>;
  return (
    typeof obj.stanzaId === 'number' &&
    typeof obj.type === 'number' &&
    'body' in obj
  );
}

/**
 * Adapter to convert backend DTO to Envelope format.
 */
function wrapInEnvelope(data: unknown, conversationId: string): Envelope {
  if (isEnvelope(data)) {
    return data;
  }

  const dto = data as Record<string, unknown>;

  // Subscription acknowledgements
  if ('success' in dto && 'conversationId' in dto) {
    if ('error' in dto || 'missedMessages' in dto) {
      return {
        stanzaId: 0,
        conversationId: (dto.conversationId as string) || conversationId,
        type: MessageType.SubscribeAck,
        body: data,
      };
    }
    // UnsubscribeAck has success and conversationId but no error/missedMessages
    return {
      stanzaId: 0,
      conversationId: (dto.conversationId as string) || conversationId,
      type: MessageType.UnsubscribeAck,
      body: data,
    };
  }

  if ('syncedMessages' in dto) {
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.SyncResponse,
      body: data,
    };
  } else if ('id' in dto && 'contents' in dto && 'conversationId' in dto) {
    // UserMessage or AssistantMessage - extract conversationId from the DTO
    const role = (dto as Record<string, unknown>).role;
    return {
      stanzaId: 0,
      conversationId: (dto.conversationId as string) || conversationId,
      type: role === 'user' ? MessageType.UserMessage : MessageType.AssistantMessage,
      body: data,
    };
  } else if ('acknowledgedStanzaId' in dto) {
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.Acknowledgement,
      body: data,
    };
  } else if ('id' in dto && 'conversationId' in dto && 'previousId' in dto && ('answerType' in dto || 'plannedSentenceCount' in dto)) {
    return {
      stanzaId: 0,
      conversationId: (dto.conversationId as string) || conversationId,
      type: MessageType.StartAnswer,
      body: data,
    };
  } else if ('conversationId' in dto && 'sequence' in dto && 'text' in dto && 'previousId' in dto) {
    return {
      stanzaId: 0,
      conversationId: (dto.conversationId as string) || conversationId,
      type: MessageType.AssistantSentence,
      body: data,
    };
  } else if ('id' in dto && 'messageId' in dto && 'toolName' in dto && 'parameters' in dto) {
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.ToolUseRequest,
      body: data,
    };
  } else if ('requestId' in dto && 'success' in dto) {
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.ToolUseResult,
      body: data,
    };
  } else if ('messageId' in dto && 'sequence' in dto && 'content' in dto && !('text' in dto)) {
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.ReasoningStep,
      body: data,
    };
  } else if ('format' in dto && 'sequence' in dto && 'durationMs' in dto) {
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.AudioChunk,
      body: data,
    };
  } else if ('text' in dto && 'final' in dto && typeof (dto as Record<string, unknown>).final === 'boolean') {
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.Transcription,
      body: data,
    };
  } else if ('memoryId' in dto && 'messageId' in dto && 'content' in dto && 'relevance' in dto) {
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.MemoryTrace,
      body: data,
    };
  } else if ('conversationId' in dto && 'updatedAt' in dto && ('title' in dto || 'status' in dto)) {
    return {
      stanzaId: 0,
      conversationId: (dto.conversationId as string) || conversationId,
      type: MessageType.ConversationUpdate,
      body: data,
    };
  }

  return {
    stanzaId: 0,
    conversationId,
    type: MessageType.ErrorMessage,
    body: data,
  };
}

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const wsRef = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionError, setConnectionError] = useState<Error | null>(null);
  const [activeSubscriptions, setActiveSubscriptions] = useState<Set<string>>(new Set());

  // Use a ref to track subscriptions for reconnection (avoids dependency cycle)
  const activeSubscriptionsRef = useRef<Set<string>>(new Set());

  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const isCleaningUpRef = useRef(false);
  const stanzaIdRef = useRef(0);

  // Pending subscription promises
  const pendingSubscriptionsRef = useRef<Map<string, PendingSubscription>>(new Map());

  // Connection store for status indicator
  const setConnectionStatus = useConnectionStore((state) => state.setConnectionStatus);
  const setStoreError = useConnectionStore((state) => state.setError);

  const nextStanzaId = useCallback(() => {
    stanzaIdRef.current++;
    return stanzaIdRef.current;
  }, []);

  // Server sends WebSocket pings every 30 seconds, browser auto-responds with pongs.
  // No client-side heartbeat needed.

  const handleEnvelope = useCallback((envelope: Envelope) => {

    const conversationId = envelope.conversationId;

    switch (envelope.type) {
      case MessageType.SubscribeAck: {
        const ack = envelope.body as SubscribeAck;
        const pending = pendingSubscriptionsRef.current.get(ack.conversationId);
        if (pending) {
          pendingSubscriptionsRef.current.delete(ack.conversationId);
          if (ack.success) {
            // Update both ref and state
            activeSubscriptionsRef.current.add(ack.conversationId);
            setActiveSubscriptions((prev) => new Set([...prev, ack.conversationId]));
            pending.resolve(ack);
          } else {
            pending.reject(new Error(ack.error || 'Subscription failed'));
          }
        }
        break;
      }

      case MessageType.UnsubscribeAck: {
        const ack = envelope.body as UnsubscribeAck;
        if (ack.success) {
          // Update both ref and state
          activeSubscriptionsRef.current.delete(ack.conversationId);
          setActiveSubscriptions((prev) => {
            const next = new Set(prev);
            next.delete(ack.conversationId);
            return next;
          });
        }
        break;
      }

      case MessageType.SyncResponse: {
        const response = envelope.body as SyncResponse;
        response.syncedMessages.forEach((syncedMsg) => {
          const existingMessage = messageRepository.findByLocalId(syncedMsg.localId);

          if (existingMessage) {
            messageRepository.update(existingMessage.id, {
              server_id: syncedMsg.serverId,
              sequence_number: syncedMsg.message?.sequenceNumber,
              sync_status: syncedMsg.status === 'synced' ? 'synced' : 'conflict',
            });
          } else if (syncedMsg.message) {
            const message = messageResponseToMessage(syncedMsg.message);
            messageRepository.insert({
              ...message,
              server_id: syncedMsg.serverId,
              sync_status: syncedMsg.status === 'synced' ? 'synced' : 'conflict',
            });
          }
        });
        if (conversationId) {
          notifySyncHandlers(conversationId);
        }
        break;
      }

      case MessageType.UserMessage:
      case MessageType.AssistantMessage: {
        const messageResponse = envelope.body as MessageResponse;
        const message = messageResponseToMessage(messageResponse);

        const existingByServerId = messageRepository.findByServerId(message.id);
        if (existingByServerId) break;

        if (message.local_id) {
          const existingByLocalId = messageRepository.findByLocalId(message.local_id);
          if (existingByLocalId) {
            messageRepository.replaceId(existingByLocalId.id, message.id, {
              ...message,
              local_id: existingByLocalId.local_id,
              server_id: message.id,
              sync_status: 'synced',
            });
            if (conversationId) {
              notifyMessageHandlers(conversationId, message);
            }
            break;
          }
        }

        const existingById = messageRepository.findById(message.id);
        if (existingById) break;

        if (message.role === 'user' && conversationId) {
          const pendingMessages = messageRepository.getPending(conversationId);
          const duplicate = pendingMessages.find(
            (m) => m.contents === message.contents && m.role === message.role
          );
          if (duplicate) {
            messageRepository.replaceId(duplicate.id, message.id, {
              ...message,
              local_id: duplicate.local_id,
              server_id: message.id,
              sync_status: 'synced',
            });
            notifyMessageHandlers(conversationId, message);
            break;
          }
        }

        messageRepository.upsert({
          ...message,
          sync_status: 'synced',
        });
        if (conversationId) {
          notifyMessageHandlers(conversationId, message);
        }
        break;
      }

      case MessageType.Acknowledgement:
        console.log('Message acknowledged:', envelope.body);
        break;

      case MessageType.StartAnswer:
      case MessageType.AssistantSentence:
      case MessageType.ToolUseRequest:
      case MessageType.ToolUseResult:
      case MessageType.ReasoningStep:
      case MessageType.AudioChunk:
      case MessageType.Transcription:
      case MessageType.MemoryTrace:
        handleProtocolMessage(envelope);
        break;

      case MessageType.ConversationUpdate: {
        const update = envelope.body as ConversationUpdate;
        notifyConversationUpdateHandlers(update);
        break;
      }

      default:
        console.warn('Unknown envelope type:', envelope.type);
    }
  }, []);

  const connect = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState !== WebSocket.CLOSED) {
      console.log('WebSocket already connected or connecting, skipping');
      return;
    }

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${location.host}/api/v1/ws`;

    setConnectionStatus(ConnectionStatus.Connecting);

    try {
      const ws = new WebSocket(wsUrl);
      ws.binaryType = 'arraybuffer';

      ws.onopen = () => {
        console.log('Multiplexed WebSocket connected');
        setIsConnected(true);
        setConnectionError(null);
        setConnectionStatus(ConnectionStatus.Connected);
        setStoreError(null);
        reconnectAttemptsRef.current = 0;

        // Re-subscribe to all previously active conversations using the ref
        activeSubscriptionsRef.current.forEach((convId) => {
          const subscribeEnvelope: Envelope = {
            stanzaId: stanzaIdRef.current++,
            conversationId: convId,
            type: MessageType.Subscribe,
            body: { conversationId: convId } as SubscribeRequest,
          };
          ws.send(pack(subscribeEnvelope));
        });
      };

      ws.onclose = () => {
        console.log('Multiplexed WebSocket disconnected');
        handleConnectionLost();
        setIsConnected(false);
        wsRef.current = null;

        if (isCleaningUpRef.current) {
          setConnectionStatus(ConnectionStatus.Disconnected);
          return;
        }

        setConnectionStatus(ConnectionStatus.Reconnecting);
        const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
        reconnectAttemptsRef.current++;

        reconnectTimeoutRef.current = setTimeout(() => {
          connect();
        }, delay);
      };

      ws.onerror = (event) => {
        console.error('WebSocket error:', event);
        handleConnectionLost();
        setConnectionError(new Error('WebSocket connection error'));
        setStoreError('WebSocket connection error');
      };

      ws.onmessage = (event) => {
        try {
          const dto = unpack(new Uint8Array(event.data));
          const envelope = wrapInEnvelope(dto, '');
          handleEnvelope(envelope);
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      wsRef.current = ws;
    } catch (err) {
      setConnectionError(err instanceof Error ? err : new Error('Failed to create WebSocket'));
      setConnectionStatus(ConnectionStatus.Error);
      setStoreError('Failed to create WebSocket');
    }
  // Note: activeSubscriptions removed from deps - we use activeSubscriptionsRef instead
  }, [handleEnvelope, setConnectionStatus, setStoreError]);

  const subscribe = useCallback((conversationId: string): Promise<SubscribeAck> => {
    return new Promise((resolve, reject) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        reject(new Error('WebSocket not connected'));
        return;
      }

      // Store pending promise
      pendingSubscriptionsRef.current.set(conversationId, { resolve, reject });

      // Send subscribe message
      const subscribeEnvelope: Envelope = {
        stanzaId: nextStanzaId(),
        conversationId,
        type: MessageType.Subscribe,
        body: { conversationId } as SubscribeRequest,
      };
      wsRef.current.send(pack(subscribeEnvelope));

      // Timeout after 10 seconds
      setTimeout(() => {
        const pending = pendingSubscriptionsRef.current.get(conversationId);
        if (pending) {
          pendingSubscriptionsRef.current.delete(conversationId);
          reject(new Error('Subscribe timeout'));
        }
      }, 10000);
    });
  }, [nextStanzaId]);

  const unsubscribe = useCallback((conversationId: string) => {
    // Update ref immediately
    activeSubscriptionsRef.current.delete(conversationId);

    // Optimistically remove from subscriptions state
    setActiveSubscriptions((prev) => {
      const next = new Set(prev);
      next.delete(conversationId);
      return next;
    });

    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      return;
    }

    const unsubscribeEnvelope: Envelope = {
      stanzaId: nextStanzaId(),
      conversationId,
      type: MessageType.Unsubscribe,
      body: { conversationId } as UnsubscribeRequest,
    };
    wsRef.current.send(pack(unsubscribeEnvelope));
  }, [nextStanzaId]);

  const isSubscribed = useCallback((conversationId: string) => {
    return activeSubscriptionsRef.current.has(conversationId);
  }, []);

  const send = useCallback((envelope: Envelope) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(pack(envelope));
    } else {
      console.warn('WebSocket not connected, cannot send message');
    }
  }, []);

  const syncConversation = useCallback((conversationId: string) => {
    if (!conversationId) return;

    const pendingMessages = messageRepository.getPending(conversationId);
    if (pendingMessages.length > 0) {
      const syncRequest: SyncRequest = {
        messages: pendingMessages.map((msg) => ({
          localId: msg.local_id!,
          sequenceNumber: msg.sequence_number,
          previousId: msg.previous_id,
          role: msg.role,
          contents: msg.contents,
          createdAt: msg.created_at,
          updatedAt: msg.updated_at,
        })),
      };
      const envelope: Envelope = {
        stanzaId: nextStanzaId(),
        conversationId,
        type: MessageType.SyncRequest,
        body: syncRequest,
      };
      send(envelope);
    }
  }, [send, nextStanzaId]);

  // Connect on mount
  useEffect(() => {
    isCleaningUpRef.current = false;
    connect();

    const handleOnline = () => {
      console.log('Network online - attempting immediate reconnection');
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      reconnectAttemptsRef.current = 0;
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        connect();
      }
    };

    const handleOffline = () => {
      console.log('Network offline - connection will be lost');
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      isCleaningUpRef.current = true;
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }

      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect]);

  return (
    <WebSocketContext.Provider
      value={{
        isConnected,
        connectionError,
        subscribe,
        unsubscribe,
        isSubscribed,
        activeSubscriptions,
        send,
        syncConversation,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket(): WebSocketContextType {
  const context = useContext(WebSocketContext);
  if (context === undefined) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}
