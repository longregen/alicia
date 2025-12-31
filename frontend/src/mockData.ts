import type {
  LanguageData,
  MessageState,
  MessageRole,
  RecordingState,
  AudioState,
} from './types/components';

// These constants mirror types from types/components.ts
// Consider importing directly if synchronization becomes an issue

// Message states constants
export const MESSAGE_STATES = {
  IDLE: 'idle',
  TYPING: 'typing',
  SENDING: 'sending',
  STREAMING: 'streaming',
  COMPLETED: 'completed',
  ERROR: 'error',
} as const satisfies Record<string, MessageState>;

export const MESSAGE_TYPES = {
  USER: 'user',
  ASSISTANT: 'assistant',
  SYSTEM: 'system',
} as const satisfies Record<string, MessageRole>;

export const RECORDING_STATES = {
  IDLE: 'idle',
  RECORDING: 'recording',
  PROCESSING: 'processing',
  COMPLETED: 'completed',
  ERROR: 'error',
} as const satisfies Record<string, RecordingState>;

export const AUDIO_STATES = {
  IDLE: 'idle',
  LOADING: 'loading',
  PLAYING: 'playing',
  PAUSED: 'paused',
  ERROR: 'error',
} as const satisfies Record<string, AudioState>;

// Language data
export const languages: LanguageData[] = [
  { code: 'auto', name: 'Auto-detect', nativeName: 'Auto-detect', flag: '🌐' },
  { code: 'en', name: 'English', nativeName: 'English', flag: '🇺🇸' },
  { code: 'es', name: 'Spanish', nativeName: 'Español', flag: '🇪🇸' },
  { code: 'fr', name: 'French', nativeName: 'Français', flag: '🇫🇷' },
  { code: 'de', name: 'German', nativeName: 'Deutsch', flag: '🇩🇪' },
  { code: 'it', name: 'Italian', nativeName: 'Italiano', flag: '🇮🇹' },
  { code: 'pt', name: 'Portuguese', nativeName: 'Português', flag: '🇵🇹' },
  { code: 'ru', name: 'Russian', nativeName: 'Русский', flag: '🇷🇺' },
  { code: 'zh', name: 'Chinese', nativeName: '中文', flag: '🇨🇳' },
  { code: 'ja', name: 'Japanese', nativeName: '日本語', flag: '🇯🇵' },
  { code: 'ko', name: 'Korean', nativeName: '한국어', flag: '🇰🇷' },
];
