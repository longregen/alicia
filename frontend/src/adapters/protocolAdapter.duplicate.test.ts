import { describe, it, expect, beforeEach, vi } from 'vitest';
import { handleTranscription } from './protocolAdapter';
import { Transcription } from '../types/protocol';
import {
  NormalizedMessage,
  MessageStatus,
  createMessageId,
  createConversationId,
  MessageId,
  ConversationId,
} from '../types/streaming';


// Mock the conversation store
const createMockStore = () => {
  const store = {
    messages: {} as Record<string, NormalizedMessage>,
    currentStreamingMessageId: null as MessageId | null,
    currentConversationId: null as ConversationId | null,

    addMessage: vi.fn((message: NormalizedMessage) => {
      store.messages[message.id] = message;
    }),

    updateMessageStatus: vi.fn((id: MessageId, status: MessageStatus) => {
      if (store.messages[id]) {
        store.messages[id].status = status;
      }
    }),

    updateMessageContent: vi.fn((id: MessageId, content: string) => {
      if (store.messages[id]) {
        store.messages[id].content = content;
      }
    }),

    setCurrentStreamingMessageId: vi.fn((id: MessageId | null) => {
      store.currentStreamingMessageId = id;
    }),

    setCurrentConversationId: vi.fn((id: ConversationId | null) => {
      store.currentConversationId = id;
    }),
  };

  return store;
};

vi.mock('../stores/conversationStore', () => ({
  useConversationStore: {
    getState: vi.fn(),
  },
}));

import { useConversationStore } from '../stores/conversationStore';

