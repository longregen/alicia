export type ConversationStatus = 'active' | 'archived' | 'deleted';

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
export type SyncStatus = 'pending' | 'synced' | 'conflict';

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

  // Related entities (loaded separately)
  sentences?: any[];
  audio?: any;
  tool_uses?: any[];
  reasoning_steps?: any[];
}

export interface CreateConversationRequest {
  title?: string;
}

export interface CreateMessageRequest {
  contents: string;
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
