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
  VoiceJoinRequest,
  VoiceJoinAck,
  VoiceLeaveRequest,
  VoiceLeaveAck,
  VoiceSpeaking,
  VoiceStatus,
  ConversationTitleUpdate,
  WhatsAppQR,
  WhatsAppStatus,
  WhatsAppDebug,
} from '../types/protocol';
import { Message } from '../types/models';
import { setMessageSender } from '../adapters/protocolAdapter';
import { handleChatProtocolMessage, handleChatConnectionLost } from '../adapters/chatAdapter';
import { useConnectionStore, ConnectionStatus } from '../stores/connectionStore';
import { useVoiceConnectionStore, VoiceConnectionStatus } from '../stores/voiceConnectionStore';
import { useWhatsAppStore, WhatsAppRole } from '../stores/whatsappStore';
import { injectTraceContext, startSpan, SpanStatusCode } from '../lib/otel';
import { getUserId } from '../utils/deviceId';

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
  sendVoiceJoinRequest: (conversationId: string) => void;
  sendVoiceLeaveRequest: (conversationId: string) => void;
  voiceConnectionStatus: VoiceConnectionStatus;
  voiceConnectionError: string | null;
  voiceRetryCount: number;
  retryVoiceJoin: () => void;
  sendWhatsAppPairRequest: (role: WhatsAppRole) => void;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

type MessageHandler = (message: Message) => void;
type TitleUpdateHandler = (conversationId: string, title: string) => void;

const messageHandlers = new Map<string, Set<MessageHandler>>();
const titleUpdateHandlers = new Set<TitleUpdateHandler>();

export function registerMessageHandler(conversationId: string, handler: MessageHandler) {
  if (!messageHandlers.has(conversationId)) {
    messageHandlers.set(conversationId, new Set());
  }
  messageHandlers.get(conversationId)!.add(handler);
  return () => {
    messageHandlers.get(conversationId)?.delete(handler);
  };
}

export function registerTitleUpdateHandler(handler: TitleUpdateHandler) {
  titleUpdateHandlers.add(handler);
  return () => {
    titleUpdateHandlers.delete(handler);
  };
}

function notifyMessageHandlers(conversationId: string, message: Message) {
  messageHandlers.get(conversationId)?.forEach((handler) => handler(message));
}

function notifyTitleUpdateHandlers(conversationId: string, title: string) {
  titleUpdateHandlers.forEach((handler) => handler(conversationId, title));
}

function isEnvelope(message: unknown): message is Envelope {
  if (!message || typeof message !== 'object') return false;
  const obj = message as Record<string, unknown>;
  return (
    typeof obj.type === 'number' &&
    'body' in obj
  );
}

function wrapInEnvelope(data: unknown, conversationId: string): Envelope {
  if (isEnvelope(data)) {
    return data;
  }

  const dto = data as Record<string, unknown>;

  if ('messageId' in dto && 'success' in dto && 'conversationId' in dto) {
    return {
      conversationId: (dto.conversationId as string) || conversationId,
      type: MessageType.GenerationComplete,
      body: data,
    };
  }

  if ('success' in dto && 'conversationId' in dto) {
    return {
      conversationId: (dto.conversationId as string) || conversationId,
      type: MessageType.SubscribeAck,
      body: data,
    };
  }

  if ('id' in dto && 'content' in dto && 'conversationId' in dto && 'role' in dto) {
    const role = dto.role;
    return {
      conversationId: (dto.conversationId as string) || conversationId,
      type: role === 'user' ? MessageType.UserMessage : MessageType.AssistantMessage,
      body: data,
    };
  }

  if ('conversationId' in dto && 'sequence' in dto && 'text' in dto) {
    return {
      conversationId: (dto.conversationId as string) || conversationId,
      type: MessageType.AssistantSentence,
      body: data,
    };
  }

  if ('messageId' in dto && 'conversationId' in dto && 'previousId' in dto && !('sequence' in dto) && !('text' in dto) && !('content' in dto)) {
    return {
      conversationId: (dto.conversationId as string) || conversationId,
      type: MessageType.StartAnswer,
      body: data,
    };
  }

  if ('id' in dto && 'messageId' in dto && 'toolName' in dto && 'arguments' in dto) {
    return {
      conversationId: (dto.conversationId as string) || conversationId,
      type: MessageType.ToolUseRequest,
      body: data,
    };
  }

  if ('requestId' in dto && 'success' in dto) {
    return {
      conversationId,
      type: MessageType.ToolUseResult,
      body: data,
    };
  }

  if ('memoryId' in dto && 'messageId' in dto && 'content' in dto && 'relevance' in dto) {
    return {
      conversationId,
      type: MessageType.MemoryTrace,
      body: data,
    };
  }

  if ('code' in dto && 'message' in dto) {
    return {
      conversationId,
      type: MessageType.ErrorMessage,
      body: data,
    };
  }

  console.warn('WebSocket: Unknown message format, treating as error:', data);
  return {
    conversationId,
    type: MessageType.ErrorMessage,
    body: data,
  };
}