describe('protocolAdapter - Duplicate Message Race Condition', () => {
  let mockStore: ReturnType<typeof createMockStore>;

  beforeEach(() => {
    mockStore = createMockStore();
    vi.mocked(useConversationStore.getState).mockReturnValue(mockStore as any);
  });

  it('should NOT create duplicate when user message already exists with different ID (content-based deduplication)', () => {
    const conversationId = createConversationId('conv-123');
    const messageContent = 'Hello, how are you?';

    // STEP 1: Simulate optimistic save from REST API
    // This happens when user sends a message via the UI
    const optimisticMessage: NormalizedMessage = {
      id: createMessageId('msg-optimistic-456'),
      conversationId,
      role: 'user',
      content: messageContent,
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };
    mockStore.messages[optimisticMessage.id] = optimisticMessage;

    // STEP 2: Simulate WebSocket broadcast from server with different ID
    // Server processed the message and broadcasts it to all clients
    const transcription: Transcription = {
      id: 'msg-server-789', // DIFFERENT ID
      conversationId: 'conv-123',
      text: messageContent, // SAME CONTENT
      final: true,
    };

    // STEP 3: Call handleTranscription with the broadcast message
    handleTranscription(transcription, mockStore as any);

    // ASSERTION: Should NOT create a duplicate message
    // Content-based deduplication should prevent the duplicate (lines 449-462)
    const allMessages = Object.values(mockStore.messages);
    const userMessages = allMessages.filter((m) => m.role === 'user');

    expect(userMessages).toHaveLength(1);
    expect(userMessages[0].content).toBe(messageContent);
    expect(userMessages[0].id).toBe(optimisticMessage.id); // Should keep the original
  });

  it('should demonstrate duplicate creation when whitespace differs (fragile deduplication)', () => {
    const conversationId = createConversationId('conv-456');
    const messageContent = 'Test message';

    // STEP 1: Optimistic save with trailing whitespace
    const optimisticMessage: NormalizedMessage = {
      id: createMessageId('msg-optimistic-111'),
      conversationId,
      role: 'user',
      content: messageContent + '  ', // Extra whitespace
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };
    mockStore.messages[optimisticMessage.id] = optimisticMessage;

    // STEP 2: Server broadcast without trailing whitespace
    const transcription: Transcription = {
      id: 'msg-server-222',
      conversationId: 'conv-456',
      text: messageContent, // No extra whitespace
      final: true,
    };

    // STEP 3: Call handleTranscription
    handleTranscription(transcription, mockStore as any);

    // ASSERTION: Content-based deduplication uses .trim() so this should work
    // But if implementation changes or has bugs, this could create duplicates
    const allMessages = Object.values(mockStore.messages);
    const userMessages = allMessages.filter((m) => m.role === 'user');

    // This should still be 1 because of .trim() in the deduplication logic
    expect(userMessages).toHaveLength(1);
  });

  it('should NOT create duplicate when status is streaming (deduplication checks all statuses)', () => {
    const conversationId = createConversationId('conv-789');
    const messageContent = 'Streaming test';

    // STEP 1: Optimistic save with streaming status
    const optimisticMessage: NormalizedMessage = {
      id: createMessageId('msg-optimistic-333'),
      conversationId,
      role: 'user',
      content: messageContent,
      status: MessageStatus.Streaming, // NOT complete
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };
    mockStore.messages[optimisticMessage.id] = optimisticMessage;

    // STEP 2: Server broadcast with final transcription
    const transcription: Transcription = {
      id: 'msg-server-444',
      conversationId: 'conv-789',
      text: messageContent,
      final: true,
    };

    // STEP 3: Call handleTranscription
    handleTranscription(transcription, mockStore as any);

    // ASSERTION: Should NOT create a duplicate - deduplication now checks
    // messages regardless of status to handle race condition
    const allMessages = Object.values(mockStore.messages);
    const userMessages = allMessages.filter((m) => m.role === 'user');

    expect(userMessages).toHaveLength(1); // No duplicate!
    expect(userMessages[0].status).toBe(MessageStatus.Streaming);
  });

  it('should create duplicate when message is from different conversation', () => {
    const conversationId1 = createConversationId('conv-aaa');
    const conversationId2 = createConversationId('conv-bbb');
    const messageContent = 'Same content';

    // STEP 1: Message in conversation 1
    const message1: NormalizedMessage = {
      id: createMessageId('msg-555'),
      conversationId: conversationId1,
      role: 'user',
      content: messageContent,
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };
    mockStore.messages[message1.id] = message1;

    // STEP 2: Broadcast for conversation 2 with same content
    const transcription: Transcription = {
      id: 'msg-666',
      conversationId: 'conv-bbb',
      text: messageContent,
      final: true,
    };

    // STEP 3: Call handleTranscription
    handleTranscription(transcription, mockStore as any);

    // ASSERTION: Should create separate message (correct behavior)
    // Deduplication checks conversationId match (line 454)
    const allMessages = Object.values(mockStore.messages);
    expect(allMessages).toHaveLength(2);
    expect(allMessages[0].conversationId).toBe(conversationId1);
    expect(allMessages[1].conversationId).toBe(conversationId2);
  });

  it('should demonstrate the actual race condition: same conversation, same content, different IDs', () => {
    const conversationId = createConversationId('conv-race');
    const messageContent = 'This is the race condition';

    // SCENARIO: User clicks send button
    // 1. UI optimistically adds message with REST API ID
    const restApiMessage: NormalizedMessage = {
      id: createMessageId('rest-api-message-id'),
      conversationId,
      role: 'user',
      content: messageContent,
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };
    mockStore.addMessage(restApiMessage);

    // 2. Server processes and broadcasts via WebSocket with its own ID
    const wsTranscription: Transcription = {
      id: 'websocket-message-id', // DIFFERENT from REST API ID
      conversationId: 'conv-race',
      text: messageContent,
      final: true,
    };

    // 3. handleTranscription receives WebSocket message
    handleTranscription(wsTranscription, mockStore as any);

    // VERIFICATION: Deduplication should work (content + conversationId + status match)
    const allMessages = Object.values(mockStore.messages);
    const userMessages = allMessages.filter(
      (m) => m.role === 'user' && m.conversationId === conversationId
    );

    // This test documents current behavior:
    // - If deduplication works: 1 message
    // - If deduplication fails: 2 messages (DUPLICATE)

    // Current implementation SHOULD prevent duplicate (lines 449-462)
    expect(userMessages).toHaveLength(1);
    expect(mockStore.addMessage).toHaveBeenCalledTimes(1); // Only the optimistic add
  });


  it('should correctly handle race condition when optimistic message is not yet marked complete', () => {
    const conversationId = createConversationId('conv-race-critical');
    const messageContent = 'User message that will race';

    // REAL SCENARIO:
    // 1. User clicks send
    // 2. REST API call is made to save message
    // 3. UI optimistically adds message to store with status=Streaming (not Complete yet)
    // 4. Server processes and broadcasts via WebSocket BEFORE REST API responds
    // 5. WebSocket message arrives with final=true

    // STEP 1: Optimistic save - might still be streaming while waiting for REST response
    const optimisticMessage: NormalizedMessage = {
      id: createMessageId('rest-id-123'),
      conversationId,
      role: 'user',
      content: messageContent,
      status: MessageStatus.Streaming, // Still waiting for confirmation!
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };
    mockStore.messages[optimisticMessage.id] = optimisticMessage;

    // STEP 2: WebSocket broadcast arrives BEFORE REST API completes
    const wsTranscription: Transcription = {
      id: 'ws-broadcast-456', // Different ID from REST
      conversationId: 'conv-race-critical',
      text: messageContent,
      final: true,
    };

    handleTranscription(wsTranscription, mockStore as any);

    // FIX VERIFIED: Deduplication now checks regardless of status
    // Since optimistic message has same content, duplicate is prevented!
    const allMessages = Object.values(mockStore.messages);
    const userMessages = allMessages.filter((m) => m.role === 'user');

    // No duplicate created - race condition is handled
    expect(userMessages).toHaveLength(1);
    expect(userMessages[0].status).toBe(MessageStatus.Streaming);
    expect(userMessages[0].content).toBe(messageContent);
  });

  it('should prevent duplicate when optimistic message is Complete', () => {
    const conversationId = createConversationId('conv-works');
    const messageContent = 'Message that deduplicates correctly';

    // STEP 1: Optimistic save already marked Complete
    const optimisticMessage: NormalizedMessage = {
      id: createMessageId('rest-complete-789'),
      conversationId,
      role: 'user',
      content: messageContent,
      status: MessageStatus.Complete, // Already complete
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };
    mockStore.messages[optimisticMessage.id] = optimisticMessage;

    // STEP 2: WebSocket broadcast
    const wsTranscription: Transcription = {
      id: 'ws-complete-999',
      conversationId: 'conv-works',
      text: messageContent,
      final: true,
    };

    handleTranscription(wsTranscription, mockStore as any);

    // This SHOULD work - deduplication catches it
    const allMessages = Object.values(mockStore.messages);
    const userMessages = allMessages.filter((m) => m.role === 'user');

    expect(userMessages).toHaveLength(1); // No duplicate!
  });
});
