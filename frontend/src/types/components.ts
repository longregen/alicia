import { ReactNode } from 'react';

// Common component props
export interface BaseComponentProps {
  className?: string;
  children?: ReactNode;
}

// Size variants
export type Size = 'sm' | 'md' | 'lg';

// Color variants
export type Variant = 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning' | 'error';

// Recording states
export type RecordingState = 'idle' | 'recording' | 'processing' | 'completed' | 'error';

// Message roles - imported from models.ts (canonical definition)
import type { MessageRole } from './models';
export type { MessageRole };

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
export type AddonType = 'icon' | 'tool' | 'audio' | 'feedback' | 'memory';

// Feedback addon data
export interface FeedbackAddonData {
  currentVote: 'up' | 'down' | null;
  onVote: (vote: 'up' | 'down' | 'critical') => void;
  upvotes: number;
  downvotes: number;
  isLoading: boolean;
}

// Memory addon data (single trace)
export interface MemoryAddonData {
  id: string;
  content: string;
  relevance: number;
}

// Message addon interface
export interface MessageAddon {
  id: string;
  type: AddonType;
  position?: 'inline' | 'below';
  emoji?: string; // Emoji for inline display (not required for feedback/memory)
  tooltip?: string; // Tooltip text (not required for feedback/memory)
  content?: React.ReactNode; // Optional content for below addons or expanded state
  feedbackData?: FeedbackAddonData; // Data for feedback addon type
  memoryData?: MemoryAddonData[]; // Data for memory addon type
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

/**
 * @deprecated Use ToolData with type='reasoning' instead
 */
export interface ReasoningData {
  steps?: string[];
  conclusion?: string;
}

/**
 * @deprecated Use ToolData instead
 */
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
  /** @deprecated Use toolData instead */
  toolUse?: ToolUseData | null;
  responseTime?: number;
  tokensGenerated?: number;
  [key: string]: unknown; // Allow additional properties
}

// Message data structure
export interface MessageData {
  id: string;
  type: MessageRole;
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

/**
 * Re-export MemoryTrace from protocol.ts.
 * This is the canonical definition for memory traces used in the protocol.
 * Import from here for component usage.
 */
export type { MemoryTrace } from './protocol';
