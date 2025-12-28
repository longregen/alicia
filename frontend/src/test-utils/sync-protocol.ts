import { pack } from 'msgpackr';
import { Message } from '../types/models';
import { SyncResponse, SyncedMessage, SyncRequest, MessageResponse } from '../types/sync';
import { Envelope, MessageType } from '../types/protocol';

// Helper to convert domain model (snake_case) to wire format (camelCase)
function messageToMessageResponse(msg: Message): MessageResponse {
  return {
    id: msg.id,
    conversationId: msg.conversation_id,
    sequenceNumber: msg.sequence_number,
    previousId: msg.previous_id,
    role: msg.role,
    contents: msg.contents,
    createdAt: msg.created_at,
    updatedAt: msg.updated_at,
    localId: msg.local_id,
    serverId: msg.server_id,
    syncStatus: msg.sync_status,
  };
}

/**
 * Builder for creating sync protocol messages using standard Envelope format
 */
export class SyncProtocolBuilder {
  /**
   * Create a sync request envelope
   */
  static createSyncRequest(messages: Message[], conversationId: string): Envelope {
    const syncRequest: SyncRequest = {
      messages: messages.map((msg) => ({
        localId: msg.local_id!,
        sequenceNumber: msg.sequence_number,
        previousId: msg.previous_id,
        role: msg.role,
        contents: msg.contents,
        createdAt: msg.created_at,
        updatedAt: msg.updated_at,
      })),
    };

    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.SyncRequest,
      body: syncRequest,
    };
  }

  /**
   * Create a sync request as MessagePack binary.
   * Note: Backend expects raw DTO, not wrapped in Envelope.
   */
  static createSyncRequestBinary(messages: Message[], conversationId: string): Uint8Array {
    const envelope = this.createSyncRequest(messages, conversationId);
    // Extract body for wire format (backend expects raw DTO)
    return pack(envelope.body);
  }

  /**
   * Create a sync response envelope
   */
  static createSyncResponse(
    syncedMessages: SyncedMessage[],
    conversationId: string
  ): Envelope {
    const syncResponse: SyncResponse = {
      syncedMessages: syncedMessages,
      syncedAt: new Date().toISOString(),
    };

    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.SyncResponse,
      body: syncResponse,
    };
  }

  /**
   * Create a sync response as MessagePack binary.
   * Note: Backend sends raw DTO, not wrapped in Envelope.
   */
  static createSyncResponseBinary(
    syncedMessages: SyncedMessage[],
    conversationId: string
  ): Uint8Array {
    const envelope = this.createSyncResponse(syncedMessages, conversationId);
    // Extract body for wire format (backend sends raw DTO)
    return pack(envelope.body);
  }

  /**
   * Create a message envelope for incoming messages
   */
  static createMessageEnvelope(message: Message, conversationId: string): Envelope {
    return {
      stanzaId: 0,
      conversationId,
      type: message.role === 'user' ? MessageType.UserMessage : MessageType.AssistantMessage,
      body: message,
    };
  }

  /**
   * Create a message envelope as MessagePack binary.
   * Note: Backend sends raw message DTO, not wrapped in Envelope.
   */
  static createMessageEnvelopeBinary(message: Message, conversationId: string): Uint8Array {
    const envelope = this.createMessageEnvelope(message, conversationId);
    // Extract body for wire format (backend sends raw DTO)
    return pack(envelope.body);
  }

  /**
   * Create an acknowledgement envelope
   */
  static createAckEnvelope(stanzaId: number, conversationId: string, success = true): Envelope {
    return {
      stanzaId: 0,
      conversationId,
      type: MessageType.Acknowledgement,
      body: {
        conversationId,
        acknowledgedStanzaId: stanzaId,
        success,
      },
    };
  }

  /**
   * Create an acknowledgement envelope as MessagePack binary.
   * Note: Backend sends raw DTO, not wrapped in Envelope.
   */
  static createAckEnvelopeBinary(
    stanzaId: number,
    conversationId: string,
    success = true
  ): Uint8Array {
    const envelope = this.createAckEnvelope(stanzaId, conversationId, success);
    // Extract body for wire format (backend sends raw DTO)
    return pack(envelope.body);
  }

  /**
   * Create a successful sync response for pending messages
   */
  static createSuccessfulSyncResponse(
    pendingMessages: Message[]
  ): SyncResponse {
    return {
      syncedMessages: pendingMessages.map((msg) => ({
        localId: msg.local_id!,
        serverId: msg.id,
        status: 'synced',
        message: messageToMessageResponse({
          ...msg,
          sync_status: 'synced',
        }),
      })),
      syncedAt: new Date().toISOString(),
    };
  }

  /**
   * Create a conflict sync response
   */
  static createConflictSyncResponse(
    localMessage: Message,
    serverMessage: Message,
    reason = 'Content mismatch'
  ): SyncResponse {
    return {
      syncedMessages: [
        {
          localId: localMessage.local_id!,
          serverId: serverMessage.id,
          status: 'conflict',
          conflict: {
            reason,
            serverMessage: messageToMessageResponse(serverMessage),
            resolution: 'server_wins',
          },
        },
      ],
      syncedAt: new Date().toISOString(),
    };
  }

  /**
   * Create a mixed sync response (some synced, some conflicts)
   */
  static createMixedSyncResponse(
    syncedMessages: Message[],
    conflictPairs: Array<{ local: Message; server: Message }>
  ): SyncResponse {
    const syncedMessageResults: SyncedMessage[] = syncedMessages.map((msg) => ({
      localId: msg.local_id!,
      serverId: msg.id,
      status: 'synced',
      message: messageToMessageResponse({
        ...msg,
        sync_status: 'synced',
      }),
    }));

    const conflictResults: SyncedMessage[] = conflictPairs.map(
      ({ local, server }) => ({
        localId: local.local_id!,
        serverId: server.id,
        status: 'conflict',
        conflict: {
          reason: 'Sequence mismatch',
          serverMessage: messageToMessageResponse(server),
          resolution: 'server_wins',
        },
      })
    );

    return {
      syncedMessages: [...syncedMessageResults, ...conflictResults],
      syncedAt: new Date().toISOString(),
    };
  }

  /**
   * Create a batch of messages in sequence
   */
  static createMessageSequence(
    count: number,
    conversationId: string,
    startSequence = 1
  ): Message[] {
    const messages: Message[] = [];
    const now = Date.now();

    for (let i = 0; i < count; i++) {
      messages.push({
        id: `msg-seq-${startSequence + i}`,
        conversation_id: conversationId,
        sequence_number: startSequence + i,
        role: i % 2 === 0 ? 'user' : 'assistant',
        contents: `Message ${startSequence + i}`,
        local_id: i % 2 === 0 ? `local-${startSequence + i}` : undefined,
        sync_status: 'pending',
        created_at: new Date(now + i * 1000).toISOString(),
        updated_at: new Date(now + i * 1000).toISOString(),
      });
    }

    return messages;
  }
}

