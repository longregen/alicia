import { describe, it, expect, beforeEach } from 'vitest';
import { useConversationStore, selectMessages, selectCurrentStreamingMessage } from './conversationStore';
import {
  createMessageId,
  createSentenceId,
  createToolCallId,
  createAudioRefId,
  createMemoryTraceId,
  createConversationId,
  MessageStatus,
  type Message,
  type MessageSentence,
  type ToolCall,
  type AudioRef,
  type MemoryTrace,
} from '../types/streaming';

describe('conversationStore', () => {
  beforeEach(() => {
    // Reset store state before each test
    useConversationStore.getState().clearConversation();
  });

  describe('addMessage', () => {
    it('should add a message to the store', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'Hello',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(message);

      const state = useConversationStore.getState();
      expect(state.messages['msg-1']).toBeDefined();
      expect(state.messages['msg-1']).toEqual(message);
    });

    it('should create message with correct initial arrays', () => {
      const message: Message = {
        id: createMessageId('msg-2'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Hi there',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(message);

      const state = useConversationStore.getState();
      expect(state.messages['msg-2'].sentenceIds).toEqual([]);
      expect(state.messages['msg-2'].toolCallIds).toEqual([]);
      expect(state.messages['msg-2'].memoryTraceIds).toEqual([]);
    });
  });

  describe('updateMessageStatus', () => {
    it('should update message status correctly', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Processing...',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().updateMessageStatus(message.id, MessageStatus.Complete);

      const state = useConversationStore.getState();
      expect(state.messages['msg-1'].status).toBe(MessageStatus.Complete);
    });

    it('should handle status transitions from streaming to error', () => {
      const message: Message = {
        id: createMessageId('msg-2'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Failed message',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().updateMessageStatus(message.id, MessageStatus.Error);

      const state = useConversationStore.getState();
      expect(state.messages['msg-2'].status).toBe(MessageStatus.Error);
    });

    it('should not throw when updating non-existent message', () => {
      expect(() => {
        useConversationStore.getState().updateMessageStatus(
          createMessageId('non-existent'),
          MessageStatus.Complete
        );
      }).not.toThrow();
    });
  });

  describe('addSentence', () => {
    it('should add sentence to store and update message.sentenceIds', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Hello world',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const sentence: MessageSentence = {
        id: createSentenceId('sent-1'),
        messageId: message.id,
        content: 'Hello',
        sequence: 0,
        isComplete: true,
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addSentence(sentence);

      const state = useConversationStore.getState();
      expect(state.sentences['sent-1']).toBeDefined();
      expect(state.sentences['sent-1']).toEqual(sentence);
      expect(state.messages['msg-1'].sentenceIds).toContain(createSentenceId('sent-1'));
    });

    it('should maintain bidirectional reference integrity', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const sentence: MessageSentence = {
        id: createSentenceId('sent-1'),
        messageId: message.id,
        content: 'Test sentence',
        sequence: 0,
        isComplete: true,
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addSentence(sentence);

      const state = useConversationStore.getState();
      // Forward reference: message -> sentence
      expect(state.messages['msg-1'].sentenceIds).toContain(sentence.id);
      // Backward reference: sentence -> message
      expect(state.sentences['sent-1'].messageId).toBe(message.id);
    });

    it('should not add duplicate sentenceIds to message', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const sentence: MessageSentence = {
        id: createSentenceId('sent-1'),
        messageId: message.id,
        content: 'Test',
        sequence: 0,
        isComplete: true,
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addSentence(sentence);
      useConversationStore.getState().addSentence(sentence); // Add again

      const state = useConversationStore.getState();
      expect(state.messages['msg-1'].sentenceIds).toHaveLength(1);
    });
  });

  describe('addToolCall', () => {
    it('should add tool call to store and update message.toolCallIds', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Using tool',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const toolCall: ToolCall = {
        id: createToolCallId('tool-1'),
        toolName: 'calculator',
        arguments: { operation: 'add', values: [1, 2] },
        messageId: message.id,
        status: 'pending',
        startTimeMs: Date.now(),
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addToolCall(toolCall);

      const state = useConversationStore.getState();
      expect(state.toolCalls['tool-1']).toBeDefined();
      expect(state.toolCalls['tool-1']).toEqual(toolCall);
      expect(state.messages['msg-1'].toolCallIds).toContain(createToolCallId('tool-1'));
    });

    it('should not add duplicate toolCallIds to message', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Using tool',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const toolCall: ToolCall = {
        id: createToolCallId('tool-1'),
        toolName: 'calculator',
        arguments: {},
        messageId: message.id,
        status: 'pending',
        startTimeMs: Date.now(),
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addToolCall(toolCall);
      useConversationStore.getState().addToolCall(toolCall); // Add again

      const state = useConversationStore.getState();
      expect(state.messages['msg-1'].toolCallIds).toHaveLength(1);
    });
  });

  describe('updateToolCall', () => {
    it('should update tool call status from pending to executing', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Using tool',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const toolCall: ToolCall = {
        id: createToolCallId('tool-1'),
        toolName: 'calculator',
        arguments: {},
        messageId: message.id,
        status: 'pending',
        startTimeMs: Date.now(),
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addToolCall(toolCall);
      useConversationStore.getState().updateToolCall(toolCall.id, { status: 'executing' });

      const state = useConversationStore.getState();
      expect(state.toolCalls['tool-1'].status).toBe('executing');
    });

    it('should update tool call status from executing to success with result', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Using tool',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const toolCall: ToolCall = {
        id: createToolCallId('tool-1'),
        toolName: 'calculator',
        arguments: {},
        messageId: message.id,
        status: 'executing',
        startTimeMs: Date.now(),
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addToolCall(toolCall);
      useConversationStore.getState().updateToolCall(toolCall.id, {
        status: 'success',
        endTimeMs: Date.now(),
        resultContent: '42',
      });

      const state = useConversationStore.getState();
      const updated = state.toolCalls['tool-1'] as Extract<ToolCall, { status: 'success' }>;
      expect(updated.status).toBe('success');
      expect(updated.resultContent).toBe('42');
      expect(updated.endTimeMs).toBeDefined();
    });

    it('should update tool call status from executing to error', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Using tool',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const toolCall: ToolCall = {
        id: createToolCallId('tool-1'),
        toolName: 'calculator',
        arguments: {},
        messageId: message.id,
        status: 'executing',
        startTimeMs: Date.now(),
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addToolCall(toolCall);
      useConversationStore.getState().updateToolCall(toolCall.id, {
        status: 'error',
        endTimeMs: Date.now(),
        error: 'Calculation failed',
      });

      const state = useConversationStore.getState();
      const updated = state.toolCalls['tool-1'] as Extract<ToolCall, { status: 'error' }>;
      expect(updated.status).toBe('error');
      expect(updated.error).toBe('Calculation failed');
      expect(updated.endTimeMs).toBeDefined();
    });
  });

  describe('addAudioRef', () => {
    it('should populate audioRefs map', () => {
      const audioRef: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      useConversationStore.getState().addAudioRef(audioRef);

      const state = useConversationStore.getState();
      expect(state.audioRefs['audio-1']).toBeDefined();
      expect(state.audioRefs['audio-1']).toEqual(audioRef);
    });

    it('should store multiple audio refs independently', () => {
      const audioRef1: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      const audioRef2: AudioRef = {
        id: createAudioRefId('audio-2'),
        sizeBytes: 2048,
        durationMs: 10000,
        sampleRate: 48000,
      };

      useConversationStore.getState().addAudioRef(audioRef1);
      useConversationStore.getState().addAudioRef(audioRef2);

      const state = useConversationStore.getState();
      expect(state.audioRefs['audio-1']).toEqual(audioRef1);
      expect(state.audioRefs['audio-2']).toEqual(audioRef2);
    });
  });

  describe('addMemoryTrace', () => {
    it('should add memory trace to store and update message.memoryTraceIds', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Remembering context',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const memoryTrace: MemoryTrace = {
        id: createMemoryTraceId('trace-1'),
        messageId: message.id,
        content: 'Previous conversation context',
        relevance: 0.95,
        source: 'long-term-memory',
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addMemoryTrace(memoryTrace);

      const state = useConversationStore.getState();
      expect(state.memoryTraces['trace-1']).toBeDefined();
      expect(state.memoryTraces['trace-1']).toEqual(memoryTrace);
      expect(state.messages['msg-1'].memoryTraceIds).toContain(createMemoryTraceId('trace-1'));
    });

    it('should not add duplicate memoryTraceIds to message', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const memoryTrace: MemoryTrace = {
        id: createMemoryTraceId('trace-1'),
        messageId: message.id,
        content: 'Context',
        relevance: 0.8,
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addMemoryTrace(memoryTrace);
      useConversationStore.getState().addMemoryTrace(memoryTrace); // Add again

      const state = useConversationStore.getState();
      expect(state.messages['msg-1'].memoryTraceIds).toHaveLength(1);
    });
  });

  describe('clearConversation', () => {
    it('should reset all state to initial values', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'Hello',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const sentence: MessageSentence = {
        id: createSentenceId('sent-1'),
        messageId: message.id,
        content: 'Hello',
        sequence: 0,
        isComplete: true,
      };

      const audioRef: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addSentence(sentence);
      useConversationStore.getState().addAudioRef(audioRef);
      useConversationStore.getState().setCurrentConversationId(createConversationId('conv-1'));
      useConversationStore.getState().setCurrentStreamingMessageId(message.id);

      useConversationStore.getState().clearConversation();

      const state = useConversationStore.getState();
      expect(Object.keys(state.messages)).toHaveLength(0);
      expect(Object.keys(state.sentences)).toHaveLength(0);
      expect(Object.keys(state.toolCalls)).toHaveLength(0);
      expect(Object.keys(state.audioRefs)).toHaveLength(0);
      expect(Object.keys(state.memoryTraces)).toHaveLength(0);
      expect(state.currentStreamingMessageId).toBeNull();
      expect(state.currentConversationId).toBeNull();
    });
  });

  describe('loadConversation', () => {
    it('should clear old state and load new messages', () => {
      const oldMessage: Message = {
        id: createMessageId('old-msg'),
        conversationId: createConversationId('old-conv'),
        role: 'user',
        content: 'Old message',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const newMessage1: Message = {
        id: createMessageId('new-msg-1'),
        conversationId: createConversationId('new-conv'),
        role: 'user',
        content: 'New message 1',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const newMessage2: Message = {
        id: createMessageId('new-msg-2'),
        conversationId: createConversationId('new-conv'),
        role: 'assistant',
        content: 'New message 2',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(oldMessage);
      useConversationStore.getState().loadConversation(
        createConversationId('new-conv'),
        [newMessage1, newMessage2]
      );

      const state = useConversationStore.getState();
      expect(state.currentConversationId).toBe('new-conv');
      expect(state.messages['old-msg']).toBeUndefined();
      expect(state.messages['new-msg-1']).toBeDefined();
      expect(state.messages['new-msg-2']).toBeDefined();
      expect(Object.keys(state.messages)).toHaveLength(2);
    });

    it('should set currentConversationId correctly', () => {
      const conversationId = createConversationId('conv-123');
      useConversationStore.getState().loadConversation(conversationId, []);

      const state = useConversationStore.getState();
      expect(state.currentConversationId).toBe('conv-123');
    });

    it('should handle empty message array', () => {
      useConversationStore.getState().loadConversation(createConversationId('empty-conv'), []);

      const state = useConversationStore.getState();
      expect(Object.keys(state.messages)).toHaveLength(0);
      expect(state.currentConversationId).toBe('empty-conv');
    });
  });

  describe('getMessageSentences selector', () => {
    it('should return sentences in correct sequence order', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Multi-sentence message',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const sentence1: MessageSentence = {
        id: createSentenceId('sent-1'),
        messageId: message.id,
        content: 'First sentence.',
        sequence: 2,
        isComplete: true,
      };

      const sentence2: MessageSentence = {
        id: createSentenceId('sent-2'),
        messageId: message.id,
        content: 'Second sentence.',
        sequence: 0,
        isComplete: true,
      };

      const sentence3: MessageSentence = {
        id: createSentenceId('sent-3'),
        messageId: message.id,
        content: 'Third sentence.',
        sequence: 1,
        isComplete: true,
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addSentence(sentence1);
      useConversationStore.getState().addSentence(sentence2);
      useConversationStore.getState().addSentence(sentence3);

      const sentences = useConversationStore.getState().getMessageSentences(message.id);

      expect(sentences).toHaveLength(3);
      expect(sentences[0].sequence).toBe(0);
      expect(sentences[1].sequence).toBe(1);
      expect(sentences[2].sequence).toBe(2);
      expect(sentences[0].content).toBe('Second sentence.');
      expect(sentences[1].content).toBe('Third sentence.');
      expect(sentences[2].content).toBe('First sentence.');
    });

    it('should return empty array for non-existent message', () => {
      const sentences = useConversationStore
        .getState()
        .getMessageSentences(createMessageId('non-existent'));

      expect(sentences).toEqual([]);
    });

    it('should filter out missing sentence references', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [createSentenceId('sent-1'), createSentenceId('missing')],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const sentence: MessageSentence = {
        id: createSentenceId('sent-1'),
        messageId: message.id,
        content: 'Valid sentence.',
        sequence: 0,
        isComplete: true,
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addSentence(sentence);

      const sentences = useConversationStore.getState().getMessageSentences(message.id);

      expect(sentences).toHaveLength(1);
      expect(sentences[0].id).toBe('sent-1');
    });
  });

  describe('getMessageToolCalls selector', () => {
    it('should return all tool calls for a message', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Using tools',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const toolCall1: ToolCall = {
        id: createToolCallId('tool-1'),
        toolName: 'calculator',
        arguments: {},
        messageId: message.id,
        status: 'success',
        startTimeMs: Date.now(),
        endTimeMs: Date.now(),
        resultContent: '42',
      };

      const toolCall2: ToolCall = {
        id: createToolCallId('tool-2'),
        toolName: 'weather',
        arguments: {},
        messageId: message.id,
        status: 'pending',
        startTimeMs: Date.now(),
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addToolCall(toolCall1);
      useConversationStore.getState().addToolCall(toolCall2);

      const toolCalls = useConversationStore.getState().getMessageToolCalls(message.id);

      expect(toolCalls).toHaveLength(2);
      expect(toolCalls.map((tc) => tc.id)).toContain('tool-1');
      expect(toolCalls.map((tc) => tc.id)).toContain('tool-2');
    });

    it('should return empty array for non-existent message', () => {
      const toolCalls = useConversationStore
        .getState()
        .getMessageToolCalls(createMessageId('non-existent'));

      expect(toolCalls).toEqual([]);
    });

    it('should filter out missing tool call references', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [createToolCallId('tool-1'), createToolCallId('missing')],
        memoryTraceIds: [],
      };

      const toolCall: ToolCall = {
        id: createToolCallId('tool-1'),
        toolName: 'test',
        arguments: {},
        messageId: message.id,
        status: 'pending',
        startTimeMs: Date.now(),
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addToolCall(toolCall);

      const toolCalls = useConversationStore.getState().getMessageToolCalls(message.id);

      expect(toolCalls).toHaveLength(1);
      expect(toolCalls[0].id).toBe('tool-1');
    });
  });

  describe('getMessageMemoryTraces selector', () => {
    it('should return memory traces sorted by relevance (descending)', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Using memory',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const trace1: MemoryTrace = {
        id: createMemoryTraceId('trace-1'),
        messageId: message.id,
        content: 'Low relevance',
        relevance: 0.3,
      };

      const trace2: MemoryTrace = {
        id: createMemoryTraceId('trace-2'),
        messageId: message.id,
        content: 'High relevance',
        relevance: 0.95,
      };

      const trace3: MemoryTrace = {
        id: createMemoryTraceId('trace-3'),
        messageId: message.id,
        content: 'Medium relevance',
        relevance: 0.6,
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addMemoryTrace(trace1);
      useConversationStore.getState().addMemoryTrace(trace2);
      useConversationStore.getState().addMemoryTrace(trace3);

      const traces = useConversationStore.getState().getMessageMemoryTraces(message.id);

      expect(traces).toHaveLength(3);
      expect(traces[0].relevance).toBe(0.95);
      expect(traces[1].relevance).toBe(0.6);
      expect(traces[2].relevance).toBe(0.3);
    });

    it('should return empty array for non-existent message', () => {
      const traces = useConversationStore
        .getState()
        .getMessageMemoryTraces(createMessageId('non-existent'));

      expect(traces).toEqual([]);
    });

    it('should filter out missing memory trace references', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [createMemoryTraceId('trace-1'), createMemoryTraceId('missing')],
      };

      const trace: MemoryTrace = {
        id: createMemoryTraceId('trace-1'),
        messageId: message.id,
        content: 'Valid trace',
        relevance: 0.8,
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().addMemoryTrace(trace);

      const traces = useConversationStore.getState().getMessageMemoryTraces(message.id);

      expect(traces).toHaveLength(1);
      expect(traces[0].id).toBe('trace-1');
    });
  });

  describe('selectMessages utility selector', () => {
    it('should return messages sorted by createdAt timestamp', () => {
      const now = new Date();
      const message1: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'Third',
        status: MessageStatus.Complete,
        createdAt: new Date(now.getTime() + 2000),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const message2: Message = {
        id: createMessageId('msg-2'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'First',
        status: MessageStatus.Complete,
        createdAt: new Date(now.getTime()),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const message3: Message = {
        id: createMessageId('msg-3'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'Second',
        status: MessageStatus.Complete,
        createdAt: new Date(now.getTime() + 1000),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(message1);
      useConversationStore.getState().addMessage(message2);
      useConversationStore.getState().addMessage(message3);

      const messages = selectMessages(useConversationStore.getState());

      expect(messages).toHaveLength(3);
      expect(messages[0].content).toBe('First');
      expect(messages[1].content).toBe('Second');
      expect(messages[2].content).toBe('Third');
    });
  });

  describe('selectCurrentStreamingMessage utility selector', () => {
    it('should return the current streaming message', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Streaming...',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(message);
      useConversationStore.getState().setCurrentStreamingMessageId(message.id);

      const currentMessage = selectCurrentStreamingMessage(useConversationStore.getState());

      expect(currentMessage).toBeDefined();
      expect(currentMessage?.id).toBe('msg-1');
      expect(currentMessage?.status).toBe(MessageStatus.Streaming);
    });

    it('should return null when no streaming message is set', () => {
      const currentMessage = selectCurrentStreamingMessage(useConversationStore.getState());

      expect(currentMessage).toBeNull();
    });

    it('should return null when streaming message ID does not exist', () => {
      useConversationStore
        .getState()
        .setCurrentStreamingMessageId(createMessageId('non-existent'));

      const currentMessage = selectCurrentStreamingMessage(useConversationStore.getState());

      expect(currentMessage).toBeUndefined();
    });
  });

  describe('bidirectional reference integrity', () => {
    it('should maintain sentence bidirectional references', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const sentences: MessageSentence[] = [
        {
          id: createSentenceId('sent-1'),
          messageId: message.id,
          content: 'First',
          sequence: 0,
          isComplete: true,
        },
        {
          id: createSentenceId('sent-2'),
          messageId: message.id,
          content: 'Second',
          sequence: 1,
          isComplete: true,
        },
      ];

      useConversationStore.getState().addMessage(message);
      sentences.forEach((s) => useConversationStore.getState().addSentence(s));

      const state = useConversationStore.getState();

      // Check forward references (message -> sentences)
      expect(state.messages['msg-1'].sentenceIds).toHaveLength(2);
      expect(state.messages['msg-1'].sentenceIds).toContain(sentences[0].id);
      expect(state.messages['msg-1'].sentenceIds).toContain(sentences[1].id);

      // Check backward references (sentence -> message)
      expect(state.sentences['sent-1'].messageId).toBe(message.id);
      expect(state.sentences['sent-2'].messageId).toBe(message.id);
    });

    it('should maintain tool call bidirectional references', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const toolCalls: ToolCall[] = [
        {
          id: createToolCallId('tool-1'),
          toolName: 'tool1',
          arguments: {},
          messageId: message.id,
          status: 'pending',
          startTimeMs: Date.now(),
        },
        {
          id: createToolCallId('tool-2'),
          toolName: 'tool2',
          arguments: {},
          messageId: message.id,
          status: 'pending',
          startTimeMs: Date.now(),
        },
      ];

      useConversationStore.getState().addMessage(message);
      toolCalls.forEach((tc) => useConversationStore.getState().addToolCall(tc));

      const state = useConversationStore.getState();

      // Check forward references (message -> tool calls)
      expect(state.messages['msg-1'].toolCallIds).toHaveLength(2);
      expect(state.messages['msg-1'].toolCallIds).toContain(toolCalls[0].id);
      expect(state.messages['msg-1'].toolCallIds).toContain(toolCalls[1].id);

      // Check backward references (tool call -> message)
      expect(state.toolCalls['tool-1'].messageId).toBe(message.id);
      expect(state.toolCalls['tool-2'].messageId).toBe(message.id);
    });

    it('should maintain memory trace bidirectional references', () => {
      const message: Message = {
        id: createMessageId('msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const traces: MemoryTrace[] = [
        {
          id: createMemoryTraceId('trace-1'),
          messageId: message.id,
          content: 'Memory 1',
          relevance: 0.9,
        },
        {
          id: createMemoryTraceId('trace-2'),
          messageId: message.id,
          content: 'Memory 2',
          relevance: 0.7,
        },
      ];

      useConversationStore.getState().addMessage(message);
      traces.forEach((t) => useConversationStore.getState().addMemoryTrace(t));

      const state = useConversationStore.getState();

      // Check forward references (message -> memory traces)
      expect(state.messages['msg-1'].memoryTraceIds).toHaveLength(2);
      expect(state.messages['msg-1'].memoryTraceIds).toContain(traces[0].id);
      expect(state.messages['msg-1'].memoryTraceIds).toContain(traces[1].id);

      // Check backward references (memory trace -> message)
      expect(state.memoryTraces['trace-1'].messageId).toBe(message.id);
      expect(state.memoryTraces['trace-2'].messageId).toBe(message.id);
    });
  });
});
