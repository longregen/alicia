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
  // WebSocket sync message types (frontend-only, used for local routing)
  SyncRequest = 17,
  SyncResponse = 18,
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
  stanza_id: number;
  conversation_id: string;
  type: MessageType;
  meta?: Record<string, unknown>;
  body: unknown;
}

// Message type definitions

export interface ErrorMessage {
  id: string;
  conversation_id: string;
  code: number;
  message: string;
  severity: Severity;
  recoverable: boolean;
  originating_id?: string;
}

export interface UserMessage {
  id: string;
  previous_id?: string;
  conversation_id: string;
  content: string;
  timestamp?: number;
}

export interface AssistantMessage {
  id: string;
  previous_id?: string;
  conversation_id: string;
  content: string;
  timestamp?: number;
}

export interface AudioChunk {
  conversation_id: string;
  format: string;
  sequence: number;
  duration_ms: number;
  track_sid?: string;
  data?: Uint8Array;
  is_last?: boolean;
  timestamp?: number;
}

export interface ReasoningStep {
  id: string;
  message_id: string;
  conversation_id: string;
  sequence: number;
  content: string;
}

export interface ToolUseRequest {
  id: string;
  message_id: string;
  conversation_id: string;
  tool_name: string;
  parameters: Record<string, unknown>;
  execution: ToolExecution;
  timeout_ms?: number;
}

export interface ToolUseResult {
  id: string;
  request_id: string;
  conversation_id: string;
  success: boolean;
  result?: unknown;
  error_code?: string;
  error_message?: string;
}

export interface Acknowledgement {
  conversation_id: string;
  acknowledged_stanza_id: number;
  success: boolean;
}

export interface Transcription {
  id: string;
  previous_id?: string;
  conversation_id: string;
  text: string;
  final: boolean;
  confidence?: number;
  language?: string;
}

export interface ControlStop {
  conversation_id: string;
  target_id?: string;
  reason?: string;
  stop_type?: StopType;
}

export interface ControlVariation {
  conversation_id: string;
  target_id: string;
  mode: VariationType;
  new_content?: string;
}

export interface Configuration {
  conversation_id?: string;
  last_sequence_seen?: number;
  client_version?: string;
  preferred_language?: string;
  device?: string;
  features?: string[];
}

export interface StartAnswer {
  id: string;
  previous_id: string;
  conversation_id: string;
  answer_type?: AnswerType;
  planned_sentence_count?: number;
}

export interface MemoryTrace {
  id: string;
  message_id: string;
  conversation_id: string;
  memory_id: string;
  content: string;
  relevance: number;
}

export interface Commentary {
  id: string;
  message_id: string;
  conversation_id: string;
  content: string;
  commentary_type?: string;
}

export interface AssistantSentence {
  id?: string;
  previous_id: string;
  conversation_id: string;
  sequence: number;
  text: string;
  is_final?: boolean;
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
