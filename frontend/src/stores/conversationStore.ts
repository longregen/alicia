import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import {
  MessageId,
  SentenceId,
  ToolCallId,
  ConversationId,
  NormalizedMessage,
  MessageSentence,
  ToolCall,
  AudioRef,
  MemoryTrace,
  MessageStatus,
  ConversationStoreState,
} from '../types/streaming';

interface ConversationStoreActions {
  // Message actions
  addMessage: (message: NormalizedMessage) => void;
  updateMessageStatus: (id: MessageId, status: MessageStatus) => void;
  updateMessageContent: (id: MessageId, content: string) => void;

  // Sentence actions
  addSentence: (sentence: MessageSentence) => void;
  updateSentence: (id: SentenceId, update: Partial<MessageSentence>) => void;

  // Tool call actions
  addToolCall: (toolCall: ToolCall) => void;
  updateToolCall: (id: ToolCallId, update: Partial<ToolCall>) => void;

  // Audio ref actions
  addAudioRef: (audioRef: AudioRef) => void;

  // Memory trace actions
  addMemoryTrace: (trace: MemoryTrace) => void;

  // Streaming state actions
  setCurrentStreamingMessageId: (id: MessageId | null) => void;
  setCurrentConversationId: (id: ConversationId | null) => void;

  // Bulk operations
  clearConversation: () => void;
  loadConversation: (conversationId: ConversationId, messages: NormalizedMessage[]) => void;
  mergeMessages: (conversationId: ConversationId, messages: NormalizedMessage[]) => void;

  // Selectors (computed helpers)
  getMessageSentences: (messageId: MessageId) => MessageSentence[];
  getMessageToolCalls: (messageId: MessageId) => ToolCall[];
  getMessageMemoryTraces: (messageId: MessageId) => MemoryTrace[];
}

type ConversationStore = ConversationStoreState & ConversationStoreActions;

const initialState: ConversationStoreState = {
  messages: {},
  sentences: {},
  toolCalls: {},
  audioRefs: {},
  memoryTraces: {},
  currentStreamingMessageId: null,
  currentConversationId: null,
};

