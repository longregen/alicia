import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import {
  MessageId,
  SentenceId,
  ToolCallId,
  ConversationId,
  Message,
  MessageSentence,
  ToolCall,
  AudioRef,
  MemoryTrace,
  MessageStatus,
  ConversationStoreState,
} from '../types/streaming';

interface ConversationStoreActions {
  // Message actions
  addMessage: (message: Message) => void;
  updateMessageStatus: (id: MessageId, status: MessageStatus) => void;

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
  loadConversation: (conversationId: ConversationId, messages: Message[]) => void;

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
        state.messages[message.id] = message;
      }),

    updateMessageStatus: (id, status) =>
      set((state) => {
        if (state.messages[id]) {
          state.messages[id].status = status;
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
export const selectMessages = (state: ConversationStore) =>
  Object.values(state.messages).sort((a, b) =>
    a.createdAt.getTime() - b.createdAt.getTime()
  );

export const selectCurrentStreamingMessage = (state: ConversationStore) =>
  state.currentStreamingMessageId
    ? state.messages[state.currentStreamingMessageId]
    : null;
