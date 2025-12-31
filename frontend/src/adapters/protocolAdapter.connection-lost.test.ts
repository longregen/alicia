import { describe, it, expect, beforeEach, vi } from 'vitest';
import { handleStartAnswer, handleConnectionLost } from './protocolAdapter';
import {
  createMessageId,
  MessageStatus,
} from '../types/streaming';

// Mock the messageRepository to avoid database initialization
vi.mock('../db/repository', () => ({
  messageRepository: {
    upsert: vi.fn(),
    findById: vi.fn(),
  },
}));

// Mock the conversation store
vi.mock('../stores/conversationStore', () => {
  const mockStore = {
    messages: {} as Record<string, any>,
    currentStreamingMessageId: null as string | null,
    currentConversationId: null as string | null,
    addMessage: vi.fn((message: any) => {
      mockStore.messages[message.id] = message;
    }),
    updateMessageStatus: vi.fn((id: string, status: any) => {
      if (mockStore.messages[id]) {
        mockStore.messages[id].status = status;
      }
    }),
    setCurrentStreamingMessageId: vi.fn((id: string | null) => {
      mockStore.currentStreamingMessageId = id;
    }),
    setCurrentConversationId: vi.fn(),
    updateMessageContent: vi.fn(),
    addSentence: vi.fn(),
    updateSentence: vi.fn(),
    getMessageSentences: vi.fn(() => []),
    addToolCall: vi.fn(),
    updateToolCall: vi.fn(),
    addAudioRef: vi.fn(),
    addMemoryTrace: vi.fn(),
  };

  return {
    useConversationStore: {
      getState: () => mockStore,
    },
  };
});

import { useConversationStore } from '../stores/conversationStore';

describe('handleConnectionLost', () => {
  beforeEach(() => {
    const store = useConversationStore.getState();
    // Clear store state
    store.messages = {};
    store.currentStreamingMessageId = null;
    store.currentConversationId = null;
    vi.clearAllMocks();
  });

  it('clears currentStreamingMessageId and marks message as error when connection is lost during streaming', () => {
    const store = useConversationStore.getState();
    const messageId = createMessageId('msg-001');

    // Start streaming (this will use the mocked store)
    handleStartAnswer(
      {
        id: 'msg-001',
        conversationId: 'conv-001',
        previousId: '',
      },
      store as any
    );

    // Verify message is streaming
    expect(store.currentStreamingMessageId).toBe(messageId);
    const message = store.messages[messageId];
    expect(message.status).toBe(MessageStatus.Streaming);

    // Simulate connection loss
    handleConnectionLost();

    // Verify streaming state is cleared
    expect(store.setCurrentStreamingMessageId).toHaveBeenLastCalledWith(null);

    // Verify message is marked as error
    expect(store.updateMessageStatus).toHaveBeenCalledWith(messageId, MessageStatus.Error);
  });

  it('does nothing when no streaming message exists', () => {
    const store = useConversationStore.getState();

    // Call handleConnectionLost without any streaming message
    handleConnectionLost();

    // Should not call any store methods
    expect(store.setCurrentStreamingMessageId).not.toHaveBeenCalled();
    expect(store.updateMessageStatus).not.toHaveBeenCalled();
  });

  it('does nothing when streaming message is already complete', () => {
    const store = useConversationStore.getState();
    const messageId = createMessageId('msg-001');

    // Start streaming
    handleStartAnswer(
      {
        id: 'msg-001',
        conversationId: 'conv-001',
        previousId: '',
      },
      store as any
    );

    // Mark as complete (simulating isFinal=true)
    store.messages[messageId].status = MessageStatus.Complete;
    store.currentStreamingMessageId = null;

    vi.clearAllMocks();

    // Call handleConnectionLost
    handleConnectionLost();

    // Should not update anything since message is already complete
    expect(store.updateMessageStatus).not.toHaveBeenCalled();
  });
});
