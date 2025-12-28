import { useEffect, useRef, useState, useCallback } from 'react';
import { pack, unpack } from 'msgpackr';
import { Message } from '../types/models';
import { messageRepository } from '../db/repository';
import { Envelope, MessageType } from '../types/protocol';
import { SyncRequest, SyncResponse } from '../types/sync';

/**
 * Converts a backend MessageResponse DTO (camelCase msgpack fields) to frontend Message type (snake_case).
 * The backend uses camelCase for msgpack serialization but the frontend expects snake_case.
 */
function dtoToMessage(dto: Record<string, unknown>): Message {
  return {
    id: dto.id as string,
    conversation_id: (dto.conversationId ?? dto.conversation_id) as string,
    sequence_number: (dto.sequenceNumber ?? dto.sequence_number) as number,
    previous_id: (dto.previousId ?? dto.previous_id) as string | undefined,
    role: dto.role as 'user' | 'assistant' | 'system',
    contents: dto.contents as string,
    created_at: (dto.createdAt ?? dto.created_at) as string,
    updated_at: (dto.updatedAt ?? dto.updated_at) as string,
    // Preserve any existing sync fields if present
    local_id: (dto.localId ?? dto.local_id) as string | undefined,
    server_id: (dto.serverId ?? dto.server_id) as string | undefined,
    sync_status: (dto.syncStatus ?? dto.sync_status) as 'pending' | 'synced' | 'conflict' | undefined,
  };
}

/**
 * Converts a backend SyncResponse DTO (camelCase msgpack fields) to frontend SyncResponse type (snake_case).
 */
function dtoToSyncResponse(dto: Record<string, unknown>): SyncResponse {
  const syncedMessages = (dto.syncedMessages ?? dto.synced_messages) as Array<Record<string, unknown>> | undefined;

  return {
    synced_messages: (syncedMessages ?? []).map(sm => ({
      local_id: (sm.localId ?? sm.local_id) as string,
      server_id: (sm.serverId ?? sm.server_id) as string,
      status: sm.status as 'synced' | 'conflict',
      message: sm.message ? dtoToMessage(sm.message as Record<string, unknown>) : undefined,
      conflict: sm.conflict ? {
        reason: (sm.conflict as Record<string, unknown>).reason as string,
        server_message: (sm.conflict as Record<string, unknown>).serverMessage
          ? dtoToMessage((sm.conflict as Record<string, unknown>).serverMessage as Record<string, unknown>)
          : (sm.conflict as Record<string, unknown>).server_message
            ? dtoToMessage((sm.conflict as Record<string, unknown>).server_message as Record<string, unknown>)
            : undefined,
        resolution: (sm.conflict as Record<string, unknown>).resolution as string,
      } : undefined,
    })),
    synced_at: (dto.syncedAt ?? dto.synced_at) as string,
  };
}

/**
 * Adapter to convert backend DTO to Envelope format.
 * The backend currently sends raw DTOs, but we use Envelope internally for consistency.
 */
function wrapInEnvelope(data: unknown, conversationId: string): Envelope {
  // Detect message type based on DTO structure
  const dto = data as Record<string, unknown>;

  if ('synced_messages' in dto) {
    // SyncResponse DTO
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
  } else if ('message_id' in dto || 'acknowledgedStanzaId' in dto) {
    // Acknowledgement DTO
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.Acknowledgement,
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
        // Convert from backend DTO (camelCase) to frontend SyncResponse (snake_case)
        const response = dtoToSyncResponse(envelope.body as Record<string, unknown>);
        // Update local database with synced messages
        response.synced_messages.forEach(syncedMsg => {
          if (syncedMsg.message) {
            messageRepository.upsert({
              ...syncedMsg.message,
              sync_status: 'synced',
            });
          }
        });
        onSyncRef.current?.();
        break;
      }

      case MessageType.UserMessage:
      case MessageType.AssistantMessage: {
        // Incoming message broadcast from server (e.g., from another client)
        // Convert from backend DTO (camelCase) to frontend Message (snake_case)
        const message = dtoToMessage(envelope.body as Record<string, unknown>);
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
          const syncRequest: SyncRequest = {
            messages: pendingMessages.map(msg => ({
              local_id: msg.local_id!,
              sequence_number: msg.sequence_number,
              previous_id: msg.previous_id,
              role: msg.role,
              contents: msg.contents,
              created_at: msg.created_at,
              updated_at: msg.updated_at,
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

    const pendingMessages = messageRepository.getPending();
    if (pendingMessages.length > 0) {
      const syncRequest: SyncRequest = {
        messages: pendingMessages.map(msg => ({
          local_id: msg.local_id!,
          sequence_number: msg.sequence_number,
          previous_id: msg.previous_id,
          role: msg.role,
          contents: msg.contents,
          created_at: msg.created_at,
          updated_at: msg.updated_at,
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
