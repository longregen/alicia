import { ReactNode } from 'react';

// Common component props
export interface BaseComponentProps {
  className?: string;
  children?: ReactNode;
}

// Size variants
export type Size = 'sm' | 'md' | 'lg';

// Color variants
export type Variant = 'default' | 'primary' | 'success' | 'warning' | 'error';

// Recording states
export type RecordingState = 'idle' | 'recording' | 'processing' | 'completed' | 'error';

// Message types
export type MessageType = 'user' | 'assistant' | 'system';

// Message states
export type MessageState = 'idle' | 'typing' | 'sending' | 'streaming' | 'completed' | 'error';

// Audio states
export type AudioState = 'idle' | 'playing' | 'paused' | 'loading' | 'error';

// Connection states
export type ConnectionState = 'connected' | 'connecting' | 'disconnected' | 'reconnecting' | 'error';

// Language codes
export type LanguageCode = string; // ISO 639-1 language codes

// Audio visualization data
export type AudioLevels = number[];

// Input method types
export type InputMethod = 'text' | 'voice';

// Addon type for message bubbles
export type AddonType = 'icon' | 'tool' | 'audio';

// Message addon interface
export interface MessageAddon {
  id: string;
  type: AddonType;
  position?: 'inline' | 'below';
  emoji: string; // Required emoji representation for inline display
  tooltip: string; // Required tooltip text
  content?: React.ReactNode; // Optional content for below addons or expanded state
}

// Tool data interface (updated from ToolUseData)
export interface ToolData {
  id: string;
  name: string;
  description: string;
  status?: 'running' | 'completed' | 'error';
  result?: string;
  type?: string; // 'reasoning' | 'search' | 'calculation' etc.
}

// Reasoning data interface (deprecated - use ToolData with type='reasoning')
export interface ReasoningData {
  steps?: string[];
  conclusion?: string;
}

// Tool use data interface (deprecated - use ToolData)
export interface ToolUseData {
  name: string;
  description: string;
  status?: 'running' | 'completed' | 'error';
  result?: string;
}

// Message metadata structure
export interface MessageMetadata {
  hasAudio?: boolean;
  audioUrl?: string | null;
  transcriptionConfidence?: number;
  speechDuration?: number;
  reasoning?: string;
  toolUse?: ToolUseData | null;
  responseTime?: number;
  tokensGenerated?: number;
  [key: string]: unknown; // Allow additional properties
}

// Message data structure
export interface MessageData {
  id: string;
  type: MessageType;
  content: string;
  timestamp: Date;
  state?: MessageState;
  metadata?: MessageMetadata;
}

// Language data structure
export interface LanguageData {
  code: LanguageCode;
  name: string;
  nativeName: string;
  flag: string;
}

// Memory trace data structure
export interface MemoryTrace {
  id: string;
  messageId: string;
  content: string;
  relevance: number; // 0-1 score
}
