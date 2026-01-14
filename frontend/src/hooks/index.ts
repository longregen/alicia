// Barrel export for hooks
export { useFeedback } from './useFeedback';
export { useFeedbackVisibility } from './useFeedbackVisibility';
export { useFeedbackAggregates } from './useFeedbackAggregates';
export type { SentimentIndicator, FeedbackAggregates } from './useFeedbackAggregates';
export { useServerInfo, useConnectionStatus, useSessionStats } from './useServerInfo';
export type { ConnectionQuality } from './useServerInfo';
export { useNotes } from './useNotes';
export { useMemories } from './useMemories';
export type {
  MemoryAPIResponse,
  MemoryListResponse,
  SearchResultResponse,
  SearchResultsResponse,
} from './useMemories';
export { useTheme } from './useTheme';
export type { Theme } from './useTheme';
export { useAsync } from './useAsync';
export { useConversations } from './useConversations';
export { useMessages } from './useMessages';
export { useLiveQuery } from './useLiveQuery';
export { useConversationStore } from './useConversationStore';

// Internal-only hooks (not exported):
// - useDatabase: Internal DB abstraction
// - useSync: Internal sync layer (wrapped by useMessages)
// - useVAD: Internal voice activity detection
// - useWebSocketSync: Internal sync transport (wrapped by useSync)
// - useAudioManager: Internal audio handling
// - useLiveKit: Internal LiveKit transport
