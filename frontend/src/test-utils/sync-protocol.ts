import { pack } from 'msgpackr';
import { Message } from '../types/models';
import { SyncResponse, SyncedMessage } from '../types/sync';

/**
 * Builder for creating sync protocol messages
 */
export class SyncProtocolBuilder {
  /**
   * Create a sync request envelope
   */
  static createSyncRequest(messages: Message[]): unknown {
    return {
      type: 'sync_request',
      payload: {
        messages: messages.map((msg) => ({
          local_id: msg.local_id,
          sequence_number: msg.sequence_number,
          role: msg.role,
          contents: msg.contents,
          created_at: msg.created_at,
        })),
      },
    };
  }

  /**
   * Create a sync request as MessagePack binary
   */
  static createSyncRequestBinary(messages: Message[]): Uint8Array {
    return pack(this.createSyncRequest(messages));
  }

  /**
   * Create a sync response envelope
   */
  static createSyncResponse(
    syncedMessages: SyncedMessage[]
  ): unknown {
    return {
      type: 'sync_response',
      payload: {
        messages: syncedMessages.map((sm) => sm.message),
      },
    };
  }

  /**
   * Create a sync response as MessagePack binary
   */
  static createSyncResponseBinary(
    syncedMessages: SyncedMessage[]
  ): Uint8Array {
    return pack(this.createSyncResponse(syncedMessages));
  }

  /**
   * Create a message envelope for incoming messages
   */
  static createMessageEnvelope(message: Message): unknown {
    return {
      type: 'message',
      payload: {
        message,
      },
    };
  }

  /**
   * Create a message envelope as MessagePack binary
   */
  static createMessageEnvelopeBinary(message: Message): Uint8Array {
    return pack(this.createMessageEnvelope(message));
  }

  /**
   * Create an acknowledgement envelope
   */
  static createAckEnvelope(messageId: string, status = 'received'): unknown {
    return {
      type: 'ack',
      payload: {
        message_id: messageId,
        status,
      },
    };
  }

  /**
   * Create an acknowledgement envelope as MessagePack binary
   */
  static createAckEnvelopeBinary(
    messageId: string,
    status = 'received'
  ): Uint8Array {
    return pack(this.createAckEnvelope(messageId, status));
  }

  /**
   * Create a successful sync response for pending messages
   */
  static createSuccessfulSyncResponse(
    pendingMessages: Message[]
  ): SyncResponse {
    return {
      synced_messages: pendingMessages.map((msg) => ({
        local_id: msg.local_id!,
        server_id: msg.id,
        status: 'synced',
        message: {
          ...msg,
          sync_status: 'synced',
        },
      })),
      synced_at: new Date().toISOString(),
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
      synced_messages: [
        {
          local_id: localMessage.local_id!,
          server_id: serverMessage.id,
          status: 'conflict',
          conflict: {
            reason,
            server_message: serverMessage,
            resolution: 'server_wins',
          },
        },
      ],
      synced_at: new Date().toISOString(),
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
      local_id: msg.local_id!,
      server_id: msg.id,
      status: 'synced',
      message: {
        ...msg,
        sync_status: 'synced',
      },
    }));

    const conflictResults: SyncedMessage[] = conflictPairs.map(
      ({ local, server }) => ({
        local_id: local.local_id!,
        server_id: server.id,
        status: 'conflict',
        conflict: {
          reason: 'Sequence mismatch',
          server_message: server,
          resolution: 'server_wins',
        },
      })
    );

    return {
      synced_messages: [...syncedMessageResults, ...conflictResults],
      synced_at: new Date().toISOString(),
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
