import { Message } from './models';

export interface SyncMessageRequest {
  local_id: string;
  sequence_number: number;
  previous_id?: string;
  role: string;
  contents: string;
  created_at: string;
  updated_at?: string;
}

export interface SyncRequest {
  messages: SyncMessageRequest[];
}

export interface ConflictDetails {
  reason: string;
  server_message?: Message;
  resolution: string;
}

export interface SyncedMessage {
  local_id: string;
  server_id: string;
  status: 'synced' | 'conflict';
  message?: Message;
  conflict?: ConflictDetails;
}

export interface SyncResponse {
  synced_messages: SyncedMessage[];
  synced_at: string;
}

export interface SyncStatusResponse {
  conversation_id: string;
  pending_count: number;
  synced_count: number;
  conflict_count: number;
  last_synced_at?: string;
}

export interface SyncState {
  isSyncing: boolean;
  lastSyncTime: Date | null;
  syncError: string | null;
}
