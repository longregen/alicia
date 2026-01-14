// Branded types for type-safe IDs
declare const brand: unique symbol;
type Brand<T, TBrand extends string> = T & { [brand]: TBrand };

export type MessageId = Brand<string, 'MessageId'>;
export type SentenceId = Brand<string, 'SentenceId'>;
export type ToolCallId = Brand<string, 'ToolCallId'>;
export type AudioRefId = Brand<string, 'AudioRefId'>;
export type MemoryTraceId = Brand<string, 'MemoryTraceId'>;
export type ConversationId = Brand<string, 'ConversationId'>;

// Helper functions to create branded IDs
export const createMessageId = (id: string): MessageId => id as MessageId;
export const createSentenceId = (id: string): SentenceId => id as SentenceId;
export const createToolCallId = (id: string): ToolCallId => id as ToolCallId;
export const createAudioRefId = (id: string): AudioRefId => id as AudioRefId;
export const createMemoryTraceId = (id: string): MemoryTraceId => id as MemoryTraceId;
export const createConversationId = (id: string): ConversationId => id as ConversationId;

// Message status enum
export enum MessageStatus {
  Streaming = 'streaming',
  Complete = 'complete',
  Error = 'error',
}

// Microphone status enum
export enum MicrophoneStatus {
  Inactive = 'inactive',
  RequestingPermission = 'requesting_permission',
  Active = 'active',
  Recording = 'recording',
  Error = 'error'
}

// Message sentence structure
export interface MessageSentence {
  id: SentenceId;
  messageId: MessageId;
  content: string;
  sequence: number;
  audioRefId?: AudioRefId;
  isComplete: boolean;
}

// Tool call with discriminated union for status
export type ToolCall =
  | {
      status: 'pending';
      id: ToolCallId;
      toolName: string;
      arguments: Record<string, unknown>;
      messageId: MessageId;
      startTimeMs: number;
    }
  | {
      status: 'executing';
      id: ToolCallId;
      toolName: string;
      arguments: Record<string, unknown>;
      messageId: MessageId;
      startTimeMs: number;
    }
  | {
      status: 'success';
      id: ToolCallId;
      toolName: string;
      arguments: Record<string, unknown>;
      messageId: MessageId;
      startTimeMs: number;
      endTimeMs: number;
      resultContent: string;
    }
  | {
      status: 'error';
      id: ToolCallId;
      toolName: string;
      arguments: Record<string, unknown>;
      messageId: MessageId;
      startTimeMs: number;
      endTimeMs: number;
      error: string;
    };

// Audio reference metadata
export interface AudioRef {
  id: AudioRefId;
  sizeBytes: number;
  durationMs: number;
  sampleRate: number;
}

// Memory trace structure
export interface MemoryTrace {
  id: MemoryTraceId;
  messageId: MessageId;
  content: string;
  relevance: number;
  source?: string;
}

// Message structure for normalized store
// Uses snake_case for sync_status to maintain database schema compatibility
export interface NormalizedMessage {
  id: MessageId;
  conversationId: ConversationId;
  role: 'user' | 'assistant' | 'system';
  content: string;
  status: MessageStatus;
  createdAt: Date;
  previousId?: MessageId; // Optional link to previous message in conversation chain
  sentenceIds: SentenceId[];
  toolCallIds: ToolCallId[];
  memoryTraceIds: MemoryTraceId[];
  sync_status?: 'pending' | 'synced' | 'conflict'; // snake_case for database compatibility
  local_id?: string; // Client-generated ID for offline sync deduplication
}

// Normalized conversation store state
export interface ConversationStoreState {
  // Normalized entities
  messages: Record<string, NormalizedMessage>;
  sentences: Record<string, MessageSentence>;
  toolCalls: Record<string, ToolCall>;
  audioRefs: Record<string, AudioRef>;
  memoryTraces: Record<string, MemoryTrace>;

  // Current streaming state
  currentStreamingMessageId: MessageId | null;

  // Conversation context
  currentConversationId: ConversationId | null;
}
