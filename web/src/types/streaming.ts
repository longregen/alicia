declare const brand: unique symbol;
type Brand<T, TBrand extends string> = T & { [brand]: TBrand };

export type MessageId = Brand<string, 'MessageId'>;
export type ConversationId = Brand<string, 'ConversationId'>;
export type ToolCallId = Brand<string, 'ToolCallId'>;
export type MemoryTraceId = Brand<string, 'MemoryTraceId'>;

export const createMessageId = (id: string): MessageId => id as MessageId;
export const createConversationId = (id: string): ConversationId => id as ConversationId;
export const createToolCallId = (id: string): ToolCallId => id as ToolCallId;
export const createMemoryTraceId = (id: string): MemoryTraceId => id as MemoryTraceId;

export enum MicrophoneStatus {
  Inactive = 'inactive',
  Loading = 'loading',
  RequestingPermission = 'requesting_permission',
  Active = 'active',
  Recording = 'recording',
  Sending = 'sending',
  Error = 'error',
}
