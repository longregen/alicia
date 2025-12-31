import { Message } from './models';

// Wire format types using camelCase to match Go msgpack serialization
// The Go backend uses msgpack tags like `msgpack:"syncedMessages"`

export interface SyncMessageRequest {
  localId: string;
  sequenceNumber: number;
  previousId?: string;
  role: string;
  contents: string;
  createdAt: string;
  updatedAt?: string;
}

export interface SyncRequest {
  messages: SyncMessageRequest[];
}

export interface ConflictDetails {
  reason: string;
  serverMessage?: MessageResponse;
  resolution: string;
  /** When the local version was last modified */
  localModifiedAt?: string;
  /** When the server version was last modified */
  serverModifiedAt?: string;
  /** The type of conflict (content, metadata, etc.) */
  conflictType?: 'content' | 'metadata' | 'both';
  /** Additional context about the conflict */
  description?: string;
}

export interface SyncedMessage {
  localId: string;
  serverId: string;
  status: 'synced' | 'conflict';
  message?: MessageResponse;
  conflict?: ConflictDetails;
}

export interface SyncResponse {
  syncedMessages: SyncedMessage[];
  syncedAt: string;
}

export interface SyncStatusResponse {
  conversationId: string;
  pendingCount: number;
  syncedCount: number;
  conflictCount: number;
  lastSyncedAt?: string;
}

// MessageResponse is the wire format for messages (camelCase)
// This matches the Go msgpack serialization
export interface MessageResponse {
  id: string;
  conversationId: string;
  sequenceNumber: number;
  previousId?: string;
  role: 'user' | 'assistant' | 'system';
  contents: string;
  createdAt: string;
  updatedAt: string;
  localId?: string;
  serverId?: string;
  syncStatus?: 'pending' | 'synced' | 'conflict';
}

// Helper to convert wire format (camelCase) to domain model (snake_case)
export function messageResponseToMessage(response: MessageResponse): Message {
  return {
    id: response.id,
    conversation_id: response.conversationId,
    sequence_number: response.sequenceNumber,
    previous_id: response.previousId,
    role: response.role,
    contents: response.contents,
    created_at: response.createdAt,
    updated_at: response.updatedAt,
    local_id: response.localId,
    server_id: response.serverId,
    sync_status: response.syncStatus,
  };
}

export interface SyncState {
  isSyncing: boolean;
  lastSyncTime: Date | null;
  syncError: string | null;
}
