import { renderHook } from '@testing-library/react';
import { describe, it, expect, beforeEach } from 'vitest';
import { useFeedbackAggregates } from './useFeedbackAggregates';
import { useFeedbackStore } from '../stores/feedbackStore';
import { useConversationStore } from '../stores/conversationStore';
import {
  createMessageId,
  createToolCallId,
  createMemoryTraceId,
  createConversationId,
  MessageStatus,
  type Message,
  type ToolCall,
  type MemoryTrace,
} from '../types/streaming';

describe('useFeedbackAggregates', () => {
  beforeEach(() => {
    // Reset stores before each test
    useFeedbackStore.getState().clearFeedback();
    useConversationStore.getState().clearConversation();
  });

  it('should return neutral sentiment with no votes', () => {
    const messageId = createMessageId('msg-1');

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Test message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);

    const { result } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.sentiment).toBe('neutral');
    expect(result.current.totalVotes).toBe(0);
    expect(result.current.totalPositive).toBe(0);
    expect(result.current.totalNegative).toBe(0);
  });

  it('should aggregate tool use votes', () => {
    const messageId = createMessageId('msg-1');
    const toolCall1: ToolCall = {
      id: createToolCallId('tool-1'),
      toolName: 'test-tool',
      arguments: {},
      messageId,
      status: 'success',
      startTimeMs: Date.now(),
      endTimeMs: Date.now(),
      resultContent: 'result',
    };
    const toolCall2: ToolCall = {
      id: createToolCallId('tool-2'),
      toolName: 'test-tool-2',
      arguments: {},
      messageId,
      status: 'success',
      startTimeMs: Date.now(),
      endTimeMs: Date.now(),
      resultContent: 'result2',
    };

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Test message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);
    useConversationStore.getState().addToolCall(toolCall1);
    useConversationStore.getState().addToolCall(toolCall2);

    // Add votes
    useFeedbackStore.getState().setAggregates('tool_use', 'tool-1', {
      upvotes: 3,
      downvotes: 1,
      special: {},
    });
    useFeedbackStore.getState().setAggregates('tool_use', 'tool-2', {
      upvotes: 2,
      downvotes: 0,
      special: {},
    });

    const { result } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.toolUseFeedback.upvotes).toBe(5);
    expect(result.current.toolUseFeedback.downvotes).toBe(1);
    expect(result.current.totalPositive).toBe(5);
    expect(result.current.totalNegative).toBe(1);
    expect(result.current.totalVotes).toBe(6);
  });

  it('should aggregate memory trace votes', () => {
    const messageId = createMessageId('msg-1');
    const memoryTrace1: MemoryTrace = {
      id: createMemoryTraceId('trace-1'),
      messageId,
      content: 'Memory 1',
      relevance: 0.9,
    };
    const memoryTrace2: MemoryTrace = {
      id: createMemoryTraceId('trace-2'),
      messageId,
      content: 'Memory 2',
      relevance: 0.8,
    };

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Test message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);
    useConversationStore.getState().addMemoryTrace(memoryTrace1);
    useConversationStore.getState().addMemoryTrace(memoryTrace2);

    // Add votes
    useFeedbackStore.getState().setAggregates('memory', 'trace-1', {
      upvotes: 2,
      downvotes: 0,
      special: {},
    });
    useFeedbackStore.getState().setAggregates('memory', 'trace-2', {
      upvotes: 1,
      downvotes: 1,
      special: {},
    });

    const { result } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.memoryFeedback.upvotes).toBe(3);
    expect(result.current.memoryFeedback.downvotes).toBe(1);
    expect(result.current.totalPositive).toBe(3);
    expect(result.current.totalNegative).toBe(1);
  });

  it('should calculate positive sentiment (>= 80% positive)', () => {
    const messageId = createMessageId('msg-1');
    const toolCall: ToolCall = {
      id: createToolCallId('tool-1'),
      toolName: 'test-tool',
      arguments: {},
      messageId,
      status: 'success',
      startTimeMs: Date.now(),
      endTimeMs: Date.now(),
      resultContent: 'result',
    };

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Test message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);
    useConversationStore.getState().addToolCall(toolCall);

    // 8 upvotes, 2 downvotes = 80% positive
    useFeedbackStore.getState().setAggregates('tool_use', 'tool-1', {
      upvotes: 8,
      downvotes: 2,
      special: {},
    });

    const { result } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.sentiment).toBe('positive');
  });

  it('should calculate mixed sentiment (40-79% positive)', () => {
    const messageId = createMessageId('msg-1');
    const toolCall: ToolCall = {
      id: createToolCallId('tool-1'),
      toolName: 'test-tool',
      arguments: {},
      messageId,
      status: 'success',
      startTimeMs: Date.now(),
      endTimeMs: Date.now(),
      resultContent: 'result',
    };

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Test message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);
    useConversationStore.getState().addToolCall(toolCall);

    // 5 upvotes, 5 downvotes = 50% positive
    useFeedbackStore.getState().setAggregates('tool_use', 'tool-1', {
      upvotes: 5,
      downvotes: 5,
      special: {},
    });

    const { result } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.sentiment).toBe('mixed');
  });

  it('should calculate negative sentiment (< 40% positive)', () => {
    const messageId = createMessageId('msg-1');
    const toolCall: ToolCall = {
      id: createToolCallId('tool-1'),
      toolName: 'test-tool',
      arguments: {},
      messageId,
      status: 'success',
      startTimeMs: Date.now(),
      endTimeMs: Date.now(),
      resultContent: 'result',
    };

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Test message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);
    useConversationStore.getState().addToolCall(toolCall);

    // 2 upvotes, 8 downvotes = 20% positive
    useFeedbackStore.getState().setAggregates('tool_use', 'tool-1', {
      upvotes: 2,
      downvotes: 8,
      special: {},
    });

    const { result } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.sentiment).toBe('negative');
  });

  it('should include critical votes in positive count', () => {
    const messageId = createMessageId('msg-1');
    const toolCall: ToolCall = {
      id: createToolCallId('tool-1'),
      toolName: 'test-tool',
      arguments: {},
      messageId,
      status: 'success',
      startTimeMs: Date.now(),
      endTimeMs: Date.now(),
      resultContent: 'result',
    };

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Test message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);
    useConversationStore.getState().addToolCall(toolCall);

    // 2 upvotes, 3 critical, 0 downvotes
    useFeedbackStore.getState().setAggregates('tool_use', 'tool-1', {
      upvotes: 2,
      downvotes: 0,
      special: { critical: 3 },
    });

    const { result } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.toolUseFeedback.critical).toBe(3);
    expect(result.current.totalPositive).toBe(5); // 2 upvotes + 3 critical
    expect(result.current.totalVotes).toBe(5);
    expect(result.current.sentiment).toBe('positive');
  });

  it('should aggregate votes from multiple tool calls and memories', () => {
    const messageId = createMessageId('msg-1');
    const toolCall: ToolCall = {
      id: createToolCallId('tool-1'),
      toolName: 'test-tool',
      arguments: {},
      messageId,
      status: 'success',
      startTimeMs: Date.now(),
      endTimeMs: Date.now(),
      resultContent: 'result',
    };
    const memoryTrace: MemoryTrace = {
      id: createMemoryTraceId('trace-1'),
      messageId,
      content: 'Memory',
      relevance: 0.9,
    };

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Test message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);
    useConversationStore.getState().addToolCall(toolCall);
    useConversationStore.getState().addMemoryTrace(memoryTrace);

    // Add votes
    useFeedbackStore.getState().setAggregates('tool_use', 'tool-1', {
      upvotes: 3,
      downvotes: 1,
      special: {},
    });
    useFeedbackStore.getState().setAggregates('memory', 'trace-1', {
      upvotes: 2,
      downvotes: 0,
      special: {},
    });

    const { result } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.totalPositive).toBe(5);
    expect(result.current.totalNegative).toBe(1);
    expect(result.current.totalVotes).toBe(6);
  });

  it('should handle message with no tool calls or memories', () => {
    const messageId = createMessageId('msg-1');

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Simple message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);

    const { result } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.toolUseFeedback).toEqual({ upvotes: 0, downvotes: 0, critical: 0 });
    expect(result.current.memoryFeedback).toEqual({ upvotes: 0, downvotes: 0, critical: 0 });
    expect(result.current.reasoningFeedback).toEqual({ upvotes: 0, downvotes: 0, critical: 0 });
    expect(result.current.sentiment).toBe('neutral');
  });

  it('should recalculate when votes change', () => {
    const messageId = createMessageId('msg-1');
    const toolCall: ToolCall = {
      id: createToolCallId('tool-1'),
      toolName: 'test-tool',
      arguments: {},
      messageId,
      status: 'success',
      startTimeMs: Date.now(),
      endTimeMs: Date.now(),
      resultContent: 'result',
    };

    const message: Message = {
      id: messageId,
      conversationId: createConversationId('conv-1'),
      role: 'assistant',
      content: 'Test message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    useConversationStore.getState().addMessage(message);
    useConversationStore.getState().addToolCall(toolCall);

    useFeedbackStore.getState().setAggregates('tool_use', 'tool-1', {
      upvotes: 1,
      downvotes: 0,
      special: {},
    });

    const { result, rerender } = renderHook(() => useFeedbackAggregates(messageId));

    expect(result.current.totalVotes).toBe(1);
    expect(result.current.sentiment).toBe('positive');

    // Update votes
    useFeedbackStore.getState().setAggregates('tool_use', 'tool-1', {
      upvotes: 1,
      downvotes: 9,
      special: {},
    });

    rerender();

    expect(result.current.totalVotes).toBe(10);
    expect(result.current.sentiment).toBe('negative');
  });
});
