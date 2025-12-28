import { describe, it, expect, beforeEach } from 'vitest';
import { act } from '@testing-library/react';
import {
  useConversationStore,
  selectMessages,
  selectCurrentStreamingMessage,
  selectMessage,
  selectMessageSentences,
  selectMessageToolCalls,
  selectMessageMemoryTraces,
  selectToolCall,
  selectSentence,
  selectCurrentConversationId,
  selectCurrentStreamingMessageId,
  selectActions,
} from './useConversationStore';
import type { Message, MessageSentence, ToolCall, MemoryTrace } from '../types/streaming';
import { MessageStatus } from '../types/streaming';

describe('useConversationStore', () => {
  // Reset store before each test
  beforeEach(() => {
    const store = useConversationStore.getState();
    store.clearConversation();
  });

  const createMockMessage = (id: string, createdAt: Date = new Date()): Message => ({
    id: id as any,
    conversationId: 'conv-1' as any,
    role: 'assistant',
    content: '',
    status: MessageStatus.Complete,
    createdAt,
    sentenceIds: [],
    toolCallIds: [],
    memoryTraceIds: [],
  });

  const createMockSentence = (id: string, messageId: string, sequence: number): MessageSentence => ({
    id: id as any,
    messageId: messageId as any,
    content: `Sentence ${sequence}`,
    sequence,
    isComplete: true,
  });

  const createMockToolCall = (id: string, messageId: string): ToolCall => ({
    id: id as any,
    messageId: messageId as any,
    toolName: 'test-tool',
    arguments: {},
    status: 'success',
    startTimeMs: Date.now(),
    endTimeMs: Date.now(),
    resultContent: '',
  });

  const createMockMemoryTrace = (id: string, messageId: string, relevance: number): MemoryTrace => ({
    id: id as any,
    messageId: messageId as any,
    content: 'Test memory',
    relevance,
  });

  describe('selectMessage', () => {
    it('returns message by id', () => {
      const store = useConversationStore.getState();
      const message = createMockMessage('msg-1');

      act(() => {
        store.addMessage(message);
      });

      const selector = selectMessage('msg-1' as any);
      const result = selector(useConversationStore.getState());

      expect(result).toEqual(message);
    });

    it('returns undefined for non-existent message', () => {
      const selector = selectMessage('non-existent' as any);
      const result = selector(useConversationStore.getState());

      expect(result).toBeUndefined();
    });
  });

  describe('selectMessages', () => {
    it('returns messages sorted by createdAt', () => {
      const store = useConversationStore.getState();
      const msg1 = createMockMessage('msg-1', new Date('2024-01-01T10:00:00'));
      const msg2 = createMockMessage('msg-2', new Date('2024-01-01T09:00:00'));
      const msg3 = createMockMessage('msg-3', new Date('2024-01-01T11:00:00'));

      act(() => {
        store.addMessage(msg1);
        store.addMessage(msg2);
        store.addMessage(msg3);
      });

      const result = selectMessages(useConversationStore.getState());

      expect(result).toHaveLength(3);
      expect(result[0].id).toBe('msg-2');
      expect(result[1].id).toBe('msg-1');
      expect(result[2].id).toBe('msg-3');
    });

    it('returns empty array when no messages', () => {
      const result = selectMessages(useConversationStore.getState());
      expect(result).toEqual([]);
    });
  });

  describe('selectCurrentStreamingMessage', () => {
    it('returns current streaming message when set', () => {
      const store = useConversationStore.getState();
      const message = createMockMessage('msg-1');

      act(() => {
        store.addMessage(message);
        store.setCurrentStreamingMessageId('msg-1' as any);
      });

      const result = selectCurrentStreamingMessage(useConversationStore.getState());

      expect(result).toEqual(message);
    });

    it('returns null when no streaming message', () => {
      const result = selectCurrentStreamingMessage(useConversationStore.getState());
      expect(result).toBeNull();
    });
  });

  describe('selectMessageSentences', () => {
    it('returns sentences for a message sorted by sequence', () => {
      const store = useConversationStore.getState();
      const message = createMockMessage('msg-1');
      const sentence1 = createMockSentence('sent-1', 'msg-1', 2);
      const sentence2 = createMockSentence('sent-2', 'msg-1', 1);
      const sentence3 = createMockSentence('sent-3', 'msg-1', 3);

      act(() => {
        store.addMessage(message);
        store.addSentence(sentence1);
        store.addSentence(sentence2);
        store.addSentence(sentence3);
      });

      const selector = selectMessageSentences('msg-1' as any);
      const result = selector(useConversationStore.getState());

      expect(result).toHaveLength(3);
      expect(result[0].sequence).toBe(1);
      expect(result[1].sequence).toBe(2);
      expect(result[2].sequence).toBe(3);
    });

    it('returns empty array for non-existent message', () => {
      const selector = selectMessageSentences('non-existent' as any);
      const result = selector(useConversationStore.getState());

      expect(result).toEqual([]);
    });
  });

  describe('selectMessageToolCalls', () => {
    it('returns tool calls for a message', () => {
      const store = useConversationStore.getState();
      const message = createMockMessage('msg-1');
      const toolCall = createMockToolCall('tool-1', 'msg-1');

      act(() => {
        store.addMessage(message);
        store.addToolCall(toolCall);
      });

      const selector = selectMessageToolCalls('msg-1' as any);
      const result = selector(useConversationStore.getState());

      expect(result).toHaveLength(1);
      expect(result[0]).toEqual(toolCall);
    });

    it('returns empty array for message with no tool calls', () => {
      const store = useConversationStore.getState();
      const message = createMockMessage('msg-1');

      act(() => {
        store.addMessage(message);
      });

      const selector = selectMessageToolCalls('msg-1' as any);
      const result = selector(useConversationStore.getState());

      expect(result).toEqual([]);
    });
  });

  describe('selectMessageMemoryTraces', () => {
    it('returns memory traces sorted by relevance (descending)', () => {
      const store = useConversationStore.getState();
      const message = createMockMessage('msg-1');
      const trace1 = createMockMemoryTrace('trace-1', 'msg-1', 0.5);
      const trace2 = createMockMemoryTrace('trace-2', 'msg-1', 0.9);
      const trace3 = createMockMemoryTrace('trace-3', 'msg-1', 0.7);

      act(() => {
        store.addMessage(message);
        store.addMemoryTrace(trace1);
        store.addMemoryTrace(trace2);
        store.addMemoryTrace(trace3);
      });

      const selector = selectMessageMemoryTraces('msg-1' as any);
      const result = selector(useConversationStore.getState());

      expect(result).toHaveLength(3);
      expect(result[0].relevance).toBe(0.9);
      expect(result[1].relevance).toBe(0.7);
      expect(result[2].relevance).toBe(0.5);
    });
  });

  describe('selectToolCall', () => {
    it('returns tool call by id', () => {
      const store = useConversationStore.getState();
      const message = createMockMessage('msg-1');
      const toolCall = createMockToolCall('tool-1', 'msg-1');

      act(() => {
        store.addMessage(message);
        store.addToolCall(toolCall);
      });

      const selector = selectToolCall('tool-1' as any);
      const result = selector(useConversationStore.getState());

      expect(result).toEqual(toolCall);
    });
  });

  describe('selectSentence', () => {
    it('returns sentence by id', () => {
      const store = useConversationStore.getState();
      const message = createMockMessage('msg-1');
      const sentence = createMockSentence('sent-1', 'msg-1', 1);

      act(() => {
        store.addMessage(message);
        store.addSentence(sentence);
      });

      const selector = selectSentence('sent-1' as any);
      const result = selector(useConversationStore.getState());

      expect(result).toEqual(sentence);
    });
  });

  describe('selectCurrentConversationId', () => {
    it('returns current conversation id', () => {
      const store = useConversationStore.getState();

      act(() => {
        store.setCurrentConversationId('conv-1' as any);
      });

      const result = selectCurrentConversationId(useConversationStore.getState());

      expect(result).toBe('conv-1');
    });

    it('returns null when not set', () => {
      const result = selectCurrentConversationId(useConversationStore.getState());
      expect(result).toBeNull();
    });
  });

  describe('selectCurrentStreamingMessageId', () => {
    it('returns current streaming message id', () => {
      const store = useConversationStore.getState();

      act(() => {
        store.setCurrentStreamingMessageId('msg-1' as any);
      });

      const result = selectCurrentStreamingMessageId(useConversationStore.getState());

      expect(result).toBe('msg-1');
    });
  });

  describe('selectActions', () => {
    it('returns all store actions', () => {
      const actions = selectActions(useConversationStore.getState());

      expect(actions).toHaveProperty('addMessage');
      expect(actions).toHaveProperty('updateMessageStatus');
      expect(actions).toHaveProperty('addSentence');
      expect(actions).toHaveProperty('updateSentence');
      expect(actions).toHaveProperty('addToolCall');
      expect(actions).toHaveProperty('updateToolCall');
      expect(actions).toHaveProperty('addAudioRef');
      expect(actions).toHaveProperty('addMemoryTrace');
      expect(actions).toHaveProperty('setCurrentStreamingMessageId');
      expect(actions).toHaveProperty('setCurrentConversationId');
      expect(actions).toHaveProperty('clearConversation');
      expect(actions).toHaveProperty('loadConversation');
    });

    it('actions work correctly', () => {
      const actions = selectActions(useConversationStore.getState());
      const message = createMockMessage('msg-1');

      act(() => {
        actions.addMessage(message);
      });

      const storedMessage = selectMessage('msg-1' as any)(useConversationStore.getState());
      expect(storedMessage).toEqual(message);
    });
  });

  describe('loadConversation', () => {
    it('loads multiple messages and sets conversation id', () => {
      const store = useConversationStore.getState();
      const messages = [
        createMockMessage('msg-1', new Date('2024-01-01T10:00:00')),
        createMockMessage('msg-2', new Date('2024-01-01T11:00:00')),
      ];

      act(() => {
        store.loadConversation('conv-1' as any, messages);
      });

      const state = useConversationStore.getState();
      expect(state.currentConversationId).toBe('conv-1');
      expect(Object.keys(state.messages)).toHaveLength(2);
    });
  });
});
