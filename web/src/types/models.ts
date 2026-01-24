export type ConversationStatus = 'active' | 'archived';

export interface Conversation {
  id: string;
  user_id: string;
  title: string;
  status: ConversationStatus;
  tip_message_id?: string;
  created_at: string;
  updated_at: string;
}

export type MessageRole = 'user' | 'assistant';
export type MessageStatus = 'pending' | 'streaming' | 'completed' | 'error';

export interface Message {
  id: string;
  conversation_id: string;
  previous_id?: string;
  branch_index: number;
  role: MessageRole;
  content: string;
  reasoning?: string;
  status: MessageStatus;
  created_at: string;
}

export interface CreateConversationRequest {
  title?: string;
}

export interface CreateMessageRequest {
  content: string;
  previous_id?: string;
  use_pareto?: boolean;
}

export interface ConversationsResponse {
  conversations: Conversation[];
}

export interface MessagesResponse {
  messages: Message[];
}