/**
 * Helper to simulate WebSocket sync flow
 */
export class SyncFlowSimulator {
  private sentMessages: Message[] = [];
  private receivedMessages: Message[] = [];

  /**
   * Simulate sending messages for sync
   */
  sendForSync(messages: Message[]): void {
    this.sentMessages.push(...messages);
  }

  /**
   * Simulate receiving synced messages
   */
  receiveSync(messages: Message[]): void {
    this.receivedMessages.push(...messages);
  }

  /**
   * Get messages that should be marked as synced
   */
  getSyncedMessages(): Message[] {
    return this.sentMessages.filter((sent) =>
      this.receivedMessages.some(
        (received) => received.local_id === sent.local_id
      )
    );
  }

  /**
   * Get messages still pending sync
   */
  getPendingMessages(): Message[] {
    return this.sentMessages.filter(
      (sent) =>
        !this.receivedMessages.some(
          (received) => received.local_id === sent.local_id
        )
    );
  }

  /**
   * Clear all tracked messages
   */
  clear(): void {
    this.sentMessages = [];
    this.receivedMessages = [];
  }
}

/**
 * Create a sync protocol builder
 */
export function createSyncProtocolBuilder(): typeof SyncProtocolBuilder {
  return SyncProtocolBuilder;
}

/**
 * Create a sync flow simulator
 */
export function createSyncFlowSimulator(): SyncFlowSimulator {
  return new SyncFlowSimulator();
}
