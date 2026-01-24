import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import { enableMapSet } from 'immer';

import {
  ChatMessage,
  MessageId,
  ConversationId,
  ToolCall,
  MemoryTrace,
  MessageStatus,
} from '../types/chat';

enableMapSet();

export function computeActiveBranch(
  messages: Map<MessageId, ChatMessage>,
  tipMessageId: MessageId | null
): ChatMessage[] {
  if (!tipMessageId) {
    return Array.from(messages.values()).sort(
      (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    );
  }

  const branch: ChatMessage[] = [];
  let current = messages.get(tipMessageId);
  const visited = new Set<MessageId>();

  while (current && !visited.has(current.id)) {
    visited.add(current.id);
    branch.push(current);
    current = current.previous_id ? messages.get(current.previous_id) : undefined;
  }

  branch.reverse();
  return branch;
}

export interface ConversationState {
  messages: Map<MessageId, ChatMessage>;
  tipMessageId: MessageId | null;
  streamingMessageId: MessageId | null;
  optimisticMessageId: MessageId | null;
}

interface ChatState {
  conversations: Map<ConversationId, ConversationState>;
  activeConversationId: ConversationId | null;
}

export function createEmptyConversationState(): ConversationState {
  return {
    messages: new Map(),
    tipMessageId: null,
    streamingMessageId: null,
    optimisticMessageId: null,
  };
}

interface ChatActions {
  // Per-conversation actions
  setMessages: (conversationId: ConversationId, messages: ChatMessage[]) => void;
  addMessage: (conversationId: ConversationId, message: ChatMessage) => void;
  updateMessage: (conversationId: ConversationId, id: MessageId, updates: Partial<ChatMessage>) => void;
  deleteMessage: (conversationId: ConversationId, id: MessageId) => void;
  appendContent: (conversationId: ConversationId, id: MessageId, text: string) => void;
  updateMessageStatus: (conversationId: ConversationId, id: MessageId, status: MessageStatus) => void;
  addToolCall: (conversationId: ConversationId, messageId: MessageId, toolCall: ToolCall) => void;
  updateToolCall: (conversationId: ConversationId, messageId: MessageId, toolCallId: string, updates: Partial<ToolCall>) => void;
  addMemoryTrace: (conversationId: ConversationId, messageId: MessageId, trace: MemoryTrace) => void;
  startStreaming: (conversationId: ConversationId, messageId: MessageId) => void;
  finishStreaming: (conversationId: ConversationId) => void;
  initiateRegeneration: (conversationId: ConversationId, optimisticMessage: ChatMessage) => void;
  setTipMessageId: (conversationId: ConversationId, id: MessageId | null) => void;
  setOptimisticMessageId: (conversationId: ConversationId, id: MessageId | null) => void;
  getMessage: (conversationId: ConversationId, id: MessageId) => ChatMessage | undefined;
  getConversationState: (conversationId: ConversationId) => ConversationState | undefined;

  // Lifecycle actions
  setActiveConversation: (conversationId: ConversationId | null) => void;
  ensureConversation: (conversationId: ConversationId) => void;
  clearConversation: (conversationId: ConversationId) => void;
}

type ChatStore = ChatState & ChatActions;

function getOrCreateConv(state: ChatState, conversationId: ConversationId): ConversationState {
  let conv = state.conversations.get(conversationId);
  if (!conv) {
    conv = createEmptyConversationState();
    state.conversations.set(conversationId, conv);
  }
  return conv;
}

const MAX_INACTIVE_CONVERSATIONS = 5;

const initialState: ChatState = {
  conversations: new Map(),
  activeConversationId: null,
};

export const useChatStore = create<ChatStore>()(
  immer((set, get) => ({
    ...initialState,

    setMessages: (conversationId, messages) =>
      set((state) => {
        const conv = getOrCreateConv(state, conversationId);
        const newMap = new Map(messages.map((m) => [m.id, m]));

        // Preserve active streaming message if not in new set
        if (conv.streamingMessageId) {
          const streamingMsg = conv.messages.get(conv.streamingMessageId);
          if (streamingMsg && !newMap.has(conv.streamingMessageId)) {
            newMap.set(conv.streamingMessageId, streamingMsg);
          }
        }

        conv.messages = newMap;
      }),

    addMessage: (conversationId, message) =>
      set((state) => {
        const conv = getOrCreateConv(state, conversationId);
        conv.messages.set(message.id, message);
      }),

    updateMessage: (conversationId, id, updates) =>
      set((state) => {
        const conv = getOrCreateConv(state, conversationId);
        const msg = conv.messages.get(id);
        if (msg) {
          if (updates.id && updates.id !== id) {
            const newId = updates.id;
            Object.assign(msg, updates);
            conv.messages.delete(id);
            conv.messages.set(newId, msg);
          } else {
            Object.assign(msg, updates);
          }
        }
      }),

    deleteMessage: (conversationId, id) =>
      set((state) => {
        const conv = state.conversations.get(conversationId);
        if (conv) {
          conv.messages.delete(id);
        }
      }),

    appendContent: (conversationId, id, text) =>
      set((state) => {
        const conv = state.conversations.get(conversationId);
        if (conv) {
          const msg = conv.messages.get(id);
          if (msg) {
            msg.content += text;
          }
        }
      }),

    updateMessageStatus: (conversationId, id, status) =>
      set((state) => {
        const conv = state.conversations.get(conversationId);
        if (conv) {
          const msg = conv.messages.get(id);
          if (msg) {
            msg.status = status;
          }
        }
      }),

    addToolCall: (conversationId, messageId, toolCall) =>
      set((state) => {
        const conv = state.conversations.get(conversationId);
        if (conv) {
          const msg = conv.messages.get(messageId);
          if (msg) {
            const idx = msg.tool_calls.findIndex((tc) => tc.id === toolCall.id);
            if (idx >= 0) {
              msg.tool_calls[idx] = toolCall;
            } else {
              msg.tool_calls.push(toolCall);
            }
          }
        }
      }),

    updateToolCall: (conversationId, messageId, toolCallId, updates) =>
      set((state) => {
        const conv = state.conversations.get(conversationId);
        if (conv) {
          const msg = conv.messages.get(messageId);
          if (msg) {
            const tc = msg.tool_calls.find((t) => t.id === toolCallId);
            if (tc) {
              Object.assign(tc, updates);
            }
          }
        }
      }),

    addMemoryTrace: (conversationId, messageId, trace) =>
      set((state) => {
        const conv = state.conversations.get(conversationId);
        if (conv) {
          const msg = conv.messages.get(messageId);
          if (msg) {
            const idx = msg.memory_traces.findIndex((t) => t.id === trace.id);
            if (idx >= 0) {
              msg.memory_traces[idx] = trace;
            } else {
              msg.memory_traces.push(trace);
            }
          }
        }
      }),

    startStreaming: (conversationId, messageId) =>
      set((state) => {
        const conv = getOrCreateConv(state, conversationId);
        conv.streamingMessageId = messageId;
        const msg = conv.messages.get(messageId);
        if (msg) {
          msg.status = 'streaming';
        }
      }),

    finishStreaming: (conversationId) =>
      set((state) => {
        const conv = state.conversations.get(conversationId);
        if (conv && conv.streamingMessageId) {
          const msg = conv.messages.get(conv.streamingMessageId);
          if (msg && msg.status === 'streaming') {
            msg.status = 'completed';
          }
          conv.streamingMessageId = null;
        }
      }),

    initiateRegeneration: (conversationId, optimisticMessage) =>
      set((state) => {
        const conv = getOrCreateConv(state, conversationId);
        optimisticMessage.status = 'streaming';
        conv.messages.set(optimisticMessage.id, optimisticMessage);
        // Don't change tipMessageId here - the previous_id chain may be incomplete.
        // MessageList appends streaming messages not in the branch at the end.
        // The tip will be set correctly when StartAnswer arrives with the server's previous_id.
        conv.optimisticMessageId = optimisticMessage.id;
        conv.streamingMessageId = optimisticMessage.id;
      }),

    setTipMessageId: (conversationId, id) =>
      set((state) => {
        const conv = getOrCreateConv(state, conversationId);
        conv.tipMessageId = id;
      }),

    setOptimisticMessageId: (conversationId, id) =>
      set((state) => {
        const conv = getOrCreateConv(state, conversationId);
        conv.optimisticMessageId = id;
      }),

    getMessage: (conversationId, id) => {
      const conv = get().conversations.get(conversationId);
      return conv ? conv.messages.get(id) : undefined;
    },

    getConversationState: (conversationId) => {
      return get().conversations.get(conversationId);
    },

    setActiveConversation: (conversationId) =>
      set((state) => {
        state.activeConversationId = conversationId;

        // Eviction: if more than 5 non-streaming conversations exist, drop the oldest
        if (state.conversations.size > MAX_INACTIVE_CONVERSATIONS) {
          const entries = Array.from(state.conversations.entries());
          const inactive = entries.filter(
            ([id, conv]) => id !== conversationId && conv.streamingMessageId === null
          );

          // Sort by most recent message time ascending (oldest first)
          const getMaxTime = (msgs: Iterable<ChatMessage>): number => {
            let max = 0;
            for (const m of msgs) {
              const t = new Date(m.created_at).getTime();
              if (t > max) max = t;
            }
            return max;
          };

          inactive.sort((a, b) => {
            const aTime = getMaxTime(a[1].messages.values());
            const bTime = getMaxTime(b[1].messages.values());
            return aTime - bTime;
          });

          const toEvict = state.conversations.size - MAX_INACTIVE_CONVERSATIONS;
          for (let i = 0; i < toEvict && i < inactive.length; i++) {
            state.conversations.delete(inactive[i][0]);
          }
        }
      }),

    ensureConversation: (conversationId) =>
      set((state) => {
        if (!state.conversations.has(conversationId)) {
          state.conversations.set(conversationId, createEmptyConversationState());
        }
      }),

    clearConversation: (conversationId) =>
      set((state) => {
        state.conversations.delete(conversationId);
      }),

  }))
);

export const selectActiveConversationId = (state: ChatStore) => state.activeConversationId;

const emptyMessages = new Map<MessageId, ChatMessage>();
const emptyBranch: ChatMessage[] = [];

export const selectConversationMessages = (conversationId: ConversationId | null) =>
  (state: ChatStore): Map<MessageId, ChatMessage> => {
    if (!conversationId) return emptyMessages;
    return state.conversations.get(conversationId)?.messages ?? emptyMessages;
  };

export const selectConversationTipMessageId = (conversationId: ConversationId | null) =>
  (state: ChatStore): MessageId | null => {
    if (!conversationId) return null;
    return state.conversations.get(conversationId)?.tipMessageId ?? null;
  };

export const selectConversationStreamingMessageId = (conversationId: ConversationId | null) =>
  (state: ChatStore): MessageId | null => {
    if (!conversationId) return null;
    return state.conversations.get(conversationId)?.streamingMessageId ?? null;
  };

export const selectConversationStreamingMessage = (conversationId: ConversationId | null) =>
  (state: ChatStore): ChatMessage | null => {
    if (!conversationId) return null;
    const conv = state.conversations.get(conversationId);
    if (!conv || !conv.streamingMessageId) return null;
    return conv.messages.get(conv.streamingMessageId) ?? null;
  };

const branchCache = new WeakMap<ConversationState, { tip: MessageId | null; branch: ChatMessage[] }>();

export const selectConversationActiveBranch = (conversationId: ConversationId | null) =>
  (state: ChatStore): ChatMessage[] => {
    if (!conversationId) return emptyBranch;
    const conv = state.conversations.get(conversationId);
    if (!conv) return emptyBranch;
    const cached = branchCache.get(conv);
    if (cached && cached.tip === conv.tipMessageId) return cached.branch;
    const branch = computeActiveBranch(conv.messages, conv.tipMessageId);
    branchCache.set(conv, { tip: conv.tipMessageId, branch });
    return branch;
  };
