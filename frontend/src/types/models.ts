import type { AssistantSentence, AudioChunk, ReasoningStep } from './protocol';

export type ConversationStatus = 'active' | 'archived' | 'deleted';

// ToolUseResponse matches the DTO format from the server's REST API
export interface ToolUseResponse {
  id: string;
  message_id: string;
  tool_name: string;
  arguments?: Record<string, unknown>;
  result?: unknown;
  status: 'pending' | 'running' | 'success' | 'error' | 'cancelled';
  error_message?: string;
  sequence_number: number;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ConversationPreferences {
  tts_voice?: string;
  tts_speed?: number;
  language?: string;
  response_style?: string;
  enable_reasoning?: boolean;
  enable_memory?: boolean;
  max_response_tokens?: number;
}

export interface Conversation {
  id: string;
  title: string;
  status: ConversationStatus;
  livekit_room_name?: string;
  preferences?: ConversationPreferences;
  last_client_stanza_id: number;
  last_server_stanza_id: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
}

export type MessageRole = 'user' | 'assistant' | 'system';
/** Sync status for offline-first operations. Canonical definition - import from here. */
export type SyncStatus = 'pending' | 'synced' | 'conflict';

/**
 * Message domain model. Uses snake_case to match database schema.
 * For normalized/streaming use, see NormalizedMessage in streaming.ts
 */
export interface Message {
  id: string;
  conversation_id: string;
  sequence_number: number;
  previous_id?: string;
  role: MessageRole;
  contents: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string;

  // Offline sync tracking
  local_id?: string;
  server_id?: string;
  sync_status?: SyncStatus;
  synced_at?: string;
  retry_count?: number;

  // Related entities (loaded separately)
  sentences?: AssistantSentence[];
  audio?: AudioChunk;
  tool_uses?: ToolUseResponse[];
  reasoning_steps?: ReasoningStep[];
}

export interface CreateConversationRequest {
  title?: string;
}

export interface CreateMessageRequest {
  contents: string;
  local_id?: string;
}

export interface ConversationsResponse {
  conversations: Conversation[];
}

export interface MessagesResponse {
  messages: Message[];
}

export interface TTSRequest {
  model: string;
  input: string;
  voice: string;
  response_format?: string;
  speed?: number;
}
