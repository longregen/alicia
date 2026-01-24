import {
  type MessageId,
  type ConversationId,
  type ToolCallId,
  type MemoryTraceId,
  createMessageId,
  createConversationId,
} from './streaming';

export type { MessageId, ConversationId, ToolCallId, MemoryTraceId };
export { createMessageId, createConversationId };

export const createToolCallId = (id: string): ToolCallId => id as unknown as ToolCallId;
export const createMemoryTraceId = (id: string): MemoryTraceId => id as unknown as MemoryTraceId;

export type MessageStatus = 'pending' | 'streaming' | 'completed' | 'error';
export type ToolCallStatus = 'pending' | 'success' | 'error';

export interface ToolCall {
  id: ToolCallId;
  tool_name: string;
  arguments: Record<string, unknown>;
  result?: unknown;
  status: ToolCallStatus;
  error?: string;
  created_at: string;
}

export interface MemoryTrace {
  id: MemoryTraceId;
  memory_id: string;
  content: string;
  relevance: number;
}

export interface ChatMessage {
  id: MessageId;
  conversation_id: ConversationId;
  previous_id?: MessageId;
  branch_index: number;
  role: 'user' | 'assistant';
  content: string;
  reasoning?: string;
  status: MessageStatus;
  created_at: string;
  tool_calls: ToolCall[];
  memory_traces: MemoryTrace[];
}

export function createEmptyMessage(
  id: MessageId,
  conversationId: ConversationId,
  role: 'user' | 'assistant'
): ChatMessage {
  return {
    id,
    conversation_id: conversationId,
    role,
    content: '',
    branch_index: 0,
    status: 'streaming',
    created_at: new Date().toISOString(),
    tool_calls: [],
    memory_traces: [],
  };
}
