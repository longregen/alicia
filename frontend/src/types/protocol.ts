// Protocol types for Alicia real-time binary protocol
// Based on backend protocol in pkg/protocol/

export enum MessageType {
  ErrorMessage = 1,
  UserMessage = 2,
  AssistantMessage = 3,
  AudioChunk = 4,
  ReasoningStep = 5,
  ToolUseRequest = 6,
  ToolUseResult = 7,
  Acknowledgement = 8,
  Transcription = 9,
  ControlStop = 10,
  ControlVariation = 11,
  Configuration = 12,
  StartAnswer = 13,
  MemoryTrace = 14,
  Commentary = 15,
  AssistantSentence = 16,
}

export enum Severity {
  Info = 0,
  Warning = 1,
  Error = 2,
  Critical = 3,
}

export type StopType = 'generation' | 'speech' | 'all';
export type ToolExecution = 'server' | 'client' | 'either';
export type AnswerType = 'text' | 'voice' | 'text+voice';
export type VariationType = 'regenerate' | 'edit' | 'continue';

// Envelope wraps all protocol messages
export interface Envelope {
  stanzaId: number;
  conversationId: string;
  type: MessageType;
  meta?: Record<string, unknown>;
  body: unknown;
}

// Message type definitions

export interface ErrorMessage {
  id: string;
  conversationId: string;
  code: number;
  message: string;
  severity: Severity;
  recoverable: boolean;
  originatingId?: string;
}

export interface UserMessage {
  id: string;
  previousId?: string;
  conversationId: string;
  content: string;
  timestamp?: number;
}

export interface AssistantMessage {
  id: string;
  previousId?: string;
  conversationId: string;
  content: string;
  timestamp?: number;
}

export interface AudioChunk {
  conversationId: string;
  format: string;
  sequence: number;
  durationMs: number;
  trackSid?: string;
  data?: Uint8Array;
  isLast?: boolean;
  timestamp?: number;
}

export interface ReasoningStep {
  id: string;
  messageId: string;
  conversationId: string;
  sequence: number;
  content: string;
}

export interface ToolUseRequest {
  id: string;
  messageId: string;
  conversationId: string;
  toolName: string;
  parameters: Record<string, unknown>;
  execution: ToolExecution;
  timeoutMs?: number;
}

export interface ToolUseResult {
  id: string;
  requestId: string;
  conversationId: string;
  success: boolean;
  result?: unknown;
  errorCode?: string;
  errorMessage?: string;
}

export interface Acknowledgement {
  conversationId: string;
  acknowledgedStanzaId: number;
  success: boolean;
}

export interface Transcription {
  id: string;
  previousId?: string;
  conversationId: string;
  text: string;
  final: boolean;
  confidence?: number;
  language?: string;
}

export interface ControlStop {
  conversationId: string;
  targetId?: string;
  reason?: string;
  stopType?: StopType;
}

export interface ControlVariation {
  conversationId: string;
  targetId: string;
  mode: VariationType;
  newContent?: string;
}

export interface Configuration {
  conversationId?: string;
  lastSequenceSeen?: number;
  clientVersion?: string;
  preferredLanguage?: string;
  device?: string;
  features?: string[];
}

export interface StartAnswer {
  id: string;
  previousId: string;
  conversationId: string;
  answerType?: AnswerType;
  plannedSentenceCount?: number;
}

export interface MemoryTrace {
  id: string;
  messageId: string;
  conversationId: string;
  memoryId: string;
  content: string;
  relevance: number;
}

export interface Commentary {
  id: string;
  messageId: string;
  conversationId: string;
  content: string;
  commentaryType?: string;
}

export interface AssistantSentence {
  id?: string;
  previousId: string;
  conversationId: string;
  sequence: number;
  text: string;
  isFinal?: boolean;
  audio?: Uint8Array;
}

// Common features
export const Features = {
  STREAMING: 'streaming',
  PARTIAL_RESPONSES: 'partial_responses',
  AUDIO_OUTPUT: 'audio_output',
  REASONING_STEPS: 'reasoning_steps',
  TOOL_USE: 'tool_use',
} as const;
