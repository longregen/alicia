import { ReactNode } from 'react';

export interface BaseComponentProps {
  className?: string;
  children?: ReactNode;
}

import type { MessageRole } from './models';
export type { MessageRole };

export type MessageState = 'idle' | 'typing' | 'sending' | 'streaming' | 'completed' | 'error';

export type AudioState = 'idle' | 'playing' | 'paused' | 'loading' | 'error';

export type VoiceState = 'idle' | 'listening' | 'processing' | 'speaking';

export type LanguageCode = string; // ISO 639-1 language codes

export type AddonType = 'icon' | 'tool' | 'audio' | 'feedback' | 'memory';

export interface FeedbackAddonData {
  currentVote: 'up' | 'down' | null;
  onVote: (vote: 'up' | 'down' | 'critical') => void;
  upvotes: number;
  downvotes: number;
  isLoading: boolean;
}

export interface MemoryAddonData {
  id: string;
  content: string;
  relevance: number;
}

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

