import { useEffect, useRef, useState, useCallback } from 'react';
import { pack, unpack } from 'msgpackr';
import { Message } from '../types/models';
import { messageRepository } from '../db/repository';
import { Envelope, MessageType } from '../types/protocol';
import { SyncRequest, SyncResponse, MessageResponse, messageResponseToMessage } from '../types/sync';
import { handleProtocolMessage } from '../adapters/protocolAdapter';

/**
 * Adapter to convert backend DTO to Envelope format.
 * The backend currently sends raw DTOs, but we use Envelope internally for consistency.
 * NOTE: Backend uses camelCase for msgpack wire format (e.g., syncedMessages, localId)
 */
function wrapInEnvelope(data: unknown, conversationId: string): Envelope {
  // Detect message type based on DTO structure (camelCase from msgpack)
  const dto = data as Record<string, unknown>;

  if ('syncedMessages' in dto) {
    // SyncResponse DTO (camelCase wire format)
    return {
      stanzaId: 0, // Backend doesn't send stanzaId for sync messages
      conversationId,
      type: MessageType.SyncResponse,
      body: data,
    };
  } else if ('id' in dto && 'contents' in dto) {
    // MessageResponse DTO (broadcast from other clients)
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.AssistantMessage, // Could be user or assistant
      body: data,
    };
  } else if ('acknowledgedStanzaId' in dto) {
    // Acknowledgement DTO (camelCase wire format)
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.Acknowledgement,
      body: data,
    };
  }
  // Protocol streaming messages
  else if ('id' in dto && 'conversationId' in dto && 'previousId' in dto && ('answerType' in dto || 'plannedSentenceCount' in dto)) {
    // StartAnswer: has id, conversationId, previousId, optionally answerType/plannedSentenceCount
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.StartAnswer,
      body: data,
    };
  } else if ('conversationId' in dto && 'sequence' in dto && 'text' in dto && 'previousId' in dto) {
    // AssistantSentence: has conversationId, sequence, text, previousId
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.AssistantSentence,
      body: data,
    };
  } else if ('id' in dto && 'messageId' in dto && 'toolName' in dto && 'parameters' in dto) {
    // ToolUseRequest: has id, messageId, toolName, parameters
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.ToolUseRequest,
      body: data,
    };
  } else if ('requestId' in dto && 'success' in dto) {
    // ToolUseResult: has requestId, success
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.ToolUseResult,
      body: data,
    };
  } else if ('messageId' in dto && 'sequence' in dto && 'content' in dto && !('text' in dto)) {
    // ReasoningStep: has messageId, sequence, content (but not text like AssistantSentence)
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.ReasoningStep,
      body: data,
    };
  } else if ('format' in dto && 'sequence' in dto && 'durationMs' in dto) {
    // AudioChunk: has format, sequence, durationMs
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.AudioChunk,
      body: data,
    };
  } else if ('text' in dto && 'final' in dto && typeof (dto as Record<string, unknown>).final === 'boolean') {
    // Transcription: has text, final (boolean)
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.Transcription,
      body: data,
    };
  } else if ('memoryId' in dto && 'messageId' in dto && 'content' in dto && 'relevance' in dto) {
    // MemoryTrace: has memoryId, messageId, content, relevance
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.MemoryTrace,
      body: data,
    };
  }

  // Default to unknown
  return {
    stanzaId: 0,
    conversationId,
    type: MessageType.ErrorMessage,
    body: data,
  };
}

/**
 * Adapter to extract DTO from Envelope for sending to backend.
 * The backend expects raw DTOs, not Envelope-wrapped messages.
 */
function unwrapEnvelope(envelope: Envelope): unknown {
  return envelope.body;
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

  const handleEnvelope = useCallback((envelope: Envelope) => {
    switch (envelope.type) {
      case MessageType.SyncResponse: {
        const response = envelope.body as SyncResponse;
        // Update local database with synced messages
        response.syncedMessages.forEach(syncedMsg => {
          // Look up existing message by local_id for proper mapping
          const existingMessage = messageRepository.findByLocalId(syncedMsg.localId);

          if (existingMessage) {
            // Update existing local message with server data
            messageRepository.update(existingMessage.id, {
              server_id: syncedMsg.serverId,
              sequence_number: syncedMsg.message?.sequenceNumber,
              sync_status: syncedMsg.status === 'synced' ? 'synced' : 'conflict',
            });
          } else if (syncedMsg.message) {
            // New message from server (e.g., from another device)
            // Convert wire format (camelCase) to domain model (snake_case)
            const message = messageResponseToMessage(syncedMsg.message);
            messageRepository.insert({
              ...message,
              server_id: syncedMsg.serverId,
              sync_status: syncedMsg.status === 'synced' ? 'synced' : 'conflict',
            });
          }
        });
        onSyncRef.current?.();
        break;
      }

      case MessageType.UserMessage:
      case MessageType.AssistantMessage: {
        // Incoming message broadcast from server (e.g., from another client)
        // Wire format uses camelCase, convert to domain model (snake_case)
        const messageResponse = envelope.body as MessageResponse;
        const message = messageResponseToMessage(messageResponse);

        // Check if this message already exists (handles REST+WebSocket race condition)
        // This happens when the same client sends via REST API and receives the broadcast
        const existingByServerId = messageRepository.findByServerId(message.id);
        if (existingByServerId) {
          // Already have this message from REST API response, skip to avoid duplicate
          break;
        }

        // Also check by local_id if present (another way to detect same message)
        if (message.local_id) {
          const existingByLocalId = messageRepository.findByLocalId(message.local_id);
          if (existingByLocalId) {
            break;
          }
        }

        // Also check by the message ID itself (in case it was already inserted)
        const existingById = messageRepository.findById(message.id);
        if (existingById) {
          break;
        }

        // Save incoming message to database
        messageRepository.upsert({
          ...message,
          sync_status: 'synced',
        });
        onMessageRef.current?.(message);
        break;
      }

      case MessageType.Acknowledgement: {
        // Message acknowledged by server
        console.log('Message acknowledged:', envelope.body);
        break;
      }

      // Protocol streaming messages - route to protocol adapter
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

        // Send initial sync request to get pending messages for this conversation
        const pendingMessages = messageRepository.getPending(conversationId);
        if (pendingMessages.length > 0) {
          const syncRequest: SyncRequest = {
            messages: pendingMessages.map(msg => ({
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
            stanzaId: 0,
            conversationId: conversationId,
            type: MessageType.SyncRequest,
            body: syncRequest,
          };
          // Backend expects raw DTO, not envelope
          ws.send(pack(unwrapEnvelope(envelope)));
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
          // Backend sends raw DTOs, wrap in Envelope for consistent handling
          const dto = unpack(new Uint8Array(event.data));
          const envelope = wrapInEnvelope(dto, conversationId);
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

  const send = useCallback((envelope: Envelope) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      // Backend expects raw DTO, not envelope
      wsRef.current.send(pack(unwrapEnvelope(envelope)));
    } else {
      console.warn('WebSocket not connected, cannot send message');
    }
  }, []);

  const syncNow = useCallback(() => {
    if (!conversationId) return;

    const pendingMessages = messageRepository.getPending(conversationId);
    if (pendingMessages.length > 0) {
      const syncRequest: SyncRequest = {
        messages: pendingMessages.map(msg => ({
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
        stanzaId: 0,
        conversationId,
        type: MessageType.SyncRequest,
        body: syncRequest,
      };
      send(envelope);
    }
  }, [conversationId, send]);

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