export const useConversationStore = create<ConversationStore>()(
  immer((set, get) => ({
    ...initialState,

    // Message actions
    addMessage: (message) =>
      set((state) => {
        // Deduplication: Check if message already exists by ID
        if (state.messages[message.id]) {
          // Message with this ID already exists - update it instead of duplicating
          state.messages[message.id] = message;
          return;
        }

        // If message has local_id, check if we already have a message with that local_id
        // This handles the case where a message was added with a local_id and later
        // we receive the same message with a server-assigned ID
        const localId = message.local_id;
        if (localId) {
          const existingMessage = Object.values(state.messages).find(
            (m) => m.local_id === localId
          );
          if (existingMessage) {
            // Found existing message with same local_id - update it
            delete state.messages[existingMessage.id];
            state.messages[message.id] = message;
            return;
          }
        }

        // No duplicate found - add the new message
        state.messages[message.id] = message;
      }),

    updateMessageStatus: (id, status) =>
      set((state) => {
        if (state.messages[id]) {
          state.messages[id].status = status;
        }
      }),

    updateMessageContent: (id, content) =>
      set((state) => {
        if (state.messages[id]) {
          state.messages[id].content = content;
        }
      }),

    // Sentence actions
    addSentence: (sentence) =>
      set((state) => {
        state.sentences[sentence.id] = sentence;
        // Also update the message's sentenceIds array
        const message = state.messages[sentence.messageId];
        if (message && !message.sentenceIds.includes(sentence.id)) {
          message.sentenceIds.push(sentence.id);
        }
      }),

    updateSentence: (id, update) =>
      set((state) => {
        if (state.sentences[id]) {
          Object.assign(state.sentences[id], update);
        }
      }),

    // Tool call actions
    addToolCall: (toolCall) =>
      set((state) => {
        state.toolCalls[toolCall.id] = toolCall;
        // Also update the message's toolCallIds array
        const message = state.messages[toolCall.messageId];
        if (message && !message.toolCallIds.includes(toolCall.id)) {
          message.toolCallIds.push(toolCall.id);
        }
      }),

    updateToolCall: (id, update) =>
      set((state) => {
        if (state.toolCalls[id]) {
          Object.assign(state.toolCalls[id], update);
        }
      }),

    // Audio ref actions
    addAudioRef: (audioRef) =>
      set((state) => {
        state.audioRefs[audioRef.id] = audioRef;
      }),

    // Memory trace actions
    addMemoryTrace: (trace) =>
      set((state) => {
        state.memoryTraces[trace.id] = trace;
        // Also update the message's memoryTraceIds array
        const message = state.messages[trace.messageId];
        if (message && !message.memoryTraceIds.includes(trace.id)) {
          message.memoryTraceIds.push(trace.id);
        }
      }),

    // Streaming state actions
    setCurrentStreamingMessageId: (id) =>
      set((state) => {
        state.currentStreamingMessageId = id;
      }),

    setCurrentConversationId: (id) =>
      set((state) => {
        state.currentConversationId = id;
      }),

    // Bulk operations
    clearConversation: () =>
      set((state) => {
        Object.assign(state, initialState);
      }),

    loadConversation: (conversationId, messages) =>
      set((state) => {
        state.currentConversationId = conversationId;
        state.messages = {};
        messages.forEach((msg) => {
          state.messages[msg.id] = msg;
        });
      }),

    mergeMessages: (conversationId, messages) =>
      set((state) => {
        // Get the set of message IDs that should exist
        const newMessageIds = new Set(messages.map((m) => m.id));

        // Find messages to remove (in this conversation but not in new set)
        const messagesToRemove = Object.values(state.messages).filter(
          (m) => m.conversationId === conversationId && !newMessageIds.has(m.id)
        );

        // Collect all related IDs from removed messages for cleanup
        const sentenceIdsToRemove = new Set<SentenceId>();
        const toolCallIdsToRemove = new Set<ToolCallId>();
        const memoryTraceIdsToRemove = new Set<string>();

        for (const msg of messagesToRemove) {
          msg.sentenceIds.forEach((id) => sentenceIdsToRemove.add(id));
          msg.toolCallIds.forEach((id) => toolCallIdsToRemove.add(id));
          msg.memoryTraceIds.forEach((id) => memoryTraceIdsToRemove.add(id));
          delete state.messages[msg.id];
        }

        // Collect audioRefIds from sentences being removed
        const audioRefIdsToRemove = new Set<string>();
        for (const sentenceId of sentenceIdsToRemove) {
          const sentence = state.sentences[sentenceId];
          if (sentence?.audioRefId) {
            audioRefIdsToRemove.add(sentence.audioRefId);
          }
          delete state.sentences[sentenceId];
        }

        // Clean up tool calls
        for (const toolCallId of toolCallIdsToRemove) {
          delete state.toolCalls[toolCallId];
        }

        // Clean up memory traces
        for (const traceId of memoryTraceIdsToRemove) {
          delete state.memoryTraces[traceId];
        }

        // Only remove audioRefs that are not referenced by any remaining sentence
        const referencedAudioRefIds = new Set<string>();
        for (const sentence of Object.values(state.sentences)) {
          if (sentence.audioRefId) {
            referencedAudioRefIds.add(sentence.audioRefId);
          }
        }
        for (const audioRefId of audioRefIdsToRemove) {
          if (!referencedAudioRefIds.has(audioRefId)) {
            delete state.audioRefs[audioRefId];
          }
        }

        // Add/update messages from the new set
        // PRESERVE existing streaming state (sentenceIds, toolCallIds, memoryTraceIds)
        for (const msg of messages) {
          const existing = state.messages[msg.id];
          if (existing) {
            // Merge new message data while preserving streaming arrays
            state.messages[msg.id] = {
              ...msg,
              sentenceIds: existing.sentenceIds,
              toolCallIds: existing.toolCallIds,
              memoryTraceIds: existing.memoryTraceIds,
            };
          } else {
            // New message - just add it
            state.messages[msg.id] = msg;
          }
        }
      }),

    // Selectors
    getMessageSentences: (messageId) => {
      const state = get();
      const message = state.messages[messageId];
      if (!message) return [];
      return message.sentenceIds
        .map((id) => state.sentences[id])
        .filter(Boolean)
        .sort((a, b) => a.sequence - b.sequence);
    },

    getMessageToolCalls: (messageId) => {
      const state = get();
      const message = state.messages[messageId];
      if (!message) return [];
      return message.toolCallIds
        .map((id) => state.toolCalls[id])
        .filter(Boolean);
    },

    getMessageMemoryTraces: (messageId) => {
      const state = get();
      const message = state.messages[messageId];
      if (!message) return [];
      return message.memoryTraceIds
        .map((id) => state.memoryTraces[id])
        .filter(Boolean)
        .sort((a, b) => b.relevance - a.relevance);
    },
  }))
);

// Utility selectors for common patterns
// Note: These selectors return derived state. Components will re-render when
// the selected value changes (using Object.is comparison by default).
export const selectMessages = (state: ConversationStore) => state.messages;
export const selectSentences = (state: ConversationStore) => state.sentences;

export const selectCurrentStreamingMessage = (state: ConversationStore) =>
  state.currentStreamingMessageId
    ? state.messages[state.currentStreamingMessageId]
    : null;