interface MessageResponse {
  id: string;
  conversationId?: string;
  conversation_id?: string;
  previousId?: string;
  previous_id?: string;
  role: 'user' | 'assistant';
  content: string;
  createdAt?: string;
  created_at?: string;
}

function messageResponseToMessage(response: MessageResponse): Message {
  return {
    id: response.id,
    conversation_id: response.conversationId || response.conversation_id || '',
    previous_id: response.previousId || response.previous_id,
    branch_index: 0,
    role: response.role,
    content: response.content,
    status: 'completed',
    created_at: response.createdAt || response.created_at || new Date().toISOString(),
  };
}

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const wsRef = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionError, setConnectionError] = useState<Error | null>(null);
  const [activeSubscriptions, setActiveSubscriptions] = useState<Set<string>>(new Set());

  const activeSubscriptionsRef = useRef<Set<string>>(new Set());
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const isCleaningUpRef = useRef(false);

  const pendingSubscriptionsRef = useRef<Map<string, PendingSubscription>>(new Map());
  const voiceRetryTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const voiceRetryConversationRef = useRef<string | null>(null);

  const setConnectionStatus = useConnectionStore((state) => state.setConnectionStatus);
  const setStoreError = useConnectionStore((state) => state.setError);

  const voiceConnectionStatus = useVoiceConnectionStore((state) => state.status);
  const voiceConnectionError = useVoiceConnectionStore((state) => state.error);
  const voiceRetryCount = useVoiceConnectionStore((state) => state.retryCount);
  const setVoiceConnecting = useVoiceConnectionStore((state) => state.setConnecting);
  const setVoiceConnected = useVoiceConnectionStore((state) => state.setConnected);
  const setVoiceRetrying = useVoiceConnectionStore((state) => state.setRetrying);
  const setVoiceError = useVoiceConnectionStore((state) => state.setError);
  const setVoiceSpeaking = useVoiceConnectionStore((state) => state.setSpeaking);
  const resetVoiceConnection = useVoiceConnectionStore((state) => state.reset);

  const setWhatsAppQR = useWhatsAppStore((state) => state.setQR);
  const setWhatsAppStatus = useWhatsAppStore((state) => state.setStatus);
  const addWhatsAppDebug = useWhatsAppStore((state) => state.addDebugEvent);

  const handleEnvelope = useCallback((envelope: Envelope) => {
    const conversationId = envelope.conversationId;

    switch (envelope.type) {
      case MessageType.SubscribeAck: {
        const ack = envelope.body as SubscribeAck;
        const pending = pendingSubscriptionsRef.current.get(ack.conversationId || '');
        if (pending) {
          pendingSubscriptionsRef.current.delete(ack.conversationId || '');
          if (ack.success) {
            activeSubscriptionsRef.current.add(ack.conversationId || '');
            setActiveSubscriptions((prev) => new Set([...prev, ack.conversationId || '']));
            pending.resolve(ack);
          } else {
            pending.reject(new Error(ack.error || 'Subscription failed'));
          }
        }
        break;
      }

      case MessageType.UserMessage:
      case MessageType.AssistantMessage: {
        const messageResponse = envelope.body as MessageResponse;
        const message = messageResponseToMessage(messageResponse);

        if (conversationId) {
          notifyMessageHandlers(conversationId, message);
        }

        if (envelope.type === MessageType.AssistantMessage) {
          handleChatProtocolMessage(envelope);
        }
        break;
      }

      case MessageType.Acknowledgement:
        break;

      case MessageType.StartAnswer:
      case MessageType.AssistantSentence:
      case MessageType.ToolUseRequest:
      case MessageType.ToolUseResult:
      case MessageType.MemoryTrace:
      case MessageType.ErrorMessage:
      case MessageType.ReasoningStep:
      case MessageType.ThinkingSummary:
      case MessageType.BranchUpdate:
      case MessageType.GenerationComplete:
        handleChatProtocolMessage(envelope);
        break;

      case MessageType.ConversationTitleUpdate: {
        const update = envelope.body as ConversationTitleUpdate;
        notifyTitleUpdateHandlers(update.conversationId, update.title);
        break;
      }

      case MessageType.VoiceJoinAck: {
        const ack = envelope.body as VoiceJoinAck;
        if (ack.success) {
          setVoiceConnected();
          voiceRetryConversationRef.current = null;
          if (voiceRetryTimeoutRef.current) {
            clearTimeout(voiceRetryTimeoutRef.current);
            voiceRetryTimeoutRef.current = null;
          }
        } else {
          console.error('Voice join failed for conversation:', ack.conversationId, ack.error);
          const currentRetryCount = useVoiceConnectionStore.getState().retryCount;
          const maxRetries = useVoiceConnectionStore.getState().maxRetries;

          if (currentRetryCount < maxRetries) {
            const nextRetry = currentRetryCount + 1;
            setVoiceRetrying(nextRetry);
            voiceRetryConversationRef.current = ack.conversationId;
            const delay = 2000 * Math.pow(2, currentRetryCount); // 2s, 4s, 8s
            voiceRetryTimeoutRef.current = setTimeout(() => {
              const convId = voiceRetryConversationRef.current;
              if (convId && wsRef.current?.readyState === WebSocket.OPEN) {
                const retryEnvelope: Envelope = {
                  conversationId: convId,
                  type: MessageType.VoiceJoinRequest,
                  body: { conversationId: convId } as VoiceJoinRequest,
                };
                try {
                  wsRef.current.send(pack(retryEnvelope));
                } catch (err) {
                  console.error('Failed to send voice join retry:', err);
                  setVoiceError('Failed to send voice join retry');
                }
              } else {
                setVoiceError('WebSocket not connected for voice retry');
              }
            }, delay);
          } else {
            setVoiceError(ack.error || 'Voice connection failed after maximum retries');
            voiceRetryConversationRef.current = null;
          }
        }
        break;
      }

      case MessageType.VoiceLeaveAck: {
        const ack = envelope.body as VoiceLeaveAck;
        if (ack.success) {
          resetVoiceConnection();
        } else {
          console.error('Voice leave failed for conversation:', ack.conversationId, ack.error);
        }
        break;
      }

      case MessageType.VoiceStatus: {
        const status = envelope.body as VoiceStatus;
        if (status.status === 'queue_full') {
          console.warn(
            `Voice TTS queue full for conversation ${status.conversationId}:`,
            status.error || 'sentences may be dropped',
            `(queue length: ${status.queueLength})`
          );
        }
        break;
      }

      case MessageType.VoiceSpeaking: {
        const speaking = envelope.body as VoiceSpeaking;
        setVoiceSpeaking(
          speaking.speaking,
          speaking.messageId || null,
          speaking.sentenceSeq ?? null,
        );
        break;
      }

      case MessageType.WhatsAppQR: {
        const qr = envelope.body as WhatsAppQR;
        if (qr.role !== 'reader' && qr.role !== 'alicia') break;
        setWhatsAppQR(qr.role, qr.code, qr.event);
        break;
      }

      case MessageType.WhatsAppStatus: {
        const status = envelope.body as WhatsAppStatus;
        if (status.role !== 'reader' && status.role !== 'alicia') break;
        setWhatsAppStatus(status.role, status.connected, status.phone, status.error);
        break;
      }

      case MessageType.WhatsAppDebug: {
        const debug = envelope.body as WhatsAppDebug;
        if (debug.role !== 'reader' && debug.role !== 'alicia') break;
        addWhatsAppDebug(debug.role, debug.event, debug.detail);
        break;
      }

      default:
        console.warn('Unknown envelope type:', envelope.type);
    }
  }, [setVoiceConnected, setVoiceRetrying, setVoiceError, setVoiceSpeaking, resetVoiceConnection, setWhatsAppQR, setWhatsAppStatus, addWhatsAppDebug]);

  const connect = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState !== WebSocket.CLOSED) {
      return;
    }

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${location.host}/api/v1/ws`;

    setConnectionStatus(ConnectionStatus.Connecting);

    try {
      const ws = new WebSocket(wsUrl);
      ws.binaryType = 'arraybuffer';

      ws.onopen = () => {
        setIsConnected(true);
        setConnectionError(null);
        setConnectionStatus(ConnectionStatus.Connected);
        setStoreError(null);
        reconnectAttemptsRef.current = 0;

        setMessageSender((envelope: Envelope) => {
          if (ws.readyState === WebSocket.OPEN) {
            try {
              ws.send(pack(envelope));
            } catch (err) {
              console.error('Failed to pack envelope:', err);
            }
          }
        });

        activeSubscriptionsRef.current.forEach((convId) => {
          const subscribeEnvelope: Envelope = {
            conversationId: convId,
            type: MessageType.Subscribe,
            body: { conversationId: convId } as SubscribeRequest,
          };
          try {
            ws.send(pack(subscribeEnvelope));
          } catch (err) {
            console.error('Failed to re-subscribe:', err);
          }
        });
      };

      ws.onclose = () => {
        handleChatConnectionLost();
        setMessageSender(null);
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
        handleChatConnectionLost();
        setConnectionError(new Error('WebSocket connection error'));
        setStoreError('WebSocket connection error');
      };

      ws.onmessage = (event) => {
        try {
          const data = new Uint8Array(event.data);
          const dto = unpack(data);
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
  }, [handleEnvelope, setConnectionStatus, setStoreError]);

  const subscribe = useCallback((conversationId: string): Promise<SubscribeAck> => {
    return new Promise((resolve, reject) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        reject(new Error('WebSocket not connected'));
        return;
      }

      const span = startSpan('websocket.subscribe', {
        'conversation.id': conversationId,
      });

      pendingSubscriptionsRef.current.set(conversationId, {
        resolve: (ack) => {
          span.end();
          resolve(ack);
        },
        reject: (error) => {
          span.setStatus({ code: SpanStatusCode.ERROR });
          span.recordException(error);
          span.end();
          reject(error);
        },
      });

      const subscribeEnvelope: Envelope = {
        conversationId,
        type: MessageType.Subscribe,
        body: { conversationId } as SubscribeRequest,
      };
      try {
        // Inject trace context into the envelope (pass the span since it's not active)
        const tracedEnvelope = injectTraceContext(subscribeEnvelope, conversationId, undefined, span);
        wsRef.current.send(pack(tracedEnvelope));
      } catch (err) {
        span.setStatus({ code: SpanStatusCode.ERROR });
        span.recordException(err as Error);
        span.end();
        reject(err instanceof Error ? err : new Error('Failed to send subscribe'));
        return;
      }

      setTimeout(() => {
        const pending = pendingSubscriptionsRef.current.get(conversationId);
        if (pending) {
          pendingSubscriptionsRef.current.delete(conversationId);
          // Note: span may already be ended by resolve/reject
          reject(new Error('Subscribe timeout'));
        }
      }, 10000);
    });
  }, []);

  const unsubscribe = useCallback((conversationId: string) => {
    activeSubscriptionsRef.current.delete(conversationId);

    setActiveSubscriptions((prev) => {
      const next = new Set(prev);
      next.delete(conversationId);
      return next;
    });

    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      return;
    }

    const unsubscribeEnvelope: Envelope = {
      conversationId,
      type: MessageType.Unsubscribe,
      body: { conversationId } as UnsubscribeRequest,
    };
    try {
      wsRef.current.send(pack(unsubscribeEnvelope));
    } catch (err) {
      console.error('Failed to unsubscribe:', err);
    }
  }, []);

  const isSubscribed = useCallback((conversationId: string) => {
    return activeSubscriptionsRef.current.has(conversationId);
  }, []);

  const send = useCallback((envelope: Envelope) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      const span = startSpan('websocket.send', {
        'message.type': MessageType[envelope.type] || envelope.type,
        'conversation.id': envelope.conversationId || '',
      });
      try {
        // Inject trace context into the envelope for distributed tracing (pass span since it's not active)
        const tracedEnvelope = injectTraceContext(envelope, envelope.conversationId, undefined, span);
        wsRef.current.send(pack(tracedEnvelope));
        span.end();
      } catch (err) {
        console.error('Failed to send:', err);
        span.setStatus({ code: SpanStatusCode.ERROR });
        span.recordException(err as Error);
        span.end();
      }
    } else {
      console.warn('WebSocket not connected, cannot send message');
    }
  }, []);

  const sendVoiceJoinRequest = useCallback((conversationId: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn('WebSocket not connected, cannot send voice join request');
      return;
    }

    setVoiceConnecting(conversationId);

    const span = startSpan('websocket.voice_join', {
      'conversation.id': conversationId,
    });

    const userId = getUserId();
    const envelope: Envelope = {
      conversationId,
      type: MessageType.VoiceJoinRequest,
      user_id: userId,
      body: { conversationId, userId } as VoiceJoinRequest,
    };

    try {
      // Inject trace context into the envelope (pass span since it's not active)
      const tracedEnvelope = injectTraceContext(envelope, conversationId, undefined, span);
      wsRef.current.send(pack(tracedEnvelope));
      span.end();
    } catch (err) {
      console.error('Failed to send voice join request:', err);
      setVoiceError('Failed to send voice join request');
      span.setStatus({ code: SpanStatusCode.ERROR });
      span.recordException(err as Error);
      span.end();
    }
  }, [setVoiceConnecting, setVoiceError]);

  const sendVoiceLeaveRequest = useCallback((conversationId: string) => {
    if (voiceRetryTimeoutRef.current) {
      clearTimeout(voiceRetryTimeoutRef.current);
      voiceRetryTimeoutRef.current = null;
    }
    voiceRetryConversationRef.current = null;
    resetVoiceConnection();

    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn('WebSocket not connected, cannot send voice leave request');
      return;
    }

    const span = startSpan('websocket.voice_leave', {
      'conversation.id': conversationId,
    });

    const envelope: Envelope = {
      conversationId,
      type: MessageType.VoiceLeaveRequest,
      body: { conversationId } as VoiceLeaveRequest,
    };

    try {
      const tracedEnvelope = injectTraceContext(envelope, conversationId, undefined, span);
      wsRef.current.send(pack(tracedEnvelope));
      span.end();
    } catch (err) {
      console.error('Failed to send voice leave request:', err);
      span.setStatus({ code: SpanStatusCode.ERROR });
      span.recordException(err as Error);
      span.end();
    }
  }, [resetVoiceConnection]);

  const retryVoiceJoin = useCallback(() => {
    const convId = voiceRetryConversationRef.current || useVoiceConnectionStore.getState().conversationId;
    if (!convId) {
      console.warn('No conversation ID available for voice retry');
      return;
    }

    useVoiceConnectionStore.getState().reset();
    sendVoiceJoinRequest(convId);
  }, [sendVoiceJoinRequest]);

  const sendWhatsAppPairRequest = useCallback((role: WhatsAppRole) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn('WebSocket not connected, cannot send WhatsApp pair request');
      return;
    }

    useWhatsAppStore.getState().setPairing(role);

    const envelope: Envelope = {
      conversationId: '',
      type: MessageType.WhatsAppPairRequest,
      body: { role },
    };

    try {
      wsRef.current.send(pack(envelope));
    } catch (err) {
      console.error('Failed to send WhatsApp pair request:', err);
    }
  }, []);

  useEffect(() => {
    isCleaningUpRef.current = false;
    connect();

    const handleOnline = () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      reconnectAttemptsRef.current = 0;
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        connect();
      }
    };

    window.addEventListener('online', handleOnline);

    return () => {
      isCleaningUpRef.current = true;
      window.removeEventListener('online', handleOnline);

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }

      if (voiceRetryTimeoutRef.current) {
        clearTimeout(voiceRetryTimeoutRef.current);
        voiceRetryTimeoutRef.current = null;
      }

      queueMicrotask(() => {
        if (isCleaningUpRef.current && wsRef.current) {
          wsRef.current.close();
          wsRef.current = null;
        }
      });
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
        sendVoiceJoinRequest,
        sendVoiceLeaveRequest,
        voiceConnectionStatus,
        voiceConnectionError,
        voiceRetryCount,
        retryVoiceJoin,
        sendWhatsAppPairRequest,
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
