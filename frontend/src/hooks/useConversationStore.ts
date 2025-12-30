import { useConversationStore as useStore, selectMessages, selectSentences, selectCurrentStreamingMessage } from '../stores/conversationStore';
import type { MessageId, ToolCallId, SentenceId } from '../types/streaming';

/**
 * Hook wrapper for conversation store with convenient selectors.
 *
 * Re-exports the Zustand store hook and provides typed selector functions
 * for common use cases throughout the application.
 */

// Re-export the base store hook
export const useConversationStore = useStore;

// Re-export utility selectors
export { selectMessages, selectSentences, selectCurrentStreamingMessage };

// Typed selector functions for common patterns
export const selectMessage = (messageId: MessageId) => (state: ReturnType<typeof useStore.getState>) =>
  state.messages[messageId];

// IMPORTANT: These selectors return raw state references. Components using these
// should compute derived data with useMemo to avoid infinite re-render loops.
export const selectMessageSentenceIds = (messageId: MessageId) => (state: ReturnType<typeof useStore.getState>) =>
  state.messages[messageId]?.sentenceIds ?? [];

export const selectMessageToolCallIds = (messageId: MessageId) => (state: ReturnType<typeof useStore.getState>) =>
  state.messages[messageId]?.toolCallIds ?? [];

export const selectMessageMemoryTraceIds = (messageId: MessageId) => (state: ReturnType<typeof useStore.getState>) =>
  state.messages[messageId]?.memoryTraceIds ?? [];

export const selectToolCall = (toolCallId: ToolCallId) => (state: ReturnType<typeof useStore.getState>) =>
  state.toolCalls[toolCallId];

export const selectSentence = (sentenceId: SentenceId) => (state: ReturnType<typeof useStore.getState>) =>
  state.sentences[sentenceId];

export const selectCurrentConversationId = (state: ReturnType<typeof useStore.getState>) =>
  state.currentConversationId;

export const selectCurrentStreamingMessageId = (state: ReturnType<typeof useStore.getState>) =>
  state.currentStreamingMessageId;

// Action selectors for convenience
export const selectActions = (state: ReturnType<typeof useStore.getState>) => ({
  addMessage: state.addMessage,
  updateMessageStatus: state.updateMessageStatus,
  addSentence: state.addSentence,
  updateSentence: state.updateSentence,
  addToolCall: state.addToolCall,
  updateToolCall: state.updateToolCall,
  addAudioRef: state.addAudioRef,
  addMemoryTrace: state.addMemoryTrace,
  setCurrentStreamingMessageId: state.setCurrentStreamingMessageId,
  setCurrentConversationId: state.setCurrentConversationId,
  clearConversation: state.clearConversation,
  loadConversation: state.loadConversation,
});
